package cmd

import "github.com/libops/sitectl/pkg/plugin"

const (
	createRepo                    = "https://github.com/libops/archivesspace"
	createBranch                  = "main"
	pluginName                    = "archivesspace"
	defaultPath                   = "./archivesspace"
	defaultDatabaseService        = "mysql"
	defaultDatabaseUser           = "as"
	defaultDatabasePasswordSecret = "ARCHIVESSPACE_DB_PASSWORD"
	defaultDatabaseName           = "archivesspace"
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
	s.SetComposeProjectDiscovery(plugin.ComposeProjectDiscovery{
		RequiredServices: []string{"archivesspace"},
		Reason:           "archivesspace service",
	})
	s.RegisterComposeTemplateCreateRunner(createDefinition(), plugin.ComposeTemplateCreateOptions{
		DefaultPath:                   defaultPath,
		DefaultPlugin:                 pluginName,
		DefaultDatabaseService:        defaultDatabaseService,
		DefaultDatabaseUser:           defaultDatabaseUser,
		DefaultDatabasePasswordSecret: defaultDatabasePasswordSecret,
		DefaultDatabaseName:           defaultDatabaseName,
		ReadyMessage:                  "ArchivesSpace is ready for use through sitectl.",
	})
	s.RegisterHealthcheckRunner(archivesSpaceHealthcheckRunner{})
	registerArchivesSpaceCommands(s)
}
