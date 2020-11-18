// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"github.com/go-vela/pkg-runtime/runtime"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// Opt represents a configuration option to initialize the client.
type Opt func(*client) error

// WithBuild sets the library build in the client.
func WithBuild(b *library.Build) Opt {
	return func(c *client) error {
		return nil
	}
}

// WithHostname sets the hostname in the client.
func WithHostname(hostname string) Opt {
	return func(c *client) error {
		return nil
	}
}

// WithPipeline sets the pipeline build in the client.
func WithPipeline(p *pipeline.Build) Opt {
	return func(c *client) error {
		return nil
	}
}

// WithRepo sets the library repo in the client.
func WithRepo(r *library.Repo) Opt {
	return func(c *client) error {
		return nil
	}
}

// WithRuntime sets the runtime engine in the client.
func WithRuntime(r runtime.Engine) Opt {
	return func(c *client) error {
		return nil
	}
}

// WithUser sets the library user in the client.
func WithUser(u *library.User) Opt {
	return func(c *client) error {
		return nil
	}
}

// WithVelaClient sets the Vela client in the client.
func WithVelaClient(cli *vela.Client) Opt {
	return func(c *client) error {
		return nil
	}
}
