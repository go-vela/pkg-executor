// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

type (
	// client manages communication with the build resources
	client struct{}

	svc struct{}
)

// New returns an Executor implementation that integrates with a Linux instance.
func New(opts ...Opt) (*client, error) {
	return nil, nil
}
