package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testbox/internal/deployer"
	"testbox/internal/docker"
	"testbox/internal/parser"
	"testbox/pkg/models"
)

type Engine struct {
	builder *docker.Builder
	// McpRef is the testbox git ref used to build the mcp image. Default master.
	McpRef string
}

func NewEngine() *Engine {
	return &Engine{
		builder: &docker.Builder{},
		McpRef:  "master",
	}
}

// LoadConfig reads make.yaml from repoPath.
func (e *Engine) LoadConfig(repoPath string) (*models.MakeYaml, error) {
	configPath := filepath.Join(repoPath, "make.yaml")
	return parser.ParseMakeYaml(configPath)
}

// ensureTestboxDir creates .testbox dir. It holds generated files.
func (e *Engine) ensureTestboxDir(repoPath string) (string, error) {
	testboxDir := filepath.Join(repoPath, ".testbox")
	if err := os.MkdirAll(testboxDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .testbox directory: %w", err)
	}
	return testboxDir, nil
}

// resolveComposeFile returns the compose file to use.
// If auto_generate is true, it generates the file and returns its path.
// If false, it resolves the user file path.
func (e *Engine) resolveComposeFile(repoPath string, config *models.MakeYaml) (string, error) {
	testboxDir, err := e.ensureTestboxDir(repoPath)
	if err != nil {
		return "", err
	}
	if config.LocalDeploy.AutoGenerateCompose {
		if err := deployer.GenerateComposeFile(repoPath, config, e.McpRef); err != nil {
			return "", err
		}
		return filepath.Join(testboxDir, "docker-compose.local.yaml"), nil
	}
	p := config.LocalDeploy.ComposeFile
	if !filepath.IsAbs(p) {
		p = filepath.Join(repoPath, p)
	}
	return p, nil
}

// Build parses config and builds the image.
func (e *Engine) Build(repoPath string) error {
	if _, err := e.ensureTestboxDir(repoPath); err != nil {
		return err
	}
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	// Join repoPath with relative context.
	// Example: repoPath=/tmp/app + context=. = /tmp/app
	config.Docker.Context = filepath.Join(repoPath, config.Docker.Context)
	fmt.Printf("Building app: %s\n", config.App.Name)
	return e.builder.Build(config.Docker)
}

// Up prepares the compose file and starts services.
func (e *Engine) Up(repoPath string) error {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	composeFile, err := e.resolveComposeFile(repoPath, config)
	if err != nil {
		return err
	}
	return deployer.Up(repoPath, composeFile)
}

// Down stops services.
func (e *Engine) Down(repoPath string) error {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	composeFile, err := e.resolveComposeFile(repoPath, config)
	if err != nil {
		return err
	}
	return deployer.Down(repoPath, composeFile)
}

// Status returns compose ps output. Read-only. It regenerates the file if auto_generate is true.
func (e *Engine) Status(repoPath string) (string, error) {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return "", err
	}
	composeFile, err := e.resolveComposeFile(repoPath, config)
	if err != nil {
		return "", err
	}
	return deployer.Status(repoPath, composeFile)
}

// Logs returns compose logs. service is optional. tail is line count, default 100.
func (e *Engine) Logs(repoPath string, service string, tail string) (string, error) {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return "", err
	}
	composeFile, err := e.resolveComposeFile(repoPath, config)
	if err != nil {
		return "", err
	}
	return deployer.Logs(repoPath, composeFile, service, tail)
}

// Run does Build then Up. Full local deploy in one step.
func (e *Engine) Run(repoPath string) error {
	if err := e.Build(repoPath); err != nil {
		return err
	}
	// Load again for deploy step. Config is small. This keeps steps separate.
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	composeFile, err := e.resolveComposeFile(repoPath, config)
	if err != nil {
		return err
	}
	if err := deployer.Up(repoPath, composeFile); err != nil {
		return err
	}
	fmt.Println("Deployment configuration ready.")
	return nil
}
