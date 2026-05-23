package cmd

import "github.com/libops/sitectl/pkg/plugin"

const (
	createRepo   = "https://github.com/libops/archivesspace"
	createBranch = "main"
	pluginName   = "archivesspace"
	defaultPath  = "./archivesspace"
	displayName  = "ArchivesSpace"
)

func createDefinition() plugin.CreateSpec {
	return plugin.CreateSpec{
		Name:                 "default",
		Description:          "Create an ArchivesSpace stack",
		Default:              true,
		MinCPUCores:          4,
		MinMemory:            "8 GiB",
		MinDiskSpace:         "50 GiB",
		DockerComposeRepo:    createRepo,
		DockerComposeBranch:  createBranch,
		DockerComposeBuild:   []string{"make build"},
		DockerComposeInit:    []string{"make init"},
		DockerComposeUp:      []string{"make up"},
		DockerComposeDown:    []string{"make down"},
		DockerComposeRollout: []string{"make rollout"},
	}
}

// RegisterCommands registers ArchivesSpace commands with the plugin SDK.
func RegisterCommands(s *plugin.SDK) {
	s.AddCommand(s.GetDiscoveryMetadataCommand())
	plugin.RegisterStandardComposeTemplate(s, createDefinition(), plugin.StandardComposeTemplateOptions{
		DefaultPath:   defaultPath,
		DefaultPlugin: pluginName,
		ReadyMessage:  "ArchivesSpace is ready for use through sitectl.",
		DisplayName:   displayName,
	})
	registerArchivesSpaceCommands(s)
}
