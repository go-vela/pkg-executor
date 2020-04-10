// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package main

import (
	"github.com/go-vela/sdk-go/vela"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// helper function to setup the queue from the CLI arguments.
func setupClient(c *cli.Context) (*vela.Client, error) {
	logrus.Debug("creating Vela client from CLI configuration")

	// create new Vela client from provided server address
	vela, err := vela.NewClient(c.String("server.addr"), nil)
	if err != nil {
		return nil, err
	}

	// set token used in authentication for Vela client
	vela.Authentication.SetTokenAuth(c.String("server.token"))

	return vela, nil
}
