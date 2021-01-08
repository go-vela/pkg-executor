// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package local

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/drone/envsubst"

	"github.com/go-vela/types/constants"
	"github.com/go-vela/types/library"
	"github.com/go-vela/types/pipeline"
)

// CreateService configures the service for execution.
func (c *client) CreateService(ctx context.Context, ctn *pipeline.Container) error {
	ctn.Environment["BUILD_HOST"] = c.Hostname
	ctn.Environment["VELA_HOST"] = c.Hostname
	ctn.Environment["VELA_VERSION"] = "v0.6.0"
	// TODO: remove hardcoded reference
	ctn.Environment["VELA_RUNTIME"] = "docker"
	ctn.Environment["VELA_DISTRIBUTION"] = "linux"

	// setup the runtime container
	err := c.Runtime.SetupContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// TODO: will be uncommented in a future PR
	// // inject secrets for container
	// err = injectSecrets(ctn, c.Secrets)
	// if err != nil {
	// 	return err
	// }

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

	// substitute the environment variables
	//
	// https://pkg.go.dev/github.com/drone/envsubst?tab=doc#Eval
	subStep, err := envsubst.Eval(string(body), subFunc)
	if err != nil {
		return fmt.Errorf("unable to substitute environment variables: %v", err)
	}

	// unmarshal container configuration
	err = json.Unmarshal([]byte(subStep), ctn)
	if err != nil {
		// define a new buffer to capture the output, which doesn't need to be written
		buf := new(bytes.Buffer)

		// create new encoder for buffer
		enc := json.NewEncoder(buf)
		// ctn is modified via pointer through enc.Encode
		err = enc.Encode(ctn)
		if err != nil {
			return fmt.Errorf("unable to unmarshal configuration: %v", err)
		}
	}

	return nil
}

// PlanService prepares the service for execution.
func (c *client) PlanService(ctx context.Context, ctn *pipeline.Container) error {
	var err error

	b := c.build
	r := c.repo

	// update the engine service object
	s := new(library.Service)
	s.SetName(ctn.Name)
	s.SetNumber(ctn.Number)
	s.SetStatus(constants.StatusRunning)
	s.SetStarted(time.Now().UTC().Unix())
	s.SetHost(ctn.Environment["VELA_HOST"])
	s.SetRuntime(ctn.Environment["VELA_RUNTIME"])
	s.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])

	// send API call to update the service
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SvcService.Update
	s, _, err = c.Vela.Svc.Update(r.GetOrg(), r.GetName(), b.GetNumber(), s)
	if err != nil {
		return err
	}

	s.SetStatus(constants.StatusSuccess)

	// add a service to a map
	c.services.Store(ctn.ID, s)

	// send API call to capture the service log
	//
	// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#LogService.GetService
	l, _, err := c.Vela.Log.GetService(r.GetOrg(), r.GetName(), b.GetNumber(), s.GetNumber())
	if err != nil {
		return err
	}

	// add a service log to a map
	c.serviceLogs.Store(ctn.ID, l)

	return nil
}

// ExecService runs a service.
func (c *client) ExecService(ctx context.Context, ctn *pipeline.Container) error {
	// run the runtime container
	err := c.Runtime.RunContainer(ctx, ctn, c.pipeline)
	if err != nil {
		return err
	}

	go func() {
		// stream logs from container
		err := c.StreamService(ctx, ctn)
		if err != nil {
			// TODO: Should this be changed or removed?
			fmt.Println(err)
		}
	}()

	return nil
}

// StreamService tails the output for a service.
func (c *client) StreamService(ctx context.Context, ctn *pipeline.Container) error {
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
		// ensure we output to stdout
		fmt.Fprintln(os.Stdout, scanner.Text())
	}

	return scanner.Err()
}

// DestroyService cleans up services after execution.
func (c *client) DestroyService(ctx context.Context, ctn *pipeline.Container) error {
	// load the service from the client
	service, err := c.loadService(ctn.ID)
	if err != nil {
		// create the service from the container
		service = new(library.Service)
		service.SetName(ctn.Name)
		service.SetNumber(ctn.Number)
		service.SetStatus(constants.StatusPending)
		service.SetHost(ctn.Environment["VELA_HOST"])
		service.SetRuntime(ctn.Environment["VELA_RUNTIME"])
		service.SetDistribution(ctn.Environment["VELA_DISTRIBUTION"])
	}

	defer func() {
		// send API call to update the step
		//
		// https://pkg.go.dev/github.com/go-vela/sdk-go/vela?tab=doc#SvcService.Update
		_, _, err := c.Vela.Svc.Update(c.repo.GetOrg(), c.repo.GetName(), c.build.GetNumber(), service)
		if err != nil {
			// TODO: Should this be changed or removed?
			fmt.Println(err)
		}
	}()

	// check if the service is in a pending state
	if service.GetStatus() == constants.StatusPending {
		// update the service fields
		service.SetExitCode(137)
		service.SetFinished(time.Now().UTC().Unix())
		service.SetStatus(constants.StatusKilled)

		// check if the service was not started
		if service.GetStarted() == 0 {
			// set the started time to the finished time
			service.SetStarted(service.GetFinished())
		}
	}

	// inspect the runtime container
	err = c.Runtime.InspectContainer(ctx, ctn)
	if err != nil {
		return err
	}

	// check if the service finished
	if service.GetFinished() == 0 {
		// update the service fields
		service.SetFinished(time.Now().UTC().Unix())
		service.SetStatus(constants.StatusSuccess)

		// check the container for an unsuccessful exit code
		if ctn.ExitCode > 0 {
			// update the service fields
			service.SetExitCode(ctn.ExitCode)
			service.SetStatus(constants.StatusFailure)
		}
	}

	// remove the runtime container
	err = c.Runtime.RemoveContainer(ctx, ctn)
	if err != nil {
		return err
	}

	return nil
}

// loadService is a helper function to capture
// a service from the client.
func (c *client) loadService(name string) (*library.Service, error) {
	// load the service key from the client
	result, ok := c.services.Load(name)
	if !ok {
		return nil, fmt.Errorf("unable to load service %s", name)
	}

	// cast the service key to the expected type
	s, ok := result.(*library.Service)
	if !ok {
		return nil, fmt.Errorf("service %s had unexpected value", name)
	}

	return s, nil
}

// loadServiceLog is a helper function to capture
// the logs for a service from the client.
func (c *client) loadServiceLogs(name string) (*library.Log, error) {
	// load the service log key from the client
	result, ok := c.serviceLogs.Load(name)
	if !ok {
		return nil, fmt.Errorf("unable to load logs for service %s", name)
	}

	// cast the service log key to the expected type
	l, ok := result.(*library.Log)
	if !ok {
		return nil, fmt.Errorf("logs for service %s had unexpected value", name)
	}

	return l, nil
}
