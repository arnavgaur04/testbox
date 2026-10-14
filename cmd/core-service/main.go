package main

import (
	"flag"
	"fmt"
	"log"
	"testbox/internal/engine"
)

func main() {
	repoPath := flag.String("repo", ".", "Path to the repository containing make.yaml")
	flag.Parse()

	eng := engine.NewEngine()
	if err := eng.Run(*repoPath); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Println("Process completed successfully!")
}
