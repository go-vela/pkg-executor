// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package executor

import (
	"github.com/go-vela/types/constants"

	"github.com/urfave/cli/v2"
)

// Flags represents all supported command line
// interface (CLI) flags for the executor.
var Flags = []cli.Flag{

	&cli.StringFlag{
		EnvVars: []string{"EXECUTOR_LOG_LEVEL", "VELA_LOG_LEVEL", "LOG_LEVEL"},
		Name:    "executor.log.level",
		Usage:   "set log level - options: (trace|debug|info|warn|error|fatal|panic)",
		Value:   "info",
	},

	// Executor Flags

	&cli.StringFlag{
		EnvVars: []string{"VELA_EXECUTOR_DRIVER", "EXECUTOR_DRIVER"},
		Name:    "executor.driver",
		Usage:   "executor driver",
		Value:   constants.DriverLinux,
	},
}
