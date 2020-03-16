// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"fmt"
	"os"
	"sync"

	"github.com/go-vela/pkg-runtime/runtime"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"

	"github.com/sirupsen/logrus"
)

type client struct {
	Vela     *vela.Client
	Runtime  runtime.Engine
	Secrets  map[string]*library.Secret
	Hostname string

	// private fields
	logger      *logrus.Entry
	build       *library.Build
	pipeline    *pipeline.Build
	repo        *library.Repo
	services    sync.Map
	serviceLogs sync.Map
	steps       sync.Map
	stepLogs    sync.Map
	user        *library.User
	err         error
}

// New returns an Executor implementation that integrates with a Linux instance.
func New(opts ...Opt) (*client, error) {
	// create new Linux client
	c := new(client)

	// apply all provided configuration options
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	// capture the hostname
	c.Hostname, _ = os.Hostname()

	// create the logger object
	c.logger = logrus.WithFields(logrus.Fields{
		"host": c.Hostname,
	})

	return c, nil
}

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
