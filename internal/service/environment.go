// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package service

import (
	"fmt"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// Environment attempts to update the environment variables
// for the container based off the library resources.
func Environment(c *pipeline.Container, b *library.Build, r *library.Repo, s *library.Service) error {
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
			//
			// TODO: consider making this a constant
			workspace = "/vela"
		}

		// populate environment variables from build library
		c.Environment = appendMap(c.Environment, b.Environment(workspace, channel))
	}

	// check if the repo provided is empty
	if r != nil {
		// populate environment variables from build library
		c.Environment = appendMap(c.Environment, r.Environment())
	}

	// check if the service provided is empty
	if s != nil {
		// populate environment variables from service library
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
