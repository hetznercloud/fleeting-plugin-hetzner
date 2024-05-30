package hetzner

import (
	"errors"
	"fmt"
	"os"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func (g *InstanceGroup) validate() error {
	errs := []error{}

	// Defaults
	if g.settings.Protocol == "" {
		g.settings.Protocol = provider.ProtocolSSH
	}

	if g.settings.Username == "" {
		g.settings.Username = "root"
	}

	// Checks
	if g.Name == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: name"))
	}

	if g.Token == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: token"))
	}

	if g.Location == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: location"))
	}

	if g.ServerType == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: server_type"))
	}

	if g.Image == "" {
		errs = append(errs, fmt.Errorf("missing required plugin config: image"))
	}

	if g.UserData != "" && g.UserDataFile != "" {
		errs = append(errs, fmt.Errorf("mutually exclusive plugin config provided: user_data, user_data_file"))
	}

	if g.settings.Protocol == provider.ProtocolWinRM {
		errs = append(errs, fmt.Errorf("unsupported connector config protocol: %s", g.settings.Protocol))
	}

	return errors.Join(errs...)
}

func (g *InstanceGroup) populate() error {
	if g.UserDataFile != "" {
		userData, err := os.ReadFile(g.UserDataFile)
		if err != nil {
			return fmt.Errorf("failed to read user data file: %w", err)
		}
		g.UserData = string(userData)
	}

	g.labels = map[string]string{
		"managed-by": g.Name,
	}

	return nil
}
