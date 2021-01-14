// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-vela/pkg-executor/internal/service"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// create a service logging pattern.
const servicePattern = "[service: %s]"

// CreateService configures the service for execution.
func (c *client) CreateService(ctx context.Context, ctn *pipeline.Container) error {
	ctn.Environment["BUILD_HOST"] = c.Hostname
	ctn.Environment["VELA_HOST"] = c.Hostname
	ctn.Environment["VELA_VERSION"] = "v0.6.0"
	// TODO: remove hardcoded reference
	ctn.Environment["VELA_RUNTIME"] = "docker"
	ctn.Environment["VELA_DISTRIBUTION"] = "linux"

	// setup the runtime container
	err := c.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// substitute container configuration
	err = ctn.Substitute()
	if err != nil {
		return err
	}

	return nil
}

// PlanService prepares the service for execution.
func (c *client) PlanService(ctx context.Context, ctn *pipeline.Container) error {
	// update the engine service object
	_service := new(library.Service)
	_service.SetName(ctn.Name)
	_service.SetNumber(ctn.Number)
	_service.SetStatus(constants.StatusRunning)
	_service.SetStarted(time.Now().UTC().Unix())
	_service.SetHost(ctn.Environment["VELA_HOST"])
	_service.SetRuntime(ctn.Environment["VELA_RUNTIME"])
	_service.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])

	// add a service to a map
	c.services.Store(ctn.ID, _service)

	return nil
}

// ExecService runs a service.
func (c *client) ExecService(ctx context.Context, ctn *pipeline.Container) error {
	// run the runtime container
	err := c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		// stream logs from container
		err := c.StreamService(ctx, ctn)
		if err != nil {
			fmt.Fprintln(os.Stdout, "unable to stream logs for service:", err)
		}
	}()

	return nil
}

// StreamService tails the output for a service.
func (c *client) StreamService(ctx context.Context, ctn *pipeline.Container) error {
	// tail the runtime container
	rc, err := c.Runtime.TailContainer(ctx, ctn)
	if err != nil {
		return err
	}
	defer rc.Close()

	// create a service pattern for log output
	_pattern := fmt.Sprintf(servicePattern, ctn.Name)

	// create new scanner from the container output
	scanner := bufio.NewScanner(rc)

	// scan entire container output
	for scanner.Scan() {
		// ensure we output to stdout
		fmt.Fprintln(os.Stdout, _pattern, scanner.Text())
	}

	return scanner.Err()
}

// DestroyService cleans up services after execution.
func (c *client) DestroyService(ctx context.Context, ctn *pipeline.Container) error {
	// load the service from the client
	_service, err := service.Load(ctn, &c.services)
	if err != nil {
		// create the service from the container
		_service = new(library.Service)
		_service.SetName(ctn.Name)
		_service.SetNumber(ctn.Number)
		_service.SetStatus(constants.StatusPending)
		_service.SetHost(ctn.Environment["VELA_HOST"])
		_service.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		_service.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	// check if the service is in a pending state
	if _service.GetStatus() == constants.StatusPending {
		// update the service fields
		//
		// TODO: consider making this a constant
		//
		// nolint: gomnd // ignore magic number 137
		_service.SetExitCode(137)
		_service.SetFinished(time.Now().UTC().Unix())
		_service.SetStatus(constants.StatusKilled)

		// check if the service was not started
		if _service.GetStarted() == 0 {
			// set the started time to the finished time
			_service.SetStarted(_service.GetFinished())
		}
	}

	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the service finished
	if _service.GetFinished() == 0 {
		// update the service fields
		_service.SetFinished(time.Now().UTC().Unix())
		_service.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the service fields
			_service.SetExitCode(ctn.ExitCode)
			_service.SetStatus(constants.StatusFailure)
		}
	}

	// remove the runtime container
	err = c.Runtime.RemoveContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}
