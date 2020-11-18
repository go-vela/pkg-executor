// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"context"

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// secretSvc handles communication with secret processes during a build.
type secretSvc svc

// create configures the secret plugin for execution.
func (s *secretSvc) create(ctx context.Context, ctn *pipeline.Container) error {
	return nil
}

// destroy cleans up secret plugin after execution.
func (s *secretSvc) destroy(ctx context.Context, ctn *pipeline.Container) error {
	return nil
}

// exec runs a secret plugins for a pipeline.
func (s *secretSvc) exec(ctx context.Context, p *pipeline.SecretSlice) error {
	return nil
}

// pull defines a function that pulls the secrets from the server for a given pipeline.
func (s *secretSvc) pull(secret *pipeline.Secret) (*library.Secret, error) {
	return nil, nil
}

// stream tails the output for a secret plugin.
func (s *secretSvc) stream(ctx context.Context, ctn *pipeline.Container) error {
	return nil
}
