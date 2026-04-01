package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOAuthJSON_Valid(t *testing.T) {
	raw := `{
		"claudeAiOauth": {"accessToken": "tok_abc123"},
		"claudeAiSubscriptionType": "max"
	}`
	token, subType := parseOAuthJSON(raw)
	assert.Equal(t, "tok_abc123", token)
	assert.Equal(t, "Max", subType)
}

func TestParseOAuthJSON_ProDefault(t *testing.T) {
	raw := `{"claudeAiOauth": {"accessToken": "tok_xyz"}}`
	token, subType := parseOAuthJSON(raw)
	assert.Equal(t, "tok_xyz", token)
	assert.Equal(t, "Pro", subType)
}

func TestParseOAuthJSON_NoToken(t *testing.T) {
	raw := `{"claudeAiOauth": {}}`
	token, subType := parseOAuthJSON(raw)
	assert.Empty(t, token)
	assert.Empty(t, subType)
}

func TestParseOAuthJSON_InvalidJSON(t *testing.T) {
	token, subType := parseOAuthJSON("not json")
	assert.Empty(t, token)
	assert.Empty(t, subType)
}

func TestParseOAuthJSON_EmptyString(t *testing.T) {
	token, subType := parseOAuthJSON("")
	assert.Empty(t, token)
	assert.Empty(t, subType)
}

func TestResolveAuth_EnvVar(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test123")
	auth, err := ResolveAuth()
	assert.NoError(t, err)
	assert.Equal(t, "sk-ant-test123", auth.Key)
	assert.False(t, auth.IsOAuth)
	assert.Equal(t, "API Key", auth.Display)
}

func TestResolveAuth_NoAuth(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	// This will fail to find keychain/file on CI but should return an error
	auth, err := ResolveAuth()
	if err != nil {
		assert.Nil(t, auth)
		assert.Contains(t, err.Error(), "no authentication found")
	} else {
		// Running on a machine with Claude Code credentials — that's fine
		assert.NotEmpty(t, auth.Key)
		assert.True(t, auth.IsOAuth)
	}
}
