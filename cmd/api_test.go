package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
)

func TestArchivesSpaceAPIRequestKeepsSessionOutOfCommandArguments(t *testing.T) {
	const session = "sentinel-session-value"
	t.Setenv(archivesSpaceSessionEnv, session)

	original := archivesSpaceRunCurl
	t.Cleanup(func() { archivesSpaceRunCurl = original })

	var gotArgs []string
	var gotInput string
	archivesSpaceRunCurl = func(_ context.Context, _ *plugin.SDK, _ *cobra.Command, args []string, input string) (string, error) {
		gotArgs = append([]string(nil), args...)
		gotInput = input
		return `{"archivesSpaceVersion":"4.2.0"}`, nil
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	cmd := archivesSpaceVersionCommand(sdk)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("version command error = %v", err)
	}

	joined := strings.Join(gotArgs, " ")
	if strings.Contains(joined, session) {
		t.Fatalf("session leaked into curl argv: %q", joined)
	}
	if !strings.Contains(joined, "--config -") {
		t.Fatalf("curl argv does not read protected configuration from stdin: %q", joined)
	}
	if !strings.Contains(gotInput, session) {
		t.Fatalf("curl stdin does not contain the session header: %q", gotInput)
	}
	if strings.Contains(stdout.String(), session) || strings.Contains(stderr.String(), session) {
		t.Fatalf("session leaked into command output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if cmd.Flags().Lookup("session") != nil {
		t.Fatal("raw --session flag must not be exposed in process arguments")
	}
	if cmd.Flags().Lookup("session-file") == nil {
		t.Fatal("expected --session-file credential reference")
	}
}

func TestArchivesSpaceLoginKeepsPasswordOutOfCommandArguments(t *testing.T) {
	const password = "sentinel password value"
	t.Setenv(archivesSpacePasswordEnv, password)

	original := archivesSpaceRunCurl
	t.Cleanup(func() { archivesSpaceRunCurl = original })

	var gotArgs []string
	var gotInput string
	archivesSpaceRunCurl = func(_ context.Context, _ *plugin.SDK, _ *cobra.Command, args []string, input string) (string, error) {
		gotArgs = append([]string(nil), args...)
		gotInput = input
		return `{"session":"session-value"}`, nil
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	cmd := archivesSpaceLoginCommand(sdk)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("login command error = %v", err)
	}

	joined := strings.Join(gotArgs, " ")
	if strings.Contains(joined, password) {
		t.Fatalf("password leaked into curl argv: %q", joined)
	}
	if !strings.Contains(gotInput, password) {
		t.Fatalf("curl stdin does not contain the login form: %q", gotInput)
	}
	if strings.Contains(stdout.String(), password) || strings.Contains(stderr.String(), password) {
		t.Fatalf("password leaked into command output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if cmd.Flags().Lookup("password") != nil {
		t.Fatal("raw --password flag must not be exposed in process arguments")
	}
	if cmd.Flags().Lookup("password-file") == nil {
		t.Fatal("expected --password-file credential reference")
	}
}

func TestCurlConfigValueRejectsLineInjection(t *testing.T) {
	for _, value := range []string{"token\nheader = injected", "token\rheader = injected", "token\x00tail"} {
		if _, err := curlConfigValue(value); err == nil {
			t.Fatalf("curlConfigValue(%q) unexpectedly succeeded", value)
		}
	}
}

func TestCurlConfigValueEscapesQuotesAndBackslashes(t *testing.T) {
	got, err := curlConfigValue(`a\"b`)
	if err != nil {
		t.Fatalf("curlConfigValue() error = %v", err)
	}
	if got != `"a\\\"b"` {
		t.Fatalf("curlConfigValue() = %q", got)
	}
}

func TestArchivesSpaceAPIRequestReadsBodyFileLocallyAndUsesStdin(t *testing.T) {
	const body = "{\n  \"title\": \"Sentinel \\\\ body\"\n}\n"
	bodyFile := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(bodyFile, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	original := archivesSpaceRunCurl
	t.Cleanup(func() { archivesSpaceRunCurl = original })
	archivesSpaceRunCurl = func(_ context.Context, _ *plugin.SDK, _ *cobra.Command, args []string, input string) (string, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, bodyFile) || strings.Contains(joined, "Sentinel") {
			t.Fatalf("request body or local path leaked into container argv: %q", joined)
		}
		if !strings.Contains(joined, "--config -") || !strings.Contains(input, `data-binary = "{\n  \"title\": \"Sentinel \\\\ body\"\n}\n"`) {
			t.Fatalf("request body was not encoded through curl stdin: args=%q input=%q", joined, input)
		}
		return `{}`, nil
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	cmd := &cobra.Command{Use: "request"}
	if _, err := executeArchivesSpaceAPIRequest(sdk, cmd, "POST", "repositories/2/resources", archivesSpaceAPIOptions{
		baseURL: archivesSpaceInternalAPIURL,
		file:    bodyFile,
	}); err != nil {
		t.Fatalf("executeArchivesSpaceAPIRequest() error = %v", err)
	}
}

func TestArchivesSpaceAPIRequestRejectsAmbiguousBodies(t *testing.T) {
	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	cmd := &cobra.Command{Use: "request"}
	_, err := executeArchivesSpaceAPIRequest(sdk, cmd, "POST", "repositories/2/resources", archivesSpaceAPIOptions{
		baseURL: archivesSpaceInternalAPIURL,
		data:    `{}`,
		file:    "request.json",
	})
	if err == nil || !strings.Contains(err.Error(), "only one of --data or --file") {
		t.Fatalf("error = %v, want ambiguous body rejection", err)
	}
}

func TestBuildArchivesSpaceAPIURLRejectsCurlOptionsAndEmbeddedCredentials(t *testing.T) {
	for _, baseURL := range []string{
		"--config=/tmp/attacker",
		"file:///etc/passwd",
		"http://user:password@example.invalid/api",
		"//example.invalid/api",
	} {
		if _, err := buildArchivesSpaceAPIURL(baseURL, "version", nil); err == nil {
			t.Fatalf("buildArchivesSpaceAPIURL(%q) unexpectedly succeeded", baseURL)
		}
	}
}

func TestArchivesSpaceAPIRequestTerminatesCurlOptionParsingBeforeURL(t *testing.T) {
	original := archivesSpaceRunCurl
	t.Cleanup(func() { archivesSpaceRunCurl = original })
	archivesSpaceRunCurl = func(_ context.Context, _ *plugin.SDK, _ *cobra.Command, args []string, _ string) (string, error) {
		if len(args) < 2 || args[len(args)-2] != "--" || args[len(args)-1] != archivesSpaceInternalAPIURL+"/version" {
			t.Fatalf("curl arguments do not terminate option parsing before URL: %#v", args)
		}
		return `{}`, nil
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	cmd := &cobra.Command{Use: "request"}
	if _, err := executeArchivesSpaceAPIRequest(sdk, cmd, "GET", "version", archivesSpaceAPIOptions{baseURL: archivesSpaceInternalAPIURL}); err != nil {
		t.Fatalf("executeArchivesSpaceAPIRequest() error = %v", err)
	}
}
