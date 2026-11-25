package config

import (
	"strings"
	"testing"
)

func setDeps(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://ether:change-me@localhost:5432/ether?sslmode=disable")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MEILISEARCH_URL", "http://localhost:7700")
	t.Setenv("KEYCLOAK_URL", "http://localhost:8080")
}

func TestLoad_Defaults(t *testing.T) {
	setDeps(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "8080" || cfg.KeycloakRealm != "ether" || cfg.IdentityStore != "postgres" {
		t.Fatalf("defaults = port %q realm %q store %q", cfg.Port, cfg.KeycloakRealm, cfg.IdentityStore)
	}
	if cfg.KeycloakAudience != "" || cfg.KeycloakClientID != "" {
		t.Fatalf("audience/client = %q/%q, want empty when unset", cfg.KeycloakAudience, cfg.KeycloakClientID)
	}
}

func TestLoad_IdentitySettings(t *testing.T) {
	setDeps(t)
	t.Setenv("KEYCLOAK_REALM", "custom")
	t.Setenv("KEYCLOAK_CLIENT_ID", "ether-api")
	t.Setenv("KEYCLOAK_AUDIENCE", "ether-api")
	t.Setenv("IDENTITY_STORE", "memory")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.KeycloakRealm != "custom" || cfg.KeycloakClientID != "ether-api" ||
		cfg.KeycloakAudience != "ether-api" || cfg.IdentityStore != "memory" {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestLoad_InvalidIdentityStore(t *testing.T) {
	setDeps(t)
	t.Setenv("IDENTITY_STORE", "banana")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "IDENTITY_STORE") {
		t.Fatalf("err = %v, want IDENTITY_STORE error", err)
	}
}

func TestLoad_MissingDependency(t *testing.T) {
	setDeps(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("err = %v, want DATABASE_URL missing error", err)
	}
}
