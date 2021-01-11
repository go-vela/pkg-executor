// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/drone/envsubst"

	"github.com/go-vela/pkg-executor/internal/service"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// CreateService configures the service for execution.
func (c *client) CreateService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	ctn.Environment["BUILD_HOST"] = c.Hostname
	ctn.Environment["VELA_HOST"] = c.Hostname
	ctn.Environment["VELA_VERSION"] = "v0.6.0"
	// TODO: remove hardcoded reference
	ctn.Environment["VELA_RUNTIME"] = "docker"
	ctn.Environment["VELA_DISTRIBUTION"] = "linux"

	logger.Debug("setting up container")
	// setup the runtime container
	err := c.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	logger.Debug("injecting secrets")
	// inject secrets for container
	err = injectSecrets(ctn, c.Secrets)
	if err != nil {
		return err
	}

	logger.Debug("marshaling configuration")
	// marshal container configuration
	body, err := json.Marshal(ctn)
	if err != nil {
		return fmt.Errorf("unable to marshal configuration: %v", err)
	}

	// create substitute function
	subFunc := func(name string) string {
		env := ctn.Environment[name]
		if strings.Contains(env, "\n") {
			env = fmt.Sprintf("%q", env)
		}

		return env
	}

	logger.Debug("substituting environment")
	// substitute the environment variables
	//
	// https://pkg.go.dev/github.com/drone/envsubst?tab=doc#Eval
	subStep, err := envsubst.Eval(string(body), subFunc)
	if err != nil {
		return fmt.Errorf("unable to substitute environment variables: %v", err)
	}

	logger.Debug("unmarshaling configuration")
	// unmarshal container configuration
	err = json.Unmarshal([]byte(subStep), ctn)
	if err != nil {
		return fmt.Errorf("unable to unmarshal configuration: %v", err)
	}

	return nil
}

// PlanService prepares the service for execution.
func (c *client) PlanService(ctx context.Context, ctn *pipeline.Container) error {
	var err error

	b := c.build
	r := c.repo

	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	// update the engine service object
	s := new(library.Service)
	s.SetName(ctn.Name)
	s.SetNumber(ctn.Number)
	s.SetStatus(constants.StatusRunning)
	s.SetStarted(time.Now().UTC().Unix())
	s.SetHost(ctn.Environment["VELA_HOST"])
	s.SetRuntime(ctn.Environment["VELA_RUNTIME"])
	s.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])

	logger.Debug("uploading service state")
	// send API call to update the service
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SvcService.Update
	s, _, err = c.Vela.Svc.Update(r.GetOrg(), r.GetName(), b.GetNumber(), s)
	if err != nil {
		return err
	}

	s.SetStatus(constants.StatusSuccess)

	// add a service to a map
	c.services.Store(ctn.ID, s)

	// get the service log here
	logger.Debug("retrieve service log")
	// send API call to capture the service log
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.GetService
	l, _, err := c.Vela.Log.GetService(r.GetOrg(), r.GetName(), b.GetNumber(), s.GetNumber())
	if err != nil {
		return err
	}

	// add a service log to a map
	c.serviceLogs.Store(ctn.ID, l)

	return nil
}

// ExecService runs a service.
func (c *client) ExecService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	logger.Debug("running container")
	// run the runtime container
	err := c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		logger.Debug("streaming logs for container")
		// stream logs from container
		err := c.StreamService(ctx, ctn)
		if err != nil {
			logger.Error(err)
		}
	}()

	return nil
}

// StreamService tails the output for a service.
func (c *client) StreamService(ctx context.Context, ctn *pipeline.Container) error {
	b := c.build
	r := c.repo

	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	// load the logs for the service from the client
	l, err := service.LoadLogs(ctn, &c.serviceLogs)
	if err != nil {
		return err
	}

	// create new buffer for uploading logs
	logs := new(bytes.Buffer)

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
		l.SetData(data)

		logger.Debug("uploading logs")
		// send API call to update the logs for the service
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateService
		_, _, err = c.Vela.Log.UpdateService(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
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

	// create new scanner from the container output
	scanner := bufio.NewScanner(rc)

	// scan entire container output
	for scanner.Scan() {
		// write all the logs from the scanner
		logs.Write(append(scanner.Bytes(), []byte("\n")...))

		// if we have at least 1000 bytes in our buffer
		if logs.Len() > 1000 {
			logger.Trace(logs.String())

			// update the existing log with the new bytes
			//
			// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
			l.AppendData(logs.Bytes())

			logger.Debug("appending logs")
			// send API call to append the logs for the service
			//
			// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateService
			l, _, err = c.Vela.Log.UpdateService(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
			if err != nil {
				return err
			}

			// flush the buffer of logs
			logs.Reset()
		}
	}

	return scanner.Err()
}

// DestroyService cleans up services after execution.
func (c *client) DestroyService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	// load the service from the client
	s, err := service.Load(ctn, &c.services)
	if err != nil {
		// create the service from the container
		s = new(library.Service)
		s.SetName(ctn.Name)
		s.SetNumber(ctn.Number)
		s.SetStatus(constants.StatusPending)
		s.SetHost(ctn.Environment["VELA_HOST"])
		s.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		s.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	defer func() {
		logger.Info("uploading service snapshot")
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SvcService.Update
		_, _, err := c.Vela.Svc.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), s)
		if err != nil {
			logger.Errorf("unable to upload service snapshot: %v", err)
		}
	}()

	// check if the service is in a pending state
	if s.GetStatus() == constants.StatusPending {
		// update the service fields
		//
		// TODO: consider making this a constant
		//
		// nolint: gomnd // ignore magic number 137
		s.SetExitCode(137)
		s.SetFinished(time.Now().UTC().Unix())
		s.SetStatus(constants.StatusKilled)

		// check if the service was not started
		if s.GetStarted() == 0 {
			// set the started time to the finished time
			s.SetStarted(s.GetFinished())
		}
	}

	logger.Debug("inspecting container")
	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the service finished
	if s.GetFinished() == 0 {
		// update the service fields
		s.SetFinished(time.Now().UTC().Unix())
		s.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the service fields
			s.SetExitCode(ctn.ExitCode)
			s.SetStatus(constants.StatusFailure)
		}
	}

	logger.Debug("removing container")
	// remove the runtime container
	err = c.Runtime.RemoveContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}
