package main

import (
	"flag"
	"fmt"
	"log"
	"testbox/internal/engine"
)

func usage() {
	fmt.Println("Usage: testbox-tool [-repo <path>] [-tail <n>] <command> [service]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  build   Parse make.yaml and build the Docker image.")
	fmt.Println("  up      Prepare compose file and start services.")
	fmt.Println("  down    Stop services.")
	fmt.Println("  status  Show compose ps output.")
	fmt.Println("  logs [service]  Show compose logs. Optional service name.")
	fmt.Println("  run     Build then up. Default when no command is given.")
	fmt.Println("")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {
	repoPath := flag.String("repo", ".", "Path to the repository containing make.yaml")
	tail := flag.String("tail", "100", "Lines of log output for logs command")
	flag.Parse()

	// Command is first non-flag arg. Default is run.
	cmd := "run"
	rest := flag.Args()
	if len(rest) > 0 {
		cmd = rest[0]
	}
	// Optional service name for logs.
	service := ""
	if len(rest) > 1 {
		service = rest[1]
	}

	eng := engine.NewEngine()

	var err error
	switch cmd {
	case "build":
		err = eng.Build(*repoPath)
	case "up":
		err = eng.Up(*repoPath)
	case "down":
		err = eng.Down(*repoPath)
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
