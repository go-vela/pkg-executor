// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-vela/pkg-executor/internal/build"
	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
)

// CreateBuild configures the build for execution.
func (c *client) CreateBuild(ctx context.Context) error {
	// defer taking a snapshot of the build
	defer build.Snapshot(c.build, c.Vela, c.err, c.logger, c.repo)

	// update the build fields
	c.build.SetStatus(constants.StatusRunning)
	c.build.SetStarted(time.Now().UTC().Unix())
	c.build.SetHost(c.Hostname)
	c.build.SetDistribution(constants.DriverLinux)
	c.build.SetRuntime(c.Runtime.Driver())

	c.logger.Info("uploading build state")
	// send API call to update the build
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#BuildService.Update
	c.build, _, c.err = c.Vela.Build.Update(c.repo.GetOrg(), c.repo.GetName(), c.build)
	if c.err != nil {
		return fmt.Errorf("unable to upload build state: %v", c.err)
	}

	// load the init container from the pipeline
	c.init = c.loadInitContainer(c.pipeline)

	c.logger.Infof("creating %s step", c.init.Name)
	// create the step
	c.err = c.CreateStep(ctx, c.init)
	if c.err != nil {
		return fmt.Errorf("unable to create %s step: %w", c.init.Name, c.err)
	}

	c.logger.Infof("planning %s step", c.init.Name)
	// plan the step
	c.err = c.PlanStep(ctx, c.init)
	if c.err != nil {
		return fmt.Errorf("unable to plan %s step: %w", c.init.Name, c.err)
	}

	return c.err
}

