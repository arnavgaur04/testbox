package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testbox/internal/docker"
	"testbox/internal/parser"
)

type Engine struct {
	builder *docker.Builder
}

func NewEngine() *Engine {
	return &Engine{
		builder: &docker.Builder{},
	}
}

func (e *Engine) Run(repoPath string) error {
	// 1. Setup .testbox directory
	testboxDir := filepath.Join(repoPath, ".testbox")
	if err := os.MkdirAll(testboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create .testbox directory: %w", err)
	}

	// 2. Parse config
	configPath := filepath.Join(repoPath, "make.yaml")
	config, err := parser.ParseMakeYaml(configPath)
	if err != nil {
		return err
	}

// 3. Build image
	// Adjust context if it's relative to repoPath
	config.Docker.Context = filepath.Join(repoPath, config.Docker.Context)
	fmt.Printf("Building app: %s\n", config.App.Name)
	if err := e.builder.Build(config.Docker); err != nil {
		return err
	}

	// 4. Deployment placeholder
	fmt.Println("Deployment logic would go here.")
	
	return nil
}
