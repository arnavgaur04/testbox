package models

type MakeYaml struct {
	App         AppConfig         `yaml:"app"`
	Docker      DockerConfig      `yaml:"docker"`
	LocalDeploy LocalDeployConfig `yaml:"local_deploy"`
	Helm        *HelmConfig       `yaml:"helm,omitempty"` // omitempty if Helm is optional
	Mcp         *McpConfig        `yaml:"mcp,omitempty"`  // omitempty: no MCP server unless user opts in
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type DockerConfig struct {
	Context    string   `yaml:"context"`
	Dockerfile string   `yaml:"dockerfile"`
	ImageName  string   `yaml:"image_name"`
	BuildArgs  []string `yaml:"build_args,omitempty"`
	Target     string   `yaml:"target,omitempty"`
}

type LocalDeployConfig struct {
	AutoGenerateCompose bool                 `yaml:"auto_generate_compose,omitempty"`
	ComposeFile         string               `yaml:"compose_file,omitempty"` // omitted if auto_generate_compose is true
	Services            []LocalServiceConfig `yaml:"services"`
}

type LocalServiceConfig struct {
	ServiceName   string `yaml:"service_name"`
	Port          int    `yaml:"port"`
	ContainerPort int    `yaml:"container_port,omitempty"`
}

type HelmConfig struct {
	ChartPath   string `yaml:"chart_path"`
	ValuesFile  string `yaml:"values_file,omitempty"`
	ReleaseName string `yaml:"release_name"`
}

// Optional per-service MCP server.
// Off unless enabled is true.
type McpConfig struct {
	Enabled   bool   `yaml:"enabled"`
	OpenAPI   string `yaml:"openapi,omitempty"`    // file path or http(s) URL of OpenAPI spec
	ToolsFile string `yaml:"tools_file,omitempty"` // file path with custom tool defs and overrides
	Port      int    `yaml:"port,omitempty"`       // host port for MCP HTTP. Default 8082.
	Image     string `yaml:"image,omitempty"`      // optional prebuilt image. Skips local build.
}
