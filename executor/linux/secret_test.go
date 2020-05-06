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

	"github.com/google/go-cmp/cmp"
)

func TestLinux_PullSecret(t *testing.T) {
	// setup types
	_build := testBuild()
	_repo := testRepo()
	_user := testUser()
	_stages := testStages()
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
				Steps: pipeline.ContainerSlice{
					{
						ID:          "step_github_octocat_1_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
				Secrets: pipeline.SecretSlice{
					{
						Name:   "foo",
						Key:    "github/octocat/foo",
						Engine: "invalid",
						Type:   "repo",
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
						ID:          "step_github_octocat_1_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
				Secrets: pipeline.SecretSlice{
					{
						Name:   "foo",
						Key:    "github/octocat/foo",
						Engine: "native",
						Type:   "invalid",
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
						ID:          "step_github_octocat_1_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
				Secrets: pipeline.SecretSlice{
					{
						Name:   "foo",
						Key:    "/",
						Engine: "native",
						Type:   "repo",
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
						ID:          "step_github_octocat_1_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
				Secrets: pipeline.SecretSlice{
					{
						Name:   "foo",
						Key:    "/",
						Engine: "native",
						Type:   "shared",
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
						ID:          "step_github_octocat_1_clone",
						Directory:   "/home/github/octocat",
						Environment: map[string]string{"FOO": "bar"},
						Image:       "target/vela-git:v0.3.0",
						Name:        "clone",
						Number:      2,
						Pull:        true,
					},
				},
				Secrets: pipeline.SecretSlice{
					{
						Name:   "foo",
						Key:    "github/not-found/foo",
						Engine: "native",
						Type:   "shared",
					},
				},
			},
		},
	}

	// run tests
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

		err = _engine.PullSecret(context.Background())

		if test.failure {
			if err == nil {
				t.Errorf("PullSecret should have returned err")
			}

			continue
		}

		if err != nil {
			t.Errorf("PullSecret returned err: %v", err)
		}
	}
}

func TestLinux_Secret_injectSecret(t *testing.T) {
	// name and value of secret
	v := "foo"

	// setup types
	tests := []struct {
		step *pipeline.Container
		msec map[string]*library.Secret
		want *pipeline.Container
	}{
		// Tests for secrets with image ACLs
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: make(map[string]string),
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Images: &[]string{""}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: make(map[string]string),
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: make(map[string]string),
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Images: &[]string{"alpine"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"FOO": "foo"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: make(map[string]string),
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Images: &[]string{"alpine:latest"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"FOO": "foo"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: make(map[string]string),
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Images: &[]string{"centos"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: make(map[string]string),
			},
		},

		// Tests for secrets with event ACLs
		{ // push event checks
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"push"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"FOO": "foo", "BUILD_EVENT": "push"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"deployment"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
			},
		},
		{ // pull_request event checks
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "pull_request"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"pull_request"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"FOO": "foo", "BUILD_EVENT": "pull_request"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "pull_request"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"deployment"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "pull_request"},
			},
		},
		{ // tag event checks
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "tag"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"tag"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"FOO": "foo", "BUILD_EVENT": "tag"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "tag"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"deployment"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "tag"},
			},
		},
		{ // deployment event checks
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "deployment"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"deployment"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"FOO": "foo", "BUILD_EVENT": "deployment"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "deployment"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"tag"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "deployment"},
			},
		},

		// Tests for secrets with event and image ACLs
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"push"}, Images: &[]string{"centos"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "centos:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"pull_request"}, Images: &[]string{"centos"}}},
			want: &pipeline.Container{
				Image:       "centos:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
			},
		},
		{
			step: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"BUILD_EVENT": "push"},
				Secrets:     pipeline.StepSecretSlice{{Source: "FOO", Target: "FOO"}},
			},
			msec: map[string]*library.Secret{"FOO": {Name: &v, Value: &v, Events: &[]string{"push"}, Images: &[]string{"alpine"}}},
			want: &pipeline.Container{
				Image:       "alpine:latest",
				Environment: map[string]string{"FOO": "foo", "BUILD_EVENT": "push"},
			},
		},
	}

	// run test
	for _, test := range tests {
		_ = injectSecrets(test.step, test.msec)
		got := test.step

		// Preferred use of reflect.DeepEqual(x, y interface) is giving false positives.
		// Switching to a Google library for increased clarity.
		// https://github.com/google/go-cmp
		if diff := cmp.Diff(test.want.Environment, got.Environment); diff != "" {
			t.Errorf("injectSecrets mismatch (-want +got):\n%s", diff)
		}
	}
}
