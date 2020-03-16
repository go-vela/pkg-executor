// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"reflect"
	"testing"

	"github.com/go-vela/sdk-go/vela"

	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// setup global variables used for testing
var (
	_build = &library.Build{
		ID:           vela.Int64(1),
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
		ID:          vela.Int64(1),
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
		ID:        vela.Int64(1),
		Name:      vela.String("octocat"),
		Token:     vela.String("superSecretToken"),
		Hash:      vela.String("MzM4N2MzMDAtNmY4Mi00OTA5LWFhZDAtNWIzMTlkNTJkODMy"),
		Favorites: vela.Strings([]string{"github/octocat"}),
		Active:    vela.Bool(true),
		Admin:     vela.Bool(false),
	}
)

func TestLinux_GetBuild(t *testing.T) {
	// setup types
	want := _build

	e, err := New(
		WithBuild(_build),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	// run test
	got, err := e.GetBuild()
	if err != nil {
		t.Errorf("unable to get build from executor: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetBuild is %v, want %v", got, want)
	}
}

func TestLinux_GetPipeline(t *testing.T) {
	// setup types
	p := &pipeline.Build{ID: "1"}

	want := p

	e, err := New(
		WithPipeline(p),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	// run test
	got, err := e.GetPipeline()
	if err != nil {
		t.Errorf("unable to get pipeline from compiler: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetPipeline is %v, want %v", got, want)
	}
}

func TestLinux_GetRepo(t *testing.T) {
	// setup types
	want := _repo

	e, err := New(
		WithRepo(_repo),
	)
	if err != nil {
		t.Errorf("unable to create executor engine: %v", err)
	}

	// run test
	got, err := e.GetRepo()
	if err != nil {
		t.Errorf("unable to get repo from compiler: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetRepo is %v, want %v", got, want)
	}
}
