// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"errors"
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
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_stages := testStages()

	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, "", nil)
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
						Pull:        "always",
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
						Pull:        "always",
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
						Image:       "target/vela-git:notfound",
						Name:        "clone",
						Number:      2,
						Pull:        "always",
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
						Image:       "target/vela-git:ignorenotfound",
						Name:        "clone",
						Number:      2,
						Pull:        "always",
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
			// run create to init steps to be created properly
			err = _engine.CreateBuild(context.Background())
		}

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

func TestLinux_PlanStage(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_stages := testStages()

	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, "", nil)
	if err != nil {
		t.Errorf("unable to create Vela API client: %v", err)
	}

	_runtime, err := docker.NewMock()
	if err != nil {
		t.Errorf("unable to create runtime engine: %v", err)
	}

	// setup tests
	tests := []struct {
		failure  bool
		err      error
		stageMap map[string]chan error
		stage    *pipeline.Stage
	}{
		{
			failure:  false,
			err:      nil,
			stageMap: map[string]chan error{},
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
						Pull:        "always",
					},
				},
			},
		},
		{
			failure: true,
			err:     errors.New("simulated error for stage"),
			stageMap: map[string]chan error{
				"init": make(chan error, 1),
			},
			stage: &pipeline.Stage{
				Name:  "clone",
				Needs: []string{"init"},
				Steps: pipeline.ContainerSlice{
					{
						ID:          "github_octocat_1_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        "always",
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

		if len(test.stageMap) > 0 {
			if test.err != nil {
				test.stageMap["init"] <- test.err
			}

			close(test.stageMap["init"])
		}

		err = _engine.PlanStage(context.Background(), test.stage, test.stageMap)

		if test.failure {
			if err == nil {
				t.Errorf("PlanStage should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("PlanStage returned err: %v", err)
		}
	}
}

func TestLinux_ExecStage(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_stages := testStages()

	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, "", nil)
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
						Pull:        "always",
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
						ID:          "github_octocat_1_bad_clone_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      0,
						Pull:        "always",
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
						Pull:        "always",
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

		_engine.steps.Store(_stages.Stages[0].Steps[0].ID, new(library.Step))
		_engine.stepLogs.Store(_stages.Stages[0].Steps[0].ID, new(library.Log))

		err = _engine.CreateStep(context.Background(), test.stage.Steps[0])
		if err != nil {
			t.Errorf("unable to create step: %v", err)
		}

		// create volume for runtime host config
		err = _runtime.CreateVolume(context.Background(), _stages)
		if err != nil {
			t.Errorf("unable to create runtime volume: %w", err)
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
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_stages := testStages()

	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, "", nil)
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
		step    *library.Step
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
						Pull:        "always",
					},
				},
			},
			step: new(library.Step),
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
						Pull:        "always",
					},
				},
			},
			step: new(library.Step),
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

		_engine.steps.Store(test.stage.Steps[0].ID, test.step)

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
