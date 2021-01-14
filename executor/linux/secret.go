// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-vela/pkg-executor/internal/step"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"

	"github.com/sirupsen/logrus"
)

// secretSvc handles communication with secret processes during a build.
type secretSvc svc

var (
	// ErrUnrecognizedSecretType defines the error type when the
	// SecretType provided to the client is unsupported.
	ErrUnrecognizedSecretType = errors.New("unrecognized secret type")

	// ErrUnableToRetrieve defines the error type when the
	// secret is not able to be retrieved from the server.
	ErrUnableToRetrieve = errors.New("unable to retrieve secret")
)

// create configures the secret plugin for execution.
func (s *secretSvc) create(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := s.client.logger.WithField("secret", ctn.Name)

	ctn.Environment["BUILD_HOST"] = s.client.Hostname
	ctn.Environment["VELA_HOST"] = s.client.Hostname

	// TODO: remove hardcoded reference
	ctn.Environment["VELA_VERSION"] = "v0.6.0"
	ctn.Environment["VELA_RUNTIME"] = "docker"
	ctn.Environment["VELA_DISTRIBUTION"] = "linux"

	logger.Debug("setting up container")
	// setup the runtime container
	err := s.client.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	logger.Debug("injecting secrets")
	// inject secrets for container
	err = injectSecrets(ctn, s.client.Secrets)
	if err != nil {
		return err
	}

	logger.Debug("substituting container configuration")
	// substitute container configuration
	err = ctn.Substitute()
	if err != nil {
		return fmt.Errorf("unable to substitute container configuration")
	}

	return nil
}

// destroy cleans up secret plugin after execution.
func (s *secretSvc) destroy(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := s.client.logger.WithField("secret", ctn.Name)

	// load the step from the client
	_secret, err := step.Load(s.client.init, &s.client.steps)
	if err != nil {
		// create the step from the container
		_secret = new(library.Step)
		_secret.SetName(ctn.Name)
		_secret.SetNumber(ctn.Number)
		_secret.SetStatus(constants.StatusPending)
		_secret.SetHost(ctn.Environment["VELA_HOST"])
		_secret.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		_secret.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	// TODO: evaluate if we need this
	//
	// Do we upload external secret container results to Vela API?
	defer func() {
		logger.Info("uploading step snapshot")
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err := s.client.Vela.Step.Update(s.client.repo.GetOrg(), s.client.repo.GetName(), s.client.build.GetNumber(), _secret)
		if err != nil {
			logger.Errorf("unable to upload step snapshot: %v", err)
		}
	}()

	// check if the step is in a pending state
	if _secret.GetStatus() == constants.StatusPending {
		// update the step fields
		//
		// TODO: consider making this a constant
		//
		// nolint: gomnd // ignore magic number 137
		_secret.SetExitCode(137)
		_secret.SetFinished(time.Now().UTC().Unix())
		_secret.SetStatus(constants.StatusKilled)

		// check if the step was not started
		if _secret.GetStarted() == 0 {
			// set the started time to the finished time
			_secret.SetStarted(_secret.GetFinished())
		}
	}

	logger.Debug("inspecting container")
	// inspect the runtime container
	err = s.client.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the step finished
	if _secret.GetFinished() == 0 {
		// update the step fields
		_secret.SetFinished(time.Now().UTC().Unix())
		_secret.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the step fields
			_secret.SetExitCode(ctn.ExitCode)
			_secret.SetStatus(constants.StatusFailure)
		}
	}

	logger.Debug("removing container")
	// remove the runtime container
	err = s.client.Runtime.RemoveContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}

// exec runs a secret plugins for a pipeline.
func (s *secretSvc) exec(ctx context.Context, p *pipeline.SecretSlice) error {
	// stream all the logs to the init step
	_init, err := step.Load(s.client.init, &s.client.steps)
	if err != nil {
		return err
	}

	defer func() {
		_init.SetFinished(time.Now().UTC().Unix())

		s.client.logger.Infof("uploading %s step state", _init.GetName())
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err = s.client.Vela.Step.Update(s.client.repo.GetOrg(), s.client.repo.GetName(), s.client.build.GetNumber(), _init)
		if err != nil {
			s.client.logger.Errorf("unable to upload init state: %v", err)
		}
	}()

	// execute the secrets for the pipeline
	for _, _secret := range *p {
		// skip over non-plugin secrets
		if _secret.Origin.Empty() {
			continue
		}

		// update engine logger with secret metadata
		//
		// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
		logger := s.client.logger.WithField("secret", _secret.Origin.Name)

		logger.Debug("running container")
		// run the runtime container
		err := s.client.Runtime.RunContainer(ctx, _secret.Origin, s.client.pipeline)
		if err != nil {
			return err
		}

		go func() {
			logger.Debug("stream logs for container")
			// stream logs from container
			err = s.client.secret.stream(ctx, _secret.Origin)
			if err != nil {
				logger.Error(err)
			}
		}()

		logger.Debug("waiting for container")
		// wait for the runtime container
		err = s.client.Runtime.WaitContainer(ctx, _secret.Origin)
		if err != nil {
			return err
		}

		logger.Debug("inspecting container")
		// inspect the runtime container
		err = s.client.Runtime.InspectContainer(ctx, _secret.Origin)
		if err != nil {
			return err
		}

		// check the step exit code
		if _secret.Origin.ExitCode != 0 {
			// check if we ignore step failures
			if !_secret.Origin.Ruleset.Continue {
				// set build status to failure
				s.client.build.SetStatus(constants.StatusFailure)
			}

			// update the step fields
			_init.SetExitCode(_secret.Origin.ExitCode)
			_init.SetStatus(constants.StatusFailure)

			return fmt.Errorf("%s container exited with non-zero code", _secret.Origin.Name)
		}

		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err = s.client.Vela.Step.Update(s.client.repo.GetOrg(), s.client.repo.GetName(), s.client.build.GetNumber(), _init)
		if err != nil {
			s.client.logger.Errorf("unable to upload init state: %v", err)
		}
	}

	return nil
}

