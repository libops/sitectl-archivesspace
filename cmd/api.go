package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/libops/sitectl/pkg/docker"
	sitectlplugin "github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
)

const (
	archivesSpaceService         = "archivesspace"
	archivesSpaceInternalAPIURL  = "http://127.0.0.1:8089"
	archivesSpaceSessionEnv      = "ARCHIVESSPACE_SESSION"
	archivesSpaceSessionFileEnv  = "ARCHIVESSPACE_SESSION_FILE"
	archivesSpacePasswordEnv     = "ARCHIVESSPACE_PASSWORD"
	archivesSpacePasswordFileEnv = "ARCHIVESSPACE_PASSWORD_FILE"
)

var archivesSpaceRunCurl = runArchivesSpaceContainerCurl

type archivesSpaceAPIOptions struct {
	baseURL      string
	sessionFile  string
	sessionValue string
	query        []string
	data         string
	file         string
}

type archivesSpaceLoginOptions struct {
	baseURL      string
	username     string
	passwordFile string
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
		baseURL:  archivesSpaceInternalAPIURL,
		username: "admin",
	}
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to ArchivesSpace and print the session response",
		Long: `Authenticate to the ArchivesSpace backend API for the active site.

The password is read from a local file or environment reference and sent to
curl over stdin, so it is not placed in a shell command or process argument.
The JSON response contains a session credential; redirect it to a protected
file instead of allowing automation to capture it in logs.`,
		Example: `  umask 077
  ARCHIVESSPACE_PASSWORD_FILE=/secure/path/archivesspace-password \
    sitectl archivesspace api login > .archivesspace-login.json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := readArchivesSpaceSecret(opts.passwordFile, archivesSpacePasswordFileEnv, archivesSpacePasswordEnv)
			if err != nil {
				return fmt.Errorf("load ArchivesSpace password: %w", err)
			}
			output, err := executeArchivesSpaceLogin(s, cmd, opts.baseURL, opts.username, password)
			if err != nil {
				return err
			}
			return writeArchivesSpaceOutput(cmd, output)
		},
	}
	cmd.Flags().StringVar(&opts.baseURL, "url", opts.baseURL, "Base ArchivesSpace API URL reachable from the ArchivesSpace container.")
	cmd.Flags().StringVar(&opts.username, "username", opts.username, "ArchivesSpace username.")
	cmd.Flags().StringVar(&opts.passwordFile, "password-file", "", "Read the ArchivesSpace password from this local file instead of process arguments. Defaults to ARCHIVESSPACE_PASSWORD_FILE or ARCHIVESSPACE_PASSWORD.")
	return cmd
}

func archivesSpaceAPIRequestCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultArchivesSpaceAPIOptions()
	cmd := &cobra.Command{
		Use:   "request METHOD PATH",
		Short: "Call an arbitrary ArchivesSpace API path",
		Long: `Call an arbitrary path on the ArchivesSpace backend API for the active site.

