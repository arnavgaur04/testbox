package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// maxBody limits tool output. AI context is finite. 8000 chars is enough for most APIs.
const maxBody = 8000

// BuildServer registers one MCP tool per ToolDef. All tools run HTTP GET.
func BuildServer(name string, defs []ToolDef, targetBase string) *server.MCPServer {
	s := server.NewMCPServer(name, "0.1.0")
	for _, def := range defs {
		def := def
		opts := []mcplib.ToolOption{mcplib.WithDescription(def.Description)}
		for _, p := range def.Params {
			opts = append(opts, paramOption(p))
		}
		s.AddTool(mcplib.NewTool(def.Name, opts...), func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			out, err := runGET(ctx, targetBase, def, req)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			return mcplib.NewToolResultText(out), nil
		})
	}
	return s
}

// paramOption maps a ParamDef to an mcp-go tool option.
func paramOption(p ParamDef) mcplib.ToolOption {
	desc := fmt.Sprintf("%s parameter %s", p.In, p.Name)
	switch p.Type {
	case "integer", "number":
		if p.Required {
			return mcplib.WithNumber(p.Name, mcplib.Description(desc), mcplib.Required())
		}
		return mcplib.WithNumber(p.Name, mcplib.Description(desc))
	case "boolean":
		if p.Required {
			return mcplib.WithBoolean(p.Name, mcplib.Description(desc), mcplib.Required())
		}
		return mcplib.WithBoolean(p.Name, mcplib.Description(desc))
	default:
		if p.Required {
			return mcplib.WithString(p.Name, mcplib.Description(desc), mcplib.Required())
		}
		return mcplib.WithString(p.Name, mcplib.Description(desc))
	}
}

// runGET builds the URL from path template + query params and runs the request.
func runGET(ctx context.Context, base string, def ToolDef, req mcplib.CallToolRequest) (string, error) {
	return ExecGET(ctx, base, def, req.GetArguments())
}

// ExecGET is the testable core of runGET. args maps param name to value.
func ExecGET(ctx context.Context, base string, def ToolDef, args map[string]any) (string, error) {
	path := def.Path
	// Path params fill {name} slots. Missing required param is an error.
	for _, p := range def.Params {
		if p.In != "path" {
			continue
		}
		val, ok := args[p.Name]
		if !ok || val == nil || fmt.Sprint(val) == "" {
			return "", fmt.Errorf("missing required path param: %s", p.Name)
		}
		path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(fmt.Sprint(val)))
	}
	// Query params go to the query string. Skip empty optional ones.
	query := url.Values{}
	for _, p := range def.Params {
		if p.In != "query" {
			continue
		}
		val, ok := args[p.Name]
		if !ok || val == nil {
			continue
		}
		s := fmt.Sprint(val)
		if s == "" && !p.Required {
			continue
		}
		query.Set(p.Name, s)
	}
	full := strings.TrimRight(base, "/") + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return "", fmt.Errorf("bad request: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}
	out := fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body))
	if len(out) > maxBody {
		out = out[:maxBody] + "\n...[truncated]"
	}
	return out, nil
}
