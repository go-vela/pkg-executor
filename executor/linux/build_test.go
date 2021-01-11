// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"context"
	"flag"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/go-vela/compiler/compiler/native"
	"github.com/go-vela/mock/server"
	"github.com/urfave/cli/v2"

	"github.com/go-vela/pkg-runtime/runtime/docker"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/library"

	"github.com/gin-gonic/gin"
)

func TestLinux_CreateBuild(t *testing.T) {
	// setup types
	compiler, _ := native.New(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))

	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_metadata := testMetadata()

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
		pipeline string
	}{
		{ // basic steps pipeline
			failure:  false,
			build:    _build,
			pipeline: "testdata/build/steps/basic.yml",
		},
		{ // basic stages pipeline
			failure:  false,
			build:    _build,
			pipeline: "testdata/build/stages/basic.yml",
		},
		{ // pipeline with empty build
			failure:  true,
			build:    new(library.Build),
			pipeline: "testdata/build/steps/basic.yml",
		},
	}

	// run test
	for _, test := range tests {
		file, _ := ioutil.ReadFile(test.pipeline)

		p, _ := compiler.
			WithBuild(_build).
			WithRepo(_repo).
			WithUser(_user).
			WithMetadata(_metadata).
			Compile(file)

		_engine, err := New(
			WithBuild(test.build),
			WithPipeline(p),
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
	compiler, _ := native.New(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))

	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_metadata := testMetadata()

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
		pipeline string
	}{
		{ // basic steps pipeline
			failure:  false,
			pipeline: "testdata/build/steps/basic.yml",
		},
		{ // basic stages pipeline
			failure:  false,
			pipeline: "testdata/build/stages/basic.yml",
		},
	}

	// run test
	for _, test := range tests {
		file, _ := ioutil.ReadFile(test.pipeline)

		p, _ := compiler.
			WithBuild(_build).
			WithRepo(_repo).
			WithUser(_user).
			WithMetadata(_metadata).
			Compile(file)

		_engine, err := New(
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

		// run create to init steps to be created properly
		err = _engine.CreateBuild(context.Background())

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

func TestLinux_AssembleBuild(t *testing.T) {
	// setup types
	compiler, _ := native.New(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))

	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_metadata := testMetadata()

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
		pipeline string
	}{
		{ // basic steps pipeline
			failure:  false,
			pipeline: "testdata/build/steps/basic.yml",
		},
		{ // pipeline with steps image tag not found
			failure:  true,
			pipeline: "testdata/build/steps/img_notfound.yml",
		},
		{ // pipeline with steps image tag ignoring not found
			failure:  true,
			pipeline: "testdata/build/steps/img_ignorenotfound.yml",
		},
		{ // basic stages pipeline
			failure:  false,
			pipeline: "testdata/build/stages/basic.yml",
		},
		{ // pipeline with stages image tag not found
			failure:  true,
			pipeline: "testdata/build/stages/img_notfound.yml",
		},
		{ // pipeline with stages image tag ignoring not found
			failure:  true,
			pipeline: "testdata/build/stages/img_ignorenotfound.yml",
		},
		{ // pipeline with service image tag not found
			failure:  true,
			pipeline: "testdata/build/services/img_notfound.yml",
		},
		{ // pipeline with service image tag ignoring not found
			failure:  true,
			pipeline: "testdata/build/services/img_ignorenotfound.yml",
		},
		{ // pipeline with stages image tag not found
			failure:  true,
			pipeline: "testdata/build/secrets/img_notfound.yml",
		},
		{ // pipeline with stages image tag ignoring not found
			failure:  true,
			pipeline: "testdata/build/secrets/img_ignorenotfound.yml",
		},
	}

	// run test
	for _, test := range tests {
		file, _ := ioutil.ReadFile(test.pipeline)

		p, _ := compiler.
			WithBuild(_build).
			WithRepo(_repo).
			WithUser(_user).
			WithMetadata(_metadata).
			Compile(file)

		_engine, err := New(
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

		// run create to init steps to be created properly
		err = _engine.CreateBuild(context.Background())

		err = _engine.AssembleBuild(context.Background())

		if test.failure {
			if err == nil {
				t.Errorf("AssembleBuild should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("AssembleBuild returned err: %v", err)
		}
	}
}

func TestLinux_ExecBuild(t *testing.T) {
	// setup types
	compiler, _ := native.New(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))

	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_metadata := testMetadata()

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
		pipeline string
	}{
		{ // basic steps pipeline
			failure:  false,
			pipeline: "testdata/build/steps/basic.yml",
		},
		{ // pipeline with step image tag not found
			failure:  true,
			pipeline: "testdata/build/steps/img_notfound.yml",
		},
		{ // pipeline with step name not found
			failure:  true,
			pipeline: "testdata/build/steps/name_notfound.yml",
		},
		{ // basic stages pipeline
			failure:  false,
			pipeline: "testdata/build/stages/basic.yml",
		},
		{ // pipeline with stage step image tag not found
			failure:  true,
			pipeline: "testdata/build/stages/img_notfound.yml",
		},
		{ // pipeline with stage step name not found
			failure:  true,
			pipeline: "testdata/build/stages/name_notfound.yml",
		},
		{ // basic services pipeline
			failure:  false,
			pipeline: "testdata/build/services/basic.yml",
		},
		{ // pipeline with service image tag not found
			failure:  true,
			pipeline: "testdata/build/services/img_notfound.yml",
		},
		{ // pipeline with service name not found
			failure:  true,
			pipeline: "testdata/build/services/name_notfound.yml",
		},
		{ // basic secrets pipeline
			failure:  false,
			pipeline: "testdata/build/secrets/basic.yml",
		},
	}

	// run test
	for _, test := range tests {
		file, _ := ioutil.ReadFile(test.pipeline)

		p, _ := compiler.
			WithBuild(_build).
			WithRepo(_repo).
			WithUser(_user).
			WithMetadata(_metadata).
			Compile(file)

		_engine, err := New(
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

		for _, service := range p.Services {
			s := &library.Service{
				Name:   &service.Name,
				Number: &service.Number,
			}

			_engine.services.Store(service.ID, s)
			_engine.serviceLogs.Store(service.ID, new(library.Log))
		}

		for _, stage := range p.Stages {
			for _, step := range stage.Steps {
				s := &library.Step{
					Name:   &step.Name,
					Number: &step.Number,
				}

				_engine.steps.Store(step.ID, s)
				_engine.stepLogs.Store(step.ID, new(library.Log))
			}
		}

		for _, step := range p.Steps {
			s := &library.Step{
				Name:   &step.Name,
				Number: &step.Number,
			}

			_engine.steps.Store(step.ID, s)
			_engine.stepLogs.Store(step.ID, new(library.Log))
		}

		// run create to init steps to be created properly
		err = _engine.CreateBuild(context.Background())
		if err != nil {
			t.Errorf("unable to create build: %v", err)
		}

		// run plan to create network and volume
		err = _engine.PlanBuild(context.Background())
		if err != nil {
			t.Errorf("unable to create build: %v", err)
		}

		// TODO: hack - remove this
		//
		// When calling CreateBuild(), it will automatically set the
		// test build object to a status of `created`. This happens
		// because we use a mock for the go-vela/server in our tests
		// which only returns dummy based responses.
		//
		// The problem this causes is that our container.Execute()
		// function isn't setup to handle builds in a `created` state.
		//
		// In a real world scenario, we never would have a build
		// in this state when we call ExecBuild() because the
		// go-vela/server has logic to set it to an expected state.
		_engine.build.SetStatus("running")

		err = _engine.ExecBuild(context.Background())

		if test.failure {
			if err == nil {
				t.Errorf("ExecBuild for %s should have returned err", test.pipeline)
			}

			continue
		}

		if err != nil {
			t.Errorf("ExecBuild for %s returned err: %v", test.pipeline, err)
		}
	}
}

func TestLinux_DestroyBuild(t *testing.T) {
	// setup types
	compiler, _ := native.New(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))

	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_metadata := testMetadata()

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
		pipeline string
	}{
		// { // pipeline empty
		// 	failure:  true,
		// 	pipeline:     "testdata/build/empty.yml",
		// },
		{ // basic steps pipeline
			failure:  false,
			pipeline: "testdata/build/steps/basic.yml",
		},
		{ // pipeline with step image tag not found
			failure:  false,
			pipeline: "testdata/build/steps/img_notfound.yml",
		},
		{ // basic stages pipeline
			failure:  false,
			pipeline: "testdata/build/stages/basic.yml",
		},
		{ // pipeline with stage step image tag not found
			failure:  false,
			pipeline: "testdata/build/stages/img_notfound.yml",
		},
		{ // basic services pipeline
			failure:  false,
			pipeline: "testdata/build/services/basic.yml",
		},
		{ // pipeline with service image tag not found
			failure:  false,
			pipeline: "testdata/build/services/img_notfound.yml",
		},
		{ // basic stages pipeline
			failure:  false,
			pipeline: "testdata/build/secrets/basic.yml",
		},
		{ // pipeline with secret image tag not found
			failure:  false,
			pipeline: "testdata/build/secrets/img_notfound.yml",
		},
	}

	// run test
	for _, test := range tests {
		file, _ := ioutil.ReadFile(test.pipeline)

		p, _ := compiler.
			WithBuild(_build).
			WithRepo(_repo).
			WithUser(_user).
			WithMetadata(_metadata).
			Compile(file)

		_engine, err := New(
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

		// run create to init steps to be created properly
		err = _engine.CreateBuild(context.Background())

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
