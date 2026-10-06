package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSecretFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secret")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestFileEnvOverrideSecretsKeys(t *testing.T) {
	path := writeSecretFile(t, "first-key-that-is-long\nsecond-key-that-is-long\n")
	t.Setenv("SECRETS_KEYS_FILE", path)

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"first-key-that-is-long", "second-key-that-is-long"}, cfg.Secrets.Keys)
}

func TestFileEnvOverrideSecretsKeysCommaSeparated(t *testing.T) {
	path := writeSecretFile(t, "first-key-that-is-long,second-key-that-is-long")
	t.Setenv("SECRETS_KEYS_FILE", path)

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"first-key-that-is-long", "second-key-that-is-long"}, cfg.Secrets.Keys)
}

func TestFileEnvOverrideDatabasePassword(t *testing.T) {
	path := writeSecretFile(t, "database-password\n")
	t.Setenv("DATABASE_PASSWORD_FILE", path)

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)

	assert.Equal(t, "database-password", cfg.Database.Password)
}

func TestFileEnvOverrideThirdPartyProviderSecret(t *testing.T) {
	path := writeSecretFile(t, "google-client-secret")
	t.Setenv("THIRD_PARTY_PROVIDERS_GOOGLE_SECRET_FILE", path)

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)

	assert.Equal(t, "google-client-secret", cfg.ThirdParty.Providers.Google.Secret)
}

func TestFileEnvOverrideEmailDeliverySmtpPassword(t *testing.T) {
	path := writeSecretFile(t, "smtp-password")
	t.Setenv("EMAIL_DELIVERY_SMTP_PASSWORD_FILE", path)

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)

	assert.Equal(t, "smtp-password", cfg.EmailDelivery.SMTP.Password)
}

func TestFileEnvOverrideRateLimiterRedisPassword(t *testing.T) {
	path := writeSecretFile(t, "redis-password")
	t.Setenv("RATE_LIMITER_REDIS_PASSWORD_FILE", path)

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)

	require.NotNil(t, cfg.RateLimiter.Redis)
	assert.Equal(t, "redis-password", cfg.RateLimiter.Redis.Password)
}

func TestFileEnvOverrideExplicitEnvWins(t *testing.T) {
	path := writeSecretFile(t, "password-from-file")
	t.Setenv("DATABASE_PASSWORD_FILE", path)
	t.Setenv("DATABASE_PASSWORD", "password-from-env")

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)

	assert.Equal(t, "password-from-env", cfg.Database.Password)
}

func TestFileEnvOverrideMissingFileFails(t *testing.T) {
	t.Setenv("SECRETS_KEYS_FILE", filepath.Join(t.TempDir(), "does-not-exist"))

	configPath := "./minimal-config.yaml"
	_, err := Load(&configPath)

	assert.Error(t, err)
}

func TestFileEnvOverrideUnknownVariableIgnored(t *testing.T) {
	t.Setenv("HANKO_UNKNOWN_SETTING_FILE", "/does/not/exist")

	configPath := "./minimal-config.yaml"
	cfg, err := Load(&configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}
