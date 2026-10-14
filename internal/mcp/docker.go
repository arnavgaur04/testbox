package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mcpHTTPPort is the port the MCP server listens on inside its container.
const mcpHTTPPort = 8082

// ServiceInfo is the data the compose template needs for the mcp service.
type ServiceInfo struct {
	Enabled       bool
	Target        string   // base URL of the app, reachable inside compose network
	HostPort      int      // host port for MCP HTTP
	ContainerPort int      // container port for MCP HTTP. Always 8082.
	Image         string   // prebuilt image. Empty means local build.
	BuildContext  string   // absolute path of .testbox/mcp. Empty when Image is set.
	StageDir      string   // absolute path of staged spec files. Used for volumes with Image.
	DependsOn     []string // app service names
	HasOpenAPI    bool     // local openapi file staged
	HasTools      bool     // local tools file staged
	OpenAPIRef    string   // /mcp/openapi.yaml or original URL
}

// mcpDockerfileBase builds the mcp-server binary from the public testbox repo.
// No testbox source needed on the host. Only network for git + go modules.
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

// dockerfileFor returns the Dockerfile with COPY lines for staged spec files.
// COPY of a missing file breaks the build, so lines are added only when present.
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

// WriteBuildContext prepares .testbox/mcp for the compose mcp service.
// Spec files are staged there. Built images bake them via Dockerfile.
// Prebuilt images mount them via volumes. No Docker calls here.
func WriteBuildContext(repoPath string, openAPI string, toolsFile string, target string, hostPort int, image string, dependsOn []string) (*ServiceInfo, error) {
	info := &ServiceInfo{
		Enabled:       true,
		Target:        target,
		HostPort:      hostPort,
		ContainerPort: mcpHTTPPort,
		Image:         image,
		DependsOn:     dependsOn,
	}
	dir := filepath.Join(repoPath, ".testbox", "mcp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mcp build dir: %w", err)
	}
	info.StageDir = dir
	// Local spec files are staged for bake or mount. URLs pass through.
	if openAPI != "" && !isURL(openAPI) {
		data, err := os.ReadFile(resolveRepoPath(repoPath, openAPI))
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
		data, err := os.ReadFile(resolveRepoPath(repoPath, toolsFile))
		if err != nil {
			return nil, fmt.Errorf("failed to read mcp tools file: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), data, 0644); err != nil {
			return nil, fmt.Errorf("failed to copy mcp tools file: %w", err)
		}
		info.HasTools = true
	}
	if image == "" {
		// Dockerfile is written last, so COPY lines match staged files.
		content := dockerfileFor(info.HasOpenAPI, info.HasTools)
		if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write mcp Dockerfile: %w", err)
		}
		info.BuildContext = dir
	}
	return info, nil
}

// Command returns the container command flags for the mcp service.
func (s *ServiceInfo) Command() []string {
	cmd := []string{"--transport", "http", "--addr", fmt.Sprintf(":%d", mcpHTTPPort), "--target", s.Target}
	if s.OpenAPIRef != "" {
		cmd = append(cmd, "--openapi", s.OpenAPIRef)
	}
	if s.HasTools {
		cmd = append(cmd, "--tools", "/mcp/tools.yaml")
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
