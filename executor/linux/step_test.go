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

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

func TestLinux_CreateStep(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		failure   bool
		container *pipeline.Container
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_init",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "#init",
				Name:        "init",
				Number:      1,
				Pull:        true,
			},
		},
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      2,
				Pull:        true,
			},
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:notfound",
				Name:        "clone",
				Number:      2,
				Pull:        true,
			},
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:        "step_github_octocat_1_clone",
				Commands:  []string{"echo", "${BAR}", "${FOO}"},
				Directory: "/home/github/octocat",
				Environment: map[string]string{
					"BAR": "1\n2\n",
					"FOO": "!@#$%^&*()\\",
				},
				Image:  "target/vela-git:v0.3.0",
				Name:   "clone",
				Number: 2,
				Pull:   true,
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

		err = _engine.CreateStep(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("CreateStep should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("CreateStep returned err: %v", err)
		}
	}
}

func TestLinux_PlanStep(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		failure   bool
		container *pipeline.Container
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      2,
				Pull:        true,
			},
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      0,
				Pull:        true,
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

		err = _engine.PlanStep(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("PlanStep should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("PlanStep returned err: %v", err)
		}
	}
}

func TestLinux_ExecStep(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		failure   bool
		container *pipeline.Container
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_init",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "#init",
				Name:        "init",
				Number:      1,
				Pull:        true,
			},
		},
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      2,
				Pull:        true,
			},
		},
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Detach:      true,
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      2,
				Pull:        true,
			},
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_notfound",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "notfound",
				Number:      2,
				Pull:        true,
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

		_engine.stepLogs.Store(test.container.ID, new(library.Log))
		_engine.steps.Store(test.container.ID, new(library.Step))

		err = _engine.ExecStep(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("ExecStep should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("ExecStep returned err: %v", err)
		}
	}
}

func TestLinux_StreamStep(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		failure   bool
		logs      *library.Log
		container *pipeline.Container
	}{
		{
			failure: false,
			logs:    new(library.Log),
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_init",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "#init",
				Name:        "init",
				Number:      1,
				Pull:        true,
			},
		},
		{
			failure: true,
			logs:    nil,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      2,
				Pull:        true,
			},
		},
		{
			failure: true,
			logs:    new(library.Log),
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_notfound",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "notfound",
				Number:      2,
				Pull:        true,
			},
		},
		{
			failure: true,
			logs:    new(library.Log),
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      0,
				Pull:        true,
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

		if test.logs != nil {
			_engine.stepLogs.Store(test.container.ID, test.logs)
		}

		_engine.steps.Store(test.container.ID, new(library.Step))

		err = _engine.StreamStep(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("StreamStep should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("StreamStep returned err: %v", err)
		}
	}
}

func TestLinux_DestroyStep(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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

	_step := new(library.Step)
	_step.SetName("clone")
	_step.SetNumber(2)
	_step.SetStatus(constants.StatusPending)

	// setup tests
	tests := []struct {
		failure   bool
		container *pipeline.Container
		step      *library.Step
	}{
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_init",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "#init",
				Name:        "init",
				Number:      1,
				Pull:        true,
			},
			step: new(library.Step),
		},
		{
			failure: false,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_clone",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "clone",
				Number:      2,
				Pull:        true,
			},
			step: _step,
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_notfound",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "notfound",
				Number:      2,
				Pull:        true,
			},
			step: new(library.Step),
		},
		{
			failure: true,
			container: &pipeline.Container{
				ID:          "step_github_octocat_1_ignorenotfound",
				Directory:   "/home/github/octocat",
				Environment: map[string]string{"FOO": "bar"},
				Image:       "target/vela-git:v0.3.0",
				Name:        "ignorenotfound",
				Number:      2,
				Pull:        true,
			},
			step: new(library.Step),
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

		_engine.steps.Store(test.container.ID, test.step)

		err = _engine.DestroyStep(context.Background(), test.container)

		if test.failure {
			if err == nil {
				t.Errorf("DestroyStep should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("DestroyStep returned err: %v", err)
		}
	}
}

func TestLinux_loadStep(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		name    string
		value   interface{}
	}{
		{
			failure: false,
			name:    "step_github_octocat_1_init",
			value:   new(library.Step),
		},
		{
			failure: true,
			name:    "step_github_octocat_1_init",
			value:   nil,
		},
		{
			failure: true,
			name:    "step_github_octocat_1_init",
			value:   new(library.Log),
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

		if test.value != nil {
			_engine.steps.Store(test.name, test.value)
		}

		_, err = _engine.loadStep(test.name)

		if test.failure {
			if err == nil {
				t.Errorf("loadStep should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("loadStep returned err: %v", err)
		}
	}
}

func TestLinux_loadStepLogs(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_steps := testSteps()

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
		name    string
		value   interface{}
	}{
		{
			failure: false,
			name:    "step_github_octocat_1_init",
			value:   new(library.Log),
		},
		{
			failure: true,
			name:    "step_github_octocat_1_init",
			value:   nil,
		},
		{
			failure: true,
			name:    "step_github_octocat_1_init",
			value:   new(library.Step),
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

		if test.value != nil {
			_engine.stepLogs.Store(test.name, test.value)
		}

		_, err = _engine.loadStepLogs(test.name)

		if test.failure {
			if err == nil {
				t.Errorf("loadStepLogs should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("loadStepLogs returned err: %v", err)
		}
	}
}
