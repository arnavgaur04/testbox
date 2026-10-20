package deployer

import (
	"fmt"
	"os"
	"path/filepath"
	"testbox/pkg/models"
)

// Default helm wait timeout.
// Long enough for image pulls.
const helmTimeout = "10m"

// Shared helm flags from config.
func helmFlags(cfg models.HelmConfig) []string {
	args := []string{}
	if cfg.Namespace != "" {
		args = append(args, "--namespace", cfg.Namespace, "--create-namespace")
	}
	if cfg.KubeContext != "" {
		args = append(args, "--kube-context", cfg.KubeContext)
	}
	return args
}

// Builds 'helm upgrade --install' args.
// Chart and values are absolute paths.
func HelmUpArgs(cfg models.HelmConfig, chartPath string, valuesPath string) []string {
	args := []string{"upgrade", "--install", cfg.ReleaseName, chartPath, "--wait", "--timeout", helmTimeout}
	args = append(args, helmFlags(cfg)...)
	if valuesPath != "" {
		args = append(args, "--values", valuesPath)
	}
	return args
}

// Builds 'helm status' args.
func HelmStatusArgs(cfg models.HelmConfig) []string {
	args := []string{"status", cfg.ReleaseName}
	if cfg.Namespace != "" {
		args = append(args, "--namespace", cfg.Namespace)
	}
	if cfg.KubeContext != "" {
		args = append(args, "--kube-context", cfg.KubeContext)
	}
	return args
}

// Builds 'helm uninstall' args.
func HelmDownArgs(cfg models.HelmConfig) []string {
	args := []string{"uninstall", cfg.ReleaseName}
	if cfg.Namespace != "" {
		args = append(args, "--namespace", cfg.Namespace)
	}
	if cfg.KubeContext != "" {
		args = append(args, "--kube-context", cfg.KubeContext)
	}
	return args
}

// Resolves chart and values paths.
// Chart must exist before calling helm.
func resolveHelmPaths(repoPath string, cfg models.HelmConfig) (string, string, error) {
	chart := cfg.ChartPath
	if !filepath.IsAbs(chart) {
		chart = filepath.Join(repoPath, chart)
	}
	if _, err := os.Stat(chart); err != nil {
		return "", "", fmt.Errorf("helm chart not found: %s", chart)
	}
	values := ""
	if cfg.ValuesFile != "" {
		values = cfg.ValuesFile
		if !filepath.IsAbs(values) {
			values = filepath.Join(repoPath, values)
		}
		if _, err := os.Stat(values); err != nil {
			return "", "", fmt.Errorf("helm values file not found: %s", values)
		}
	}
	return chart, values, nil
}

// Installs or upgrades the release.
func HelmUp(repoPath string, cfg models.HelmConfig) error {
	chart, values, err := resolveHelmPaths(repoPath, cfg)
	if err != nil {
		return err
	}
	args := HelmUpArgs(cfg, chart, values)
	fmt.Printf("Running: helm %v\n", args)
	out, err := streamCommand(repoPath, "helm", args)
	if err != nil {
		return fmt.Errorf("helm upgrade failed: %w\nOutput: %s", err, out)
	}
	fmt.Print(out)
	fmt.Println("Helm release deployed.")
	return nil
}

// Returns release status.
func HelmStatus(repoPath string, cfg models.HelmConfig) (string, error) {
	out, err := streamCommand(repoPath, "helm", HelmStatusArgs(cfg))
	if err != nil {
		return "", fmt.Errorf("helm status failed: %w\nOutput: %s", err, out)
	}
	return out, nil
}

// Removes the release.
func HelmDown(repoPath string, cfg models.HelmConfig) error {
	out, err := streamCommand(repoPath, "helm", HelmDownArgs(cfg))
	if err != nil {
		return fmt.Errorf("helm uninstall failed: %w\nOutput: %s", err, out)
	}
	fmt.Print(out)
	fmt.Println("Helm release removed.")
	return nil
}
