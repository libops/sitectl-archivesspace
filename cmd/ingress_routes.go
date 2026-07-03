package cmd

import (
	"net/url"
	"strings"

	"github.com/libops/sitectl/pkg/config"
	"github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type archivesSpaceIngressRouteProvider struct{}

func (archivesSpaceIngressRouteProvider) BindFlags(cmd *cobra.Command) {}

func (archivesSpaceIngressRouteProvider) Routes(cmd *cobra.Command, ctx *config.Context) (plugin.IngressRoutes, error) {
	scheme := "http"
	domain := "localhost"
	env, err := plugin.ContextServiceEnvironment(ctx, "archivesspace")
	if err != nil {
		return plugin.IngressRoutes{}, err
	}
	if ingressDomain := firstArchivesSpaceHostname(env["INGRESS_HOSTNAMES"]); ingressDomain != "" || strings.TrimSpace(env["INGRESS_SCHEME"]) != "" {
		scheme = firstArchivesSpaceIngressValue(env["INGRESS_SCHEME"], scheme)
		domain = firstArchivesSpaceIngressValue(ingressDomain, domain)
	} else if parsedScheme, parsedDomain := archivesSpaceSchemeDomain(env["PUBLIC_URL"]); parsedDomain != "" {
		scheme = firstArchivesSpaceIngressValue(parsedScheme, scheme)
		domain = parsedDomain
	}
	return plugin.IngressRoutes{
		Domain: domain,
		Scheme: scheme,
		Routes: []plugin.IngressRoute{
			{
				Name:          "app",
				Service:       "archivesspace",
				Router:        "archivesspace-public-web",
				DefaultScheme: scheme,
				DefaultDomain: domain,
				Primary:       true,
			},
			{
				Name:          "staff",
				Service:       "archivesspace",
				Router:        "archivesspace-staff-web",
				DefaultScheme: scheme,
				DefaultDomain: domain,
				Path:          "/staff",
			},
			{
				Name:          "api",
				Service:       "archivesspace",
				Router:        "archivesspace-api-web",
				DefaultScheme: scheme,
				DefaultDomain: domain,
				Path:          "/api",
			},
			{
				Name:          "oai",
				Service:       "archivesspace",
				Router:        "archivesspace-public-web",
				DefaultScheme: scheme,
				DefaultDomain: domain,
				Path:          "/oai",
			},
		},
	}, nil
}

func archivesSpaceSchemeDomain(value string) (string, string) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || strings.TrimSpace(parsed.Host) == "" {
		return "", ""
	}
	return strings.TrimSpace(parsed.Scheme), strings.TrimSpace(parsed.Host)
}

func firstArchivesSpaceHostname(value string) string {
	for _, hostname := range strings.Split(value, ",") {
		hostname = strings.TrimSpace(hostname)
		if hostname != "" {
			return hostname
		}
	}
	return ""
}

func firstArchivesSpaceIngressValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
