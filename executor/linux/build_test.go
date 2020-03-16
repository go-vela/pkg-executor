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

func TestExecutor_CreateBuild_Success(t *testing.T) {
	// setup global vars
	var (
		_build = &library.Build{
			Number:       vela.Int(1),
			Parent:       vela.Int(1),
			Event:        vela.String("push"),
			Status:       vela.String("success"),
			Error:        vela.String(""),
			Enqueued:     vela.Int64(1563474077),
			Created:      vela.Int64(1563474076),
			Started:      vela.Int64(1563474077),
			Finished:     vela.Int64(0),
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

		_repo = &library.Repo{
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

		_user = &library.User{
			ID:    vela.Int64(1),
			Name:  vela.String("octocat"),
			Token: vela.String("superSecretToken"),
		}
	)

	// setup context
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())
	cli, _ := vela.NewClient(s.URL, nil)

	// setup types
	r, _ := docker.NewMock()

	tests := []struct {
		build    *library.Build
		pipeline *pipeline.Build
		repo     *library.Repo
		user     *library.User
	}{
		{ // pipeline with steps
			build: _build,
			repo:  _repo,
			user:  _user,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Ports:       []string{"5432:5432"},
					},
				},
				Steps: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "__0_clone",
						Environment: map[string]string{},
						Image:       "target/vela-plugins/git:1",
						Name:        "clone",
						Number:      1,
						Pull:        true,
					},
					&pipeline.Container{
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
					&pipeline.Container{
						ID:          "__0_echo",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "echo",
						Number:      2,
						Pull:        true,
						Commands:    []string{"echo ${FOOBAR}"},
						Secrets: pipeline.StepSecretSlice{
							&pipeline.StepSecret{
								Source: "foobar",
								Target: "foobar",
							},
						},
					},
				},
			},
		},
		{ // pipeline with stages
			build: _build,
			repo:  _repo,
			user:  _user,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Ports:       []string{"5432:5432"},
					},
				},
				Stages: pipeline.StageSlice{
					&pipeline.Stage{
						Name: "clone",
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
								ID:          "__0_clone_clone",
								Environment: map[string]string{},
								Image:       "target/vela-plugins/git:1",
								Name:        "clone",
								Number:      1,
								Pull:        true,
							},
						},
					},
					&pipeline.Stage{
						Name:  "exit",
						Needs: []string{"clone"},
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
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
					&pipeline.Stage{
						Name:  "echo",
						Needs: []string{"clone"},
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
								ID:          "__0_echo_echo",
								Environment: map[string]string{},
								Image:       "alpine:latest",
								Name:        "echo",
								Number:      1,
								Pull:        true,
								Secrets: pipeline.StepSecretSlice{
									&pipeline.StepSecret{
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
			WithBuild(test.build),
			WithPipeline(test.pipeline),
			WithRepo(test.repo),
			WithRuntime(r),
			WithUser(test.user),
			WithVelaClient(cli),
		)
		if err != nil {
			t.Errorf("unable to create executor client: %v", err)
		}

		got := e.CreateBuild(context.Background())

		if got != nil {
			t.Errorf("CreateBuild is %v, want nil", got)
		}
	}
}

func TestExecutor_ExecBuild_Success(t *testing.T) {
	// setup global vars
	var (
		_build = &library.Build{
			Number:       vela.Int(1),
			Parent:       vela.Int(1),
			Event:        vela.String("push"),
			Status:       vela.String("success"),
			Error:        vela.String(""),
			Enqueued:     vela.Int64(1563474077),
			Created:      vela.Int64(1563474076),
			Started:      vela.Int64(1563474077),
			Finished:     vela.Int64(0),
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

		_repo = &library.Repo{
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

		_user = &library.User{
			ID:    vela.Int64(1),
			Name:  vela.String("octocat"),
			Token: vela.String("superSecretToken"),
		}
	)

	// setup context
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())
	cli, _ := vela.NewClient(s.URL, nil)

	// setup types
	r, _ := docker.NewMock()

	tests := []struct {
		build    *library.Build
		pipeline *pipeline.Build
		repo     *library.Repo
		user     *library.User
	}{
		{ // pipeline with steps
			build: _build,
			repo:  _repo,
			user:  _user,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
						Ports:       []string{"5432:5432"},
					},
				},
				Steps: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "__0_clone",
						Environment: map[string]string{},
						Image:       "target/vela-plugins/git:1",
						Name:        "clone",
						Number:      1,
						Pull:        true,
					},
					&pipeline.Container{
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
					&pipeline.Container{
						ID:          "__0_echo",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "echo",
						Number:      2,
						Pull:        true,
						Commands:    []string{"echo ${FOOBAR}"},
						Secrets: pipeline.StepSecretSlice{
							&pipeline.StepSecret{
								Source: "foobar",
								Target: "foobar",
							},
						},
					},
				},
			},
		},
		{ // pipeline with stages
			build: _build,
			repo:  _repo,
			user:  _user,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
						Ports:       []string{"5432:5432"},
					},
				},
				Stages: pipeline.StageSlice{
					&pipeline.Stage{
						Name: "clone",
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
								ID:          "__0_clone_clone",
								Environment: map[string]string{},
								Image:       "target/vela-plugins/git:1",
								Name:        "clone",
								Number:      1,
								Pull:        true,
							},
						},
					},
					&pipeline.Stage{
						Name:  "exit",
						Needs: []string{"clone"},
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
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
					&pipeline.Stage{
						Name:  "echo",
						Needs: []string{"clone"},
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
								ID:          "__0_echo_echo",
								Environment: map[string]string{},
								Image:       "alpine:latest",
								Name:        "echo",
								Number:      1,
								Pull:        true,
								Secrets: pipeline.StepSecretSlice{
									&pipeline.StepSecret{
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
			WithBuild(test.build),
			WithPipeline(test.pipeline),
			WithRepo(test.repo),
			WithRuntime(r),
			WithUser(test.user),
			WithVelaClient(cli),
		)
		if err != nil {
			t.Errorf("unable to create executor client: %v", err)
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

func TestExecutor_DestroyBuild_Success(t *testing.T) {
	// setup global vars
	var (
		_build = &library.Build{
			Number:       vela.Int(1),
			Parent:       vela.Int(1),
			Event:        vela.String("push"),
			Status:       vela.String("success"),
			Error:        vela.String(""),
			Enqueued:     vela.Int64(1563474077),
			Created:      vela.Int64(1563474076),
			Started:      vela.Int64(1563474077),
			Finished:     vela.Int64(0),
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

		_repo = &library.Repo{
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

		_user = &library.User{
			ID:    vela.Int64(1),
			Name:  vela.String("octocat"),
			Token: vela.String("superSecretToken"),
		}
	)

	// setup context
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(server.FakeHandler())
	cli, _ := vela.NewClient(s.URL, nil)

	// setup types
	r, _ := docker.NewMock()

	tests := []struct {
		build    *library.Build
		pipeline *pipeline.Build
		repo     *library.Repo
		user     *library.User
	}{
		{ // pipeline with steps
			build: _build,
			repo:  _repo,
			user:  _user,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
						Ports:       []string{"5432:5432"},
					},
				},
				Steps: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "__0_clone",
						Environment: map[string]string{},
						Image:       "target/vela-plugins/git:1",
						Name:        "clone",
						Number:      1,
						Pull:        true,
					},
					&pipeline.Container{
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
					&pipeline.Container{
						ID:          "__0_echo",
						Environment: map[string]string{},
						Image:       "alpine:latest",
						Name:        "echo",
						Number:      2,
						Pull:        true,
						Commands:    []string{"echo ${FOOBAR}"},
						Secrets: pipeline.StepSecretSlice{
							&pipeline.StepSecret{
								Source: "foobar",
								Target: "foobar",
							},
						},
					},
				},
			},
		},
		{ // pipeline with stages
			build: _build,
			repo:  _repo,
			user:  _user,
			pipeline: &pipeline.Build{
				Version: "1",
				ID:      "__0",
				Services: pipeline.ContainerSlice{
					&pipeline.Container{
						ID:          "service_org_repo_0_postgres;",
						Environment: map[string]string{},
						Image:       "postgres:11-alpine",
						Name:        "postgres",
						Number:      1,
						Ports:       []string{"5432:5432"},
					},
				},
				Stages: pipeline.StageSlice{
					&pipeline.Stage{
						Name: "clone",
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
								ID:          "__0_clone_clone",
								Environment: map[string]string{},
								Image:       "target/vela-plugins/git:1",
								Name:        "clone",
								Number:      1,
								Pull:        true,
							},
						},
					},
					&pipeline.Stage{
						Name:  "exit",
						Needs: []string{"clone"},
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
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
					&pipeline.Stage{
						Name:  "echo",
						Needs: []string{"clone"},
						Steps: pipeline.ContainerSlice{
							&pipeline.Container{
								ID:          "__0_echo_echo",
								Environment: map[string]string{},
								Image:       "alpine:latest",
								Name:        "echo",
								Number:      1,
								Pull:        true,
								Secrets: pipeline.StepSecretSlice{
									&pipeline.StepSecret{
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
			WithBuild(test.build),
			WithPipeline(test.pipeline),
			WithRepo(test.repo),
			WithRuntime(r),
			WithUser(test.user),
			WithVelaClient(cli),
		)
		if err != nil {
			t.Errorf("unable to create executor client: %v", err)
		}

		svc := new(library.Service)
		svc.SetNumber(1)
		e.services.Store(e.pipeline.Services[0].ID, svc)

		got := e.DestroyBuild(context.Background())

		if got != nil {
			t.Errorf("DestroyBuild is %v, want nil", got)
		}
	}
}
