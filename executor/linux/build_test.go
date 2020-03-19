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

func TestLinux_CreateBuild(t *testing.T) {
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

	tests := []struct {
		failure  bool
		build    *library.Build
		pipeline *pipeline.Build
	}{
		{
			failure:  false,
			build:    _build,
			pipeline: _stages,
		},
		{
			failure:  false,
			build:    _build,
			pipeline: _steps,
		},
		{
			failure:  true,
			build:    new(library.Build),
			pipeline: _steps,
		},
	}

	// run test
	for _, test := range tests {
		_engine, err := New(
			WithBuild(test.build),
			WithPipeline(test.pipeline),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		err = _engine.CreateBuild(context.Background())

		if test.failure {
			if err == nil {
				t.Errorf("CreateBuild should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("CreateBuild returned err: %v", err)
		}
	}
}

func TestLinux_PlanBuild(t *testing.T) {
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

	tests := []struct {
		failure  bool
		pipeline *pipeline.Build
	}{
		{
			failure:  false,
			pipeline: _stages,
		},
		{
			failure:  false,
			pipeline: _steps,
		},
		{
			failure: true,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "github_octocat_1",
				Stages: pipeline.StageSlice{
					{
						Name: "init",
						Steps: pipeline.ContainerSlice{
							{
								ID:          "github_octocat_1_init_init",
								Directory:   "/home/github/octocat",
								Environment: map[string]string{"FOO": "bar"},
								Image:       "#init",
								Name:        "init",
								Number:      0,
								Pull:        true,
							},
						},
					},
				},
			},
		},
		{
			failure: true,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "github_octocat_1",
				Steps: pipeline.ContainerSlice{
					{
						ID:          "step_github_octocat_1_init",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "#init",
						Name:        "init",
						Number:      0,
						Pull:        true,
					},
				},
			},
		},
	}

	// run test
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(test.pipeline),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		err = _engine.PlanBuild(context.Background())

		if test.failure {
			if err == nil {
				t.Errorf("PlanBuild should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("PlanBuild returned err: %v", err)
		}
	}
}

func TestLinux_ExecBuild(t *testing.T) {
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

	tests := []struct {
		failure  bool
		pipeline *pipeline.Build
	}{
		{
			failure:  false,
			pipeline: _stages,
		},
		{
			failure:  false,
			pipeline: _steps,
		},
	}

	// run test
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(test.pipeline),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		err = _engine.PlanBuild(context.Background())
		if err != nil {
			t.Errorf("PlanBuild returned err: %v", err)
		}

		err = _engine.ExecBuild(context.Background())

		if test.failure {
			if err == nil {
				t.Errorf("ExecBuild should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("ExecBuild returned err: %v", err)
		}
	}
}

func TestLinux_DestroyBuild(t *testing.T) {
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

	tests := []struct {
		failure  bool
		pipeline *pipeline.Build
		service  *library.Service
	}{
		{
			failure:  false,
			pipeline: _stages,
			service: &library.Service{
				Name:   &_stages.Services[0].Name,
				Number: &_stages.Services[0].Number,
			},
		},
		{
			failure:  false,
			pipeline: _steps,
			service: &library.Service{
				Name:   &_steps.Services[0].Name,
				Number: &_steps.Services[0].Number,
			},
		},
		{
			failure:  true,
			pipeline: _steps,
			service:  nil,
		},
		{
			failure:  true,
			pipeline: new(pipeline.Build),
			service:  nil,
		},
	}

	// run test
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(test.pipeline),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		if test.service != nil {
			_engine.services.Store(_engine.pipeline.Services[0].ID, test.service)
		}

		err = _engine.DestroyBuild(context.Background())

		if test.failure {
			if err == nil {
				t.Errorf("DestroyBuild should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("DestroyBuild returned err: %v", err)
		}
	}
}
