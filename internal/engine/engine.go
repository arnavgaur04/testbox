package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testbox/internal/deployer"
	"testbox/internal/docker"
	"testbox/internal/parser"
	"testbox/internal/report"
	"testbox/pkg/models"
	"time"
)

type Engine struct {
	builder *docker.Builder
	// Git ref for mcp image build.
	McpRef string
}

func NewEngine() *Engine {
	return &Engine{
		builder: &docker.Builder{},
		McpRef:  "master",
	}
}

// Reads make.yaml from repo.
func (e *Engine) LoadConfig(repoPath string) (*models.MakeYaml, error) {
	configPath := filepath.Join(repoPath, "make.yaml")
	return parser.ParseMakeYaml(configPath)
}

// Creates .testbox workspace dir.
func (e *Engine) ensureTestboxDir(repoPath string) (string, error) {
	testboxDir := filepath.Join(repoPath, ".testbox")
	if err := os.MkdirAll(testboxDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .testbox directory: %w", err)
	}
	return testboxDir, nil
}

// Picks compose files to use.
// Custom plus mcp adds override.
func (e *Engine) resolveComposeFiles(repoPath string, config *models.MakeYaml) ([]string, error) {
	testboxDir, err := e.ensureTestboxDir(repoPath)
	if err != nil {
		return nil, err
	}
	if config.LocalDeploy.AutoGenerateCompose {
		if err := deployer.GenerateComposeFile(repoPath, config, e.McpRef); err != nil {
			return nil, err
		}
		return []string{filepath.Join(testboxDir, "docker-compose.local.yaml")}, nil
	}
	p := config.LocalDeploy.ComposeFile
	if !filepath.IsAbs(p) {
		p = filepath.Join(repoPath, p)
	}
	files := []string{p}
	if config.Mcp != nil && config.Mcp.Enabled {
		override, err := deployer.WriteMcpOverride(repoPath, config, e.McpRef)
		if err != nil {
			return nil, err
		}
		files = append(files, override)
	}
	return files, nil
}

// Builds the image from config.
func (e *Engine) Build(repoPath string) error {
	if _, err := e.ensureTestboxDir(repoPath); err != nil {
		return err
	}
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	// Context joins repo + rel path.
	config.Docker.Context = filepath.Join(repoPath, config.Docker.Context)
	fmt.Printf("Building app: %s\n", config.App.Name)
	return e.builder.Build(config.Docker)
}

// Starts services via compose.
func (e *Engine) Up(repoPath string) error {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	files, err := e.resolveComposeFiles(repoPath, config)
	if err != nil {
		return err
	}
	return deployer.Up(repoPath, files)
}

// Stops services via compose.
func (e *Engine) Down(repoPath string, volumes bool) error {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	files, err := e.resolveComposeFiles(repoPath, config)
	if err != nil {
		return err
	}
	return deployer.Down(repoPath, files, volumes)
}

// Fresh state. Down with volumes, up.
func (e *Engine) Reset(repoPath string) error {
	if err := e.Down(repoPath, true); err != nil {
		return err
	}
	return e.Up(repoPath)
}

// Returns compose ps output.
func (e *Engine) Status(repoPath string) (string, error) {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return "", err
	}
	files, err := e.resolveComposeFiles(repoPath, config)
	if err != nil {
		return "", err
	}
	return deployer.Status(repoPath, files)
}

// Returns compose logs output.
func (e *Engine) Logs(repoPath string, service string, tail string) (string, error) {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return "", err
	}
	files, err := e.resolveComposeFiles(repoPath, config)
	if err != nil {
		return "", err
	}
	return deployer.Logs(repoPath, files, service, tail)
}

// TestOptions tunes the test run.
// Zero value uses defaults.
type TestOptions struct {
	Wait        time.Duration
	Cleanup     bool
	Volumes     bool
	ReportJSON  string
	ReportJUnit string
}

