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

// CreateStep configures the step for execution.
func (c *client) CreateStep(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	ctn.Environment["BUILD_HOST"] = c.Hostname
	ctn.Environment["VELA_HOST"] = c.Hostname
	ctn.Environment["VELA_VERSION"] = "v0.4.0"
	// TODO: remove hardcoded reference
	ctn.Environment["VELA_RUNTIME"] = "docker"
	ctn.Environment["VELA_DISTRIBUTION"] = "linux"

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

// PlanStep prepares the step for execution.
func (c *client) PlanStep(ctx context.Context, ctn *pipeline.Container) error {
	var err error

	b := c.build
	r := c.repo

	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	// update the engine step object
	s := new(library.Step)
	s.SetName(ctn.Name)
	s.SetNumber(ctn.Number)
	s.SetStatus(constants.StatusRunning)
	s.SetStarted(time.Now().UTC().Unix())
	s.SetHost(ctn.Environment["VELA_HOST"])
	s.SetRuntime(ctn.Environment["VELA_RUNTIME"])
	s.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])

	logger.Debug("uploading step state")
	// send API call to update the step
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
	s, _, err = c.Vela.Step.Update(r.GetOrg(), r.GetName(), b.GetNumber(), s)
	if err != nil {
		return err
	}

	s.SetStatus(constants.StatusSuccess)

	// add a step to a map
	c.steps.Store(ctn.ID, s)

	// get the step log here
	logger.Debug("retrieve step log")
	// send API call to capture the step log
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.GetStep
	l, _, err := c.Vela.Log.GetStep(r.GetOrg(), r.GetName(), b.GetNumber(), s.GetNumber())
	if err != nil {
		return err
	}

	// add a step log to a map
	c.stepLogs.Store(ctn.ID, l)

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

	logger.Debug("running container")
	// run the runtime container
	err := c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		logger.Debug("stream logs for container")
		// stream logs from container
		err := c.StreamStep(ctx, ctn)
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

	b := c.build
	r := c.repo

	// update engine logger with step metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("step", ctn.Name)

	// load the logs for the step from the client
	l, err := c.loadStepLogs(ctn.ID)
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
			// send API call to append the logs for the step
			//
			// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
			l, _, err = c.Vela.Log.UpdateStep(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
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
	// send API call to update the logs for the step
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
	_, _, err = c.Vela.Log.UpdateStep(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
	if err != nil {
		return err
	}

	return nil
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
	step, err := c.loadStep(ctn.ID)
	if err != nil {
		// create the step from the container
		step = new(library.Step)
		step.SetName(ctn.Name)
		step.SetNumber(ctn.Number)
		step.SetStatus(constants.StatusPending)
		step.SetHost(ctn.Environment["VELA_HOST"])
		step.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		step.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	defer func() {
		logger.Info("uploading step snapshot")
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err := c.Vela.Step.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), step)
		if err != nil {
			logger.Errorf("unable to upload step snapshot: %v", err)
		}
	}()

	// check if the step is in a pending state
	if step.GetStatus() == constants.StatusPending {
		// update the step fields
		step.SetExitCode(137)
		step.SetFinished(time.Now().UTC().Unix())
		step.SetStatus(constants.StatusKilled)

		// check if the step was not started
		if step.GetStarted() == 0 {
			// set the started time to the finished time
			step.SetStarted(step.GetFinished())
		}
	}

	logger.Debug("inspecting container")
	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the step finished
	if step.GetFinished() == 0 {
		// update the step fields
		step.SetFinished(time.Now().UTC().Unix())
		step.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the step fields
			step.SetExitCode(ctn.ExitCode)
			step.SetStatus(constants.StatusFailure)
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

// loadStep is a helper function to capture
// a step from the client.
func (c *client) loadStep(name string) (*library.Step, error) {
	// load the step key from the client
	result, ok := c.steps.Load(name)
	if !ok {
		return nil, fmt.Errorf("unable to load step %s", name)
	}

	// cast the step key to the expected type
	s, ok := result.(*library.Step)
	if !ok {
		return nil, fmt.Errorf("step %s had unexpected value", name)
	}

	return s, nil
}

// loadStepLog is a helper function to capture
// the logs for a step from the client.
func (c *client) loadStepLogs(name string) (*library.Log, error) {
	// load the step log key from the client
	result, ok := c.stepLogs.Load(name)
	if !ok {
		return nil, fmt.Errorf("unable to load logs for step %s", name)
	}

	// cast the step log key to the expected type
	l, ok := result.(*library.Log)
	if !ok {
		return nil, fmt.Errorf("logs for step %s had unexpected value", name)
	}

	return l, nil
}
