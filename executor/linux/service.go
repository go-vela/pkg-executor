// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/go-vela/pkg-executor/internal/service"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"

	"github.com/gorilla/websocket"
)

// CreateService configures the service for execution.
func (c *client) CreateService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	logger.Debug("setting up container")
	// setup the runtime container
	err := c.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// update the service container environment
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#Environment
	err = service.Environment(ctn, c.build, c.repo, nil, c.Version)
	if err != nil {
		return err
	}

	logger.Debug("injecting secrets")
	// inject secrets for container
	err = injectSecrets(ctn, c.Secrets)
	if err != nil {
		return err
	}

	logger.Debug("substituting container configuration")
	// substitute container configuration
	//
	// https://pkg.go.dev/github.com/go-vela/types/pipeline#Container.Substitute
	err = ctn.Substitute()
	if err != nil {
		return fmt.Errorf("unable to substitute container configuration")
	}

	return nil
}

// PlanService prepares the service for execution.
func (c *client) PlanService(ctx context.Context, ctn *pipeline.Container) error {
	var err error

	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	// create the library service object
	_service := new(library.Service)
	_service.SetName(ctn.Name)
	_service.SetNumber(ctn.Number)
	_service.SetImage(ctn.Image)
	_service.SetStatus(constants.StatusRunning)
	_service.SetStarted(time.Now().UTC().Unix())
	_service.SetHost(c.build.GetHost())
	_service.SetRuntime(c.build.GetRuntime())
	_service.SetDistribution(c.build.GetDistribution())

	logger.Debug("uploading service state")
	// send API call to update the service
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SvcService.Update
	_service, _, err = c.Vela.Svc.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), _service)
	if err != nil {
		return err
	}

	// update the service container environment
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#Environment
	err = service.Environment(ctn, c.build, c.repo, _service, c.Version)
	if err != nil {
		return err
	}

	// add a service to a map
	c.services.Store(ctn.ID, _service)

	// get the service log here
	logger.Debug("retrieve service log")
	// send API call to capture the service log
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.GetService
	_log, _, err := c.Vela.Log.GetService(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), _service.GetNumber())
	if err != nil {
		return err
	}

	// add a service log to a map
	c.serviceLogs.Store(ctn.ID, _log)

	return nil
}

// ExecService runs a service.
func (c *client) ExecService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	// load the service from the client
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#Load
	_service, err := service.Load(ctn, &c.services)
	if err != nil {
		return err
	}

	// defer taking a snapshot of the service
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#Snapshot
	defer func() { service.Snapshot(ctn, c.build, c.Vela, c.logger, c.repo, _service) }()

	logger.Debug("running container")
	// run the runtime container
	err = c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		logger.Debug("streaming logs for container")
		// stream logs from container
		err := c.StreamService(context.Background(), ctn)
		if err != nil {
			logger.Error(err)
		}
	}()

	return nil
}

// StreamService tails the output for a service.
func (c *client) StreamService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	// load the logs for the service from the client
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#LoadLogs
	_log, err := service.LoadLogs(ctn, &c.serviceLogs)
	if err != nil {
		return err
	}

	// nolint: dupl // ignore similar code
	defer func() {
		// tail the runtime container
		rc, err := c.Runtime.TailContainer(ctx, ctn)
		if err != nil {
			logger.Errorf("unable to tail container output for upload: %v", err)

			return
		}
		defer rc.Close()

		// read all output from the runtime container
		data, err := ioutil.ReadAll(rc)
		if err != nil {
			logger.Errorf("unable to read container output for upload: %v", err)

			return
		}

		// overwrite the existing log with all bytes
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.SetData
		_log.SetData(data)

		logger.Debug("uploading logs")
		// send API call to update the logs for the service
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateService
		_, _, err = c.Vela.Log.UpdateService(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), ctn.Number, _log)
		if err != nil {
			logger.Errorf("unable to upload container logs: %v", err)
		}
	}()

	logger.Debug("tailing container")
	// tail the runtime container
	rc, err := c.Runtime.TailContainer(ctx, ctn)
	if err != nil {
		return err
	}
	defer rc.Close()

	// TODO: consider moving most (all?) of this into the Vela Go SDK
	url := fmt.Sprintf(
		"ws://server:8080/api/v1/repos/%s/%s/builds/%d/services/%d/stream",
		c.repo.GetOrg(),
		c.repo.GetName(),
		c.build.GetNumber(),
		ctn.Number,
	)

	headers := http.Header{}
	headers.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("VELA_SERVER_SECRET")))

	logger.Debugf("creating websocket connection to %s", url)
	// create a connection to the url to stream logs
	//
	// https://pkg.go.dev/github.com/gorilla/websocket#Dialer.Dial
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	// create new scanner from the container output
	scanner := bufio.NewScanner(rc)

	logger.Debug("scanning container logs")
	// scan entire container output
	for scanner.Scan() {
		logger.Trace(scanner.Text())

		// set timeout of 10s to send the logs
		//
		// https://pkg.go.dev/github.com/gorilla/websocket#Conn.SetWriteDeadline
		err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			return err
		}

		// send call to update the logs for the service
		//
		// https://pkg.go.dev/github.com/gorilla/websocket#Conn.WriteMessage
		err = conn.WriteMessage(websocket.TextMessage, scanner.Bytes())
		if err != nil {
			return err
		}
	}

	// create close message for the websocket connection
	closeMessage := websocket.FormatCloseMessage(
		websocket.CloseNormalClosure, "finished scanning container logs",
	)

	logger.Debugf("closing websocket connection to %s", url)
	// send call to close the connection
	//
	// https://pkg.go.dev/github.com/gorilla/websocket#Conn.WriteMessage
	err = conn.WriteMessage(websocket.CloseMessage, closeMessage)
	if err != nil {
		return err
	}

	// END TODO

	return scanner.Err()
}

// DestroyService cleans up services after execution.
func (c *client) DestroyService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	// load the service from the client
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#Load
	_service, err := service.Load(ctn, &c.services)
	if err != nil {
		// create the service from the container
		//
		// https://pkg.go.dev/github.com/go-vela/types/library#ServiceFromContainer
		_service = library.ServiceFromContainer(ctn)
	}

	// defer an upload of the service
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#LoaUploadd
	defer func() { service.Upload(ctn, c.build, c.Vela, logger, c.repo, _service) }()

	logger.Debug("inspecting container")
	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	logger.Debug("removing container")
	// remove the runtime container
	err = c.Runtime.RemoveContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}
