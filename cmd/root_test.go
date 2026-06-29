package cmd

import (
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
	for _, name := range []string{"reverse-proxy", "dev-mode"} {
		if !got[name] {
			t.Fatalf("expected component %q to be registered, got %+v", name, definitions)
		}
	}
}
