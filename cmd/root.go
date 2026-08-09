package cmd

import (
	corecomponent "github.com/libops/sitectl/pkg/component"
	"github.com/libops/sitectl/pkg/plugin"
	coretraefik "github.com/libops/sitectl/pkg/services/traefik"
)

const (
	createRepo                    = "https://github.com/libops/archivesspace"
	createBranch                  = "v1.0.1"
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
			{Service: "archivesspace", Image: "libops/archivesspace:4.2.0", BuildPolicy: plugin.BuildPolicyAlways},
			{Service: "solr", Image: "libops/archivesspace-solr:4.2.0", BuildPolicy: plugin.BuildPolicyIfNotPresent},
		},
		DockerComposeInit: []string{
			"mkdir -p ./secrets",
			"docker compose run --rm init",
		},
		DockerComposeUp: []string{
			"docker compose up --remove-orphans --wait --wait-timeout 600 -d",
		},
		DockerComposeDown: []string{"docker compose down"},
		DockerComposeRollout: []string{
			"docker compose pull --ignore-buildable --quiet || docker compose pull --ignore-buildable",
			"docker compose build --pull",
			"mkdir -p ./secrets",
			"docker compose run --rm init",
			"docker compose up --remove-orphans --wait --wait-timeout 600 --pull missing --quiet-pull -d",
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
	registerApplicationComponents(s)
	s.RegisterHealthcheckRunner(archivesSpaceHealthcheckRunner)
	s.RegisterVerifyRunner(&archivesSpaceVerifyRunner{sdk: s})
	s.RegisterIngressRouteProvider(archivesSpaceIngressRouteProvider{})
	registerArchivesSpaceCommands(s)
}

func registerApplicationComponents(s *plugin.SDK) {
	s.RegisterServiceComponents(plugin.ServiceComponentRegistryOptions{
		DisplayName: "ArchivesSpace",
		Components:  applicationComponents(),
	})
}

func applicationComponents() []corecomponent.ComposeServiceComponent {
	ingress, err := coretraefik.Ingress(coretraefik.IngressOptions{
		AppService:                     "archivesspace",
		NoDefaultAppRuntimeEnvironment: true,
		HTTPEntrypoint:                 "web",
		HTTPSEntrypoint:                "websecure",
		AppEnvDeletes:                  []string{"PUBLIC_URL"},
	})
	if err != nil {
		panic(err)
	}
	return []corecomponent.ComposeServiceComponent{ingress}
}
