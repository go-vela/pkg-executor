// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

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
	l, err := step.LoadLogs(c.pipeline.Stages[0].Steps[0], &c.stepLogs)
	if err != nil {
		return err
	}

	// update the init log with progress
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData([]byte(fmt.Sprintf("> Pulling step images for stage %s...\n", s.Name)))

	// create the steps for the stage
	for _, step := range s.Steps {
		// TODO: make this not hardcoded
		// update the init log with progress
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		l.AppendData([]byte(fmt.Sprintf("$ docker image inspect %s\n", step.Image)))

		// create the step
		err := c.CreateStep(ctx, step)
		if err != nil {
			return err
		}

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
	b := c.build
	r := c.repo

	// close the stage channel at the end
	defer close(m[s.Name])

	// execute the steps for the stage
	for _, _step := range s.Steps {
		// extract rule data from build information
		ruledata := &pipeline.RuleData{
			Branch: b.GetBranch(),
			Event:  b.GetEvent(),
			Repo:   r.GetFullName(),
			Status: b.GetStatus(),
		}

		// when tag event add tag information into ruledata
		if strings.EqualFold(b.GetEvent(), constants.EventTag) {
			ruledata.Tag = strings.TrimPrefix(c.build.GetRef(), "refs/tags/")
		}

		// when deployment event add deployment information into ruledata
		if strings.EqualFold(b.GetEvent(), constants.EventDeploy) {
			ruledata.Target = b.GetDeploy()
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
				b.SetStatus(constants.StatusFailure)
			}

			// update the step fields
			cStep.SetExitCode(_step.ExitCode)
			cStep.SetStatus(constants.StatusFailure)
		}

		cStep.SetFinished(time.Now().UTC().Unix())
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
	// destroy the steps for the stage
	for _, step := range s.Steps {
		// destroy the step
		err := c.DestroyStep(ctx, step)
		if err != nil {
			return err
		}
	}

	return nil
}
