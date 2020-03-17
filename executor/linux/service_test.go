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

func TestLinux_CreateService_Success(t *testing.T) {
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
				ID:          "service_org_repo_0_postgres",
				Environment: map[string]string{},
				Image:       "postgres:11-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				Pull:        true,
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
	err = e.CreateService(context.Background(), e.pipeline.Services[0])
	if err != nil {
		t.Errorf("CreateService returned err: %v", err)
	}
}

func TestLinux_PlanService_Success(t *testing.T) {
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
				ID:          "service_org_repo_0_postgres",
				Environment: map[string]string{},
				Image:       "postgres:11-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
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
	err = e.PlanService(context.Background(), e.pipeline.Services[0])
	if err != nil {
		t.Errorf("PlanService returned err: %v", err)
	}
}

func TestLinux_ExecService_Success(t *testing.T) {
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
				ID:          "service_org_repo_0_postgres",
				Environment: map[string]string{},
				Image:       "postgres:11-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
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

	e.serviceLogs.Store(e.pipeline.Services[0].ID, new(library.Log))
	e.services.Store(e.pipeline.Services[0].ID, new(library.Service))

	// run test
	err = e.ExecService(context.Background(), e.pipeline.Services[0])
	if err != nil {
		t.Errorf("ExecService returned err: %v", err)
	}
}

func TestLinux_DestroyService_Success(t *testing.T) {
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
				ID:          "service_org_repo_0_postgres",
				Environment: map[string]string{},
				Image:       "postgres:11-alpine",
				Name:        "postgres",
				Number:      1,
				Ports:       []string{"5432:5432"},
				ExitCode:    0,
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
	err = e.DestroyService(context.Background(), e.pipeline.Services[0])
	if err != nil {
		t.Errorf("DestroyService returned err: %v", err)
	}
}
