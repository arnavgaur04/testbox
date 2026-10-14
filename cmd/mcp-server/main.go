package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/server"
	testmcp "testbox/internal/mcp"
)

func main() {
	openapi := flag.String("openapi", "", "OpenAPI file path or URL. Optional if --tools is given.")
	tools := flag.String("tools", "", "Custom tools YAML file. Optional if --openapi is given.")
	target := flag.String("target", "", "Base URL of the app. Required. Example: http://my-app:80")
	name := flag.String("name", "testbox-mcp", "MCP server name.")
	transport := flag.String("transport", "stdio", "Transport: stdio or http.")
	addr := flag.String("addr", ":8082", "Listen address for http transport.")
	flag.Parse()

	if *target == "" {
		log.Fatal("--target is required")
	}
	if *openapi == "" && *tools == "" {
		log.Fatal("one of --openapi or --tools is required")
	}
	if *transport != "stdio" && *transport != "http" {
		log.Fatal("--transport must be stdio or http")
	}

	var auto []testmcp.ToolDef
	if *openapi != "" {
		var err error
		auto, err = testmcp.LoadOpenAPITools(*openapi)
		if err != nil {
			log.Fatalf("load openapi: %v", err)
		}
	}
	var custom []testmcp.ToolDef
	if *tools != "" {
		var err error
		custom, err = testmcp.LoadCustomTools(*tools)
		if err != nil {
			log.Fatalf("load tools: %v", err)
		}
	}
	defs := testmcp.MergeTools(auto, custom)
	if len(defs) == 0 {
		log.Fatal("no tools found: spec has no GET operations and no custom tools")
	}
	fmt.Printf("Loaded %d tools (%d auto, %d custom).\n", len(defs), len(auto), len(custom))

	s := testmcp.BuildServer(*name, defs, *target)

	if *transport == "stdio" {
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("stdio server failed: %v", err)
		}
		return
	}
	httpSrv := server.NewStreamableHTTPServer(s)
	fmt.Printf("MCP HTTP listening on %s\n", *addr)
	if err := httpSrv.Start(*addr); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}
