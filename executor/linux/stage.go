// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/pipeline"
)

// CreateStage prepares the stage for execution.
func (c *client) CreateStage(ctx context.Context, s *pipeline.Stage) error {
	// load the logs for the init step from the client
	l, err := c.loadStepLogs(c.pipeline.Stages[0].Steps[0].ID)
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
	l.AppendData([]byte(fmt.Sprintf("  $ Pulling step images for stage %s...\n", s.Name)))

	// create the steps for the stage
	for _, step := range s.Steps {
		// TODO: make this not hardcoded
		// update the init log with progress
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		l.AppendData([]byte(fmt.Sprintf("    $ docker image inspect %s\n", step.Image)))

		logger.Debugf("creating %s step", step.Name)
		// create the step
		err := c.CreateStep(ctx, step)
		if err != nil {
			return err
		}

		c.logger.Infof("inspecting image %s step", step.Name)
		// inspect the step image
		image, err := c.Runtime.InspectImage(ctx, step)
		if err != nil {
			return err
		}

		// update the init log with step image info
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		l.AppendData(image)
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
	b := c.build
	r := c.repo

	// update engine logger with stage metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("stage", s.Name)

	// close the stage channel at the end
	defer close(m[s.Name])

	logger.Debug("starting execution of stage")
	// execute the steps for the stage
	for _, step := range s.Steps {
		// assume you will excute a step by setting flag
		disregard := false

		// check if the build status is successful
		if !strings.EqualFold(b.GetStatus(), constants.StatusSuccess) {
			// disregard the need to run the step
			disregard = true

			// check if you need to run a status failure ruleset
			if !(step.Ruleset.If.Empty() && step.Ruleset.Unless.Empty()) &&
				step.Ruleset.Match(&pipeline.RuleData{Status: b.GetStatus()}) {
				// approve the need to run the step
				disregard = false
			}
		}

		// check if you need to skip a status failure ruleset
		if strings.EqualFold(b.GetStatus(), constants.StatusSuccess) &&
			!(step.Ruleset.If.Empty() && step.Ruleset.Unless.Empty()) &&
			step.Ruleset.Match(&pipeline.RuleData{Status: constants.StatusFailure}) {
			// disregard the need to run the step
			disregard = true
		}

		// check if you need to excute this step
		if disregard {
			continue
		}

		logger.Debugf("planning %s step", step.Name)
		// plan the step
		err := c.PlanStep(ctx, step)
		if err != nil {
			return fmt.Errorf("unable to plan step %s: %w", step.Name, err)
		}

		logger.Infof("executing %s step", step.Name)
		// execute the step
		err = c.ExecStep(ctx, step)
		if err != nil {
			return err
		}

		// load the step from the client
		cStep, err := c.loadStep(step.ID)
		if err != nil {
			return err
		}

		// check the step exit code
		if step.ExitCode != 0 {
			// check if we ignore step failures
			if !step.Ruleset.Continue {
				// set build status to failure
				b.SetStatus(constants.StatusFailure)
			}

			// update the step fields
			cStep.SetExitCode(step.ExitCode)
			cStep.SetStatus(constants.StatusFailure)
		}

		cStep.SetFinished(time.Now().UTC().Unix())
		logger.Infof("uploading %s step state", step.Name)
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err = c.Vela.Step.Update(r.GetOrg(), r.GetName(), b.GetNumber(), cStep)
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
	for _, step := range s.Steps {
		logger.Debugf("destroying %s step", step.Name)
		// destroy the step
		err := c.DestroyStep(ctx, step)
		if err != nil {
			return err
		}
	}

	return nil
}
