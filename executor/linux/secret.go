// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package linux

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/drone/envsubst"
	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"

	"github.com/sirupsen/logrus"
)

// PullSecret defines a function that pulls the secrets for a given pipeline.
func (c *client) PullSecret(ctx context.Context) error {
	var err error

	p := c.pipeline

	secrets := make(map[string]*library.Secret)
	sec := new(library.Secret)

	// iterate through each secret provided in the pipeline
	for _, s := range p.Secrets {
		// if the secret isn't a native or vault type
		if !strings.EqualFold(s.Engine, constants.DriverNative) &&
			!strings.EqualFold(s.Engine, constants.DriverVault) {
			return fmt.Errorf("unrecognized secret engine: %s", s.Engine)
		}

		switch s.Type {
		// handle org secrets
		case constants.SecretOrg:
			c.logger.Debug("pulling org secret")
			// get org secret
			sec, err = c.getOrg(s)
			if err != nil {
				return err
			}
		// handle repo secrets
		case constants.SecretRepo:
			c.logger.Debug("pulling repo secret")
			// get repo secret
			sec, err = c.getRepo(s)
			if err != nil {
				return err
			}
		// handle shared secrets
		case constants.SecretShared:
			c.logger.Debug("pulling shared secret")
			// get shared secret
			sec, err = c.getShared(s)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unrecognized secret type: %s", s.Type)
		}

		// add secret to the map
		secrets[s.Name] = sec
	}

	// overwrite the engine secret map
	c.Secrets = secrets

	return nil
}

// CreateSecret configures the secret plugin for execution.
func (c *client) CreateSecret(ctx context.Context, ctn *pipeline.Container) error {
	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("secret", ctn.Name)

	ctn.Environment["BUILD_HOST"] = c.Hostname
	ctn.Environment["VELA_HOST"] = c.Hostname
	ctn.Environment["VELA_VERSION"] = "v0.4.0"
	// TODO: remove hardcoded reference
	ctn.Environment["VELA_RUNTIME"] = "docker"
	ctn.Environment["VELA_DISTRIBUTION"] = "linux"

	logger.Debug("setting up container")
	// setup the runtime container
	err := c.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	logger.Debug("injecting secrets")
	// inject secrets for container
	err = injectSecrets(ctn, c.Secrets)
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

// PlanSecret prepares the secret plugin for execution.
//
// this function is a no-op
func (c *client) PlanSecret(ctx context.Context, ctn *pipeline.Container) error {
	return nil
}

// ExecSecret runs a secret plugin.
func (c *client) ExecSecret(ctx context.Context, init *pipeline.Container, ctn *pipeline.Container) error {
	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("secret", ctn.Name)

	logger.Debug("running container")
	// run the runtime container
	err := c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		logger.Debug("stream logs for container")
		// stream logs from container
		err := c.StreamSecret(ctx, init, ctn)
		if err != nil {
			logger.Error(err)
		}
	}()

	return nil
}

// StreamSecret tails the output for a secret plugin.
func (c *client) StreamSecret(ctx context.Context, init *pipeline.Container, ctn *pipeline.Container) error {
	b := c.build
	r := c.repo

	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("secret", ctn.Name)

	// load the logs for the service from the client
	l, err := c.loadStepLogs(init.ID)
	if err != nil {
		return err
	}

	// create new buffer for uploading logs
	logs := new(bytes.Buffer)

	logger.Debug("tailing container")
	// tail the runtime container
	rc, err := c.Runtime.TailContainer(ctx, ctn)
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
			l, _, err = c.Vela.Log.UpdateStep(r.GetOrg(), r.GetName(), b.GetNumber(), init.Number, l)
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
	_, _, err = c.Vela.Log.UpdateStep(r.GetOrg(), r.GetName(), b.GetNumber(), ctn.Number, l)
	if err != nil {
		return err
	}

	return nil
}

// DestroySecret cleans up secret plugin after execution.
func (c *client) DestroySecret(ctx context.Context, init *pipeline.Container, ctn *pipeline.Container) error {
	// update engine logger with secret metadata
	//
	// https://pkg.go.dev/github.com/sirupsen/logrus?tab=doc#Entry.WithField
	logger := c.logger.WithField("secret", ctn.Name)

	// load the secret from the client
	step, err := c.loadStep(init.ID)
	if err != nil {
		// create the secret from the container
		step = new(library.Step)
		step.SetName(ctn.Name)
		step.SetNumber(ctn.Number)
		step.SetStatus(constants.StatusPending)
		step.SetHost(ctn.Environment["VELA_HOST"])
		step.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		step.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	// check if the secret is in a pending state
	if step.GetStatus() == constants.StatusPending {
		// update the secret fields
		step.SetExitCode(137)
		step.SetFinished(time.Now().UTC().Unix())
		step.SetStatus(constants.StatusKilled)

		// check if the secret was not started
		if step.GetStarted() == 0 {
			// set the started time to the finished time
			step.SetStarted(step.GetFinished())
		}
	}

	logger.Debug("inspecting container")
	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the secret finished
	if step.GetFinished() == 0 {
		// update the secret fields
		step.SetFinished(time.Now().UTC().Unix())
		step.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the secret fields
			step.SetExitCode(ctn.ExitCode)
			step.SetStatus(constants.StatusFailure)
		}
	}

	logger.Debug("removing container")
	// remove the runtime container
	err = c.Runtime.RemoveContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}

