package cmd

import (
	"strings"
	"testing"

	"github.com/libops/sitectl/pkg/plugin"
)

func TestRegisterCommandsRegistersApplicationComponents(t *testing.T) {
	t.Parallel()

	sdk := plugin.NewSDK(plugin.Metadata{Name: "archivesspace"})
	RegisterCommands(sdk)

	definitions := sdk.LocalComponentDefinitions()
	got := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		got[definition.Name] = true
	}
	for _, name := range []string{"ingress"} {
		if !got[name] {
			t.Fatalf("expected component %q to be registered, got %+v", name, definitions)
		}
	}
	if got["dev-mode"] {
		t.Fatalf("dev-mode must not mask bundled ArchivesSpace directories: %+v", definitions)
	}
}

func TestCreateDefinitionLifecycleContract(t *testing.T) {
	t.Parallel()
	spec := createDefinition()
	if len(spec.Images) != 2 || spec.Images[0].Image != "libops/archivesspace:4.2.0" || spec.Images[0].BuildPolicy != plugin.BuildPolicyAlways {
		t.Fatalf("unexpected ArchivesSpace image contract: %+v", spec.Images)
	}
	if spec.Images[1].Image != "libops/archivesspace-solr:4.2.0" {
		t.Fatalf("Solr must follow the ArchivesSpace release tag: %+v", spec.Images)
	}
	if len(spec.DockerComposeUp) != 1 || !strings.Contains(spec.DockerComposeUp[0], "--wait --wait-timeout 600") {
		t.Fatalf("create must wait for service health before reporting ready: %+v", spec.DockerComposeUp)
	}
	rollout := strings.Join(spec.DockerComposeRollout, "\n")
	if !strings.Contains(rollout, "docker compose build --pull") || strings.Contains(rollout, "|| true") {
		t.Fatalf("rollout must rebuild and propagate failures:\n%s", rollout)
	}
	if !strings.Contains(rollout, "--wait --wait-timeout 600") {
		t.Fatalf("rollout readiness wait must be bounded:\n%s", rollout)
	}
}