// pull defines a function that pulls the secrets from the server for a given pipeline.
func (s *secretSvc) pull(secret *pipeline.Secret) (*library.Secret, error) {
	// nolint: staticheck // reports the value is never used but we return it
	_secret := new(library.Secret)

	switch secret.Type {
	// handle repo secrets
	case constants.SecretOrg:
		org, key, err := secret.ParseOrg(s.client.repo.GetOrg())
		if err != nil {
			return nil, err
		}

		// send API call to capture the org secret
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
		_secret, _, err = s.client.Vela.Secret.Get(secret.Engine, secret.Type, org, "*", key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ErrUnableToRetrieve, err)
		}

		secret.Value = _secret.GetValue()

	// handle repo secrets
	case constants.SecretRepo:
		org, repo, key, err := secret.ParseRepo(s.client.repo.GetOrg(), s.client.repo.GetName())
		if err != nil {
			return nil, err
		}

		// send API call to capture the repo secret
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
		_secret, _, err = s.client.Vela.Secret.Get(secret.Engine, secret.Type, org, repo, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ErrUnableToRetrieve, err)
		}

		secret.Value = _secret.GetValue()

	// handle shared secrets
	case constants.SecretShared:
		org, team, key, err := secret.ParseShared()
		if err != nil {
			return nil, err
		}

		// send API call to capture the repo secret
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
		_secret, _, err = s.client.Vela.Secret.Get(secret.Engine, secret.Type, org, team, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ErrUnableToRetrieve, err)
		}

		secret.Value = _secret.GetValue()

	default:
		return nil, fmt.Errorf("%s: %s", ErrUnrecognizedSecretType, secret.Type)
	}

	return _secret, nil
}

// stream tails the output for a secret plugin.
func (s *secretSvc) stream(ctx context.Context, ctn *pipeline.Container) error {
	// stream all the logs to the init step
	_log, err := step.LoadLogs(s.client.init, &s.client.stepLogs)
	if err != nil {
		return err
	}

	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := s.client.logger.WithField("secret", ctn.Name)

	// create new buffer for uploading logs
	logs := new(bytes.Buffer)

	defer func() {
		// NOTE: Whenever the stream ends we want to ensure
		// that this function makes the call to update
		// the step logs
		logger.Trace(logs.String())

		// update the existing log with the last bytes
		//
		// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
		_log.AppendData(logs.Bytes())

		logger.Debug("uploading logs")
		// send API call to update the logs for the service
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateService
		_log, _, err = s.client.Vela.Log.UpdateStep(s.client.repo.GetOrg(), s.client.repo.GetName(), s.client.build.GetNumber(), ctn.Number, _log)
		if err != nil {
			logger.Errorf("unable to upload container logs: %v", err)
		}
	}()

	logger.Debug("tailing container")
	// tail the runtime container
	rc, err := s.client.Runtime.TailContainer(ctx, ctn)
	if err != nil {
		return err
	}
	defer rc.Close()

	// create new scanner from the container output
	scanner := bufio.NewScanner(rc)

	// scan entire container output
	for scanner.Scan() {
		// write all the logs from the scanner
		logs.Write(append(scanner.Bytes(), []byte("\n")...))

		// if we have at least 1000 bytes in our buffer
		if logs.Len() > 1000 {
			logger.Trace(logs.String())

			// update the existing log with the new bytes
			//
			// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
			_log.AppendData(logs.Bytes())

			logger.Debug("appending logs")
			// send API call to append the logs for the init step
			//
			// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
			// nolint: lll // skip line length due to variable names
			_log, _, err = s.client.Vela.Log.UpdateStep(s.client.repo.GetOrg(), s.client.repo.GetName(), s.client.build.GetNumber(), s.client.init.Number, _log)
			if err != nil {
				return err
			}

			// flush the buffer of logs
			logs.Reset()
		}
	}

	return scanner.Err()
}

// TODO: Evaluate pulling this into a "bool" types function for injecting
//
// helper function to check secret whitelist before setting value.
func injectSecrets(ctn *pipeline.Container, m map[string]*library.Secret) error {
	// inject secrets for step
	for _, _secret := range ctn.Secrets {
		logrus.Tracef("looking up secret %s from pipeline secrets", _secret.Source)
		// lookup container secret in map
		s, ok := m[_secret.Source]
		if !ok {
			continue
		}

		logrus.Tracef("matching secret %s to container %s", _secret.Source, ctn.Name)
		// ensure the secret matches with the container
		if s.Match(ctn) {
			ctn.Environment[strings.ToUpper(_secret.Target)] = s.GetValue()
		}
	}

	return nil
}
