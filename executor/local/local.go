// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"sync"

	"github.com/go-vela/pkg-runtime/runtime"
	"github.com/go-vela/sdk-go/vela"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

type (
	// client manages communication with the build resources
	client struct {
		Vela     *vela.Client
		Runtime  runtime.Engine
		Secrets  map[string]*library.Secret
		Hostname string

		// clients for build actions
		secret *secretSvc

		// private fields
		init        *pipeline.Container
		build       *library.Build
		pipeline    *pipeline.Build
		repo        *library.Repo
		secrets     sync.Map
		services    sync.Map
		serviceLogs sync.Map
		steps       sync.Map
		stepLogs    sync.Map
		user        *library.User
		err         error
	}

	svc struct {
		client *client
	}
)

// New returns an Executor implementation that integrates with the local system.
func New(opts ...Opt) (*client, error) {
	// create new local client
	c := new(client)

	// apply all provided configuration options
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	// instantiate map for non-plugin secrets
	c.Secrets = make(map[string]*library.Secret)

	// instantiate all client services
	c.secret = &secretSvc{client: c}

	return c, nil
}
