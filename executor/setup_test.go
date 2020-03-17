// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package executor

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/go-vela/mock/server"

	"github.com/go-vela/pkg-executor/executor/linux"

	"github.com/go-vela/pkg-runtime/runtime/docker"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/constants"
)

func TestExecutor_Setup_Darwin(t *testing.T) {
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

	_setup := &Setup{
		Build:    _build,
		Client:   _client,
		Driver:   constants.DriverDarwin,
		Pipeline: _pipeline,
		Repo:     _repo,
		Runtime:  _runtime,
		User:     _user,
	}

	got, err := _setup.Darwin()
	if err == nil {
		t.Errorf("Darwin should have returned err")
	}

	if got != nil {
		t.Errorf("Darwin is %v, want nil", got)
	}
}

func TestExecutor_Setup_Linux(t *testing.T) {
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

	want, err := linux.New(
		linux.WithBuild(_build),
		linux.WithPipeline(_pipeline),
		linux.WithRepo(_repo),
		linux.WithRuntime(_runtime),
		linux.WithUser(_user),
		linux.WithVelaClient(_client),
	)
	if err != nil {
		t.Errorf("unable to create linux engine: %v", err)
	}

	_setup := &Setup{
		Build:    _build,
		Client:   _client,
		Driver:   constants.DriverLinux,
		Pipeline: _pipeline,
		Repo:     _repo,
		Runtime:  _runtime,
		User:     _user,
	}

	got, err := _setup.Linux()
	if err != nil {
		t.Errorf("Linux returned err: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Linux is %v, want %v", got, want)
	}
}

func TestExecutor_Setup_Windows(t *testing.T) {
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

	_setup := &Setup{
		Build:    _build,
		Client:   _client,
		Driver:   constants.DriverWindows,
		Pipeline: _pipeline,
		Repo:     _repo,
		Runtime:  _runtime,
		User:     _user,
	}

	got, err := _setup.Windows()
	if err == nil {
		t.Errorf("Windows should have returned err")
	}

	if got != nil {
		t.Errorf("Windows is %v, want nil", got)
	}
}

func TestExecutor_Setup_Validate(t *testing.T) {
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
		setup   *Setup
		failure bool
	}{
		{
			setup: &Setup{
				Build:    _build,
				Client:   _client,
				Driver:   constants.DriverLinux,
				Pipeline: _pipeline,
				Repo:     _repo,
				Runtime:  _runtime,
				User:     _user,
			},
			failure: false,
		},
		{
			setup: &Setup{
				Build:    nil,
				Client:   _client,
				Driver:   constants.DriverLinux,
				Pipeline: _pipeline,
				Repo:     _repo,
				Runtime:  _runtime,
				User:     _user,
			},
			failure: true,
		},
		{
			setup: &Setup{
				Build:    _build,
				Client:   nil,
				Driver:   constants.DriverLinux,
				Pipeline: _pipeline,
				Repo:     _repo,
				Runtime:  _runtime,
				User:     _user,
			},
			failure: true,
		},
		{
			setup: &Setup{
				Build:    _build,
				Client:   _client,
				Driver:   "",
				Pipeline: _pipeline,
				Repo:     _repo,
				Runtime:  _runtime,
				User:     _user,
			},
			failure: true,
		},
		{
			setup: &Setup{
				Build:    _build,
				Client:   _client,
				Driver:   constants.DriverLinux,
				Pipeline: nil,
				Repo:     _repo,
				Runtime:  _runtime,
				User:     _user,
			},
			failure: true,
		},
		{
			setup: &Setup{
				Build:    _build,
				Client:   _client,
				Driver:   constants.DriverLinux,
				Pipeline: _pipeline,
				Repo:     nil,
				Runtime:  _runtime,
				User:     _user,
			},
			failure: true,
		},
		{
			setup: &Setup{
				Build:    _build,
				Client:   _client,
				Driver:   constants.DriverLinux,
				Pipeline: _pipeline,
				Repo:     _repo,
				Runtime:  nil,
				User:     _user,
			},
			failure: true,
		},
		{
			setup: &Setup{
				Build:    _build,
				Client:   _client,
				Driver:   constants.DriverLinux,
				Pipeline: _pipeline,
				Repo:     _repo,
				Runtime:  _runtime,
				User:     nil,
			},
			failure: true,
		},
	}

	// run tests
	for _, test := range tests {
		err = test.setup.Validate()

		if test.failure {
			if err == nil {
				t.Errorf("Validate should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("Validate returned err: %v", err)
		}
	}
}
