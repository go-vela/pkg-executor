// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/drone/envsubst"
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
	ctn.Environment["VELA_VERSION"] = "v0.4.0"
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

	logger.Debug("marshaling configuration")
	// marshal container configuration
	body, err := json.Marshal(ctn)
	if err != nil {
		return fmt.Errorf("unable to marshal configuration: %v", err)
	}

	// create substitute function
	subFunc := func(name string) string {
		env := ctn.Environment[name]
		if strings.Contains(env, "\n") {
			env = fmt.Sprintf("%q", env)
		}

		return env
	}

	logger.Debug("substituting environment")
	// substitute the environment variables
	//
	// https://pkg.go.dev/github.com/drone/envsubst?tab=doc#Eval
	subStep, err := envsubst.Eval(string(body), subFunc)
	if err != nil {
		return fmt.Errorf("unable to substitute environment variables: %v", err)
	}

	logger.Debug("unmarshaling configuration")
	// unmarshal container configuration
	err = json.Unmarshal([]byte(subStep), ctn)
	if err != nil {
		return fmt.Errorf("unable to unmarshal configuration: %v", err)
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
	step, err := s.client.loadStep(s.client.init.ID)
	if err != nil {
		// create the step from the container
		step = new(library.Step)
		step.SetName(ctn.Name)
		step.SetNumber(ctn.Number)
		step.SetStatus(constants.StatusPending)
		step.SetHost(ctn.Environment["VELA_HOST"])
		step.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		step.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	defer func() {
		logger.Info("uploading step snapshot")
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err := s.client.Vela.Step.Update(s.client.repo.GetOrg(), s.client.repo.GetName(), s.client.build.GetNumber(), step)
		if err != nil {
			logger.Errorf("unable to upload step snapshot: %v", err)
		}
	}()

	// check if the step is in a pending state
	if step.GetStatus() == constants.StatusPending {
		// update the step fields
		step.SetExitCode(137)
		step.SetFinished(time.Now().UTC().Unix())
		step.SetStatus(constants.StatusKilled)

		// check if the step was not started
		if step.GetStarted() == 0 {
			// set the started time to the finished time
			step.SetStarted(step.GetFinished())
		}
	}

	logger.Debug("inspecting container")
	// inspect the runtime container
	err = s.client.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the step finished
	if step.GetFinished() == 0 {
		// update the step fields
		step.SetFinished(time.Now().UTC().Unix())
		step.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the step fields
			step.SetExitCode(ctn.ExitCode)
			step.SetStatus(constants.StatusFailure)
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
	b := s.client.build
	r := s.client.repo

	// stream all the logs to the init step
	init, err := s.client.loadStep(s.client.init.ID)
	if err != nil {
		return err
	}

	defer func() {
		init.SetFinished(time.Now().UTC().Unix())
		s.client.logger.Infof("uploading %s step state", init.GetName())
		// send API call to update the build
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#StepService.Update
		_, _, err = s.client.Vela.Step.Update(r.GetOrg(), r.GetName(), b.GetNumber(), init)
		if err != nil {
			s.client.logger.Errorf("unable to upload init state: %v", err)
		}
	}()

	// execute the secrets for the pipeline
	for _, secret := range *p {
		// skip over non-plugin secrets
		if secret.Origin.Empty() {
			continue
		}

		// update engine logger with secret metadata
		//
		// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
		logger := s.client.logger.WithField("secret", secret.Origin.Name)

		logger.Debug("running container")
		// run the runtime container
		err := s.client.Runtime.RunContainer(ctx, secret.Origin, s.client.pipeline)
		if err != nil {
			return err
		}

		go func() {
			logger.Debug("stream logs for container")
			// stream logs from container
			err = s.client.secret.stream(ctx, secret.Origin)
			if err != nil {
				logger.Error(err)
			}
		}()

		logger.Debug("waiting for container")
		// wait for the runtime container
		err = s.client.Runtime.WaitContainer(ctx, secret.Origin)
		if err != nil {
			return err
		}

		logger.Debug("inspecting container")
		// inspect the runtime container
		err = s.client.Runtime.InspectContainer(ctx, secret.Origin)
		if err != nil {
			return err
		}

		// check the step exit code
		if secret.Origin.ExitCode != 0 {
			// check if we ignore step failures
			if !secret.Origin.Ruleset.Continue {
				// set build status to failure
				b.SetStatus(constants.StatusFailure)
			}

			// update the step fields
			init.SetExitCode(secret.Origin.ExitCode)
			init.SetStatus(constants.StatusFailure)
		}
	}

	return nil
}

// pull defines a function that pulls the secrets from the server for a given pipeline.
func (s *secretSvc) pull(secret *pipeline.Secret) (*library.Secret, error) {
	sec := new(library.Secret)

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
		sec, _, err = s.client.Vela.Secret.Get(secret.Engine, secret.Type, org, "*", key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ErrUnableToRetrieve, err)
		}

		secret.Value = sec.GetValue()

	// handle repo secrets
	case constants.SecretRepo:
		org, repo, key, err := secret.ParseRepo(s.client.repo.GetOrg(), s.client.repo.GetName())
		if err != nil {
			return nil, err
		}

		// send API call to capture the repo secret
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
		sec, _, err = s.client.Vela.Secret.Get(secret.Engine, secret.Type, org, repo, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ErrUnableToRetrieve, err)
		}

		secret.Value = sec.GetValue()

	// handle shared secrets
	case constants.SecretShared:
		org, team, key, err := secret.ParseShared(s.client.repo.GetOrg())
		if err != nil {
			return nil, err
		}

		// send API call to capture the repo secret
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
		sec, _, err = s.client.Vela.Secret.Get(secret.Engine, secret.Type, org, team, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ErrUnableToRetrieve, err)
		}

		secret.Value = sec.GetValue()

	default:
		return nil, fmt.Errorf("%s: %s", ErrUnrecognizedSecretType, secret.Type)
	}

	return sec, nil
}

