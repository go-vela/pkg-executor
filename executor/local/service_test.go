// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/go-vela/mock/server"

	"github.com/go-vela/pkg-runtime/runtime/docker"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

func TestLocal_CreateService(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		failure   bool
		container *pipeline.Container
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_postgres",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_postgres",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:notfound",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
		},
	}

	// run tests
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_steps),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		err = _engine.CreateService(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("CreateService should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("CreateService returned err: %v", err)
		}
	}
}

func TestLocal_PlanService(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		failure   bool
		container *pipeline.Container
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_postgres",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
		},
	}

	// run tests
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_steps),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		err = _engine.PlanService(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("PlanService should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("PlanService returned err: %v", err)
		}
	}
}

func TestLocal_ExecService(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		failure   bool
		container *pipeline.Container
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_postgres",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_notfound",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "notfound",
				Number:      2,
				Pull:        "always",
			},
		},
	}

	// run tests
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_steps),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		_engine.services.Store(test.container.ID, new(library.Service))

		err = _engine.ExecService(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("ExecService should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("ExecService returned err: %v", err)
		}
	}
}

func TestLocal_StreamService(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

	_runtime, err := docker.NewMock()
	if err != nil {
		t.Errorf("unable to create runtime engine: %v", err)
	}

	// setup tests
	tests := []struct {
		failure   bool
		container *pipeline.Container
	}{
		{ // container step succeeds
			failure: false,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_postgres",
				Directory:   "/vela/src/vcs.company.com/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
		},
		{ // container step fails because of invalid container id
			failure: true,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_notfound",
				Directory:   "/vela/src/vcs.company.com/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "notfound",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
		},
	}

	// run tests
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_steps),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		err = _engine.StreamService(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("StreamService should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("StreamService returned err: %v", err)
		}
	}
}

func TestLocal_DestroyService(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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

	_service := new(library.Service)
	_service.SetName("postgres")
	_service.SetNumber(1)
	_service.SetStatus(constants.StatusPending)

	// setup tests
	tests := []struct {
		failure   bool
		container *pipeline.Container
		service   *library.Service
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_postgres",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
			service: _service,
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_notfound",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "notfound",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
			service: new(library.Service),
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "service_github_octocat_1_ignorenotfound",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "postgres:12-alpine",
				Name:        "ignorenotfound",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        "not_present",
			},
			service: new(library.Service),
		},
	}

	// run tests
	for _, test := range tests {
		_engine, err := New(
			WithBuild(_build),
			WithPipeline(_steps),
			WithRepo(_repo),
			WithRuntime(_runtime),
			WithUser(_user),
			WithVelaClient(_client),
		)
		if err != nil {
			t.Errorf("unable to create executor engine: %v", err)
		}

		_engine.services.Store(test.container.ID, test.service)

		err = _engine.DestroyService(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("DestroyService should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("DestroyService returned err: %v", err)
		}
	}
}
