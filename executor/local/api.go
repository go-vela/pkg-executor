// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// GetBuild gets the current build in execution.
func (c *client) GetBuild() (*library.Build, error) {
	return nil, nil
}

// GetPipeline gets the current pipeline in execution.
func (c *client) GetPipeline() (*pipeline.Build, error) {
	return nil, nil
}

// GetRepo gets the current repo in execution.
func (c *client) GetRepo() (*library.Repo, error) {
	return nil, nil
}

// CancelBuild cancels the current build in execution.
func (c *client) CancelBuild() (*library.Build, error) {
	return nil, nil
}
