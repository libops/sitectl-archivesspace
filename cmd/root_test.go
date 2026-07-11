package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corecomponent "github.com/libops/sitectl/pkg/component"
	"github.com/libops/sitectl/pkg/config"
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

func TestArchivesSpaceIngressOmitsDefaultPHPAndNginxEnvironment(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	compose, err := os.ReadFile(filepath.Join("testdata", "ingress-non-php-runtime", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("ReadFile(fixture) error = %v", err)
	}
	if err := os.WriteFile(composePath, compose, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	components := applicationComponents()
	if len(components) != 1 || components[0].Name() != "ingress" {
		t.Fatalf("unexpected ArchivesSpace application components: %+v", components)
	}
	spec := components[0].SpecForWithOptions(corecomponent.StateOn, map[string]string{
		"mode":            "http",
		"domain":          "archives.example.org",
		"trusted-ip":      "203.0.113.0/24",
		"max-upload-size": "2G",
		"upload-timeout":  "10m",
	})
	ctx := &config.Context{
		DockerHostType: config.ContextLocal,
		ProjectDir:     projectDir,
	}
	manager := corecomponent.NewManager(ctx)
	if err := manager.EnableComponentWithOptions(context.Background(), spec, corecomponent.ApplyOptions{Yolo: true}); err != nil {
		t.Fatalf("EnableComponentWithOptions() error = %v", err)
	}

	data, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`INGRESS_HOSTNAMES: "archives.example.org,localhost,127.0.0.1,::1"`,
		`INGRESS_SCHEME: "http"`,
		`KEEP_ME: "true"`,
		`--entryPoints.web.transport.respondingTimeouts.readTimeout=10m`,
		`--entryPoints.web.forwardedHeaders.trustedIPs=203.0.113.0/24`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected compose to contain %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{"PHP_", "NGINX_", "PUBLIC_URL"} {
		if strings.Contains(got, notWant) {
			t.Fatalf("expected ArchivesSpace ingress to remove %q settings:\n%s", notWant, got)
		}
	}
}
