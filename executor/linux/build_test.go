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

func TestLinux_CreateBuild_Success(t *testing.T) {
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
		pipeline *pipeline.Build
	}{
		{ // pipeline with steps
			pipeline: &pipeline.Build{
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
			},
		},
		{ // pipeline with stages
			pipeline: &pipeline.Build{
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
			},
		},
	}

	// run test
	for _, test := range tests {
		e, err := New(
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

		err = e.CreateBuild(context.Background())
		if err != nil {
			t.Errorf("CreateBuild returned err: %v", err)
		}
	}
}

func TestLinux_ExecBuild_Success(t *testing.T) {
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
		pipeline *pipeline.Build
	}{
		{ // pipeline with steps
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
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
			},
		},
		{ // pipeline with stages
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
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
			},
		},
	}

	// run test
	for _, test := range tests {
		e, err := New(
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

		err = e.PlanBuild(context.Background())
		if err != nil {
			t.Errorf("PlanBuild returned err: %v", err)
		}

		err = e.ExecBuild(context.Background())
		if err != nil {
			t.Errorf("ExecBuild returned err: %v", err)
		}
	}
}

func TestLinux_DestroyBuild_Success(t *testing.T) {
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
		pipeline *pipeline.Build
	}{
		{ // pipeline with steps
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
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
			},
		},
		{ // pipeline with stages
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
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
			},
		},
	}

	// run test
	for _, test := range tests {
		e, err := New(
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

		svc := new(library.Service)
		svc.SetNumber(1)
		e.services.Store(e.pipeline.Services[0].ID, svc)

		err = e.DestroyBuild(context.Background())
		if err != nil {
			t.Errorf("DestroyBuild returned err: %v", err)
		}
	}
}
