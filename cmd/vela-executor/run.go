// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package main

import (
	"fmt"

	"github.com/go-vela/pkg-executor/executor"

	"github.com/go-vela/pkg-runtime/runtime"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	_ "github.com/joho/godotenv/autoload"
)

// run executes the package based off the configuration provided.
func run(c *cli.Context) error {
	// set the log level for the plugin
	switch c.String("log.level") {
	case "t", "trace", "Trace", "TRACE":
		logrus.SetLevel(logrus.TraceLevel)
	case "d", "debug", "Debug", "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "w", "warn", "Warn", "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case "e", "error", "Error", "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	case "f", "fatal", "Fatal", "FATAL":
		logrus.SetLevel(logrus.FatalLevel)
	case "p", "panic", "Panic", "PANIC":
		logrus.SetLevel(logrus.PanicLevel)
	case "i", "info", "Info", "INFO":
		fallthrough
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	// create a vela client
	vela, err := setupClient(c)
	if err != nil {
		return err
	}

	// setup the compiler
	compiler, err := setupCompiler(c)
	if err != nil {
		logrus.Fatal(err)
	}

	// setup the pipeline
	p, err := setupPipeline(c, compiler)
	if err != nil {
		return err
	}

	fmt.Println("Pipeline: ", p)

	// setup the runtime
	r, err := runtime.New(&runtime.Setup{
		Driver:    c.String("runtime.driver"),
		Config:    c.String("runtime.config"),
		Namespace: c.String("runtime.namespace"),
	})
	if err != nil {
		return err
	}

	fmt.Println("Runtime: ", r)

	// setup the executor
	e, err := executor.New(&executor.Setup{
		Driver:   c.String("executor.driver"),
		Client:   vela,
		Runtime:  r,
		Build:    setupBuild(),
		Pipeline: p,
		Repo:     setupRepo(),
		User:     setupUser(),
	})
	if err != nil {
		return err
	}

	fmt.Println("Executor: ", e)

	return nil
}
