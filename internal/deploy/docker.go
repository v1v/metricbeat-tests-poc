// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/compose"
	"github.com/elastic/e2e-testing/internal/docker"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// DockerDeploymentManifest deploy manifest for docker
type dockerDeploymentManifest struct {
	Context context.Context
}

func newDockerDeploy() Deployment {
	return &dockerDeploymentManifest{Context: context.Background()}
}

// Add adds services deployment
func (c *dockerDeploymentManifest) Add(services []string, env map[string]string) error {
	serviceManager := compose.NewServiceManager()

	return serviceManager.AddServicesToCompose(c.Context, services[0], services[1:], env)
}

// Bootstrap sets up environment with docker compose
func (c *dockerDeploymentManifest) Bootstrap(waitCB func() error) error {
	serviceManager := compose.NewServiceManager()
	common.ProfileEnv = map[string]string{
		"kibanaVersion": common.KibanaVersion,
		"stackVersion":  common.StackVersion,
	}

	common.ProfileEnv["kibanaDockerNamespace"] = "kibana"
	if strings.HasPrefix(common.KibanaVersion, "pr") || utils.IsCommit(common.KibanaVersion) {
		// because it comes from a PR
		common.ProfileEnv["kibanaDockerNamespace"] = "observability-ci"
	}

	profile := common.FleetProfileName
	err := serviceManager.RunCompose(c.Context, true, []string{profile}, common.ProfileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"profile": profile,
			"error":   err.Error(),
		}).Fatal("Could not run the runtime dependencies for the profile.")
	}
	err = waitCB()
	if err != nil {
		return err
	}
	return nil
}

// Destroy teardown docker environment
func (c *dockerDeploymentManifest) Destroy() error {
	serviceManager := compose.NewServiceManager()
	err := serviceManager.StopCompose(c.Context, true, []string{common.FleetProfileName})
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"profile": common.FleetProfileName,
		}).Fatal("Could not destroy the runtime dependencies for the profile.")
	}
	return nil
}

// ExecIn execute command in service
func (c *dockerDeploymentManifest) ExecIn(service string, cmd []string) (string, error) {
	output, err := docker.ExecCommandIntoContainer(c.Context, service, "root", cmd)
	if err != nil {
		return "", err
	}
	return output, nil
}

// Inspect inspects a service
func (c *dockerDeploymentManifest) Inspect(service string) (*ServiceManifest, error) {
	inspect, err := docker.InspectContainer(service)
	if err != nil {
		return &ServiceManifest{}, err
	}
	return &ServiceManifest{
		ID:         inspect.ID,
		Name:       strings.TrimPrefix(inspect.Name, "/"),
		Connection: service,
		Hostname:   inspect.NetworkSettings.Networks["fleet_default"].Aliases[0],
	}, nil
}

// Remove remove services from deployment
func (c *dockerDeploymentManifest) Remove(services []string, env map[string]string) error {
	serviceManager := compose.NewServiceManager()

	return serviceManager.RemoveServicesFromCompose(c.Context, services[0], services[1:], env)
}