// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package main

import (
	"github.com/go-vela/pkg-executor/executor"

	"github.com/urfave/cli"
)

func flags() []cli.Flag {
	f := []cli.Flag{

		cli.StringFlag{
			EnvVar: "VELA_PIPELINE_CONFIG,PIPELINE_CONFIG",
			Name:   "pipeline.config",
			Usage:  "path to pipeline configuration file",
			Value:  "testdata/steps.yml",
		},

		// Compiler Flags

		cli.BoolFlag{
			EnvVar: "VELA_COMPILER_GITHUB,COMPILER_GITHUB",
			Name:   "github.driver",
			Usage:  "github compiler driver",
		},
		cli.StringFlag{
			EnvVar: "VELA_COMPILER_GITHUB_URL,COMPILER_GITHUB_URL",
			Name:   "github.url",
			Usage:  "github url, used by compiler, for pulling registry templates",
		},
		cli.StringFlag{
			EnvVar: "VELA_COMPILER_GITHUB_TOKEN,COMPILER_GITHUB_TOKEN",
			Name:   "github.token",
			Usage:  "github token, used by compiler, for pulling registry templates",
		},

		// Server Flags

		cli.StringFlag{
			EnvVar: "EXECUTOR_SERVER_ADDR,VELA_SERVER_ADDR,VELA_SERVER,SERVER_ADDR",
			Name:   "server.addr",
			Usage:  "Vela server address as a fully qualified url (<scheme>://<host>)",
		},
		cli.StringFlag{
			EnvVar: "EXECUTOR_SERVER_SECRET,VELA_SERVER_SECRET,SERVER_SECRET",
			Name:   "server.secret",
			Usage:  "secret used for server <-> worker communication",
		},
	}

	// Executor Flags

	f = append(f, executor.Flags...)

	return f
}
