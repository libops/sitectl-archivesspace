package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/libops/sitectl/pkg/config"
	"github.com/libops/sitectl/pkg/plugin"
	sitevalidate "github.com/libops/sitectl/pkg/validate"
	"github.com/spf13/cobra"
)

func TestArchivesSpaceVerifyChecksAuthenticatedSearchWithoutLeakingSession(t *testing.T) {
	const session = "verify-session-sentinel"
	t.Setenv(archivesSpaceSessionEnv, session)

	original := archivesSpaceRunCurl
	t.Cleanup(func() { archivesSpaceRunCurl = original })

	var calls []string
	archivesSpaceRunCurl = func(_ context.Context, _ *plugin.SDK, _ *cobra.Command, args []string, input string) (string, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, session) {
			t.Fatalf("session leaked into verifier argv: %q", joined)
		}
		calls = append(calls, joined)
		switch {
		case strings.Contains(joined, "/version"):
			return `ArchivesSpace (v4.2.0)`, nil
		case strings.Contains(joined, "/repositories"):
			if !strings.Contains(input, session) {
				t.Fatalf("authenticated repository request did not receive session through stdin")
			}
			return `[{"repo_code":"TEST"}]`, nil
		case strings.Contains(joined, "/search?"):
			if !strings.Contains(input, session) {
				t.Fatalf("authenticated search request did not receive session through stdin")
			}
			if !strings.Contains(joined, "page=1") || !strings.Contains(joined, "page_size=1") || !strings.Contains(joined, "q=%2A") || strings.Contains(joined, "q%5B%5D") {
				t.Fatalf("authenticated search did not use the documented q and pagination parameters: %q", joined)
			}
			return `{"total_hits":0,"results":[]}`, nil
		default:
			t.Fatalf("unexpected verifier curl call: %q", joined)
			return "", nil
		}
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	runner := &archivesSpaceVerifyRunner{sdk: sdk}
	cmd := &cobra.Command{Use: "verify"}
	runner.BindFlags(cmd)
	results, err := runner.Run(cmd, &config.Context{Name: "test", Plugin: pluginName})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(calls) != 3 {
		t.Fatalf("verifier curl calls = %d, want 3: %v", len(calls), calls)
	}
	if len(results) != 3 {
		t.Fatalf("verification results = %d, want 3: %+v", len(results), results)
	}
	for _, result := range results {
		if result.Status != sitevalidate.StatusOK {
			t.Fatalf("verification result is not OK: %+v", result)
		}
	}
}

func TestArchivesSpaceVerifyFailsWhenCredentialReferenceIsMissing(t *testing.T) {
	t.Setenv(archivesSpaceSessionEnv, "")

	original := archivesSpaceRunCurl
	t.Cleanup(func() { archivesSpaceRunCurl = original })
	archivesSpaceRunCurl = func(_ context.Context, _ *plugin.SDK, _ *cobra.Command, args []string, _ string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "/version") {
			return `ArchivesSpace (v4.2.0)`, nil
		}
		t.Fatal("authenticated checks must not run without a credential reference")
		return "", nil
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	runner := &archivesSpaceVerifyRunner{sdk: sdk}
	cmd := &cobra.Command{Use: "verify"}
	runner.BindFlags(cmd)
	results, err := runner.Run(cmd, &config.Context{Name: "test", Plugin: pluginName})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("verification results = %d, want 3: %+v", len(results), results)
	}
	if results[0].Status != sitevalidate.StatusOK || results[1].Status != sitevalidate.StatusFailed || results[2].Status != sitevalidate.StatusFailed {
		t.Fatalf("unexpected credential-missing results: %+v", results)
	}
}

func TestArchivesSpaceVerifyRejectsUnsupportedBackendVersion(t *testing.T) {
	result := archivesSpaceVersionResult(`ArchivesSpace (v4.1.1)`, nil)
	if result.Status != sitevalidate.StatusFailed || !strings.Contains(result.Detail, archivesSpaceExpectedVersion) {
		t.Fatalf("unsupported backend version was accepted: %+v", result)
	}
}

func TestArchivesSpaceVerifyRejectsNonDocumentedVersionShape(t *testing.T) {
	result := archivesSpaceVersionResult(`{"archivesSpaceVersion":"4.2.0"}`, nil)
	if result.Status != sitevalidate.StatusFailed || !strings.Contains(result.Detail, "unexpected version response") {
		t.Fatalf("non-documented backend version response was accepted: %+v", result)
	}
}

func TestArchivesSpaceVerifyDisposableLoginKeepsDefaultCredentialOutOfArguments(t *testing.T) {
	t.Setenv(archivesSpaceSessionEnv, "")
	t.Setenv(archivesSpacePasswordEnv, "")

	original := archivesSpaceRunCurl
	t.Cleanup(func() { archivesSpaceRunCurl = original })

	var loginSeen bool
	archivesSpaceRunCurl = func(_ context.Context, _ *plugin.SDK, _ *cobra.Command, args []string, input string) (string, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "password=admin") {
			t.Fatalf("disposable password leaked into verifier argv: %q", joined)
		}
		switch {
		case strings.Contains(joined, "/version"):
			return `ArchivesSpace (v4.2.0)`, nil
		case strings.Contains(joined, "/users/admin/login"):
			loginSeen = true
			if !strings.Contains(input, "password=admin") {
				t.Fatalf("disposable login password did not arrive through stdin: %q", input)
			}
			return `{"session":"disposable-session"}`, nil
		case strings.Contains(joined, "/repositories"):
			return `[]`, nil
		case strings.Contains(joined, "/search?"):
			if !strings.Contains(joined, "page=1") || !strings.Contains(joined, "page_size=1") || !strings.Contains(joined, "q=%2A") || strings.Contains(joined, "q%5B%5D") {
				t.Fatalf("authenticated search did not use the documented q and pagination parameters: %q", joined)
			}
			return `{"total_hits":0,"results":[]}`, nil
		default:
			t.Fatalf("unexpected verifier curl call: %q", joined)
			return "", nil
		}
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	runner := &archivesSpaceVerifyRunner{sdk: sdk, disposable: true, username: "admin", baseURL: archivesSpaceInternalAPIURL}
	cmd := &cobra.Command{Use: "verify"}
	results, err := runner.Run(cmd, &config.Context{Name: "test", Plugin: pluginName})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !loginSeen {
		t.Fatal("disposable verifier did not create a session")
	}
	for _, result := range results {
		if result.Status != sitevalidate.StatusOK {
			t.Fatalf("verification result is not OK: %+v", result)
		}
	}
}
