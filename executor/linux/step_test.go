// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/go-vela/mock/server"

	"github.com/go-vela/pkg-runtime/runtime/docker"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"

	"github.com/gin-gonic/gin"
)

func TestExecutor_CreateStep_Success(t *testing.T) {
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
		Steps: pipeline.ContainerSlice{
			{
				ID:          "__0_clone",
				Environment: map[string]string{},
				Image:       "target/vela-plugins/git:1",
				Name:        "clone",
				Number:      1,
				Pull:        true,
			},
			{
				ID:          "__0_exit",
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
			{
				ID:          "__0_echo",
				Environment: map[string]string{},
				Image:       "alpine:latest",
				Name:        "echo",
				Number:      2,
				Pull:        true,
				Commands:    []string{"echo ${FOOBAR}"},
				Secrets: pipeline.StepSecretSlice{
					{
						Source: "foobar",
						Target: "foobar",
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
	got := e.CreateStep(context.Background(), e.pipeline.Steps[0])

	if got != nil {
		t.Errorf("CreateStep is %v, want nil", got)
	}
}

func TestExecutor_PlanStep_Success(t *testing.T) {
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
		Steps: pipeline.ContainerSlice{
			{
				ID:          "__0_clone",
				Environment: map[string]string{},
				Image:       "target/vela-plugins/git:1",
				Name:        "clone",
				Number:      1,
				Pull:        true,
			},
			{
				ID:          "__0_exit",
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
			{
				ID:          "__0_echo",
				Environment: map[string]string{},
				Image:       "alpine:latest",
				Name:        "echo",
				Number:      2,
				Pull:        true,
				Commands:    []string{"echo ${FOOBAR}"},
				Secrets: pipeline.StepSecretSlice{
					{
						Source: "foobar",
						Target: "foobar",
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
	err = e.PlanStep(context.Background(), e.pipeline.Steps[0])
	if err != nil {
		t.Errorf("PlanStep returned err: %v", err)
	}
}

func TestExecutor_ExecStep_Success(t *testing.T) {
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
		Steps: pipeline.ContainerSlice{
			{
				ID:          "__0_clone",
				Environment: map[string]string{},
				Image:       "target/vela-plugins/git:1",
				Name:        "clone",
				Number:      1,
				Pull:        true,
			},
			{
				ID:          "__0_exit",
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
			{
				ID:          "__0_echo",
				Environment: map[string]string{},
				Image:       "alpine:latest",
				Name:        "echo",
				Number:      2,
				Pull:        true,
				Commands:    []string{"echo ${FOOBAR}"},
				Secrets: pipeline.StepSecretSlice{
					{
						Source: "foobar",
						Target: "foobar",
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

	e.stepLogs.Store(e.pipeline.Steps[0].ID, new(library.Log))
	e.steps.Store(e.pipeline.Steps[0].ID, new(library.Step))

	// run test
	err = e.ExecStep(context.Background(), e.pipeline.Steps[0])
	if err != nil {
		t.Errorf("ExecStep returned err: %v", err)
	}
}

func TestExecutor_DestroyStep_Success(t *testing.T) {
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
		Steps: pipeline.ContainerSlice{
			{
				ID:          "__0_clone",
				Environment: map[string]string{},
				Image:       "target/vela-plugins/git:1",
				Name:        "clone",
				Number:      1,
				Pull:        true,
			},
			{
				ID:          "__0_exit",
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
			{
				ID:          "__0_echo",
				Environment: map[string]string{},
				Image:       "alpine:latest",
				Name:        "echo",
				Number:      2,
				Pull:        true,
				Commands:    []string{"echo ${FOOBAR}"},
				Secrets: pipeline.StepSecretSlice{
					{
						Source: "foobar",
						Target: "foobar",
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
	err = e.DestroyStep(context.Background(), e.pipeline.Steps[0])
	if err != nil {
		t.Errorf("DestroyStep returned err: %v", err)
	}
}
