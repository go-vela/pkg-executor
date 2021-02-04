// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/go-vela/pkg-executor/internal/service"
	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// GetBuild gets the current build in execution.
func (c *client) GetBuild() (*library.Build, error) {
	// check if the build resource is available
	if c.build == nil {
		return nil, fmt.Errorf("build resource not found")
	}

	return c.build, nil
}

// GetPipeline gets the current pipeline in execution.
func (c *client) GetPipeline() (*pipeline.Build, error) {
	// check if the pipeline resource is available
	if c.pipeline == nil {
		return nil, fmt.Errorf("pipeline resource not found")
	}

	return c.pipeline, nil
}

// GetRepo gets the current repo in execution.
func (c *client) GetRepo() (*library.Repo, error) {
	// check if the repo resource is available
	if c.repo == nil {
		return nil, fmt.Errorf("repo resource not found")
	}

	return c.repo, nil
}

// CancelBuild cancels the current build in execution.
func (c *client) CancelBuild() (*library.Build, error) {
	// get the current build from the client
	b, err := c.GetBuild()
	if err != nil {
		return nil, err
	}

	// set the build status to canceled
	b.SetStatus(constants.StatusCanceled)

	// get the current build from the client
	pipeline, err := c.GetPipeline()
	if err != nil {
		return nil, err
	}

	// cancel non successful services
	for _, _service := range pipeline.Services {
		// load the service from the client
		//
		// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/service#Load
		s, err := service.Load(_service, &c.services)
		if err != nil {
			// create the service from the container
			//
			// https://pkg.go.dev/github.com/go-vela/types/library#ServiceFromContainer
			s = library.ServiceFromContainer(_service)
		}

		// if service has not succeeded, set it as canceled
		if !strings.EqualFold(s.GetStatus(), constants.StatusSuccess) {
			s.SetStatus(constants.StatusCanceled)
		}
	}

	// cancel non successful steps
	for _, _step := range pipeline.Steps {
		// load the step from the client
		//
		// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Load
		s, err := step.Load(_step, &c.steps)
		if err != nil {
			// create the step from the container
			//
			// https://pkg.go.dev/github.com/go-vela/types/library#StepFromContainer
			s = library.StepFromContainer(_step)
		}

		// if step has not succeeded, set it as canceled
		if !strings.EqualFold(s.GetStatus(), constants.StatusSuccess) {
			s.SetStatus(constants.StatusCanceled)
		}
	}

	// cancel non successful stages
	for _, _stage := range pipeline.Stages {
		// cancel non successful steps for that stage
		for _, _step := range _stage.Steps {
			// load the step from the client
			//
			// https://pkg.go.dev/github.com/go-vela/pkg-executor/internal/step#Load
			s, err := step.Load(_step, &c.steps)
			if err != nil {
				// create the step from the container
				//
				// https://pkg.go.dev/github.com/go-vela/types/library#StepFromContainer
				s = library.StepFromContainer(_step)
			}

			// if step has not succeeded, set it as canceled
			if !strings.EqualFold(s.GetStatus(), constants.StatusSuccess) {
				s.SetStatus(constants.StatusCanceled)
			}
		}
	}

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return nil, fmt.Errorf("unable to find PID: %v", err)
	}

	err = p.Signal(syscall.SIGTERM)
	if err != nil {
		return nil, fmt.Errorf("unable to cancel PID: %v", err)
	}

	return b, nil
}
