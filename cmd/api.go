package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	sitectlplugin "github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
)

const archivesSpaceService = "archivesspace"

type archivesSpaceAPIOptions struct {
	baseURL string
	session string
	query   []string
	data    string
	file    string
}

type archivesSpaceLoginOptions struct {
	baseURL  string
	username string
	password string
}

func registerArchivesSpaceCommands(s *sitectlplugin.SDK) {
	s.AddCommand(archivesSpaceAPICommand(s))
	s.AddCommand(archivesSpaceVersionCommand(s))
	s.AddCommand(archivesSpaceRepositoriesCommand(s))
	s.AddCommand(archivesSpaceUsersCommand(s))
	s.AddCommand(archivesSpaceSearchCommand(s))
	s.AddCommand(archivesSpaceJobsCommand(s))
	s.AddCommand(archivesSpaceSchemasCommand(s))
	s.AddCommand(archivesSpaceDiagnosticsCommand(s))
	s.AddCommand(archivesSpaceScriptCommand(s))
	s.AddCommand(archivesSpaceNamedScriptCommand(s, "setup-database", "Run the ArchivesSpace setup-database script"))
	s.AddCommand(archivesSpaceNamedScriptCommand(s, "backup", "Run the ArchivesSpace backup script"))
}

func archivesSpaceAPICommand(s *sitectlplugin.SDK) *cobra.Command {
	root := &cobra.Command{
		Use:   "api",
		Short: "Call the ArchivesSpace REST API",
	}
	root.AddCommand(archivesSpaceLoginCommand(s))
	root.AddCommand(archivesSpaceAPIRequestCommand(s))
	return root
}

func archivesSpaceLoginCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := archivesSpaceLoginOptions{
		baseURL:  "http://localhost/api",
		username: "admin",
	}
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to ArchivesSpace and print the session response",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			password := opts.password
			if password == "" {
				password = os.Getenv("ARCHIVESSPACE_PASSWORD")
			}
			if password == "" {
				return fmt.Errorf("provide --password or set ARCHIVESSPACE_PASSWORD")
			}
			requestURL, err := buildArchivesSpaceAPIURL(opts.baseURL, "users/"+url.PathEscape(opts.username)+"/login", nil)
			if err != nil {
				return err
			}
			curlArgs := []string{"curl", "-fsS", "-X", "POST", "-F", "password=" + password, requestURL}
			return s.RunActiveComposeProjectCommand(cmd, sitectlplugin.ShellJoin(curlArgs))
		},
	}
	cmd.Flags().StringVar(&opts.baseURL, "url", opts.baseURL, "Base ArchivesSpace API URL.")
	cmd.Flags().StringVar(&opts.username, "username", opts.username, "ArchivesSpace username.")
	cmd.Flags().StringVar(&opts.password, "password", "", "ArchivesSpace password. Defaults to ARCHIVESSPACE_PASSWORD.")
	return cmd
}

func archivesSpaceAPIRequestCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	cmd := &cobra.Command{
		Use:   "request METHOD PATH",
		Short: "Call an arbitrary ArchivesSpace API path",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchivesSpaceAPIRequest(s, cmd, args[0], args[1], opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, true)
	return cmd
}

func archivesSpaceVersionCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the ArchivesSpace application version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchivesSpaceAPIRequest(s, cmd, "GET", "version", opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, false)
	return cmd
}

func archivesSpaceRepositoriesCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	cmd := &cobra.Command{
		Use:   "repositories [ID]",
		Short: "List or read ArchivesSpace repositories",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "repositories"
			if len(args) == 1 {
				path += "/" + strings.TrimLeft(args[0], "/")
			}
			return runArchivesSpaceAPIRequest(s, cmd, "GET", path, opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, false)
	return cmd
}

func archivesSpaceUsersCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	cmd := &cobra.Command{
		Use:   "users [ID]",
		Short: "List or read ArchivesSpace users",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "users"
			if len(args) == 1 {
				path += "/" + strings.TrimLeft(args[0], "/")
			}
			return runArchivesSpaceAPIRequest(s, cmd, "GET", path, opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, false)
	return cmd
}

func archivesSpaceSearchCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	var recordType string
	var repository string
	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search ArchivesSpace records",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := append([]string{}, opts.query...)
			query = append(query, "q[]="+args[0])
			if recordType != "" {
				query = append(query, "type[]="+recordType)
			}
			path := "search"
			if repository != "" {
				path = "repositories/" + strings.Trim(repository, "/") + "/search"
			}
			opts.query = query
			return runArchivesSpaceAPIRequest(s, cmd, "GET", path, opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, false)
	cmd.Flags().StringVar(&recordType, "type", "", "Limit search to an ArchivesSpace record type.")
	cmd.Flags().StringVar(&repository, "repo", "", "Repository ID for repository-scoped search.")
	return cmd
}

func archivesSpaceJobsCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	var repository string
	cmd := &cobra.Command{
		Use:   "jobs [ID]",
		Short: "List or read ArchivesSpace background jobs for a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if repository == "" {
				return fmt.Errorf("provide --repo with an ArchivesSpace repository ID")
			}
			path := "repositories/" + strings.Trim(repository, "/") + "/jobs"
			if len(args) == 1 {
				path += "/" + strings.TrimLeft(args[0], "/")
			}
			return runArchivesSpaceAPIRequest(s, cmd, "GET", path, opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, false)
	cmd.Flags().StringVar(&repository, "repo", "", "ArchivesSpace repository ID.")
	return cmd
}

func archivesSpaceSchemasCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	cmd := &cobra.Command{
		Use:   "schemas [NAME]",
		Short: "List or read ArchivesSpace schemas",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "schemas"
			if len(args) == 1 {
				path += "/" + strings.TrimLeft(args[0], "/")
			}
			return runArchivesSpaceAPIRequest(s, cmd, "GET", path, opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, false)
	return cmd
}

func archivesSpaceDiagnosticsCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Show ArchivesSpace diagnostic information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchivesSpaceAPIRequest(s, cmd, "GET", "diagnostics", opts)
		},
	}
	bindArchivesSpaceAPIFlags(cmd, &opts, false)
	return cmd
}

func archivesSpaceScriptCommand(s *sitectlplugin.SDK) *cobra.Command {
	return &cobra.Command{
		Use:                "script SCRIPT [args...]",
		Short:              "Run an ArchivesSpace container script",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := normalizeArchivesSpaceScriptName(args[0])
			if err != nil {
				return err
			}
			return runArchivesSpaceScript(s, cmd, script, args[1:]...)
		},
	}
}

func archivesSpaceNamedScriptCommand(s *sitectlplugin.SDK, name, short string) *cobra.Command {
	return &cobra.Command{
		Use:                name + " [args...]",
		Short:              short,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchivesSpaceScript(s, cmd, name+".sh", args...)
		},
	}
}

func defaultArchivesSpaceAPIOptions() archivesSpaceAPIOptions {
	return archivesSpaceAPIOptions{baseURL: "http://localhost/api"}
}

func bindArchivesSpaceAPIFlags(cmd *cobra.Command, opts *archivesSpaceAPIOptions, includeBody bool) {
	cmd.Flags().StringVar(&opts.baseURL, "url", opts.baseURL, "Base ArchivesSpace API URL.")
	cmd.Flags().StringVar(&opts.session, "session", "", "ArchivesSpace session token for X-ArchivesSpace-Session.")
	cmd.Flags().StringArrayVarP(&opts.query, "query", "q", nil, "Additional query parameter as name=value. May be repeated.")
	if includeBody {
		cmd.Flags().StringVar(&opts.data, "data", "", "JSON request body.")
		cmd.Flags().StringVar(&opts.file, "file", "", "Path to a JSON request body file.")
	}
}

func runArchivesSpaceAPIRequest(s *sitectlplugin.SDK, cmd *cobra.Command, method, path string, opts archivesSpaceAPIOptions) error {
	requestURL, err := buildArchivesSpaceAPIURL(opts.baseURL, path, opts.query)
	if err != nil {
		return err
	}
	args := []string{"curl", "-fsS", "-X", strings.ToUpper(method), "-H", "Accept: application/json"}
	if opts.session != "" {
		args = append(args, "-H", "X-ArchivesSpace-Session: "+opts.session)
	}
	if opts.data != "" || opts.file != "" {
		args = append(args, "-H", "Content-Type: application/json")
	}
	if opts.data != "" {
		args = append(args, "--data", opts.data)
	}
	if opts.file != "" {
		args = append(args, "--data-binary", "@"+opts.file)
	}
	args = append(args, requestURL)
	return s.RunActiveComposeProjectCommand(cmd, sitectlplugin.ShellJoin(args))
}

func runArchivesSpaceScript(s *sitectlplugin.SDK, cmd *cobra.Command, script string, args ...string) error {
	commandArgs := []string{"/archivesspace/scripts/" + script}
	commandArgs = append(commandArgs, args...)
	return s.RunActiveComposeProjectCommand(cmd, sitectlplugin.DockerComposeExecCommand(archivesSpaceService, commandArgs...))
}

func buildArchivesSpaceAPIURL(baseURL, path string, queryPairs []string) (string, error) {
	raw := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse API URL: %w", err)
	}
	values := parsed.Query()
	for _, pair := range queryPairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return "", fmt.Errorf("query parameter must be name=value: %q", pair)
		}
		values.Add(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func normalizeArchivesSpaceScriptName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, `\`) || strings.Contains(name, "..") {
		return "", fmt.Errorf("script name must be a file name, not a path")
	}
	if !strings.HasSuffix(name, ".sh") {
		name += ".sh"
	}
	return name, nil
}
