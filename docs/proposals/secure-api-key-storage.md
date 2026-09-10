# API keys stored in plaintext in config file

## Problem

API keys are stored as plaintext strings in `~/.local/share/crush/crush.json`. Any process running as the same user can read them. Backups, dotfile repos, and accidental shares leak credentials.

## Goal

Store secrets in the OS-native keychain. Fall back to current plaintext behavior when keychain is unavailable. Zero migration friction — existing configs keep working.

## Current state

- `ProviderConfig.APIKey` written as literal string to JSON (`config.go:99`)
- `ConfigStore.SetConfigField()` writes with `0o600` permissions — only file-level protection (`store.go:128`)
- OAuth tokens (`access_token`, `refresh_token`) also plaintext (`store.go:252-254`)
- No encryption, no keychain integration, no obfuscation
- `$ENV_VAR` references already supported via `VariableResolver` (`resolve.go`) — good for CI, not the default UX
- `Scope` parameter added for global vs workspace config (`store.go`)
