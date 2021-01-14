// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package step

import (
	"fmt"
	"sync"

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// Load attempts to capture the library step
// representing the container from the map.
func Load(c *pipeline.Container, m *sync.Map) (*library.Step, error) {
	// check if the container provided is empty
	if c == nil {
		return nil, fmt.Errorf("empty container provided")
	}

	// load the container ID as the step key from the map
	result, ok := m.Load(c.ID)
	if !ok {
		return nil, fmt.Errorf("unable to load step %s", c.ID)
	}

	// cast the value from the step key to the expected type
	s, ok := result.(*library.Step)
	if !ok {
		return nil, fmt.Errorf("unable to cast value for step %s", c.ID)
	}

	return s, nil
}

// LoadLogs attempts to capture the library step logs
// representing the container from the map.
func LoadLogs(c *pipeline.Container, m *sync.Map) (*library.Log, error) {
	// check if the container provided is empty
	if c == nil {
		return nil, fmt.Errorf("empty container provided")
	}

	// load the container ID as the step log key from the map
	result, ok := m.Load(c.ID)
	if !ok {
		return nil, fmt.Errorf("unable to load logs for step %s", c.ID)
	}

	// cast the value from the step log key to the expected type
	l, ok := result.(*library.Log)
	if !ok {
		return nil, fmt.Errorf("unable to cast value to logs for step %s", c.ID)
	}

	return l, nil
}
