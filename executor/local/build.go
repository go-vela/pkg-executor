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
)

// CreateBuild configures the build for execution.
func (c *client) CreateBuild(ctx context.Context) error {
	b := c.build
	p := c.pipeline
	r := c.repo
	e := c.err

	// defer taking snapshot of build
	defer build.Snapshot(b, nil, e, nil, r)

	// update the build fields
	b.SetStatus(constants.StatusRunning)
	b.SetStarted(time.Now().UTC().Unix())
	b.SetHost(c.Hostname)
	// TODO: This should not be hardcoded
	b.SetDistribution("linux")
	b.SetRuntime("docker")

	c.build = b

	// load the init container from the pipeline
	init := c.loadInitContainer(p)

	// create the step
	err := c.CreateStep(ctx, init)
	if err != nil {
		e = err
		return fmt.Errorf("unable to create %s step: %w", init.Name, err)
	}

	// plan the step
	err = c.PlanStep(ctx, init)
	if err != nil {
		e = err
		return fmt.Errorf("unable to plan %s step: %w", init.Name, err)
	}

	// add the init container to secrets client
	c.init = init

	return nil
}

// PlanBuild prepares the build for execution.
//
// nolint: funlen // ignore function length - will be refactored at a later date
func (c *client) PlanBuild(ctx context.Context) error {
	b := c.build
	p := c.pipeline
	r := c.repo
	e := c.err
	init := c.init

	// defer taking snapshot of build
	defer build.Snapshot(b, nil, e, nil, r)

	// load the init step from the client
	s, err := step.Load(init, &c.steps)
	if err != nil {
		return err
	}

	// create a step pattern for log output
	_pattern := fmt.Sprintf(stepPattern, init.Name)

	defer func() {
		s.SetFinished(time.Now().UTC().Unix())
	}()

	// create the runtime network for the pipeline
	err = c.Runtime.CreateNetwork(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to create network: %w", err)
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Inspecting runtime network...")

	// output the network command to stdout
	fmt.Fprintln(os.Stdout, _pattern, "$ docker network inspect", p.ID)

	// inspect the runtime network for the pipeline
	network, err := c.Runtime.InspectNetwork(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to inspect network: %w", err)
	}

	// output the network information to stdout
	fmt.Fprintln(os.Stdout, _pattern, string(network))

	// create the runtime volume for the pipeline
	err = c.Runtime.CreateVolume(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to create volume: %w", err)
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Inspecting runtime volume...")

	// output the volume command to stdout
	fmt.Fprintln(os.Stdout, _pattern, "$ docker volume inspect", p.ID)

	// inspect the runtime volume for the pipeline
	volume, err := c.Runtime.InspectVolume(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to inspect volume: %w", err)
	}

	// output the volume information to stdout
	fmt.Fprintln(os.Stdout, _pattern, string(volume))

	return nil
}

// AssembleBuild prepares the containers within a build for execution.
//
// nolint: funlen // ignore function length - will be refactored at a later date
func (c *client) AssembleBuild(ctx context.Context) error {
	b := c.build
	p := c.pipeline
	r := c.repo
	e := c.err
	init := c.init

	// defer taking snapshot of build
	defer build.Snapshot(b, nil, e, nil, r)

	// load the init step from the client
	sInit, err := step.Load(init, &c.steps)
	if err != nil {
		return err
	}

	// create a step pattern for log output
	_pattern := fmt.Sprintf(stepPattern, init.Name)

	defer func() {
		sInit.SetFinished(time.Now().UTC().Unix())
	}()

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Pulling service images...")

	// create the services for the pipeline
	for _, s := range p.Services {
		// TODO: remove this; but we need it for tests
		s.Detach = true

		// create the service
		err := c.CreateService(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to create %s service: %w", s.Name, err)
		}

		// output the image command to stdout
		fmt.Fprintln(os.Stdout, _pattern, "$ docker image inspect", s.Image)

		// inspect the service image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to inspect %s service: %w", s.Name, err)
		}

		// output the image information to stdout
		fmt.Fprintln(os.Stdout, _pattern, string(image))
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Pulling stage images...")

	// create the stages for the pipeline
	for _, s := range p.Stages {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// create the stage
		err := c.CreateStage(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to create %s stage: %w", s.Name, err)
		}
	}

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Pulling step images...")

	// create the steps for the pipeline
	for _, s := range p.Steps {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// create the step
		err := c.CreateStep(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to create %s step: %w", s.Name, err)
		}

		// output the image command to stdout
		fmt.Fprintln(os.Stdout, _pattern, "$ docker image inspect", s.Image)

		// inspect the step image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to inspect %s step: %w", s.Name, err)
		}

		// output the image information to stdout
		fmt.Fprintln(os.Stdout, _pattern, string(image))
	}

	// output a new line for readability to stdout
	fmt.Fprintln(os.Stdout, "")

	return nil
}

// ExecBuild runs a pipeline for a build.
//
// nolint: funlen // ignore function length - will be refactored at a later date
func (c *client) ExecBuild(ctx context.Context) error {
	b := c.build
	p := c.pipeline
	// r := c.repo
	e := c.err

	defer func() {
		// Overwrite with proper status and error only if build was not canceled
		if !strings.EqualFold(b.GetStatus(), constants.StatusCanceled) {
			// NOTE: if the build is already in a failure state we do not
			// want to update the state to be success
			if !strings.EqualFold(b.GetStatus(), constants.StatusFailure) {
				b.SetStatus(constants.StatusSuccess)
			}

			// NOTE: When an error occurs during a build that does not have to do
			// with a pipeline we should set build status to "error" not "failed"
			// because it is worker related and not build.
			if e != nil {
				b.SetError(e.Error())
				b.SetStatus(constants.StatusError)
			}
		}
		// update the build fields
		b.SetFinished(time.Now().UTC().Unix())
	}()

	// execute the services for the pipeline
	for _, s := range p.Services {
		// plan the service
		err := c.PlanService(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to plan service: %w", err)
		}

		// execute the service
		err = c.ExecService(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to execute service: %w", err)
		}
	}

	// execute the steps for the pipeline
	for _, s := range p.Steps {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// // extract rule data from build information
		// ruledata := &pipeline.RuleData{
		// 	Branch: b.GetBranch(),
		// 	Event:  b.GetEvent(),
		// 	Repo:   r.GetFullName(),
		// 	Status: b.GetStatus(),
		// }

		// // when tag event add tag information into ruledata
		// if strings.EqualFold(b.GetEvent(), constants.EventTag) {
		// 	ruledata.Tag = strings.TrimPrefix(c.build.GetRef(), "refs/tags/")
		// }

		// // when deployment event add deployment information into ruledata
		// if strings.EqualFold(b.GetEvent(), constants.EventDeploy) {
		// 	ruledata.Target = b.GetDeploy()
		// }

		// // check if you need to excute this step
		// if !s.Execute(ruledata) {
		// 	continue
		// }

		// plan the step
		err := c.PlanStep(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to plan step: %w", err)
		}

		// execute the step
		err = c.ExecStep(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to execute step: %w", err)
		}

		// load the init step from the client
		cStep, err := step.Load(s, &c.steps)
		if err != nil {
			return err
		}

		// check the step exit code
		if s.ExitCode != 0 {
			// check if we ignore step failures
			if !s.Ruleset.Continue {
				// set build status to failure
				b.SetStatus(constants.StatusFailure)
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
	for _, s := range p.Stages {
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
			err := c.PlanStage(stageCtx, stage, stageMap)
			if err != nil {
				e = err
				return fmt.Errorf("unable to plan stage: %w", err)
			}

			// execute the stage
			err = c.ExecStage(stageCtx, stage, stageMap)
			if err != nil {
				e = err
				return fmt.Errorf("unable to execute stage: %w", err)
			}

			return nil
		})
	}

	// wait for the stages to complete or return an error
	//
	// https://pkg.go.dev/golang.org/x/sync/errgroup?tab=doc#Group.Wait
	err := stages.Wait()
	if err != nil {
		e = err
		return fmt.Errorf("unable to wait for stages: %v", err)
	}

	return nil
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
			// TODO: Should this be changed or removed?
			fmt.Println(err)
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
			// TODO: Should this be changed or removed?
			fmt.Println(err)
		}
	}

	// destroy the services for the pipeline
	for _, s := range c.pipeline.Services {
		// destroy the service
		err = c.DestroyService(ctx, s)
		if err != nil {
			// TODO: Should this be changed or removed?
			fmt.Println(err)
		}
	}

	// remove the runtime volume for the pipeline
	err = c.Runtime.RemoveVolume(ctx, c.pipeline)
	if err != nil {
		// TODO: Should this be changed or removed?
		fmt.Println(err)
	}

	// remove the runtime network for the pipeline
	err = c.Runtime.RemoveNetwork(ctx, c.pipeline)
	if err != nil {
		// TODO: Should this be changed or removed?
		fmt.Println(err)
	}

	return err
}
