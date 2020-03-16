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

func TestLinux_GetBuild(t *testing.T) {
	// setup types
	b := &library.Build{ID: vela.Int64(1)}

	want := b

	e, err := New(
		WithBuild(b),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
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
		t.Errorf("unable to create executor client: %v", err)
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
	r := &library.Repo{ID: vela.Int64(1)}

	want := r

	e, err := New(
		WithRepo(r),
	)
	if err != nil {
		t.Errorf("unable to create executor client: %v", err)
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