The plugin executes curl inside the running ArchivesSpace container. Session
credentials and request bodies are carried over stdin instead of a shell
command. A --file path identifies a local JSON file read by sitectl. A literal
--data value remains visible in the local sitectl process arguments, so use
--file whenever a request body is confidential.`,
		Args: cobra.ExactArgs(2),
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
			query = append(query, "q="+args[0])
			pageConfigured := false
			for _, pair := range query {
				key, _, _ := strings.Cut(pair, "=")
				if key == "page" {
					pageConfigured = true
					break
				}
			}
			if !pageConfigured {
				query = append(query, "page=1")
			}
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
	return archivesSpaceAPIOptions{baseURL: archivesSpaceInternalAPIURL}
}

func bindArchivesSpaceAPIFlags(cmd *cobra.Command, opts *archivesSpaceAPIOptions, includeBody bool) {
	cmd.Flags().StringVar(&opts.baseURL, "url", opts.baseURL, "Base ArchivesSpace API URL reachable from the ArchivesSpace container.")
	cmd.Flags().StringVar(&opts.sessionFile, "session-file", "", "Read the ArchivesSpace session from this local file instead of process arguments. Defaults to ARCHIVESSPACE_SESSION_FILE or ARCHIVESSPACE_SESSION.")
	cmd.Flags().StringArrayVarP(&opts.query, "query", "q", nil, "Additional query parameter as name=value. May be repeated.")
	if includeBody {
		cmd.Flags().StringVar(&opts.data, "data", "", "JSON request body.")
		cmd.Flags().StringVar(&opts.file, "file", "", "Path to a JSON request body file.")
	}
}

func runArchivesSpaceAPIRequest(s *sitectlplugin.SDK, cmd *cobra.Command, method, path string, opts archivesSpaceAPIOptions) error {
	output, err := executeArchivesSpaceAPIRequest(s, cmd, method, path, opts)
	if err != nil {
		return err
	}
	return writeArchivesSpaceOutput(cmd, output)
}

func executeArchivesSpaceAPIRequest(s *sitectlplugin.SDK, cmd *cobra.Command, method, path string, opts archivesSpaceAPIOptions) (string, error) {
	requestURL, err := buildArchivesSpaceAPIURL(opts.baseURL, path, opts.query)
	if err != nil {
		return "", err
	}
	args := []string{"curl", "-fsS", "-X", strings.ToUpper(method), "-H", "Accept: application/json"}
	var curlConfig strings.Builder
	session := opts.sessionValue
	if session == "" {
		session, err = readArchivesSpaceSecretIfConfigured(opts.sessionFile, archivesSpaceSessionFileEnv, archivesSpaceSessionEnv)
		if err != nil {
			return "", fmt.Errorf("load ArchivesSpace session: %w", err)
		}
	}
	if session != "" {
		headerValue, err := curlConfigValue("X-ArchivesSpace-Session: " + session)
		if err != nil {
			return "", fmt.Errorf("encode ArchivesSpace session: %w", err)
		}
		fmt.Fprintf(&curlConfig, "header = %s\n", headerValue)
	}
	if opts.data != "" && opts.file != "" {
		return "", fmt.Errorf("provide only one of --data or --file")
	}
	body := opts.data
	if opts.file != "" {
		data, err := os.ReadFile(opts.file) // #nosec G304 -- the operator explicitly selects the request body file.
		if err != nil {
			return "", fmt.Errorf("read ArchivesSpace request body %q: %w", opts.file, err)
		}
		body = string(data)
	}
	if body != "" {
		args = append(args, "-H", "Content-Type: application/json")
		bodyValue, err := curlConfigDataValue(body)
		if err != nil {
			return "", fmt.Errorf("encode ArchivesSpace request body: %w", err)
		}
		fmt.Fprintf(&curlConfig, "data-binary = %s\n", bodyValue)
	}
	if curlConfig.Len() > 0 {
		args = append(args, "--config", "-")
	}
	// End option parsing before the operator-selected URL. This keeps a
	// malformed value from being interpreted as another curl option.
	args = append(args, "--", requestURL)
	return archivesSpaceRunCurl(cmd.Context(), s, cmd, args, curlConfig.String())
}

func executeArchivesSpaceLogin(s *sitectlplugin.SDK, cmd *cobra.Command, baseURL, username, password string) (string, error) {
	requestURL, err := buildArchivesSpaceAPIURL(baseURL, "users/"+url.PathEscape(username)+"/login", nil)
	if err != nil {
		return "", err
	}
	passwordValue, err := curlConfigValue("password=" + password)
	if err != nil {
		return "", fmt.Errorf("encode ArchivesSpace password: %w", err)
	}
	curlArgs := []string{"curl", "-fsS", "-X", "POST", "--config", "-", "--", requestURL}
	return archivesSpaceRunCurl(cmd.Context(), s, cmd, curlArgs, "form-string = "+passwordValue+"\n")
}

func runArchivesSpaceContainerCurl(runCtx context.Context, s *sitectlplugin.SDK, cmd *cobra.Command, args []string, input string) (string, error) {
	ctx, err := s.ContextFromCommand(cmd)
	if err != nil {
		return "", err
	}
	client, err := s.GetDockerClient()
	if err != nil {
		return "", fmt.Errorf("connect to Docker for ArchivesSpace API request: %w", err)
	}
	defer func() { _ = client.Close() }()

	containerName, err := client.GetContainerNameContext(runCtx, ctx, archivesSpaceService)
	if err != nil {
		return "", fmt.Errorf("find running ArchivesSpace container: %w", err)
	}
	if strings.TrimSpace(containerName) == "" {
		return "", fmt.Errorf("ArchivesSpace service is not running")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode, err := client.Exec(runCtx, docker.ExecOptions{
		Container:    containerName,
		Cmd:          append([]string(nil), args...),
		AttachStdin:  input != "",
		AttachStdout: true,
		AttachStderr: true,
		Stdin:        strings.NewReader(input),
		Stdout:       &stdout,
		Stderr:       &stderr,
	})
	if err != nil {
		return "", fmt.Errorf("run ArchivesSpace API request: %w", err)
	}
	if exitCode != 0 {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = "curl exited without diagnostic output"
		}
		return "", fmt.Errorf("ArchivesSpace API request failed with exit code %d: %s", exitCode, detail)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func writeArchivesSpaceOutput(cmd *cobra.Command, output string) error {
	if output == "" {
		return nil
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), output)
	return err
}

func readArchivesSpaceSecret(explicitFile, fileEnv, valueEnv string) (string, error) {
	value, err := readArchivesSpaceSecretIfConfigured(explicitFile, fileEnv, valueEnv)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("provide --%s-file, %s, or %s", secretFlagName(valueEnv), fileEnv, valueEnv)
	}
	return value, nil
}

func readArchivesSpaceSecretIfConfigured(explicitFile, fileEnv, valueEnv string) (string, error) {
	filename := strings.TrimSpace(explicitFile)
	if filename == "" {
		filename = strings.TrimSpace(os.Getenv(fileEnv))
	}
	value := ""
	if filename != "" {
		data, err := os.ReadFile(filename) // #nosec G304 -- the operator explicitly selects the local credential file.
		if err != nil {
			return "", fmt.Errorf("read credential file %q: %w", filename, err)
		}
		value = strings.TrimSuffix(string(data), "\n")
		value = strings.TrimSuffix(value, "\r")
	} else {
		value = os.Getenv(valueEnv)
	}
	if value == "" {
		return "", nil
	}
	if _, err := curlConfigValue(value); err != nil {
		return "", err
	}
	return value, nil
}

func secretFlagName(valueEnv string) string {
	if valueEnv == archivesSpacePasswordEnv {
		return "password"
	}
	return "session"
}

func curlConfigValue(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("credential contains a forbidden line or NUL character")
	}
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`, nil
}

