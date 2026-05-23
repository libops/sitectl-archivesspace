package main

import (
	"fmt"

	"github.com/libops/sitectl-archivesspace/cmd"
	"github.com/libops/sitectl/pkg/plugin"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	sdk := plugin.NewSDK(plugin.Metadata{
		Name:         "archivesspace",
		Version:      fmt.Sprintf("%s (Built on %s from Git SHA %s)", version, date, commit),
		Description:  "ArchivesSpace helpers",
		Author:       "libops",
		TemplateRepo: "https://github.com/libops/archivesspace",
	})

	cmd.RegisterCommands(sdk)
	sdk.Execute()
}
