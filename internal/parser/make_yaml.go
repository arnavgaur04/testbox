package parser

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"

	"testbox/pkg/models"
)

// ParseMakeYaml reads and validates a make.yaml file.
func ParseMakeYaml(filePath string) (*models.MakeYaml, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("make.yaml not found at path: %s", filePath)
	}

	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read make.yaml: %w", err)
	}

	var config models.MakeYaml
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal make.yaml: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return &config, nil
}

func validateConfig(config *models.MakeYaml) error {
	// App Validation
	if config.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if config.App.Version == "" {
		return fmt.Errorf("app.version is required")
	}

	// Docker Validation
	if config.Docker.Context == "" {
		return fmt.Errorf("docker.context is required")
	}
	if config.Docker.Dockerfile == "" {
		return fmt.Errorf("docker.dockerfile is required")
	}
	if config.Docker.ImageName == "" {
		return fmt.Errorf("docker.image_name is required")
	}

	// Local Deployment Validation
	if !config.LocalDeploy.AutoGenerateCompose && config.LocalDeploy.ComposeFile == "" {
		return fmt.Errorf("local_deploy.compose_file is required when auto_generate_compose is false")
	}

	for _, svc := range config.LocalDeploy.Services {
		if svc.ServiceName == "" {
			return fmt.Errorf("service_name is required for all local_deploy services")
		}
		if svc.Port <= 0 || svc.Port > 65535 {
			return fmt.Errorf("invalid port %d for service %s: must be between 1 and 65535", svc.Port, svc.ServiceName)
		}
	}

	return nil
}
