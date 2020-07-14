// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
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
	secrets     sync.Map
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

	// create new logger for the client
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#StandardLogger
	logger := logrus.StandardLogger()

	// create new logger for the client
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#NewEntry
	c.logger = logrus.NewEntry(logger)

	// apply all provided configuration options
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}
