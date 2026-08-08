package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/libops/sitectl/pkg/config"
	"github.com/libops/sitectl/pkg/plugin"
	sitevalidate "github.com/libops/sitectl/pkg/validate"
	"github.com/spf13/cobra"
)

const archivesSpaceExpectedVersion = "4.2.0"

type archivesSpaceVerifyRunner struct {
	sdk          *plugin.SDK
	baseURL      string
	sessionFile  string
	username     string
	passwordFile string
	disposable   bool
}

func (r *archivesSpaceVerifyRunner) BindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&r.baseURL, "url", archivesSpaceInternalAPIURL, "Base ArchivesSpace API URL reachable from the ArchivesSpace container.")
	cmd.Flags().StringVar(&r.sessionFile, "session-file", "", "Read the ArchivesSpace session from this local file. Defaults to ARCHIVESSPACE_SESSION_FILE or ARCHIVESSPACE_SESSION.")
	cmd.Flags().StringVar(&r.username, "username", "admin", "ArchivesSpace username used when verification must create a session.")
	cmd.Flags().StringVar(&r.passwordFile, "password-file", "", "Read the ArchivesSpace password used to create a verification session from this local file. Defaults to ARCHIVESSPACE_PASSWORD_FILE or ARCHIVESSPACE_PASSWORD.")
	cmd.Flags().BoolVar(&r.disposable, "disposable", false, "Use the fresh-install admin credential for disposable CI only. Never use this option for a retained site.")
}

func (r *archivesSpaceVerifyRunner) Run(cmd *cobra.Command, _ *config.Context) ([]sitevalidate.Result, error) {
	results := make([]sitevalidate.Result, 0, 3)

	versionOutput, versionErr := executeArchivesSpaceAPIRequest(r.sdk, cmd, "GET", "version", archivesSpaceAPIOptions{baseURL: r.baseURL})
	results = append(results, archivesSpaceVersionResult(versionOutput, versionErr))

	session, sessionErr := r.resolveSession(cmd)
	if sessionErr != nil || session == "" {
		detail := "no session credential reference is configured"
		if sessionErr != nil {
			detail = sessionErr.Error()
		}
		fix := "provide a session with --session-file/ARCHIVESSPACE_SESSION_FILE, or a login password with --password-file/ARCHIVESSPACE_PASSWORD_FILE"
		results = append(results,
			failedArchivesSpaceVerifyResult("verify:archivesspace:authentication", detail, fix),
			failedArchivesSpaceVerifyResult("verify:archivesspace:search", "authenticated search was not attempted", fix),
		)
		return results, nil
	}

	requestOpts := archivesSpaceAPIOptions{baseURL: r.baseURL, sessionValue: session}
	repositoriesOutput, repositoriesErr := executeArchivesSpaceAPIRequest(r.sdk, cmd, "GET", "repositories", requestOpts)
	results = append(results, archivesSpaceRepositoriesResult(repositoriesOutput, repositoriesErr))

	requestOpts.query = []string{"q=*", "page=1", "page_size=1"}
	searchOutput, searchErr := executeArchivesSpaceAPIRequest(r.sdk, cmd, "GET", "search", requestOpts)
	results = append(results, archivesSpaceSearchResult(searchOutput, searchErr))
	return results, nil
}

func (r *archivesSpaceVerifyRunner) resolveSession(cmd *cobra.Command) (string, error) {
	session, err := readArchivesSpaceSecretIfConfigured(r.sessionFile, archivesSpaceSessionFileEnv, archivesSpaceSessionEnv)
	if err != nil || session != "" {
		return session, err
	}

	password, err := readArchivesSpaceSecretIfConfigured(r.passwordFile, archivesSpacePasswordFileEnv, archivesSpacePasswordEnv)
	if err != nil {
		return "", fmt.Errorf("load ArchivesSpace password: %w", err)
	}
	if password == "" && r.disposable {
		password = "admin"
	}
	if password == "" {
		return "", nil
	}
	output, err := executeArchivesSpaceLogin(r.sdk, cmd, r.baseURL, r.username, password)
	if err != nil {
		return "", fmt.Errorf("create ArchivesSpace verification session: %w", err)
	}
	var response struct {
		Session string `json:"session"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return "", fmt.Errorf("decode ArchivesSpace login response: %w", err)
	}
	if strings.TrimSpace(response.Session) == "" {
		return "", fmt.Errorf("ArchivesSpace login response omitted session")
	}
	return response.Session, nil
}

func archivesSpaceVersionResult(output string, requestErr error) sitevalidate.Result {
	if requestErr != nil {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:api-version", requestErr.Error(), "confirm the ArchivesSpace backend API is running on the configured URL")
	}
	const prefix = "ArchivesSpace (v"
	trimmed := strings.TrimSpace(output)
	if !strings.HasPrefix(trimmed, prefix) || !strings.HasSuffix(trimmed, ")") {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:api-version", fmt.Sprintf("unexpected version response %q", trimmed), "confirm the configured URL points to the ArchivesSpace backend API")
	}
	version := strings.TrimSuffix(strings.TrimPrefix(trimmed, prefix), ")")
	if version != archivesSpaceExpectedVersion {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:api-version", fmt.Sprintf("backend reports %s, expected %s", version, archivesSpaceExpectedVersion), "rebuild the site from the supported ArchivesSpace application and Solr image pair")
	}
	return sitevalidate.Result{Name: "verify:archivesspace:api-version", Status: sitevalidate.StatusOK, Detail: version}
}

func archivesSpaceRepositoriesResult(output string, requestErr error) sitevalidate.Result {
	if requestErr != nil {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:authentication", requestErr.Error(), "confirm the session is current and authorized to read repositories")
	}
	var repositories []json.RawMessage
	if err := json.Unmarshal([]byte(output), &repositories); err != nil {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:authentication", fmt.Sprintf("decode repositories response: %v", err), "confirm the session is current and the API response matches the supported ArchivesSpace release")
	}
	return sitevalidate.Result{Name: "verify:archivesspace:authentication", Status: sitevalidate.StatusOK, Detail: fmt.Sprintf("authenticated; %d repositories visible", len(repositories))}
}

func archivesSpaceSearchResult(output string, requestErr error) sitevalidate.Result {
	if requestErr != nil {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:search", requestErr.Error(), "confirm ArchivesSpace can query its configured Solr index")
	}
	var response struct {
		TotalHits *int `json:"total_hits"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:search", fmt.Sprintf("decode search response: %v", err), "confirm the ArchivesSpace and Solr release lines are compatible")
	}
	if response.TotalHits == nil {
		return failedArchivesSpaceVerifyResult("verify:archivesspace:search", "search response omitted total_hits", "confirm the configured URL points to the ArchivesSpace backend API")
	}
	return sitevalidate.Result{Name: "verify:archivesspace:search", Status: sitevalidate.StatusOK, Detail: fmt.Sprintf("search completed; %d hits", *response.TotalHits)}
}

func failedArchivesSpaceVerifyResult(name, detail, fix string) sitevalidate.Result {
	return sitevalidate.Result{Name: name, Status: sitevalidate.StatusFailed, Detail: detail, FixHint: fix}
}

var _ plugin.VerifyRunner = (*archivesSpaceVerifyRunner)(nil)
