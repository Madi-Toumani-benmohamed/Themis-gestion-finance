package config

import "testing"

func TestLoadOpenAI(t *testing.T) {
	for _, key := range []string{"AI_PROVIDER", "OLLAMA_URL", "MISTRAL_API_KEY", "FREELLM_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENAI_MODEL"} {
		t.Setenv(key, "")
	}
	t.Setenv("OPENAI_API_KEY", "test-secret")
	cfg := Load()
	if cfg.AIProvider != "openai" || cfg.OpenAIModel != "gpt-4.1-mini" || cfg.OpenAIKey != "test-secret" {
		t.Fatal("OpenAI environment config incorrect")
	}
	t.Setenv("OPENAI_MODEL", "custom-model")
	if Load().OpenAIModel != "custom-model" {
		t.Fatal("custom model ignored")
	}
	t.Setenv("OLLAMA_URL", "http://localhost:11434")
	if Load().AIProvider != "ollama" {
		t.Fatal("local provider should retain precedence")
	}
	t.Setenv("AI_PROVIDER", "none")
	if Load().AIProvider != "none" {
		t.Fatal("explicit disable should retain precedence")
	}
}
