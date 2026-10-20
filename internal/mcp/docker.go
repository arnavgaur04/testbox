package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MCP server port in container.
const mcpHTTPPort = 8082

// Data for the mcp compose block.
type ServiceInfo struct {
	Enabled       bool
	Target        string            // base URL of the app, reachable inside compose network
	HostPort      int               // host port for MCP HTTP
	ContainerPort int               // container port for MCP HTTP. Always 8082.
	Image         string            // prebuilt image. Empty means local build.
	BuildContext  string            // absolute path of .testbox/mcp. Empty when Image is set.
	StageDir      string            // absolute path of staged spec files. Used for volumes with Image.
	DependsOn     []string          // app service names
	HasOpenAPI    bool              // local openapi file staged
	HasTools      bool              // local tools file staged
	OpenAPIRef    string            // /mcp/openapi.yaml or original URL
	Headers       map[string]string // auth headers for tool calls
}

// Builds mcp-server from source.
// Needs network for git + modules.
const mcpDockerfileBase = `FROM golang:1.25-alpine AS build
ARG TESTBOX_REF=master
RUN apk add --no-cache git && \
  git clone https://github.com/arnavgaur04/testbox /src && \
  cd /src && git checkout ${TESTBOX_REF} && \
  CGO_ENABLED=0 go build -o /mcp-server ./cmd/mcp-server
FROM alpine:3.21
COPY --from=build /mcp-server /mcp-server
ENTRYPOINT ["/mcp-server"]
`

// Dockerfile with COPY for staged.
// Missing file breaks COPY.
func dockerfileFor(hasOpenAPI bool, hasTools bool) string {
	var b strings.Builder
	b.WriteString(mcpDockerfileBase)
	if hasOpenAPI {
		b.WriteString("COPY openapi.yaml /mcp/openapi.yaml\n")
	}
	if hasTools {
		b.WriteString("COPY tools.yaml /mcp/tools.yaml\n")
	}
	return b.String()
}

// Stages .testbox/mcp for compose.
// Bake or mount specs. No Docker.
func WriteBuildContext(repoPath string, openAPI string, toolsFile string, target string, hostPort int, image string, dependsOn []string, headers map[string]string) (*ServiceInfo, error) {
	info := &ServiceInfo{
		Enabled:       true,
		Target:        target,
		HostPort:      hostPort,
		ContainerPort: mcpHTTPPort,
		Image:         image,
		DependsOn:     dependsOn,
		Headers:       headers,
	}
	// Absolute paths survive -repo.
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repo path: %w", err)
	}
	dir := filepath.Join(absRepo, ".testbox", "mcp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mcp build dir: %w", err)
	}
	info.StageDir = dir
	// Local files stage. URLs pass.
	if openAPI != "" && !isURL(openAPI) {
		data, err := os.ReadFile(resolveRepoPath(absRepo, openAPI))
		if err != nil {
			return nil, fmt.Errorf("failed to read mcp openapi file: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), data, 0644); err != nil {
			return nil, fmt.Errorf("failed to copy mcp openapi file: %w", err)
		}
		info.HasOpenAPI = true
		info.OpenAPIRef = "/mcp/openapi.yaml"
	} else {
		info.OpenAPIRef = openAPI
	}
	if toolsFile != "" {
		data, err := os.ReadFile(resolveRepoPath(absRepo, toolsFile))
		if err != nil {
			return nil, fmt.Errorf("failed to read mcp tools file: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), data, 0644); err != nil {
			return nil, fmt.Errorf("failed to copy mcp tools file: %w", err)
		}
		info.HasTools = true
	}
	if image == "" {
		// COPY must match staged files.
		content := dockerfileFor(info.HasOpenAPI, info.HasTools)
		if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write mcp Dockerfile: %w", err)
		}
		info.BuildContext = dir
	}
	return info, nil
}

// Container flags for mcp service.
func (s *ServiceInfo) Command() []string {
	cmd := []string{"--transport", "http", "--addr", fmt.Sprintf(":%d", mcpHTTPPort), "--target", s.Target}
	if s.OpenAPIRef != "" {
		cmd = append(cmd, "--openapi", s.OpenAPIRef)
	}
	if s.HasTools {
		cmd = append(cmd, "--tools", "/mcp/tools.yaml")
	}
	// Sorted so output is stable.
	names := make([]string, 0, len(s.Headers))
	for name := range s.Headers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd = append(cmd, "--header", name+"="+s.Headers[name])
	}
	return cmd
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func resolveRepoPath(repoPath string, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(repoPath, p)
}
