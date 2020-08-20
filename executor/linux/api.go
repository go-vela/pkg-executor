// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"fmt"
	"syscall"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// GetBuild gets the current build in execution.
func (c *client) GetBuild() (*library.Build, error) {
	b := c.build

	// check if the build resource is available
	if b == nil {
		return nil, fmt.Errorf("build resource not found")
	}

	return b, nil
}

// GetPipeline gets the current pipeline in execution.
func (c *client) GetPipeline() (*pipeline.Build, error) {
	p := c.pipeline

	// check if the pipeline resource is available
	if p == nil {
		return nil, fmt.Errorf("pipeline resource not found")
	}

	return p, nil
}

// GetRepo gets the current repo in execution.
func (c *client) GetRepo() (*library.Repo, error) {
	r := c.repo

	// check if the repo resource is available
	if r == nil {
		return nil, fmt.Errorf("repo resource not found")
	}

	return r, nil
}

// CancelBuild cancels the current build in execution.
func (c *client) CancelBuild() (*library.Build, error) {
	b := c.build

	// check if the build resource is available
	if b == nil {
		return nil, fmt.Errorf("build resource not found")
	}

	err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	if err != nil {
		return nil, fmt.Errorf("unable to cancel PID: %w", err)
	}

	// set the build status to killed
	b.SetStatus(constants.StatusCanceled)

	return b, nil
}