// stream tails the output for a secret plugin.
func (s *secretSvc) stream(ctx context.Context, ctn *pipeline.Container) error {
	b := s.client.build
	r := s.client.repo

	// stream all the logs to the init step
	l, err := s.client.loadStepLogs(s.client.init.ID)
	if err != nil {
		return err
	}

	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := s.client.logger.WithField("secret", ctn.Name)

	// create new buffer for uploading logs
	logs := new(bytes.Buffer)

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
			l.AppendData(logs.Bytes())

			logger.Debug("appending logs")
			// send API call to append the logs for the init step
			//
			// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateStep
			l, _, err = s.client.Vela.Log.UpdateStep(r.GetOrg(), r.GetName(), b.GetNumber(), s.client.init.Number, l)
			if err != nil {
				return err
			}

			// flush the buffer of logs
			logs.Reset()
		}
	}
	logger.Trace(logs.String())

	// update the existing log with the last bytes
	//
	// https://pkg.go.dev/github.com/go-vela/types/library?tab=doc#Log.AppendData
	l.AppendData(logs.Bytes())

	logger.Debug("uploading logs")
	// send API call to update the logs for the service
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.UpdateService
	_, _, err = s.client.Vela.Log.UpdateStep(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
	if err != nil {
		return err
	}

	return nil
}

// helper function to check secret whitelist before setting value
// TODO: Evaluate pulling this into a "bool" types function for injecting
func injectSecrets(ctn *pipeline.Container, m map[string]*library.Secret) error {
	// inject secrets for step
	for _, secret := range ctn.Secrets {
		logrus.Tracef("looking up secret %s from pipeline secrets", secret.Source)
		// lookup container secret in map
		s, ok := m[secret.Source]
		if !ok {
			continue
		}

		logrus.Tracef("matching secret %s to container %s", secret.Source, ctn.Name)
		// ensure the secret matches with the container
		if s.Match(ctn) {
			ctn.Environment[strings.ToUpper(secret.Target)] = s.GetValue()
		}
	}

	return nil
}