func curlConfigDataValue(value string) (string, error) {
	if strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("request body contains a forbidden NUL character")
	}
	var encoded strings.Builder
	encoded.Grow(len(value) + 2)
	encoded.WriteByte('"')
	for _, char := range value {
		switch char {
		case '\\':
			encoded.WriteString(`\\`)
		case '"':
			encoded.WriteString(`\"`)
		case '\n':
			encoded.WriteString(`\n`)
		case '\r':
			encoded.WriteString(`\r`)
		case '\t':
			encoded.WriteString(`\t`)
		default:
			if char < 0x20 {
				return "", fmt.Errorf("request body contains unsupported control character U+%04X", char)
			}
			encoded.WriteRune(char)
		}
	}
	encoded.WriteByte('"')
	return encoded.String(), nil
}

func runArchivesSpaceScript(s *sitectlplugin.SDK, cmd *cobra.Command, script string, args ...string) error {
	commandArgs := []string{"/archivesspace/scripts/" + script}
	commandArgs = append(commandArgs, args...)
	return s.RunActiveComposeProjectArgv(cmd, sitectlplugin.DockerComposeExecArgv(archivesSpaceService, commandArgs...))
}

func buildArchivesSpaceAPIURL(baseURL, path string, queryPairs []string) (string, error) {
	raw := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse API URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("API URL must use http or https with a host and no embedded credentials")
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
