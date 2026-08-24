package common

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repositoryRootForConfigTest(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Dir(filepath.Dir(filename))
}

func readRepositoryConfigForTest(t *testing.T, name string) string {
	t.Helper()
	root := repositoryRootForConfigTest(t)
	content, err := os.ReadFile(filepath.Join(root, name))
	require.NoError(t, err)
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

func dockerIgnoreExcludesTestPath(rules string, fileName string) bool {
	excluded := false
	for _, rawRule := range strings.Split(strings.ReplaceAll(rules, "\r\n", "\n"), "\n") {
		rule := strings.TrimSpace(rawRule)
		if rule == "" || strings.HasPrefix(rule, "#") {
			continue
		}
		negated := strings.HasPrefix(rule, "!")
		if negated {
			rule = strings.TrimPrefix(rule, "!")
		}
		rule = strings.TrimPrefix(rule, "/")
		matched, err := path.Match(rule, fileName)
		if err != nil {
			continue
		}
		if matched {
			excluded = !negated
		}
	}
	return excluded
}

func TestDockerIgnoreExcludesRuntimeEnvFilesAndKeepsExample(t *testing.T) {
	rules := readRepositoryConfigForTest(t, ".dockerignore")
	assert.Contains(t, rules, "\n.env\n")
	assert.Contains(t, rules, "\n.env-prod\n")
	assert.Contains(t, rules, "\n.env.*\n")
	assert.Contains(t, rules, "\n!.env.example\n")

	for _, fileName := range []string{".env", ".env-prod", ".env.test", ".env-prod.backup", ".env-local"} {
		assert.True(t, dockerIgnoreExcludesTestPath(rules, fileName), fileName)
	}
	assert.False(t, dockerIgnoreExcludesTestPath(rules, ".env.example"))
}

func TestRuntimeDeploymentContractLoadsLocalEnvWithoutEmbeddingIt(t *testing.T) {
	example := readRepositoryConfigForTest(t, ".env.example")
	for _, key := range []string{
		"STRIPE_SUBSCRIPTION_ENABLED",
		"STRIPE_SUBSCRIPTION_ACCOUNT_ID",
		"STRIPE_SUBSCRIPTION_PRODUCT_ID",
		"STRIPE_SUBSCRIPTION_FOUNDER_PRICE_ID",
		"STRIPE_SUBSCRIPTION_STANDARD_PRICE_ID",
		"STRIPE_SUBSCRIPTION_PORTAL_CONFIGURATION_ID",
	} {
		assert.Contains(t, example, key)
	}
	assert.Contains(t, example, "STRIPE_SUBSCRIPTION_ENABLED=false")

	for _, composeFile := range []string{"docker-compose.yml", "docker-compose.dev.yml"} {
		compose := readRepositoryConfigForTest(t, composeFile)
		assert.Contains(t, compose, "env_file:")
		assert.Contains(t, compose, "${NOVAPURA_ENV_FILE:-.env}")
	}
	assert.Contains(t, readRepositoryConfigForTest(t, "new-api.service"), "EnvironmentFile=-/path/to/new-api/.env")
	assert.Contains(t, readRepositoryConfigForTest(t, "docs/ops/08-cloud-run-secret-manager-清单.md"), "STRIPE_SUBSCRIPTION_ENABLED")
}

func TestRuntimeDeploymentSelectsTestAndProductionEnvFiles(t *testing.T) {
	for _, composeFile := range []string{"docker-compose.yml", "docker-compose.dev.yml"} {
		compose := readRepositoryConfigForTest(t, composeFile)
		assert.Contains(t, compose, "${NOVAPURA_ENV_FILE:-.env}")
	}
	productionDocs := readRepositoryConfigForTest(t, "docs/ops/05-local-deploy-and-admin.md")
	if !strings.Contains(productionDocs, "$env:NOVAPURA_ENV_FILE='.env-prod'") {
		t.Fatal("production deployment instructions must select .env-prod")
	}
	if !strings.Contains(productionDocs, "docker compose --env-file .env-prod") {
		t.Fatal("production deployment instructions must pass .env-prod to Compose")
	}
	productionUnit := readRepositoryConfigForTest(t, "new-api-prod.service")
	if !strings.Contains(productionUnit, "EnvironmentFile=-/path/to/new-api/.env-prod") {
		t.Fatal("production systemd unit must load .env-prod")
	}
	if !strings.Contains(productionUnit, "GIN_MODE=release") {
		t.Fatal("production systemd unit must select release runtime")
	}
}

func TestProductionEnvFileUsesDotenvAssignmentsOnly(t *testing.T) {
	content := readRepositoryConfigForTest(t, ".env-prod")
	assignment := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=.*$`)
	for lineNumber, rawLine := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !assignment.MatchString(line) {
			t.Fatalf("invalid dotenv assignment at line %d", lineNumber+1)
		}
	}
	for lineNumber, rawLine := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if strings.Contains(rawLine, "$env:") {
			t.Fatalf("PowerShell assignment marker remains at line %d", lineNumber+1)
		}
	}
}

func TestGetEnvOrDefaultBoolInvalidValueLogIsRedacted(t *testing.T) {
	const key = "CODEX_INVALID_BOOL_LOG_TEST"
	const invalidValue = "invalid-boolean"
	t.Setenv(key, invalidValue)

	previousWriter := gin.DefaultErrorWriter
	var output bytes.Buffer
	gin.DefaultErrorWriter = &output
	t.Cleanup(func() { gin.DefaultErrorWriter = previousWriter })

	assert.True(t, GetEnvOrDefaultBool(key, true))
	message := output.String()
	assert.Contains(t, message, key)
	assert.Contains(t, message, "invalid boolean")
	assert.Contains(t, message, "default value: true")
	assert.NotContains(t, message, invalidValue)
	assert.NotContains(t, message, "strconv")
}
