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

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

func TestLinux_Opt_WithBuild(t *testing.T) {
	// setup types
	_build := &library.Build{
		Number:       vela.Int(1),
		Parent:       vela.Int(1),
		Event:        vela.String("push"),
		Status:       vela.String("success"),
		Error:        vela.String(""),
		Enqueued:     vela.Int64(1563474077),
		Created:      vela.Int64(1563474076),
		Started:      vela.Int64(1563474078),
		Finished:     vela.Int64(1563474079),
		Deploy:       vela.String(""),
		Clone:        vela.String("https://github.com/github/octocat.git"),
		Source:       vela.String("https://github.com/github/octocat/abcdefghi123456789"),
		Title:        vela.String("push received from https://github.com/github/octocat"),
		Message:      vela.String("First commit..."),
		Commit:       vela.String("48afb5bdc41ad69bf22588491333f7cf71135163"),
		Sender:       vela.String("OctoKitty"),
		Author:       vela.String("OctoKitty"),
		Branch:       vela.String("master"),
		Ref:          vela.String("refs/heads/master"),
		BaseRef:      vela.String(""),
		Host:         vela.String("example.company.com"),
		Runtime:      vela.String("docker"),
		Distribution: vela.String("linux"),
	}

	e, err := New(
		WithBuild(_build),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
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
		t.Errorf("unable to create executor client: %v", err)
	}

	if e.pipeline != _pipeline {
		t.Errorf("WithPipeline is %v, want %v", e.pipeline, _pipeline)
	}
}

func TestLinux_Opt_WithRepo(t *testing.T) {
	// setup types
	_repo := &library.Repo{
		Org:         vela.String("github"),
		Name:        vela.String("octocat"),
		FullName:    vela.String("github/octocat"),
		Link:        vela.String("https://github.com/github/octocat"),
		Clone:       vela.String("https://github.com/github/octocat.git"),
		Branch:      vela.String("master"),
		Timeout:     vela.Int64(60),
		Visibility:  vela.String("public"),
		Private:     vela.Bool(false),
		Trusted:     vela.Bool(false),
		Active:      vela.Bool(true),
		AllowPull:   vela.Bool(false),
		AllowPush:   vela.Bool(true),
		AllowDeploy: vela.Bool(false),
		AllowTag:    vela.Bool(false),
	}

	e, err := New(
		WithRepo(_repo),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
	}

	if e.repo != _repo {
		t.Errorf("WithRepo is %v, want %v", e.repo, _repo)
	}
}

func TestLinux_Opt_WithRuntime(t *testing.T) {
	// setup types
	_runtime, _ := docker.NewMock()

	e, err := New(
		WithRuntime(_runtime),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
	}

	if e.Runtime != _runtime {
		t.Errorf("WithRuntime is %v, want %v", e.Runtime, _runtime)
	}
}

func TestLinux_Opt_WithUser(t *testing.T) {
	// setup types
	_user := &library.User{
		ID:    vela.Int64(1),
		Name:  vela.String("octocat"),
		Token: vela.String("superSecretToken"),
	}

	e, err := New(
		WithUser(_user),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
	}

	if e.user != _user {
		t.Errorf("WithUser is %v, want %v", e.user, _user)
	}
}

func TestLinux_Opt_WithVelaClient(t *testing.T) {
	// setup types
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())
	_client, _ := vela.NewClient(s.URL, nil)

	e, err := New(
		WithVelaClient(_client),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
	}

	if e.Vela != _client {
		t.Errorf("WithVelaClient is %v, want %v", e.Vela, _client)
	}
}
