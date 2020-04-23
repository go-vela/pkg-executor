// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/pipeline"
)

// CreateBuild configures the build for execution.
func (c *client) CreateBuild(ctx context.Context) error {
	b := c.build
	p := c.pipeline
	r := c.repo
	e := c.err

	defer func() {
		// NOTE: When an error occurs during a build that does not have to do
		// with a pipeline we should set build status to "error" not "failed"
		// because it is worker related and not build.
		if e != nil {
			b.SetError(e.Error())
			b.SetStatus(constants.StatusError)
			b.SetFinished(time.Now().UTC().Unix())
		}

		c.logger.Info("uploading build snapshot")
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#BuildService.Update
		_, _, err := c.Vela.Build.Update(r.GetOrg(), r.GetName(), b)
		if err != nil {
			c.logger.Errorf("unable to upload build snapshot: %v", err)
		}
	}()

	// update the build fields
	b.SetStatus(constants.StatusRunning)
	b.SetStarted(time.Now().UTC().Unix())
	b.SetHost(c.Hostname)
	// TODO: This should not be hardcoded
	b.SetDistribution("linux")
	b.SetRuntime("docker")

	c.logger.Info("uploading build state")
	// send API call to update the build
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#BuildService.Update
	b, _, err := c.Vela.Build.Update(r.GetOrg(), r.GetName(), b)
	if err != nil {
		e = err
		return fmt.Errorf("unable to upload build state: %v", err)
	}

	c.build = b

	// TODO: Pull this out into a the plan function for steps.
	c.logger.Info("pulling secrets")
	// pull secrets for the build
	err = c.PullSecret(ctx)
	if err != nil {
		e = err
		return fmt.Errorf("unable to pull secrets: %v", err)
	}

	// TODO: make this better
	init := new(pipeline.Container)
	if len(p.Steps) > 0 {
		init = p.Steps[0]
	}

	// TODO: make this better
	if len(p.Stages) > 0 {
		init = p.Stages[0].Steps[0]
	}

	c.logger.Infof("creating %s step", init.Name)
	// create the step
	err = c.CreateStep(ctx, init)
	if err != nil {
		e = err
		return fmt.Errorf("unable to create %s step: %w", init.Name, err)
	}

	c.logger.Infof("planning %s step", init.Name)
	// plan the step
	err = c.PlanStep(ctx, init)
	if err != nil {
		e = err
		return fmt.Errorf("unable to plan %s step: %w", init.Name, err)
	}

	return nil
}

