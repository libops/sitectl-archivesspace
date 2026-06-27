package cmd

import (
	corecomponent "github.com/libops/sitectl/pkg/component"
	"github.com/libops/sitectl/pkg/plugin"
	coredevmode "github.com/libops/sitectl/pkg/services/devmode"
	coretraefik "github.com/libops/sitectl/pkg/services/traefik"
)

const (
	createRepo                    = "https://github.com/libops/archivesspace"
	createBranch                  = "main"
	pluginName                    = "archivesspace"
	defaultPath                   = "./archivesspace"
	defaultDatabaseService        = "mariadb"
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
		DockerComposeBuild:   []string{"docker compose pull --ignore-buildable"},
		DockerComposeInit:    []string{"./scripts/init.sh"},
		DockerComposeUp:      []string{"./scripts/init.sh", "docker compose up --remove-orphans -d"},
		DockerComposeDown:    []string{"docker compose down"},
		DockerComposeRollout: []string{"./scripts/rollout.sh"},
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
	registerApplicationComponents(s)
	s.RegisterHealthcheckRunner(archivesSpaceHealthcheckRunner{})
	registerArchivesSpaceCommands(s)
}

func registerApplicationComponents(s *plugin.SDK) {
	reverseProxy, err := coretraefik.ReverseProxy(coretraefik.ReverseProxyOptions{NoAppService: true})
	if err != nil {
		panic(err)
	}
	devMode, err := coredevmode.Component(coredevmode.Options{
		AppService: "archivesspace",
		Volumes: []string{
			"./plugins:/archivesspace/plugins:z,rw",
			"./locales:/archivesspace/locales:z,rw",
			"./stylesheets:/archivesspace/stylesheets:z,rw",
		},
	})
	if err != nil {
		panic(err)
	}
	s.RegisterServiceComponents(plugin.ServiceComponentRegistryOptions{
		DisplayName: "ArchivesSpace",
		Components:  []corecomponent.ComposeServiceComponent{reverseProxy, devMode},
	})
}
