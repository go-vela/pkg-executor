// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/go-vela/mock/server"

	"github.com/go-vela/pkg-runtime/runtime/docker"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/pipeline"
)

func TestExecutor_CreateStage_Success(t *testing.T) {
	// setup types
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, nil)
	if err != nil {
		t.Errorf("unable to create Vela API client: %v", err)
	}

	_runtime, err := docker.NewMock()
	if err != nil {
		t.Errorf("unable to create runtime engine: %v", err)
	}

	p := &pipeline.Build{
		Version: "1",
		ID:      "__0",
		Services: pipeline.ContainerSlice{
			{
				ID:          "service_org_repo_0_postgres;",
				Environment: map[string]string{},
				Image:       "postgres:11-alpine",
				Name:        "postgres",
				Ports:       []string{"5432:5432"},
			},
		},
		Stages: pipeline.StageSlice{
			{
				Name: "init",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_init_init",
						Environment: map[string]string{},
						Image:       "#init",
						Name:        "init",
						Number:      1,
						Pull:        true,
					},
				},
			},
			{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_clone_clone",
						Environment: map[string]string{},
						Image:       "target/vela-plugins/git:1",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
			},
			{
				Name:  "exit",
				Needs: []string{"clone"},
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_exit_exit",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "exit",
						Number:      3,
						Pull:        true,
						Ruleset: pipeline.Ruleset{
							Continue: true,
						},
						Commands: []string{"exit 1"},
					},
				},
			},
			{
				Name:  "echo",
				Needs: []string{"clone"},
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_echo_echo",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "echo",
						Number:      4,
						Pull:        true,
						Secrets: pipeline.StepSecretSlice{
							&pipeline.StepSecret{
								Source: "foobar",
								Target: "foobar",
							},
						},
					},
				},
			},
		},
	}

	e, err := New(
		WithBuild(_build),
		WithPipeline(p),
		WithRepo(_repo),
		WithRuntime(_runtime),
		WithUser(_user),
		WithVelaClient(_client),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
	}

	// run test
	err = e.CreateStep(context.Background(), e.pipeline.Stages[0].Steps[0])
	if err != nil {
		t.Errorf("unable to create init step: %v", err)
	}

	err = e.CreateStage(context.Background(), e.pipeline.Stages[1])
	if err != nil {
		t.Errorf("CreateStage returned err: %v", err)
	}
}

func TestExecutor_ExecStage_Success(t *testing.T) {
	// setup types
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, nil)
	if err != nil {
		t.Errorf("unable to create Vela API client: %v", err)
	}

	_runtime, err := docker.NewMock()
	if err != nil {
		t.Errorf("unable to create runtime engine: %v", err)
	}

	stageMap := make(map[string]chan error)
	stageMap["clone"] = make(chan error)
	stageMap["exit"] = make(chan error)
	stageMap["echo"] = make(chan error)

	p := &pipeline.Build{
		Version: "1",
		ID:      "__0",
		Services: pipeline.ContainerSlice{
			{
				ID:          "service_org_repo_0_postgres;",
				Environment: map[string]string{},
				Image:       "postgres:11-alpine",
				Name:        "postgres",
				Ports:       []string{"5432:5432"},
			},
		},
		Stages: pipeline.StageSlice{
			{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_clone_clone",
						Environment: map[string]string{},
						Image:       "target/vela-plugins/git:1",
						Name:        "clone",
						Number:      1,
						Pull:        true,
					},
				},
			},
			{
				Name:  "exit",
				Needs: []string{"clone"},
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_exit_exit",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "exit",
						Number:      2,
						Pull:        true,
						Ruleset: pipeline.Ruleset{
							Continue: true,
						},
						Commands: []string{"exit 1"},
					},
				},
			},
			{
				Name:  "echo",
				Needs: []string{"clone"},
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_echo_echo",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "echo",
						Number:      1,
						Pull:        true,
						Secrets: pipeline.StepSecretSlice{
							{
								Source: "foobar",
								Target: "foobar",
							},
						},
					},
				},
			},
		},
	}

	e, err := New(
		WithBuild(_build),
		WithPipeline(p),
		WithRepo(_repo),
		WithRuntime(_runtime),
		WithUser(_user),
		WithVelaClient(_client),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	// run test
	err = e.CreateStep(context.Background(), e.pipeline.Stages[0].Steps[0])
	if err != nil {
		t.Errorf("unable to create init step: %v", err)
	}

	err = e.CreateStage(context.Background(), e.pipeline.Stages[0])
	if err != nil {
		t.Errorf("CreateStage returned err: %v", err)
	}

	err = e.ExecStage(context.Background(), e.pipeline.Stages[0], stageMap)
	if err != nil {
		t.Errorf("ExecStage returned err: %v", err)
	}
}

func TestExecutor_DestroyStage_Success(t *testing.T) {
	// setup types
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, nil)
	if err != nil {
		t.Errorf("unable to create Vela API client: %v", err)
	}

	_runtime, err := docker.NewMock()
	if err != nil {
		t.Errorf("unable to create runtime engine: %v", err)
	}

	p := &pipeline.Build{
		Version: "1",
		ID:      "__0",
		Services: pipeline.ContainerSlice{
			{
				ID:          "service_org_repo_0_postgres;",
				Environment: map[string]string{},
				Image:       "postgres:11-alpine",
				Name:        "postgres",
				Ports:       []string{"5432:5432"},
			},
		},
		Stages: pipeline.StageSlice{
			{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_clone_clone",
						Environment: map[string]string{},
						Image:       "target/vela-plugins/git:1",
						Name:        "clone",
						Number:      1,
						Pull:        true,
					},
				},
			},
			{
				Name:  "exit",
				Needs: []string{"clone"},
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_exit_exit",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "exit",
						Number:      2,
						Pull:        true,
						Ruleset: pipeline.Ruleset{
							Continue: true,
						},
						Commands: []string{"exit 1"},
					},
				},
			},
			{
				Name:  "echo",
				Needs: []string{"clone"},
				Steps: pipeline.ContainerSlice{
					{
						ID:          "__0_echo_echo",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "echo",
						Number:      1,
						Pull:        true,
						Secrets: pipeline.StepSecretSlice{
							{
								Source: "foobar",
								Target: "foobar",
							},
						},
					},
				},
			},
		},
	}

	e, err := New(
		WithBuild(_build),
		WithPipeline(p),
		WithRepo(_repo),
		WithRuntime(_runtime),
		WithUser(_user),
		WithVelaClient(_client),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	// run test
	err = e.DestroyStage(context.Background(), e.pipeline.Stages[0])
	if err != nil {
		t.Errorf("DestroyStage returned err: %v", err)
	}
}
