// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// CreateStep configures the step for execution.
func (c *client) CreateStep(ctx context.Context, ctn *pipeline.Container) error {
	ctn.Environment["BUILD_HOST"] = c.Hostname
	ctn.Environment["VELA_HOST"] = c.Hostname
	ctn.Environment["VELA_VERSION"] = "v0.6.0"
	// TODO: remove hardcoded reference
	ctn.Environment["VELA_RUNTIME"] = "docker"
	ctn.Environment["VELA_DISTRIBUTION"] = "linux"

	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	// setup the runtime container
	err := c.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// substitute container configuration
	err = ctn.Substitute()
	if err != nil {
		return err
	}

	return nil
}

// PlanStep prepares the step for execution.
func (c *client) PlanStep(ctx context.Context, ctn *pipeline.Container) error {
	b := c.build
	r := c.repo

	// update the engine step object
	s := new(library.Step)
	s.SetName(ctn.Name)
	s.SetNumber(ctn.Number)
	s.SetStatus(constants.StatusRunning)
	s.SetStarted(time.Now().UTC().Unix())
	s.SetHost(ctn.Environment["VELA_HOST"])
	s.SetRuntime(ctn.Environment["VELA_RUNTIME"])
	s.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])

	// add a step to a map
	c.steps.Store(ctn.ID, s)

	// update the engine step log object
	l := new(library.Log)
	l.SetBuildID(b.GetID())
	l.SetRepoID(r.GetID())
	l.SetStepID(s.GetID())

	// add a step log to a map
	c.stepLogs.Store(ctn.ID, l)

	return nil
}

// ExecStep runs a step.
func (c *client) ExecStep(ctx context.Context, ctn *pipeline.Container) error {
	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	// run the runtime container
	err := c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		// stream logs from container
		err := c.StreamStep(ctx, ctn)
		if err != nil {
			// TODO: Should this be changed or removed?
			fmt.Println(err)
		}
	}()

	// do not wait for detached containers
	if ctn.Detach {
		return nil
	}

	// wait for the runtime container
	err = c.Runtime.WaitContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}

// StreamStep tails the output for a step.
func (c *client) StreamStep(ctx context.Context, ctn *pipeline.Container) error {
	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	// tail the runtime container
	rc, err := c.Runtime.TailContainer(ctx, ctn)
	if err != nil {
		return err
	}
	defer rc.Close()

	// create new scanner from the container output
	scanner := bufio.NewScanner(rc)

	// scan entire container output
	for scanner.Scan() {
		// ensure we output to stdout
		fmt.Fprintln(os.Stdout, scanner.Text())
	}

	return scanner.Err()
}

// DestroyStep cleans up steps after execution.
func (c *client) DestroyStep(ctx context.Context, ctn *pipeline.Container) error {
	// TODO: remove hardcoded reference
	if ctn.Name == "init" {
		return nil
	}

	// load the step from the client
	s, err := step.Load(ctn, &c.steps)
	if err != nil {
		// create the step from the container
		s = new(library.Step)
		s.SetName(ctn.Name)
		s.SetNumber(ctn.Number)
		s.SetStatus(constants.StatusPending)
		s.SetHost(ctn.Environment["VELA_HOST"])
		s.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		s.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	// check if the step is in a pending state
	if s.GetStatus() == constants.StatusPending {
		// update the step fields
		//
		// TODO: consider making this a constant
		//
		// nolint: gomnd // ignore magic number 137
		s.SetExitCode(137)
		s.SetFinished(time.Now().UTC().Unix())
		s.SetStatus(constants.StatusKilled)

		// check if the step was not started
		if s.GetStarted() == 0 {
			// set the started time to the finished time
			s.SetStarted(s.GetFinished())
		}
	}

	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the step finished
	if s.GetFinished() == 0 {
		// update the step fields
		s.SetFinished(time.Now().UTC().Unix())
		s.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the step fields
			s.SetExitCode(ctn.ExitCode)
			s.SetStatus(constants.StatusFailure)
		}
	}

	// remove the runtime container
	err = c.Runtime.RemoveContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}

// loadInitContainer is a helper function to capture
// the init step from the client.
func (c *client) loadInitContainer(p *pipeline.Build) *pipeline.Container {
	// TODO: make this better
	init := new(pipeline.Container)
	if len(p.Steps) > 0 {
		init = p.Steps[0]
	}

	// TODO: make this better
	if len(p.Stages) > 0 {
		init = p.Stages[0].Steps[0]
	}

	return init
}