// getOrg is a helper function to parse and capture
// the org secret from the provided secret engine.
func (c *client) getOrg(s *pipeline.Secret) (*library.Secret, error) {
	c.logger.Tracef("pulling %s %s secret %s", s.Engine, s.Type, s.Name)

	// variables necessary for secret
	org := c.repo.GetOrg()
	repo := "*"
	path := s.Key

	// check if the full path was provided
	if strings.Contains(path, "/") {
		// split the full path into parts
		parts := strings.SplitN(path, "/", 2)

		// secret is invalid
		if len(parts) != 2 {
			return nil, fmt.Errorf("path %s for %s secret %s is invalid", s.Key, s.Type, s.Name)
		}

		// check if the org provided matches what we expect
		if strings.EqualFold(parts[0], org) {
			// update the variables
			org = parts[0]
			path = parts[1]
		}
	}

	// send API call to capture the org secret
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
	secret, _, err := c.Vela.Secret.Get(s.Engine, s.Type, org, repo, path)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve %s secret %s: %w", s.Type, s.Key, err)
	}

	// overwrite the secret value
	s.Value = secret.GetValue()

	return secret, nil
}

// getRepo is a helper function to parse and capture
// the repo secret from the provided secret engine.
func (c *client) getRepo(s *pipeline.Secret) (*library.Secret, error) {
	c.logger.Tracef("pulling %s %s secret %s", s.Engine, s.Type, s.Name)

	// variables necessary for secret
	org := c.repo.GetOrg()
	repo := c.repo.GetName()
	path := s.Key

	// check if the full path was provided
	if strings.Contains(path, "/") {
		// split the full path into parts
		parts := strings.SplitN(path, "/", 3)

		// secret is invalid
		if len(parts) != 3 {
			return nil, fmt.Errorf("path %s for %s secret %s is invalid", s.Key, s.Type, s.Name)
		}

		// check if the org provided matches what we expect
		if strings.EqualFold(parts[0], org) {
			// update the org variable
			org = parts[0]

			// check if the repo provided matches what we expect
			if strings.EqualFold(parts[1], repo) {
				// update the variables
				repo = parts[1]
				path = parts[2]
			}
		}
	}

	// send API call to capture the repo secret
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
	secret, _, err := c.Vela.Secret.Get(s.Engine, s.Type, org, repo, path)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve %s secret %s: %w", s.Type, s.Key, err)
	}

	// overwrite the secret value
	s.Value = secret.GetValue()

	return secret, nil
}

// getShared is a helper function to parse and capture
// the shared secret from the provided secret engine.
func (c *client) getShared(s *pipeline.Secret) (*library.Secret, error) {
	c.logger.Tracef("pulling %s %s secret %s", s.Engine, s.Type, s.Name)

	// variables necessary for secret
	var (
		team string
		org  string
	)

	path := s.Key

	// check if the full path was provided
	if strings.Contains(path, "/") {
		// split the full path into parts
		parts := strings.SplitN(path, "/", 3)

		// secret is invalid
		if len(parts) != 3 {
			return nil, fmt.Errorf("path %s for %s secret %s is invalid", s.Key, s.Type, s.Name)
		}

		// check if the org provided is not empty
		if len(parts[0]) > 0 {
			// update the org variable
			org = parts[0]

			// check if the team provided is not empty
			if len(parts[1]) > 0 {
				// update the variables
				team = parts[1]
				path = parts[2]
			}
		}
	}

	// send API call to capture the shared secret
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SecretService.Get
	secret, _, err := c.Vela.Secret.Get(s.Engine, s.Type, org, team, path)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve %s secret %s: %w", s.Type, s.Key, err)
	}

	// overwrite the secret value
	s.Value = secret.GetValue()

	return secret, nil
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

// loadService is a helper function to capture
// a service from the client.
func (c *client) loadSecret(name string) (*library.Secret, error) {
	// load the service key from the client
	result, ok := c.secrets.Load(name)
	if !ok {
		return nil, fmt.Errorf("unable to load secret %s", name)
	}

	// cast the service key to the expected type
	s, ok := result.(*library.Secret)
	if !ok {
		return nil, fmt.Errorf("secret %s had unexpected value", name)
	}

	return s, nil
}