// PlanBuild prepares the build for execution.
//
// nolint: funlen // ignore function length due to comments and logging messages
func (c *client) PlanBuild(ctx context.Context) error {
	// defer taking a snapshot of the build
	defer build.Snapshot(c.build, c.Vela, c.err, c.logger, c.repo)

	// load the init step from the client
	_init, err := step.Load(c.init, &c.steps)
	if err != nil {
		return err
	}

	// load the logs for the init step from the client
	_log, err := step.LoadLogs(c.init, &c.stepLogs)
	if err != nil {
		return err
	}

	defer func() {
		c.logger.Infof("uploading %s step state", c.init.Name)
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err := c.Vela.Step.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), _init)
		if err != nil {
			c.logger.Errorf("unable to upload %s state: %v", c.init.Name, err)
		}

		c.logger.Infof("uploading %s step logs", c.init.Name)
		// send API call to update the logs for the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
		_log, _, err = c.Vela.Log.UpdateStep(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), c.init.Number, _log)
		if err != nil {
			c.logger.Errorf("unable to upload %s logs: %v", c.init.Name, err)
		}
	}()

	c.logger.Info("creating network")
	// create the runtime network for the pipeline
	c.err = c.Runtime.CreateNetwork(ctx, c.pipeline)
	if c.err != nil {
		return fmt.Errorf("unable to create network: %w", c.err)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Inspecting runtime network...\n"))

	// inspect the runtime network for the pipeline
	network, err := c.Runtime.InspectNetwork(ctx, c.pipeline)
	if err != nil {
		c.err = err
		return fmt.Errorf("unable to inspect network: %w", err)
	}

	// update the init log with network information
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData(network)

	c.logger.Info("creating volume")
	// create the runtime volume for the pipeline
	c.err = c.Runtime.CreateVolume(ctx, c.pipeline)
	if c.err != nil {
		return fmt.Errorf("unable to create volume: %w", c.err)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Inspecting runtime volume...\n"))

	// inspect the runtime volume for the pipeline
	volume, err := c.Runtime.InspectVolume(ctx, c.pipeline)
	if err != nil {
		c.err = err
		return fmt.Errorf("unable to inspect volume: %w", err)
	}

	// update the init log with volume information
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData(volume)

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Pulling secrets...\n"))

	// iterate through each secret provided in the pipeline
	for _, secret := range c.pipeline.Secrets {
		// ignore pulling secrets coming from plugins
		if !secret.Origin.Empty() {
			continue
		}

		c.logger.Infof("pulling %s %s secret %s", secret.Engine, secret.Type, secret.Name)

		s, err := c.secret.pull(secret)
		if err != nil {
			c.err = err
			return fmt.Errorf("unable to pull secrets: %w", err)
		}

		_log.AppendData([]byte(
			fmt.Sprintf("$ vela view secret --secret.engine %s --secret.type %s --org %s --repo %s --name %s \n",
				secret.Engine, secret.Type, s.GetOrg(), s.GetRepo(), s.GetName())))

		sRaw, err := json.MarshalIndent(s.Sanitize(), "", " ")
		if err != nil {
			c.err = err
			return fmt.Errorf("unable to decode secret: %w", err)
		}

		_log.AppendData(append(sRaw, "\n"...))

		// add secret to the map
		c.Secrets[secret.Name] = s
	}

	return nil
}

// AssembleBuild prepares the containers within a build for execution.
//
// nolint: funlen // ignore function length due to comments and logging messages
func (c *client) AssembleBuild(ctx context.Context) error {
	// defer taking a snapshot of the build
	defer build.Snapshot(c.build, c.Vela, c.err, c.logger, c.repo)

	// load the init step from the client
	_init, err := step.Load(c.init, &c.steps)
	if err != nil {
		return err
	}

	// load the logs for the init step from the client
	_log, err := step.LoadLogs(c.init, &c.stepLogs)
	if err != nil {
		return err
	}

	defer func() {
		_init.SetFinished(time.Now().UTC().Unix())

		c.logger.Infof("uploading %s step state", c.init.Name)
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err := c.Vela.Step.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), _init)
		if err != nil {
			c.logger.Errorf("unable to upload %s state: %v", c.init.Name, err)
		}

		c.logger.Infof("uploading %s step logs", c.init.Name)
		// send API call to update the logs for the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
		_log, _, err = c.Vela.Log.UpdateStep(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), c.init.Number, _log)
		if err != nil {
			c.logger.Errorf("unable to upload %s logs: %v", c.init.Name, err)
		}
	}()

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Pulling service images...\n"))

	// create the services for the pipeline
	for _, s := range c.pipeline.Services {
		// TODO: remove this; but we need it for tests
		s.Detach = true

		c.logger.Infof("creating %s service", s.Name)
		// create the service
		c.err = c.CreateService(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to create %s service: %w", s.Name, c.err)
		}

		c.logger.Infof("inspecting %s service", s.Name)
		// inspect the service image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			c.err = err
			return fmt.Errorf("unable to inspect %s service: %w", s.Name, err)
		}

		// update the init log with service image info
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		_log.AppendData(image)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Pulling stage images...\n"))

	// create the stages for the pipeline
	for _, s := range c.pipeline.Stages {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		c.logger.Infof("creating %s stage", s.Name)
		// create the stage
		c.err = c.CreateStage(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to create %s stage: %w", s.Name, c.err)
		}
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Pulling step images...\n"))

	// create the steps for the pipeline
	for _, s := range c.pipeline.Steps {
		// TODO: remove hardcoded reference
		if s.Name == "init" {
			continue
		}

		c.logger.Infof("creating %s step", s.Name)
		// create the step
		c.err = c.CreateStep(ctx, s)
		if c.err != nil {
			return fmt.Errorf("unable to create %s step: %w", s.Name, c.err)
		}

		c.logger.Infof("inspecting %s step", s.Name)
		// inspect the step image
		image, err := c.Runtime.InspectImage(ctx, s)
		if err != nil {
			c.err = err
			return fmt.Errorf("unable to inspect %s step: %w", s.Name, c.err)
		}

		// update the init log with step image info
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		_log.AppendData(image)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Pulling secret images...\n"))

	// create the secrets for the pipeline
	for _, s := range c.pipeline.Secrets {
		// skip over non-plugin secrets
		if s.Origin.Empty() {
			continue
		}

		c.logger.Infof("creating %s secret", s.Origin.Name)
		// create the service
		c.err = c.secret.create(ctx, s.Origin)
		if c.err != nil {
			return fmt.Errorf("unable to create %s secret: %w", s.Origin.Name, c.err)
		}

		c.logger.Infof("inspecting %s secret", s.Origin.Name)
		// inspect the service image
		image, err := c.Runtime.InspectImage(ctx, s.Origin)
		if err != nil {
			c.err = err
			return fmt.Errorf("unable to inspect %s secret: %w", s.Origin.Name, err)
		}

		// update the init log with secret image info
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		_log.AppendData(image)
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte("> Executing secret images...\n"))

	c.logger.Info("executing secret images")
	// execute the secret
	c.err = c.secret.exec(ctx, &c.pipeline.Secrets)
	if c.err != nil {
		return fmt.Errorf("unable to execute secret: %w", c.err)
	}

	return c.err
}

// ExecBuild runs a pipeline for a build.
//
// nolint: funlen // ignore function length due to comments and log messages
func (c *client) ExecBuild(ctx context.Context) error {
	// defer an upload of the build
	defer build.Upload(c.build, c.Vela, c.err, c.logger, c.repo)

	// execute the services for the pipeline
	for _, _service := range c.pipeline.Services {
		c.logger.Infof("planning %s service", _service.Name)
		// plan the service
		c.err = c.PlanService(ctx, _service)
		if c.err != nil {
			return fmt.Errorf("unable to plan service: %w", c.err)
		}

		c.logger.Infof("executing %s service", _service.Name)
		// execute the service
		c.err = c.ExecService(ctx, _service)
		if c.err != nil {
			return fmt.Errorf("unable to execute service: %w", c.err)
		}
	}

	// execute the steps for the pipeline
	for _, _step := range c.pipeline.Steps {
		// TODO: remove hardcoded reference
		if _step.Name == "init" {
			continue
		}

		// check if you need to skip executing this step
		if step.Skip(_step, c.build, c.repo) {
			continue
		}

		c.logger.Infof("planning %s step", _step.Name)
		// plan the step
		c.err = c.PlanStep(ctx, _step)
		if c.err != nil {
			return fmt.Errorf("unable to plan step: %w", c.err)
		}

		c.logger.Infof("executing %s step", _step.Name)
		// execute the step
		c.err = c.ExecStep(ctx, _step)
		if c.err != nil {
			return fmt.Errorf("unable to execute step: %w", c.err)
		}
	}

	// create an error group with the context for each stage
	//
	// https://pkg.go.dev/golang.org/x/sync/errgroup?tab=doc#WithContext
	stages, stageCtx := errgroup.WithContext(ctx)
	// create a map to track the progress of each stage
	stageMap := make(map[string]chan error)

	// iterate through each stage in the pipeline
	for _, _stage := range c.pipeline.Stages {
		// TODO: remove hardcoded reference
		if _stage.Name == "init" {
			continue
		}

		// https://golang.org/doc/faq#closures_and_goroutines
		stage := _stage

		// create a new channel for each stage in the map
		stageMap[stage.Name] = make(chan error)

		// spawn errgroup routine for the stage
		//
		// https://pkg.go.dev/golang.org/x/sync/errgroup?tab=doc#Group.Go
		stages.Go(func() error {
			c.logger.Infof("planning %s stage", stage.Name)
			// plan the stage
			c.err = c.PlanStage(stageCtx, stage, stageMap)
			if c.err != nil {
				return fmt.Errorf("unable to plan stage: %w", c.err)
			}

			c.logger.Infof("executing %s stage", stage.Name)
			// execute the stage
			c.err = c.ExecStage(stageCtx, stage, stageMap)
			if c.err != nil {
				return fmt.Errorf("unable to execute stage: %w", c.err)
			}

			return nil
		})
	}

	c.logger.Debug("waiting for stages completion")
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
	for _, _step := range c.pipeline.Steps {
		// TODO: remove hardcoded reference
		if _step.Name == "init" {
			continue
		}

		c.logger.Infof("destroying %s step", _step.Name)
		// destroy the step
		err = c.DestroyStep(ctx, _step)
		if err != nil {
			c.logger.Errorf("unable to destroy step: %v", err)
		}
	}

	// destroy the stages for the pipeline
	for _, _stage := range c.pipeline.Stages {
		// TODO: remove hardcoded reference
		if _stage.Name == "init" {
			continue
		}

		c.logger.Infof("destroying %s stage", _stage.Name)
		// destroy the stage
		err = c.DestroyStage(ctx, _stage)
		if err != nil {
			c.logger.Errorf("unable to destroy stage: %v", err)
		}
	}

	// destroy the services for the pipeline
	for _, _service := range c.pipeline.Services {
		c.logger.Infof("destroying %s service", _service.Name)
		// destroy the service
		err = c.DestroyService(ctx, _service)
		if err != nil {
			c.logger.Errorf("unable to destroy service: %v", err)
		}
	}

	// destroy the secrets for the pipeline
	for _, _secret := range c.pipeline.Secrets {
		// skip over non-plugin secrets
		if _secret.Origin.Empty() {
			continue
		}

		c.logger.Infof("destroying %s secret", _secret.Name)
		// destroy the secret
		err = c.secret.destroy(ctx, _secret.Origin)
		if err != nil {
			c.logger.Errorf("unable to destroy secret: %v", err)
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
