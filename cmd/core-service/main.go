package main

import (
	"flag"
	"fmt"
	"log"
	"testbox/internal/engine"
	"time"
)

func usage() {
	fmt.Println("Usage: testbox-tool [-repo <path>] [-tail <n>] <command> [service]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  build   Parse make.yaml and build the Docker image.")
	fmt.Println("  up      Prepare compose file and start services.")
	fmt.Println("  down [-v]  Stop services. -v wipes volumes.")
	fmt.Println("  reset   Fresh state. Down -v, then up.")
	fmt.Println("  status  Show compose ps output.")
	fmt.Println("  logs [service]  Show compose logs. Optional service name.")
	fmt.Println("  run     Build then up. Default when no command is given.")
	fmt.Println("  test [--cleanup] [--wait 2m]")
	fmt.Println("          Up, wait ready, run tests: suites.")
	fmt.Println("")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {
	repoPath := flag.String("repo", ".", "Path to the repository containing make.yaml")
	tail := flag.String("tail", "100", "Lines of log output for logs command")
	mcpRef := flag.String("mcp-ref", "master", "Testbox git ref used to build the mcp image")
	flag.Parse()

	// First non-flag arg is command.
	// Default is run.
	cmd := "run"
	rest := flag.Args()
	if len(rest) > 0 {
		cmd = rest[0]
	}
	// Optional service name for logs.
	// Flags for down and test.
	service := ""
	volumes := false
	cleanup := false
	wait := engine.DefaultWait
	extra := []string{}
	if len(rest) > 1 {
		extra = rest[1:]
	}
	for i := 0; i < len(extra); i++ {
		a := extra[i]
		switch {
		case a == "-v" || a == "--volumes":
			volumes = true
		case a == "--cleanup":
			cleanup = true
		case a == "--wait":
			if i+1 >= len(extra) {
				log.Fatal("--wait needs a duration, for example 2m")
			}
			i++
			d, err := time.ParseDuration(extra[i])
			if err != nil {
				log.Fatalf("Invalid --wait: %v", err)
			}
			wait = d
		default:
			if service == "" {
				service = a
			}
		}
	}

	eng := engine.NewEngine()
	eng.McpRef = *mcpRef

	var err error
	switch cmd {
	case "build":
		err = eng.Build(*repoPath)
	case "up":
		err = eng.Up(*repoPath)
	case "down":
		err = eng.Down(*repoPath, volumes)
	case "reset":
		err = eng.Reset(*repoPath)
	case "status":
		var out string
		out, err = eng.Status(*repoPath)
		if err == nil {
			fmt.Print(out)
		}
	case "logs":
		var out string
		out, err = eng.Logs(*repoPath, service, *tail)
		if err == nil {
			fmt.Print(out)
		}
	case "run":
		err = eng.Run(*repoPath)
	case "test":
		_, err = eng.Test(*repoPath, engine.TestOptions{Wait: wait, Cleanup: cleanup, Volumes: volumes})
	case "help", "-h", "--help":
		usage()
		return
	default:
		log.Fatalf("Unknown command: %s. Use help to list commands.", cmd)
	}

	if err != nil {
		log.Fatalf("Command %s failed: %v", cmd, err)
	}

	fmt.Printf("Command %s completed successfully!\n", cmd)
}
