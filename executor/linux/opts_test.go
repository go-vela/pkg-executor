// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/go-vela/mock/server"

	"github.com/go-vela/pkg-runtime/runtime/docker"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/pipeline"
)

func TestLinux_Opt_WithBuild(t *testing.T) {
	// run test
	e, err := New(
		WithBuild(_build),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	if e.build != _build {
		t.Errorf("WithBuild is %v, want %v", e.build, _build)
	}
}

func TestLinux_Opt_WithPipeline(t *testing.T) {
	// setup types
	_pipeline := &pipeline.Build{
		Version: "1",
		ID:      "__0",
		Steps: pipeline.ContainerSlice{
			{
				ID:          "__0_clone",
				Environment: map[string]string{},
				Image:       "target/vela-git:latest",
				Name:        "clone",
				Number:      1,
				Pull:        true,
			},
		},
	}

	e, err := New(
		WithPipeline(_pipeline),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	if e.pipeline != _pipeline {
		t.Errorf("WithPipeline is %v, want %v", e.pipeline, _pipeline)
	}
}

func TestLinux_Opt_WithRepo(t *testing.T) {
	// run test
	e, err := New(
		WithRepo(_repo),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	if e.repo != _repo {
		t.Errorf("WithRepo is %v, want %v", e.repo, _repo)
	}
}

func TestLinux_Opt_WithRuntime(t *testing.T) {
	// setup types
	_runtime, _ := docker.NewMock()

	// run test
	e, err := New(
		WithRuntime(_runtime),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	if e.Runtime != _runtime {
		t.Errorf("WithRuntime is %v, want %v", e.Runtime, _runtime)
	}
}

func TestLinux_Opt_WithUser(t *testing.T) {
	// run test
	e, err := New(
		WithUser(_user),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	if e.user != _user {
		t.Errorf("WithUser is %v, want %v", e.user, _user)
	}
}

func TestLinux_Opt_WithVelaClient(t *testing.T) {
	// setup types
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())

	_client, err := vela.NewClient(s.URL, nil)
	if err != nil {
		t.Errorf("unable to create Vela API client: %v", err)
	}

	e, err := New(
		WithVelaClient(_client),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	if e.Vela != _client {
		t.Errorf("WithVelaClient is %v, want %v", e.Vela, _client)
	}
}
