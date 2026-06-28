package cmd

import "github.com/libops/sitectl/pkg/plugin"

var archivesSpaceHealthcheckRunner = plugin.StandardComposeWebHealthcheck(plugin.StandardComposeWebHealthcheckOptions{
	AppService:              "archivesspace",
	HTTPName:                "http:archivesspace",
	DefaultScheme:           "http",
	DefaultDomain:           "localhost",
	DatabaseService:         "mariadb",
	CheckDatabaseDependency: true,
	SolrService:             "solr",
	SolrCore:                "archivesspace",
})
