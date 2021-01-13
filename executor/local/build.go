// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-vela/pkg-executor/internal/build"
	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/pipeline"
)

// CreateBuild configures the build for execution.
func (c *client) CreateBuild(ctx context.Context) error {
	// defer taking snapshot of build
	defer build.Snapshot(c.build, nil, c.err, nil, nil)

	// update the build fields
	c.build.SetStatus(constants.StatusRunning)
	c.build.SetStarted(time.Now().UTC().Unix())
	c.build.SetHost(c.Hostname)
	// TODO: This should not be hardcoded
	c.build.SetDistribution(constants.DriverLocal)
	c.build.SetRuntime("docker")

	// load the init container from the pipeline
	c.init = c.loadInitContainer(c.pipeline)

	// create the step
	c.err = c.CreateStep(ctx, c.init)
	if c.err != nil {
		return fmt.Errorf("unable to create %s step: %w", c.init.Name, c.err)
	}

	// plan the step
	c.err = c.PlanStep(ctx, c.init)
	if c.err != nil {
		return fmt.Errorf("unable to plan %s step: %w", c.init.Name, c.err)
	}

	return c.err
}

// PlanBuild prepares the build for execution.
//
// nolint: funlen // ignore function length - will be refactored at a later date
func (c *client) PlanBuild(ctx context.Context) error {
	// defer taking snapshot of build
	defer build.Snapshot(c.build, nil, c.err, nil, nil)

	// load the init step from the client
	s, err := step.Load(c.init, &c.steps)
	if err != nil {
		return err
	}

	// create a step pattern for log output
	_pattern := fmt.Sprintf(stepPattern, c.init.Name)

	defer func() {
		s.SetFinished(time.Now().UTC().Unix())
	}()

	// create the runtime network for the pipeline
	c.err = c.Runtime.CreateNetwork(ctx, c.pipeline)
	if c.err != nil {
		return fmt.Errorf("unable to create network: %w", c.err)
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Inspecting runtime network...")

	// output the network command to stdout
	fmt.Fprintln(os.Stdout, _pattern, "$ docker network inspect", c.pipeline.ID)

	// inspect the runtime network for the pipeline
	network, err := c.Runtime.InspectNetwork(ctx, c.pipeline)
	if err != nil {
		c.err = err
		return fmt.Errorf("unable to inspect network: %w", err)
	}

	// output the network information to stdout
	fmt.Fprintln(os.Stdout, _pattern, string(network))

	// create the runtime volume for the pipeline
	err = c.Runtime.CreateVolume(ctx, c.pipeline)
	if err != nil {
		c.err = err
		return fmt.Errorf("unable to create volume: %w", err)
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Inspecting runtime volume...")

	// output the volume command to stdout
	fmt.Fprintln(os.Stdout, _pattern, "$ docker volume inspect", c.pipeline.ID)

	// inspect the runtime volume for the pipeline
	volume, err := c.Runtime.InspectVolume(ctx, c.pipeline)
	if err != nil {
		c.err = err
		return fmt.Errorf("unable to inspect volume: %w", err)
	}

	// output the volume information to stdout
	fmt.Fprintln(os.Stdout, _pattern, string(volume))

	return c.err
}

// AssembleBuild prepares the containers within a build for execution.
//
// nolint: funlen // ignore function length - will be refactored at a later date
func (c *client) AssembleBuild(ctx context.Context) error {
	// defer taking snapshot of build
	defer build.Snapshot(c.build, nil, c.err, nil, nil)

	// load the init step from the client
	sInit, err := step.Load(c.init, &c.steps)
	if err != nil {
		return err
	}

	// create a step pattern for log output
	_pattern := fmt.Sprintf(stepPattern, c.init.Name)

	defer func() {
		sInit.SetFinished(time.Now().UTC().Unix())
	}()

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Pulling service images...")

	// create the services for the pipeline
	for _, s := range c.pipeline.Services {
		// TODO: remove this; but we need it for tests
		s.Detach = true

		// create the service
		c.err = c.CreateService(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to create %s service: %w", s.Name, c.err)
		}

		// output the image command to stdout
		fmt.Fprintln(os.Stdout, _pattern, "$ docker image inspect", s.Image)

		// inspect the service image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			c.err = err
			return fmt.Errorf("unable to inspect %s service: %w", s.Name, err)
		}

		// output the image information to stdout
		fmt.Fprintln(os.Stdout, _pattern, string(image))
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Pulling stage images...")

	// create the stages for the pipeline
	for _, s := range c.pipeline.Stages {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// create the stage
		c.err = c.CreateStage(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to create %s stage: %w", s.Name, c.err)
		}
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Pulling step images...")

	// create the steps for the pipeline
	for _, s := range c.pipeline.Steps {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// create the step
		c.err = c.CreateStep(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to create %s step: %w", s.Name, c.err)
		}

		// output the image command to stdout
		fmt.Fprintln(os.Stdout, _pattern, "$ docker image inspect", s.Image)

		// inspect the step image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			c.err = err
			return fmt.Errorf("unable to inspect %s step: %w", s.Name, err)
		}

		// output the image information to stdout
		fmt.Fprintln(os.Stdout, _pattern, string(image))
	}

	// output a new line for readability to stdout
	fmt.Fprintln(os.Stdout, "")

	return c.err
}

// ExecBuild runs a pipeline for a build.
//
// nolint: funlen // ignore function length - will be refactored at a later date
func (c *client) ExecBuild(ctx context.Context) error {
	defer func() {
		// Overwrite with proper status and error only if build was not canceled
		if !strings.EqualFold(c.build.GetStatus(), constants.StatusCanceled) {
			// NOTE: if the build is already in a failure state we do not
			// want to update the state to be success
			if !strings.EqualFold(c.build.GetStatus(), constants.StatusFailure) {
				c.build.SetStatus(constants.StatusSuccess)
			}

			// NOTE: When an error occurs during a build that does not have to do
			// with a pipeline we should set build status to "error" not "failed"
			// because it is worker related and not build.
			if c.err != nil {
				c.build.SetError(c.err.Error())
				c.build.SetStatus(constants.StatusError)
			}
		}
		// update the build fields
		c.build.SetFinished(time.Now().UTC().Unix())
	}()

	// execute the services for the pipeline
	for _, s := range c.pipeline.Services {
		// plan the service
		c.err = c.PlanService(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to plan service: %w", c.err)
		}

		// execute the service
		c.err = c.ExecService(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to execute service: %w", c.err)
		}
	}

	// execute the steps for the pipeline
	for _, s := range c.pipeline.Steps {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// extract rule data from build information
		ruledata := &pipeline.RuleData{
			Branch: c.build.GetBranch(),
			Event:  c.build.GetEvent(),
			Repo:   c.repo.GetFullName(),
			Status: c.build.GetStatus(),
		}

		// when tag event add tag information into ruledata
		if strings.EqualFold(c.build.GetEvent(), constants.EventTag) {
			ruledata.Tag = strings.TrimPrefix(c.build.GetRef(), "refs/tags/")
		}

		// when deployment event add deployment information into ruledata
		if strings.EqualFold(c.build.GetEvent(), constants.EventDeploy) {
			ruledata.Target = c.build.GetDeploy()
		}

		// check if you need to excute this step
		if !s.Execute(ruledata) {
			continue
		}

		// plan the step
		c.err = c.PlanStep(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to plan step: %w", c.err)
		}

		// execute the step
		c.err = c.ExecStep(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to execute step: %w", c.err)
		}

		// load the init step from the client
		cStep, err := step.Load(s, &c.steps)
		if err != nil {
			c.err = err
			return err
		}

		// check the step exit code
		if s.ExitCode != 0 {
			// check if we ignore step failures
			if !s.Ruleset.Continue {
				// set build status to failure
				c.build.SetStatus(constants.StatusFailure)
			}

			// update the step fields
			cStep.SetExitCode(s.ExitCode)
			cStep.SetStatus(constants.StatusFailure)
		}

		cStep.SetFinished(time.Now().UTC().Unix())
	}

	// create an error group with the context for each stage
	//
	// https://pkg.go.dev/golang.org/x/sync/errgroup?tab=doc#WithContext
	stages, stageCtx := errgroup.WithContext(ctx)
	// create a map to track the progress of each stage
	stageMap := make(map[string]chan error)

	// iterate through each stage in the pipeline
	for _, s := range c.pipeline.Stages {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// https://golang.org/doc/faq#closures_and_goroutines
		stage := s

		// create a new channel for each stage in the map
		stageMap[stage.Name] = make(chan error)

		// spawn errgroup routine for the stage
		//
		// https://pkg.go.dev/golang.org/x/sync/errgroup?tab=doc#Group.Go
		stages.Go(func() error {
			// plan the stage
			c.err = c.PlanStage(stageCtx, stage, stageMap)
			if c.err != nil {
				return fmt.Errorf("unable to plan stage: %w", c.err)
			}

			// execute the stage
			c.err = c.ExecStage(stageCtx, stage, stageMap)
			if c.err != nil {
				return fmt.Errorf("unable to execute stage: %w", c.err)
			}

			return nil
		})
	}

	// wait for the stages to complete or return an error
	//
	// https://pkg.go.dev/golang.org/x/sync/errgroup?tab=doc#Group.Wait
	c.err = stages.Wait()
	if c.err != nil {
		return fmt.Errorf("unable to wait for stages: %v", c.err)
	}

	return c.err
}

// DestroyBuild cleans up the build after execution.
func (c *client) DestroyBuild(ctx context.Context) error {
	var err error

	// destroy the steps for the pipeline
	for _, s := range c.pipeline.Steps {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// destroy the step
		err = c.DestroyStep(ctx, s)
		if err != nil {
			// output the error information to stdout
			fmt.Fprintln(os.Stdout, "unable to destroy step:", err)
		}
	}

	// destroy the stages for the pipeline
	for _, s := range c.pipeline.Stages {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// destroy the stage
		err = c.DestroyStage(ctx, s)
		if err != nil {
			// output the error information to stdout
			fmt.Fprintln(os.Stdout, "unable to destroy stage:", err)
		}
	}

	// destroy the services for the pipeline
	for _, s := range c.pipeline.Services {
		// destroy the service
		err = c.DestroyService(ctx, s)
		if err != nil {
			// output the error information to stdout
			fmt.Fprintln(os.Stdout, "unable to destroy service:", err)
		}
	}

	// remove the runtime volume for the pipeline
	err = c.Runtime.RemoveVolume(ctx, c.pipeline)
	if err != nil {
		// output the error information to stdout
		fmt.Fprintln(os.Stdout, "unable to destroy runtime volume:", err)
	}

	// remove the runtime network for the pipeline
	err = c.Runtime.RemoveNetwork(ctx, c.pipeline)
	if err != nil {
		// output the error information to stdout
		fmt.Fprintln(os.Stdout, "unable to destroy runtime network:", err)
	}

	return err
}
