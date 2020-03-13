// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package main

import (
	"time"

	"github.com/go-vela/types/library"

	"github.com/sirupsen/logrus"
)

// helper function to setup the build.
func setupBuild() *library.Build {
	logrus.Trace("creating fake build")

	b := new(library.Build)

	b.SetID(1)
	b.SetRepoID(1)
	b.SetNumber(1)
	b.SetParent(1)
	b.SetEvent("push")
	b.SetStatus("pending")
	b.SetError("")
	b.SetEnqueued(time.Now().UTC().Unix())
	b.SetCreated(time.Now().UTC().Unix())
	b.SetDeploy("")
	b.SetClone("https://github.com/go-vela/pkg-runtime.git")
	b.SetSource("https://github.com/go-vela/pkg-runtime/commit/0a08eb2eea09dd58498a4325fee0cb0ab3b66fc9")
	b.SetTitle("push received from https://github.com/go-vela/pkg-runtime")
	b.SetMessage("initial commit")
	b.SetCommit("0a08eb2eea09dd58498a4325fee0cb0ab3b66fc9")
	b.SetSender("vela-worker")
	b.SetAuthor("vela-worker")
	b.SetEmail("vela@target.com")
	b.SetLink("")
	b.SetBranch("master")
	b.SetRef("refs/heads/master")
	b.SetBaseRef("")

	return b
}

// helper function to setup the repo.
func setupRepo() *library.Repo {
	logrus.Trace("creating fake repo")

	r := new(library.Repo)

	r.SetID(1)
	r.SetUserID(1)
	r.SetOrg("go-vela")
	r.SetName("pkg-runtime")
	r.SetFullName("go-vela/pkg-runtime")
	r.SetLink("https://github.com/go-vela/pkg-runtime")
	r.SetClone("https://github.com/go-vela/pkg-runtime.git")
	r.SetBranch("master")
	r.SetTimeout(30)
	r.SetVisibility("public")
	r.SetPrivate(false)
	r.SetTrusted(false)
	r.SetActive(true)
	r.SetAllowPull(true)
	r.SetAllowPush(true)
	r.SetAllowDeploy(false)
	r.SetAllowTag(false)

	return r
}

// helper function to setup the user.
func setupUser() *library.User {
	logrus.Trace("creating fake user")

	u := new(library.User)

	u.SetID(1)
	u.SetName("vela-worker")
	u.SetToken("superSecretToken")
	u.SetFavorites([]string{})
	u.SetActive(true)
	u.SetAdmin(false)

	return u
}
