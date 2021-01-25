// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"fmt"

	"github.com/go-vela/pkg-executor/internal/step"
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
		// update the container environment with stage name
		_step.Environment["VELA_STEP_STAGE"] = s.Name

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
		// check if you need to skip executing this step
		if step.Skip(_step, c.build, c.repo) {
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
			return fmt.Errorf("unable to exec step %s: %w", _step.Name, err)
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
