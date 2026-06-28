package cmd

import "github.com/libops/sitectl/pkg/plugin"

const (
	createRepo                    = "https://github.com/libops/archivesspace"
	createBranch                  = "main"
	pluginName                    = "archivesspace"
	defaultPath                   = "./archivesspace"
	defaultDatabaseService        = "mariadb"
	defaultDatabaseUser           = "archivesspace"
	defaultDatabasePasswordSecret = "ARCHIVESSPACE_DB_PASSWORD"
	defaultDatabaseName           = "archivesspace"
)

func createDefinition() plugin.CreateSpec {
	return plugin.CreateSpec{
		Name:                "default",
		Description:         "Create an ArchivesSpace stack",
		Default:             true,
		MinCPUCores:         4,
		MinMemory:           "8 GiB",
		MinDiskSpace:        "50 GiB",
		DockerComposeRepo:   createRepo,
		DockerComposeBranch: createBranch,
		DockerComposeBuild: []string{
			"docker compose pull --ignore-buildable",
			"docker compose build",
		},
		Images: []plugin.ComposeImageSpec{
			{Service: "archivesspace", Image: "libops/archivesspace:4.2.0", BuildPolicy: plugin.BuildPolicyIfNotPresent},
			{Service: "solr", Image: "libops/archivesspace-solr:4.2.0", BuildPolicy: plugin.BuildPolicyIfNotPresent},
		},
		DockerComposeInit: []string{
			"./scripts/init.sh",
		},
		InitArtifacts: []plugin.InitArtifact{
			{Path: ".env"},
			{Path: "secrets/DB_ROOT_PASSWORD"},
			{Path: "secrets/ARCHIVESSPACE_DB_PASSWORD"},
		},
		DockerComposeUp: []string{
			"docker compose up --remove-orphans -d",
		},
		DockerComposeDown: []string{"docker compose down"},
		DockerComposeRollout: []string{
			"docker compose pull --ignore-buildable --quiet || docker compose pull --ignore-buildable || true",
			"./scripts/init.sh",
			"docker compose up --remove-orphans --wait --pull missing --quiet-pull -d",
		},
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
	s.RegisterHealthcheckRunner(archivesSpaceHealthcheckRunner)
	registerArchivesSpaceCommands(s)
}
