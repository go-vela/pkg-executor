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

	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/pipeline"
)

// CreateStage prepares the stage for execution.
func (c *client) CreateStage(ctx context.Context, s *pipeline.Stage) error {
	// create a step pattern for log output
	_pattern := fmt.Sprintf(stepPattern, c.init.Name)

	// output init progress to stdout
	fmt.Fprintln(os.Stdout, _pattern, "> Pulling step images for stage", s.Name, "...")

	// create the steps for the stage
	for _, _step := range s.Steps {
		// create the step
		err := c.CreateStep(ctx, _step)
		if err != nil {
			return err
		}

		// output image command to stdout
		fmt.Fprintln(os.Stdout, _pattern, "$ docker image inspect", _step.Image)

		// inspect the step image
		image, err := c.Runtime.InspectImage(ctx, _step)
		if err != nil {
			return err
		}

		// output the image information to stdout
		fmt.Fprintln(os.Stdout, _pattern, string(image))
	}

	return nil
}

// PlanStage prepares the stage for execution.
func (c *client) PlanStage(ctx context.Context, s *pipeline.Stage, m map[string]chan error) error {
	// ensure dependent stages have completed
	for _, needs := range s.Needs {
		// check if a dependency stage has completed
		stageErr, ok := m[needs]
		if !ok { // stage not found so we continue
			continue
		}

		// wait for the stage channel to close
		select {
		case <-ctx.Done():
			return fmt.Errorf("errgroup context is done")
		case err := <-stageErr:
			if err != nil {
				return err
			}

			continue
		}
	}

	return nil
}

// ExecStage runs a stage.
func (c *client) ExecStage(ctx context.Context, s *pipeline.Stage, m map[string]chan error) error {
	// close the stage channel at the end
	defer close(m[s.Name])

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

		// plan the step
		err := c.PlanStep(ctx, _step)
		if err != nil {
			return fmt.Errorf("unable to plan step %s: %w", _step.Name, err)
		}

		// execute the step
		err = c.ExecStep(ctx, _step)
		if err != nil {
			return fmt.Errorf("unable to exec step %s: %w", _step.Name, err)
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
	}

	return nil
}

// DestroyStage cleans up the stage after execution.
func (c *client) DestroyStage(ctx context.Context, s *pipeline.Stage) error {
	// destroy the steps for the stage
	for _, _step := range s.Steps {
		// destroy the step
		err := c.DestroyStep(ctx, _step)
		if err != nil {
			return err
		}
	}

	return nil
}
