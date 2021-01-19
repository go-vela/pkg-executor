// Copyright (c) 2021 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package service

import (
	"time"

	"github.com/go-vela/sdk-go/vela"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
	"github.com/sirupsen/logrus"
)

// Snapshot creates a moment in time record of the service
// and attempts to upload it to the server.
func Snapshot(ctn *pipeline.Container, b *library.Build, c *vela.Client, l *logrus.Entry, r *library.Repo, s *library.Service) {
	// check if the logger provided is empty
	if l == nil {
		l = logrus.NewEntry(logrus.StandardLogger())
	}

	// check if the service is in a pending state
	if s.GetStatus() == constants.StatusPending {
		// update the service fields
		//
		// TODO: consider making this a constant
		//
		// nolint: gomnd // ignore magic number 137
		s.SetExitCode(137)
		s.SetFinished(time.Now().UTC().Unix())
		s.SetStatus(constants.StatusKilled)

		// check if the service was not started
		if s.GetStarted() == 0 {
			// set the started time to the finished time
			s.SetStarted(s.GetFinished())
		}
	}

	// check if the service finished
	if s.GetFinished() == 0 {
		// update the service fields
		s.SetFinished(time.Now().UTC().Unix())
		s.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the service fields
			s.SetExitCode(ctn.ExitCode)
			s.SetStatus(constants.StatusFailure)
		}
	}

	// check if the Vela client provided is empty
	if c != nil {
		l.Debug("uploading service snapshot")

		// send API call to update the service
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SvcService.Update
		_, _, err := c.Svc.Update(r.GetOrg(), r.GetName(), b.GetNumber(), s)
		if err != nil {
			l.Errorf("unable to upload service snapshot: %v", err)
		}
	}
}
