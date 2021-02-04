// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"fmt"
	"os"
	"syscall"

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
