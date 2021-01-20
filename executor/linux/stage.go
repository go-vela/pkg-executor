// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/pipeline"
)

// CreateStage prepares the stage for execution.
func (c *client) CreateStage(ctx context.Context, s *pipeline.Stage) error {
	// load the logs for the init step from the client
	_log, err := step.LoadLogs(c.init, &c.stepLogs)
	if err != nil {
		return err
	}

	// update engine logger with stage metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("stage", s.Name)

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	_log.AppendData([]byte(fmt.Sprintf("> Pulling step images for stage %s...\n", s.Name)))

	// create the steps for the stage
	for _, _step := range s.Steps {
		logger.Debugf("creating %s step", _step.Name)
		// create the step
		err := c.CreateStep(ctx, _step)
		if err != nil {
			return err
		}

		logger.Infof("inspecting image for %s step", _step.Name)
		// inspect the step image
		image, err := c.Runtime.InspectImage(ctx, _step)
		if err != nil {
			return err
		}

		// update the init log with step image info
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		_log.AppendData(image)
	}

	return nil
}

// PlanStage prepares the stage for execution.
func (c *client) PlanStage(ctx context.Context, s *pipeline.Stage, m map[string]chan error) error {
	// update engine logger with stage metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("stage", s.Name)

	logger.Debug("gathering stage dependency tree")
	// ensure dependent stages have completed
	for _, needs := range s.Needs {
		logger.Debugf("looking up dependency %s", needs)
		// check if a dependency stage has completed
		stageErr, ok := m[needs]
		if !ok { // stage not found so we continue
			continue
		}

		logger.Debugf("waiting for dependency %s", needs)
		// wait for the stage channel to close
		select {
		case <-ctx.Done():
			return fmt.Errorf("errgroup context is done")
		case err := <-stageErr:
			if err != nil {
				logger.Errorf("%s stage returned error: %v", needs, err)
				return err
			}

			continue
		}
	}

	return nil
}

// ExecStage runs a stage.
func (c *client) ExecStage(ctx context.Context, s *pipeline.Stage, m map[string]chan error) error {
	// update engine logger with stage metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("stage", s.Name)

	// close the stage channel at the end
	defer close(m[s.Name])

	logger.Debug("starting execution of stage")
	// execute the steps for the stage
	for _, _step := range s.Steps {
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
		if !_step.Execute(ruledata) {
			continue
		}

		logger.Debugf("planning %s step", _step.Name)
		// plan the step
		err := c.PlanStep(ctx, _step)
		if err != nil {
			return fmt.Errorf("unable to plan step %s: %w", _step.Name, err)
		}

		logger.Infof("executing %s step", _step.Name)
		// execute the step
		err = c.ExecStep(ctx, _step)
		if err != nil {
			return err
		}

		// load the step from the client
		cStep, err := step.Load(_step, &c.steps)
		if err != nil {
			return err
		}

		// check the step exit code
		if _step.ExitCode != 0 {
			// check if we ignore step failures
			if !_step.Ruleset.Continue {
				// set build status to failure
				c.build.SetStatus(constants.StatusFailure)
			}

			// update the step fields
			cStep.SetExitCode(_step.ExitCode)
			cStep.SetStatus(constants.StatusFailure)
		}

		cStep.SetFinished(time.Now().UTC().Unix())
		logger.Infof("uploading %s step state", _step.Name)
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err = c.Vela.Step.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), cStep)
		if err != nil {
			return fmt.Errorf("unable to upload step state: %v", err)
		}
	}

	return nil
}

// DestroyStage cleans up the stage after execution.
func (c *client) DestroyStage(ctx context.Context, s *pipeline.Stage) error {
	// update engine logger with stage metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("stage", s.Name)

	// destroy the steps for the stage
	for _, _step := range s.Steps {
		logger.Debugf("destroying %s step", _step.Name)
		// destroy the step
		err := c.DestroyStep(ctx, _step)
		if err != nil {
			return err
		}
	}

	return nil
}