// Default wait when none given.
const DefaultWait = 120 * time.Second

// Runs test suites. Up and wait first.
// Runs all suites. Errors at end.
func (e *Engine) Test(repoPath string, opts TestOptions) ([]deployer.SuiteResult, error) {
	started := time.Now()
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return nil, err
	}
	if len(config.Tests) == 0 {
		fmt.Println("no tests defined in make.yaml")
		return nil, nil
	}
	files, err := e.resolveComposeFiles(repoPath, config)
	if err != nil {
		return nil, err
	}
	if err := deployer.Up(repoPath, files); err != nil {
		return nil, err
	}
	wait := opts.Wait
	if wait == 0 {
		wait = DefaultWait
	}
	fmt.Printf("Waiting up to %s for services.\n", wait)
	if err := deployer.WaitComposeReady(repoPath, files, wait); err != nil {
		return nil, err
	}
	fmt.Println("Services ready.")
	results := []deployer.SuiteResult{}
	failed := 0
	for _, suite := range config.Tests {
		res, err := deployer.RunSuite(repoPath, files, suite)
		results = append(results, res)
		if err != nil {
			failed++
			fmt.Printf("FAIL %s\n", suite.Name)
			continue
		}
		fmt.Printf("PASS %s\n", suite.Name)
	}
	fmt.Printf("%d passed, %d failed\n", len(results)-failed, failed)
	if opts.ReportJSON != "" || opts.ReportJUnit != "" {
		run := report.Build(repoPath, config.App.Name, config.App.Version, started, results)
		if opts.ReportJSON != "" {
			if err := report.WriteJSON(opts.ReportJSON, run); err != nil {
				return results, err
			}
			fmt.Printf("JSON report: %s\n", opts.ReportJSON)
		}
		if opts.ReportJUnit != "" {
			if err := report.WriteJUnit(opts.ReportJUnit, run); err != nil {
				return results, err
			}
			fmt.Printf("JUnit report: %s\n", opts.ReportJUnit)
		}
	}
	var cleanupErr error
	if opts.Cleanup {
		fmt.Println("Cleaning up services.")
		cleanupErr = e.Down(repoPath, opts.Volumes)
	}
	if failed > 0 {
		return results, fmt.Errorf("%d suites failed", failed)
	}
	return results, cleanupErr
}

// Helm config or a clear error.
// Helm is opt-in via make.yaml.
func (e *Engine) helmConfig(repoPath string) (models.HelmConfig, error) {
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return models.HelmConfig{}, err
	}
	if config.Helm == nil {
		return models.HelmConfig{}, fmt.Errorf("helm section is not configured in make.yaml")
	}
	return *config.Helm, nil
}

// Deploys the helm release.
func (e *Engine) HelmUp(repoPath string) error {
	cfg, err := e.helmConfig(repoPath)
	if err != nil {
		return err
	}
	return deployer.HelmUp(repoPath, cfg)
}

// Returns helm release status.
func (e *Engine) HelmStatus(repoPath string) (string, error) {
	cfg, err := e.helmConfig(repoPath)
	if err != nil {
		return "", err
	}
	return deployer.HelmStatus(repoPath, cfg)
}

// Removes the helm release.
func (e *Engine) HelmDown(repoPath string) error {
	cfg, err := e.helmConfig(repoPath)
	if err != nil {
		return err
	}
	return deployer.HelmDown(repoPath, cfg)
}

// Build then Up in one step.
func (e *Engine) Run(repoPath string) error {
	if err := e.Build(repoPath); err != nil {
		return err
	}
	// Config is small. Reload it here.
	config, err := e.LoadConfig(repoPath)
	if err != nil {
		return err
	}
	files, err := e.resolveComposeFiles(repoPath, config)
	if err != nil {
		return err
	}
	if err := deployer.Up(repoPath, files); err != nil {
		return err
	}
	fmt.Println("Deployment configuration ready.")
	return nil
}
