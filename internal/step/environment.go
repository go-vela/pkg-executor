// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package step

import (
	"fmt"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// Environment attempts to update the environment variables
// for the container based off the library resources.
func Environment(c *pipeline.Container, b *library.Build, r *library.Repo, s *library.Step, version string) error {
	// check if container or container environment are empty
	if c == nil || c.Environment == nil {
		return fmt.Errorf("empty container provided for environment")
	}

	// check if the build provided is empty
	if b != nil {
		// check if the channel exists in the environment
		channel, ok := c.Environment["VELA_CHANNEL"]
		if !ok {
			// set default for channel
			channel = constants.DefaultRoute
		}

		// check if the workspace exists in the environment
		workspace, ok := c.Environment["VELA_WORKSPACE"]
		if !ok {
			// set default for workspace
			workspace = constants.WorkspaceDefault
		}

		// update environment variables
		c.Environment["VELA_DISTRIBUTION"] = b.GetDistribution()
		c.Environment["VELA_HOST"] = b.GetHost()
		c.Environment["VELA_RUNTIME"] = b.GetRuntime()
		c.Environment["VELA_VERSION"] = version

		// populate environment variables from build library
		//
		// https://pkg.go.dev/github.com/go-vela/types/library#Build.Environment
		c.Environment = appendMap(c.Environment, b.Environment(workspace, channel))
	}

	// check if the repo provided is empty
	if r != nil {
		// populate environment variables from build library
		//
		// https://pkg.go.dev/github.com/go-vela/types/library#Repo.Environment
		c.Environment = appendMap(c.Environment, r.Environment())
	}

	// check if the step provided is empty
	if s != nil {
		// populate environment variables from step library
		//
		// https://pkg.go.dev/github.com/go-vela/types/library#Service.Environment
		c.Environment = appendMap(c.Environment, s.Environment())
	}

	return nil
}

// helper function to merge two maps together.
func appendMap(originalMap, otherMap map[string]string) map[string]string {
	for key, value := range otherMap {
		originalMap[key] = value
	}

	return originalMap
}
