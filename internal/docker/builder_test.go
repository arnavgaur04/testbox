package docker

import (
	"reflect"
	"testbox/pkg/models"
	"testing"
)

func TestBuildArgsBasic(t *testing.T) {
	cfg := models.DockerConfig{
		Context:    ".",
		Dockerfile: "Dockerfile",
		ImageName:  "my-app:latest",
	}
	got := BuildArgsForTest(cfg)
	want := []string{"build", "-t", "my-app:latest", "-f", "Dockerfile", "."}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildArgsWithBuildArgsAndTarget(t *testing.T) {
	cfg := models.DockerConfig{
		Context:    "/tmp/app",
		Dockerfile: "Dockerfile.dev",
		ImageName:  "a:1",
		BuildArgs:  []string{"KEY1=VAL1", "KEY2=VAL2"},
		Target:     "runner",
	}
	got := BuildArgsForTest(cfg)
	want := []string{"build", "-t", "a:1", "-f", "Dockerfile.dev", "--build-arg", "KEY1=VAL1", "--build-arg", "KEY2=VAL2", "--target", "runner", "/tmp/app"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildEnvDefaultsBuildKit(t *testing.T) {
	got := buildEnv([]string{"A=1"})
	found := false
	for _, e := range got {
		if e == "DOCKER_BUILDKIT=1" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing default: %v", got)
	}
}

func TestBuildEnvRespectsParent(t *testing.T) {
	in := []string{"DOCKER_BUILDKIT=0"}
	got := buildEnv(in)
	if len(got) != 1 || got[0] != "DOCKER_BUILDKIT=0" {
		t.Errorf("must keep parent: %v", got)
	}
}
