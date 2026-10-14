package main

import (
	"flag"
	"fmt"
	"log"
	"testbox/internal/engine"
)

func usage() {
	fmt.Println("Usage: testbox-tool [-repo <path>] <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  build   Parse make.yaml and build the Docker image.")
	fmt.Println("  up      Prepare compose file and start services.")
	fmt.Println("  down    Stop services.")
	fmt.Println("  run     Build then up. Default when no command is given.")
	fmt.Println("")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {
	repoPath := flag.String("repo", ".", "Path to the repository containing make.yaml")
	flag.Parse()

	// Command is first non-flag arg. Default is run.
	cmd := "run"
	if args := flag.Args(); len(args) > 0 {
		cmd = args[0]
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