// PlanBuild prepares the build for execution.
func (c *client) PlanBuild(ctx context.Context) error {
	b := c.build
	p := c.pipeline
	r := c.repo
	e := c.err

	defer func() {
		// NOTE: When an error occurs during a build that does not have to do
		// with a pipeline we should set build status to "error" not "failed"
		// because it is worker related and not build.
		if e != nil {
			b.SetError(e.Error())
			b.SetStatus(constants.StatusError)
			b.SetFinished(time.Now().UTC().Unix())
		}

		c.logger.Info("uploading build snapshot")
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#BuildService.Update
		_, _, err := c.Vela.Build.Update(r.GetOrg(), r.GetName(), b)
		if err != nil {
			c.logger.Errorf("unable to upload build snapshot: %v", err)
		}
	}()

	// TODO: make this better
	init := new(pipeline.Container)
	if len(p.Steps) > 0 {
		init = p.Steps[0]
	}
	// TODO: make this better
	if len(p.Stages) > 0 {
		init = p.Stages[0].Steps[0]
	}

	// load the init step from the client
	s, err := c.loadStep(init.ID)
	if err != nil {
		return err
	}

	// load the logs for the init step from the client
	l, err := c.loadStepLogs(init.ID)
	if err != nil {
		return err
	}

	defer func() {
		s.SetFinished(time.Now().UTC().Unix())
		c.logger.Infof("uploading %s step state", init.Name)
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err := c.Vela.Step.Update(r.GetOrg(), r.GetName(), b.GetNumber(), s)
		if err != nil {
			c.logger.Errorf("unable to upload %s state: %v", init.Name, err)
		}

		c.logger.Infof("uploading %s step logs", init.Name)
		// send API call to update the logs for the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
		l, _, err = c.Vela.Log.UpdateStep(r.GetOrg(), r.GetName(), b.GetNumber(), init.Number, l)
		if err != nil {
			c.logger.Errorf("unable to upload %s logs: %v", init.Name, err)
		}
	}()

	c.logger.Info("creating network")
	// create the runtime network for the pipeline
	err = c.Runtime.CreateNetwork(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to create network: %w", err)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData([]byte("$ Inspecting runtime network...\n"))

	// inspect the runtime network for the pipeline
	network, err := c.Runtime.InspectNetwork(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to inspect network: %w", err)
	}

	// update the init log with network info
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData(network)

	c.logger.Info("creating volume")
	// create the runtime volume for the pipeline
	err = c.Runtime.CreateVolume(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to create volume: %w", err)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData([]byte("$ Inspecting runtime volume...\n"))

	// inspect the runtime volume for the pipeline
	volume, err := c.Runtime.InspectVolume(ctx, p)
	if err != nil {
		e = err
		return fmt.Errorf("unable to inspect volume: %w", err)
	}

	// update the init log with volume info
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData(volume)

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData([]byte("$ Pulling service images...\n"))

	// create the services for the pipeline
	for _, s := range p.Services {
		// TODO: remove this; but we need it for tests
		s.Detach = true
		s.Pull = true

		// TODO: remove hardcoded reference
		// update the init log with progress
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		l.AppendData([]byte(fmt.Sprintf("  $ docker image inspect %s\n", s.Image)))

		c.logger.Infof("creating %s service", s.Name)
		// create the service
		err = c.CreateService(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to create %s service: %w", s.Name, err)
		}

		c.logger.Infof("inspecting %s service", s.Name)
		// inspect the service image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to inspect %s service: %w", s.Name, err)
		}

		// update the init log with service image info
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		l.AppendData(image)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData([]byte("$ Pulling stage images...\n"))

	// create the stages for the pipeline
	for _, s := range p.Stages {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		c.logger.Infof("creating %s stage", s.Name)
		// create the stage
		err = c.CreateStage(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to create %s stage: %w", s.Name, err)
		}
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData([]byte("$ Pulling step images...\n"))

	// create the steps for the pipeline
	for _, s := range p.Steps {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		// TODO: make this not hardcoded
		// update the init log with progress
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		l.AppendData([]byte(fmt.Sprintf("  $ docker image inspect %s\n", s.Image)))

		c.logger.Infof("creating %s step", s.Name)
		// create the step
		err = c.CreateStep(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to create %s step: %w", s.Name, err)
		}

		c.logger.Infof("inspecting %s step", s.Name)
		// inspect the step image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to inspect %s step: %w", s.Name, err)
		}

		// update the init log with step image info
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		l.AppendData(image)
	}

	return nil
}

// ExecBuild runs a pipeline for a build.
func (c *client) ExecBuild(ctx context.Context) error {
	b := c.build
	p := c.pipeline
	r := c.repo
	e := c.err

	b.SetStatus(constants.StatusSuccess)
	c.build = b

	defer func() {
		// NOTE: When an error occurs during a build that does not have to do
		// with a pipeline we should set build status to "error" not "failed"
		// because it is worker related and not build.
		if e != nil {
			b.SetError(e.Error())
			b.SetStatus(constants.StatusError)
		}

		// update the build fields
		b.SetFinished(time.Now().UTC().Unix())

		c.logger.Info("uploading build state")
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#BuildService.Update
		_, _, err := c.Vela.Build.Update(r.GetOrg(), r.GetName(), b)
		if err != nil {
			c.logger.Errorf("unable to upload errorred state: %v", err)
		}
	}()

	// execute the services for the pipeline
	for _, s := range p.Services {
		c.logger.Infof("planning %s service", s.Name)
		// plan the service
		err := c.PlanService(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to plan service: %w", err)
		}

		c.logger.Infof("executing %s service", s.Name)
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

		switch b.GetStatus() {
		case constants.StatusSuccess:
		// do nothing, and continue running the build
		case constants.StatusFailure:
			fallthrough
		case constants.StatusError:
			fallthrough
		default:
			// check if you need to run a status failure ruleset
			rules := strings.Join(s.Ruleset.If.Status, ", ")
			if !strings.Contains(rules, constants.StatusFailure) {
				continue
			}
		}

		c.logger.Infof("planning %s step", s.Name)
		// plan the step
		err := c.PlanStep(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to plan step: %w", err)
		}

		c.logger.Infof("executing %s step", s.Name)
		// execute the step
		err = c.ExecStep(ctx, s)
		if err != nil {
			e = err
			return fmt.Errorf("unable to execute step: %w", err)
		}

		// load the step from the client
		cStep, err := c.loadStep(s.ID)
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
		c.logger.Infof("uploading %s step state", s.Name)
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err = c.Vela.Step.Update(r.GetOrg(), r.GetName(), b.GetNumber(), cStep)
		if err != nil {
			e = err
			return fmt.Errorf("unable to upload step state: %v", err)
		}
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
			c.logger.Infof("planning %s stage", stage.Name)
			// plan the stage
			err := c.PlanStage(stageCtx, stage, stageMap)
			if err != nil {
				e = err
				return fmt.Errorf("unable to plan stage: %w", err)
			}

			c.logger.Infof("executing %s stage", stage.Name)
			// execute the stage
			err = c.ExecStage(stageCtx, stage, stageMap)
			if err != nil {
				e = err
				return fmt.Errorf("unable to execute stage: %w", err)
			}

			fmt.Println("STAGE FINISHED: ", stage.Name)
			return nil
		})
	}

	c.logger.Debug("waiting for stages completion")
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

		c.logger.Infof("destroying %s step", s.Name)
		// destroy the step
		err = c.DestroyStep(ctx, s)
		if err != nil {
			c.logger.Errorf("unable to destroy step: %v", err)
		}
	}

	// destroy the stages for the pipeline
	for _, s := range c.pipeline.Stages {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		c.logger.Infof("destroying %s stage", s.Name)
		// destroy the stage
		err = c.DestroyStage(ctx, s)
		if err != nil {
			c.logger.Errorf("unable to destroy stage: %v", err)
		}
	}

	// destroy the services for the pipeline
	for _, s := range c.pipeline.Services {
		c.logger.Infof("destroying %s service", s.Name)
		// destroy the service
		err = c.DestroyService(ctx, s)
		if err != nil {
			c.logger.Errorf("unable to destroy service: %v", err)
		}
	}

	c.logger.Info("deleting volume")
	// remove the runtime volume for the pipeline
	err = c.Runtime.RemoveVolume(ctx, c.pipeline)
	if err != nil {
		c.logger.Errorf("unable to remove volume: %v", err)
	}

	c.logger.Info("deleting network")
	// remove the runtime network for the pipeline
	err = c.Runtime.RemoveNetwork(ctx, c.pipeline)
	if err != nil {
		c.logger.Errorf("unable to remove network: %v", err)
	}

	return err
}
