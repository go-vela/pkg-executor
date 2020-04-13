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
	"strings"
	"time"

	"github.com/drone/envsubst"

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
	ctn.Environment["VELA_VERSION"] = "v0.4.0"
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

	// TODO: add these to the library.Service
	//
	// s.SetHost(ctn.Environment["VELA_HOST"])
	// s.SetRuntime(ctn.Environment["VELA_RUNTIME"])
	// s.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])

	logger.Debug("uploading service state")
	// send API call to update the service
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
		logger.Debug("stream logs for container")
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
	l, err := c.loadServiceLogs(ctn.ID)
	if err != nil {
		return err
	}

	// create new buffer for uploading logs
	logs := new(bytes.Buffer)

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
			l, _, err = c.Vela.Log.UpdateService(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
			if err != nil {
				return err
			}

			// flush the buffer of logs
			logs.Reset()
		}
	}
	logger.Trace(logs.String())

	// update the existing log with the last bytes
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData(logs.Bytes())

	logger.Debug("uploading logs")
	// send API call to update the logs for the service
	_, _, err = c.Vela.Log.UpdateService(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
	if err != nil {
		return err
	}

	return nil
}

// DestroyService cleans up services after execution.
func (c *client) DestroyService(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with service metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("service", ctn.Name)

	logger.Debug("inspecting container")
	// inspect the runtime container
	err := c.Runtime.InspectContainer(ctx, ctn)
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

// loadService is a helper function to capture
// a service from the client.
func (c *client) loadService(name string) (*library.Service, error) {
	result, ok := c.services.Load(name)
	if !ok {
		return nil, fmt.Errorf("unable to load service %s", name)
	}

	s, ok := result.(*library.Service)
	if !ok {
		return nil, fmt.Errorf("service %s had unexpected value", name)
	}

	return s, nil
}

// loadServiceLog is a helper function to capture
// the logs for a service from the client.
func (c *client) loadServiceLogs(name string) (*library.Log, error) {
	result, ok := c.serviceLogs.Load(name)
	if !ok {
		return nil, fmt.Errorf("unable to load logs for service %s", name)
	}

	l, ok := result.(*library.Log)
	if !ok {
		return nil, fmt.Errorf("logs for service %s had unexpected value", name)
	}

	return l, nil
}
