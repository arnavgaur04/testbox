package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testbox/internal/api"
)

// Repeatable -allow-repo value.
type stringList []string

func (l *stringList) String() string {
	return strings.Join(*l, ", ")
}

func (l *stringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

// True for local-only listen addrs.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	switch host {
	case "127.0.0.1", "::1", "localhost":
		return true
	}
	return false
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8081", "Listen address for API server")
	token := flag.String("token", os.Getenv("TESTBOX_API_TOKEN"), "Bearer token. Required for non-local listen addrs.")
	var repos stringList
	flag.Var(&repos, "allow-repo", "Repo root the API may touch. Repeatable.")
	flag.Parse()

	if len(repos) == 0 {
		env := os.Getenv("TESTBOX_ALLOWED_REPOS")
		for _, p := range strings.Split(env, ",") {
			if p = strings.TrimSpace(p); p != "" {
				repos = append(repos, p)
			}
		}
	}
	if len(repos) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("resolve cwd: %v", err)
		}
		repos = append(repos, cwd)
		fmt.Printf("No -allow-repo given. Allowing %s only.\n", cwd)
	}
	if *token == "" && !isLoopback(*addr) {
		log.Fatal("refusing non-local listen without -token or TESTBOX_API_TOKEN")
	}

	guard, err := api.NewGuard(*token, repos)
	if err != nil {
		log.Fatalf("bad guard config: %v", err)
	}
	fmt.Printf("Allowed repos: %s\n", strings.Join(guard.AllowedRepos, ", "))
	if *token == "" {
		fmt.Println("Warning: no token set. Local-only access assumed.")
	}

	srv := api.NewServerWithGuard(guard)
	fmt.Printf("testbox API listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, srv.Handler()))
}
