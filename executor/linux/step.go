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

	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"

	"github.com/gorilla/websocket"
)

// CreateStep configures the step for execution.
func (c *client) CreateStep(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	logger.Debug("setting up container")
	// setup the runtime container
	err := c.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// update the step container environment
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Environment
	err = step.Environment(ctn, c.build, c.repo, nil, c.Version)
	if err != nil {
		return err
	}

	logger.Debug("escaping newlines in secrets")
	escapeNewlineSecrets(c.Secrets)

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

// PlanStep prepares the step for execution.
func (c *client) PlanStep(ctx context.Context, ctn *pipeline.Container) error {
	var err error

	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	// create the library step object
	_step := new(library.Step)
	_step.SetName(ctn.Name)
	_step.SetNumber(ctn.Number)
	_step.SetImage(ctn.Image)
	_step.SetStage(ctn.Environment["VELA_STEP_STAGE"])
	_step.SetStatus(constants.StatusRunning)
	_step.SetStarted(time.Now().UTC().Unix())
	_step.SetHost(c.build.GetHost())
	_step.SetRuntime(c.build.GetRuntime())
	_step.SetDistribution(c.build.GetDistribution())

	logger.Debug("uploading step state")
	// send API call to update the step
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
	_step, _, err = c.Vela.Step.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), _step)
	if err != nil {
		return err
	}

	// update the step container environment
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Environment
	err = step.Environment(ctn, c.build, c.repo, _step, c.Version)
	if err != nil {
		return err
	}

	// add a step to a map
	c.steps.Store(ctn.ID, _step)

	// get the step log here
	logger.Debug("retrieve step log")
	// send API call to capture the step log
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.GetStep
	_log, _, err := c.Vela.Log.GetStep(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), _step.GetNumber())
	if err != nil {
		return err
	}

	// add a step log to a map
	c.stepLogs.Store(ctn.ID, _log)

	return nil
}

// ExecStep runs a step.
func (c *client) ExecStep(ctx context.Context, ctn *pipeline.Container) error {
	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	// load the step from the client
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Load
	_step, err := step.Load(ctn, &c.steps)
	if err != nil {
		return err
	}

	// defer taking a snapshot of the step
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Snapshot
	defer func() { step.Snapshot(ctn, c.build, c.Vela, c.logger, c.repo, _step) }()

	logger.Debug("running container")
	// run the runtime container
	err = c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		logger.Debug("streaming logs for container")
		// stream logs from container
		err := c.StreamStep(context.Background(), ctn)
		if err != nil {
			logger.Error(err)
		}
	}()

	// do not wait for detached containers
	if ctn.Detach {
		return nil
	}

	logger.Debug("waiting for container")
	// wait for the runtime container
	err = c.Runtime.WaitContainer(ctx, ctn)
	if err != nil {
		return err
	}

	logger.Debug("inspecting container")
	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}

// StreamStep tails the output for a step.
func (c *client) StreamStep(ctx context.Context, ctn *pipeline.Container) error {
	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	// load the logs for the step from the client
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#LoadLogs
	_log, err := step.LoadLogs(ctn, &c.stepLogs)
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
		// send API call to update the logs for the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
		_, _, err = c.Vela.Log.UpdateStep(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), ctn.Number, _log)
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
		"ws://server:8080/api/v1/repos/%s/%s/builds/%d/steps/%d/stream",
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
	// send call to close the websocket connection
	//
	// https://pkg.go.dev/github.com/gorilla/websocket#Conn.WriteMessage
	err = conn.WriteMessage(websocket.CloseMessage, closeMessage)
	if err != nil {
		return err
	}

	// END TODO

	return scanner.Err()
}

// DestroyStep cleans up steps after execution.
func (c *client) DestroyStep(ctx context.Context, ctn *pipeline.Container) error {
	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	// load the step from the client
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Load
	_step, err := step.Load(ctn, &c.steps)
	if err != nil {
		// create the step from the container
		//
		// https://pkg.go.dev/github.com/go-vela/types/library#StepFromContainer
		_step = library.StepFromContainer(ctn)
	}

	// defer an upload of the step
	//
	// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Upload
	defer func() { step.Upload(ctn, c.build, c.Vela, logger, c.repo, _step) }()

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
