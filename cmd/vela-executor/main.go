// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package main

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	app := cli.NewApp()

	// Package Information

	app.Name = "vela-executor"
	app.HelpName = "vela-executor"
	app.Usage = "Vela executor package for integrating with different executors"
	app.Copyright = "Copyright (c) 2020 Target Brands, Inc. All rights reserved."
	app.Authors = []cli.Author{
		{
			Name:  "Vela Admins",
			Email: "vela@target.com",
		},
	}

	// Package Metadata

	app.Compiled = time.Now()
	app.Action = run

	// Package Flags

	app.Flags = []cli.Flag{

		cli.StringFlag{
			EnvVar: "VELA_LOG_LEVEL,RUNTIME_LOG_LEVEL",
			Name:   "log.level",
			Usage:  "set log level - options: (trace|debug|info|warn|error|fatal|panic)",
			Value:  "info",
		},
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

		// Executor Flags

		cli.StringFlag{
			EnvVar: "VELA_EXECUTOR_DRIVER,EXECUTOR_DRIVER",
			Name:   "executor.driver",
			Usage:  "executor driver",
			Value:  "linux",
		},
		cli.IntFlag{
			EnvVar: "VELA_EXECUTOR_THREADS,EXECUTOR_THREADS",
			Name:   "executor.threads",
			Usage:  "number of executor threads to create",
			Value:  1,
		},
		cli.DurationFlag{
			EnvVar: "VELA_EXECUTOR_TIMEOUT,EXECUTOR_TIMEOUT",
			Name:   "executor.timeout",
			Usage:  "max time an executor will run a build",
			Value:  60 * time.Minute,
		},

		// Runtime Flags

		cli.StringFlag{
			EnvVar: "VELA_RUNTIME_DRIVER,RUNTIME_DRIVER",
			Name:   "runtime.driver",
			Usage:  "name of runtime driver to use",
		},
		cli.StringFlag{
			EnvVar: "VELA_RUNTIME_CONFIG,RUNTIME_CONFIG",
			Name:   "runtime.config",
			Usage:  "path to runtime configuration file",
		},
		cli.StringFlag{
			EnvVar: "VELA_RUNTIME_NAMESPACE,RUNTIME_NAMESPACE",
			Name:   "runtime.namespace",
			Usage:  "name of namespace for runtime configuration (kubernetes runtime only)",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
