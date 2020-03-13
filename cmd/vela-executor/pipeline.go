// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package main

import (
	"fmt"

	"github.com/go-vela/compiler/compiler"

	"github.com/go-vela/types/pipeline"

	"github.com/sirupsen/logrus"

	"github.com/urfave/cli"
)

// helper function to setup the pipeline from the CLI arguments.
func setupPipeline(c *cli.Context, comp compiler.Engine) (*pipeline.Build, error) {
	logrus.Debug("creating pipeline from CLI configuration")

	// setup the fake build for the compiler
	b := setupBuild()
	// setup the fake repo for the compiler
	r := setupRepo()
	// setup the fake user for the compiler
	u := setupUser()

	logrus.Infof("compiling pipeline configuration %s", c.String("pipeline.config"))
	p, err := comp.
		WithBuild(b).
		WithFiles([]string{}).
		WithMetadata(nil).
		WithRepo(r).
		WithUser(u).
		Compile(c.String("pipeline.config"))
	if err != nil {
		return nil, err
	}

	// sanitize pipeline
	p.Sanitize(c.String("runtime.driver"))
	if p == nil {
		return nil, fmt.Errorf("unable to sanitize pipeline")
	}

	return p, nil
}
