package entity

import (
	"encoding/json"
	"testing"
)

func TestSandboxConfigJSONUsesWebFieldNames(t *testing.T) {
	cfg := SandboxConfig{
		Provider: SandboxDocker,
		Docker: &DockerSandboxConfig{
			Image:       "ghcr.io/agencycli/sandbox-cursor:latest",
			NetworkMode: "none",
			MemoryMB:    4096,
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal sandbox config: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal sandbox config: %v", err)
	}

	if _, ok := got["Provider"]; ok {
		t.Fatalf("JSON should not expose Go field name Provider: %s", data)
	}
	if got["provider"] != "docker" {
		t.Fatalf("provider = %v, want docker", got["provider"])
	}

	docker, ok := got["docker"].(map[string]any)
	if !ok {
		t.Fatalf("docker config missing or invalid: %s", data)
	}
	if _, ok := docker["NetworkMode"]; ok {
		t.Fatalf("JSON should not expose Go field name NetworkMode: %s", data)
	}
	if docker["network_mode"] != "none" {
		t.Fatalf("network_mode = %v, want none", docker["network_mode"])
	}
	if docker["memory_mb"] != float64(4096) {
		t.Fatalf("memory_mb = %v, want 4096", docker["memory_mb"])
	}
}
