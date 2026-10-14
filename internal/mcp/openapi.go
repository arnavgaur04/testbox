package mcp

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ParamDef is one tool input. Maps to an OpenAPI path or query parameter.
type ParamDef struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"` // path or query
	Required bool   `yaml:"required"`
	Type     string `yaml:"type,omitempty"` // string, integer, boolean. Default string.
}

// ToolDef is one MCP tool. It maps to one HTTP GET endpoint.
type ToolDef struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Method      string     `yaml:"method"`
	Path        string     `yaml:"path"`
	Params      []ParamDef `yaml:"params,omitempty"`
}

// openAPISpec is the minimal subset of OpenAPI we read. GET only for now.
type openAPISpec struct {
	Paths map[string]map[string]openAPIOperation `yaml:"paths"`
}

type openAPIOperation struct {
	OperationID string           `yaml:"operationId"`
	Summary     string           `yaml:"summary"`
	Description string           `yaml:"description"`
	Parameters  []openAPIParam   `yaml:"parameters"`
}

type openAPIParam struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
	Schema   struct {
		Type string `yaml:"type"`
	} `yaml:"schema"`
}

// LoadOpenAPITools reads an OpenAPI file or URL and returns one ToolDef per GET operation.
// Only GET is supported in this phase. Other methods are skipped.
func LoadOpenAPITools(location string) ([]ToolDef, error) {
	data, err := readLocation(location)
	if err != nil {
		return nil, err
	}
	var spec openAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}
	var tools []ToolDef
	for path, ops := range spec.Paths {
		op, ok := ops["get"]
		if !ok {
			continue
		}
		def := ToolDef{
			Name:        toolName(op.OperationID, "get", path),
			Description: firstNonEmpty(op.Description, op.Summary, "GET "+path),
			Method:      "GET",
			Path:        path,
		}
		for _, p := range op.Parameters {
			if p.In != "path" && p.In != "query" {
				continue
			}
			paramType := p.Schema.Type
			if paramType == "" {
				paramType = "string"
			}
			def.Params = append(def.Params, ParamDef{
				Name:     p.Name,
				In:       p.In,
				Required: p.Required || p.In == "path",
				Type:     paramType,
			})
		}
		tools = append(tools, def)
	}
	return tools, nil
}

// toolsFile is the shape of the user custom tools file.
type toolsFile struct {
	Tools []ToolDef `yaml:"tools"`
}

// LoadCustomTools reads a user tools file. Only GET tools allowed in this phase.
func LoadCustomTools(path string) ([]ToolDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools file: %w", err)
	}
	var tf toolsFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("failed to parse tools file: %w", err)
	}
	for _, t := range tf.Tools {
		if t.Name == "" {
			return nil, fmt.Errorf("tool name is required in tools file")
		}
		if t.Method != "" && t.Method != "GET" {
			return nil, fmt.Errorf("tool %s: only GET is supported in this phase", t.Name)
		}
		if t.Path == "" {
			return nil, fmt.Errorf("tool %s: path is required", t.Name)
		}
	}
	// Default empty method to GET.
	for i := range tf.Tools {
		if tf.Tools[i].Method == "" {
			tf.Tools[i].Method = "GET"
		}
	}
	return tf.Tools, nil
}

// MergeTools combines auto tools with custom tools.
// A custom tool with the same name replaces the auto tool. This is the edit path.
// A custom tool with a new name is added. This is the add path.
func MergeTools(auto []ToolDef, custom []ToolDef) []ToolDef {
	byName := map[string]int{}
	merged := append([]ToolDef{}, auto...)
	for i, t := range merged {
		byName[t.Name] = i
	}
	for _, c := range custom {
		if idx, ok := byName[c.Name]; ok {
			merged[idx] = c
			continue
		}
		byName[c.Name] = len(merged)
		merged = append(merged, c)
	}
	return merged
}

// toolName prefers operationId. Falls back to method + path.
func toolName(operationID string, method string, path string) string {
	if operationID != "" {
		return sanitize(operationID)
	}
	return sanitize(method + "_" + path)
}

// sanitize makes a safe tool name: lowercase, alnum and _ only.
func sanitize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	if out == "" {
		return "tool"
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// readLocation reads a local file or an http(s) URL.
func readLocation(location string) ([]byte, error) {
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(location)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s: %w", location, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch %s returned %s", location, resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
	data, err := os.ReadFile(location)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", location, err)
	}
	return data, nil
}
