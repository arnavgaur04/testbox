package deployer

import (
	"fmt"
	"os"
	"path/filepath"
	testmcp "testbox/internal/mcp"
	"testbox/pkg/models"
	"text/template"
)

// Template for generated compose.
// No 'version' key. V2 ignores it.
const ComposeTemplate = `services:
{{- range .Services }}
  {{ .ServiceName }}:
    image: {{ $.ImageName }}
    ports:
      - "{{ .Port }}:{{ .ContainerPort }}"
{{- end }}
{{- if .Mcp.Enabled }}
  mcp:
{{- if .Mcp.Image }}
    image: {{ .Mcp.Image }}
{{- else }}
    build:
      context: {{ .Mcp.BuildContext }}
      args:
        TESTBOX_REF: {{ .McpRef }}
{{- end }}
    command:
{{- range .Mcp.Command }}
      - "{{ . }}"
{{- end }}
    ports:
      - "{{ .Mcp.HostPort }}:{{ .Mcp.ContainerPort }}"
{{- if .Mcp.DependsOn }}
    depends_on:
{{- range .Mcp.DependsOn }}
      - {{ . }}
{{- end }}
{{- end }}
{{- if and .Mcp.Image (or .Mcp.HasOpenAPI .Mcp.HasTools) }}
    volumes:
{{- if .Mcp.HasOpenAPI }}
      - "{{ .Mcp.StageDir }}/openapi.yaml:/mcp/openapi.yaml:ro"
{{- end }}
{{- if .Mcp.HasTools }}
      - "{{ .Mcp.StageDir }}/tools.yaml:/mcp/tools.yaml:ro"
{{- end }}
{{- end }}
{{- end }}
`

// Writes docker-compose.local.yaml.
// mcpRef sets the mcp build ref.
func GenerateComposeFile(repoPath string, config *models.MakeYaml, mcpRef string) error {
	testboxDir := filepath.Join(repoPath, ".testbox")
	if err := os.MkdirAll(testboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create .testbox directory: %w", err)
	}
	composeFilePath := filepath.Join(testboxDir, "docker-compose.local.yaml")

	// MCP service is opt-in. Off by default.
	mcpInfo, err := mcpServiceInfo(repoPath, config)
	if err != nil {
		return err
	}

	// Prepare data for the template
	data := struct {
		ImageName string
		Services  []models.LocalServiceConfig
		Mcp       *testmcp.ServiceInfo
		McpRef    string
	}{
		ImageName: config.Docker.ImageName,
		Services:  config.LocalDeploy.Services,
		Mcp:       mcpInfo,
		McpRef:    mcpRef,
	}
	// Nil-safe default when mcp is off.
	if data.Mcp == nil {
		data.Mcp = &testmcp.ServiceInfo{}
	}

	// Create the file
	file, err := os.Create(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create compose file: %w", err)
	}
	defer file.Close()

	// Parse and execute the template
	tmpl, err := template.New("compose").Parse(ComposeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated compose file at: %s\n", composeFilePath)
	return nil
}

// Stages mcp build context.
// Nil when mcp is off.
func mcpServiceInfo(repoPath string, config *models.MakeYaml) (*testmcp.ServiceInfo, error) {
	if config.Mcp == nil || !config.Mcp.Enabled {
		return nil, nil
	}
	if len(config.LocalDeploy.Services) == 0 {
		return nil, fmt.Errorf("mcp needs at least one local_deploy service as target")
	}
	first := config.LocalDeploy.Services[0]
	target := fmt.Sprintf("http://%s:%d", first.ServiceName, first.ContainerPort)
	dependsOn := make([]string, 0, len(config.LocalDeploy.Services))
	for _, s := range config.LocalDeploy.Services {
		dependsOn = append(dependsOn, s.ServiceName)
	}
	return testmcp.WriteBuildContext(repoPath, config.Mcp.OpenAPI, config.Mcp.ToolsFile, target, config.Mcp.Port, config.Mcp.Image, dependsOn)
}

// Runs compose up -d.
// Tries v2 first, then v1.
func Up(repoPath string, composeFilePath string) error {
	if _, err := os.Stat(composeFilePath); err != nil {
		return fmt.Errorf("compose file not found: %s", composeFilePath)
	}
	output, err := composeStream(repoPath, composeFilePath, "up", "-d")
	if err != nil {
		return fmt.Errorf("compose up failed: %w\nOutput: %s", err, output)
	}
	fmt.Println("Containers started successfully.")
	return nil
}

// Stops compose services.
func Down(repoPath string, composeFilePath string) error {
	if _, err := os.Stat(composeFilePath); err != nil {
		return fmt.Errorf("compose file not found: %s", composeFilePath)
	}
	output, err := composeStream(repoPath, composeFilePath, "down")
	if err != nil {
		return fmt.Errorf("compose down failed: %w\nOutput: %s", err, output)
	}
	fmt.Println("Containers stopped successfully.")
	return nil
}
