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

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

func TestLinux_CreateStage(t *testing.T) {
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

	// setup tests
	tests := []struct {
		failure bool
		logs    *library.Log
		stage   *pipeline.Stage
	}{
		{
			failure: false,
			logs:    new(library.Log),
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
		{
			failure: true,
			logs:    nil,
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
		{
			failure: true,
			logs:    new(library.Log),
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      0,
						Pull:        true,
					},
				},
			},
		},
		{
			failure: true,
			logs:    new(library.Log),
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "!@#$%^&*()",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
	}

	// run tests
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_stages),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		if test.logs != nil {
			_engine.stepLogs.Store(_stages.Stages[0].Steps[0].ID, test.logs)
		}

		_engine.steps.Store(_stages.Stages[0].Steps[0].ID, new(library.Step))

		err = _engine.CreateStage(context.Background(), test.stage)

		if test.failure {
			if err == nil {
				t.Errorf("CreateStage should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("CreateStage returned err: %v", err)
		}
	}
}

func TestLinux_ExecStage(t *testing.T) {
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

	// setup tests
	tests := []struct {
		failure bool
		stage   *pipeline.Stage
	}{
		{
			failure: false,
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
		{
			failure: true,
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "!@#$%^&*()",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
		{
			failure: true,
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_notfound",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "notfound",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
	}

	// run tests
	for _, test := range tests {
		stageMap := make(map[string]chan error)
		stageMap["init"] = make(chan error)
		stageMap["clone"] = make(chan error)
		stageMap["echo"] = make(chan error)

		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_stages),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		_engine.stepLogs.Store(_stages.Stages[0].Steps[0].ID, new(library.Log))
		_engine.steps.Store(_stages.Stages[0].Steps[0].ID, new(library.Step))

		err = _engine.CreateStep(context.Background(), test.stage.Steps[0])
		if err != nil {
			t.Errorf("unable to create step: %v", err)
		}

		err = _engine.ExecStage(context.Background(), test.stage, stageMap)

		if test.failure {
			if err == nil {
				t.Errorf("ExecStage should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("ExecStage returned err: %v", err)
		}
	}
}

func TestLinux_DestroyStage(t *testing.T) {
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

	// setup tests
	tests := []struct {
		failure bool
		stage   *pipeline.Stage
	}{
		{
			failure: false,
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
		{
			failure: true,
			stage: &pipeline.Stage{
				Name: "clone",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_notfound",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "notfound",
						Number:      2,
						Pull:        true,
					},
				},
			},
		},
	}

	// run tests
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_stages),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		err = _engine.DestroyStage(context.Background(), test.stage)

		if test.failure {
			if err == nil {
				t.Errorf("DestroyStage should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("DestroyStage returned err: %v", err)
		}
	}
}
