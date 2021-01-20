// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package executor

import (
	"github.com/go-vela/types/constants"

	"github.com/urfave/cli/v2"
)

// Flags represents all supported command line
// interface (CLI) flags for the executor.
//
// https://pkg.go.dev/github.com/urfave/cli?tab=doc#Flag
var Flags = []cli.Flag{

	&cli.StringFlag{
		EnvVars: []string{"EXECUTOR_LOG_LEVEL", "VELA_LOG_LEVEL", "LOG_LEVEL"},
		Name:    "executor.log.level",
		Usage:   "sets the log level for the executor",
		Value:   "info",
	},

	// Executor Flags

	&cli.StringFlag{
		EnvVars: []string{"VELA_EXECUTOR_DRIVER", "EXECUTOR_DRIVER"},
		Name:    "executor.driver",
		Usage:   "sets the driver to be used for the executor",
		Value:   constants.DriverLinux,
	},
}
