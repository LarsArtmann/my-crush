# Implementation Plan: OS Keychain for API Keys

## Library: `zalando/go-keyring`

No CGO. Pure Go. `Set/Get/Delete` with build-tag backends. Service name: `"crush"`, user: `providerID`.

Key sentinels: `ErrNotFound`, `ErrUnsupportedPlatform`. Mock: `MockInit()`, `MockInitWithError()`.

## Gate: `keyring.Available()` only, no platform checks

| Environment | `Available()` | Storage |
|---|---|---|
| macOS | Always true | Keychain |
| Windows | Always true | Credential Manager |
| Linux desktop (+ SSH) | True — D-Bus shared with desktop session | GNOME Keyring / KWallet |
| Linux headless (CI, container, WSL) | False | Plaintext `0o600` |

## Config format

```jsonc
// Before (legacy, auto-migrated)
{ "providers": { "openai": { "api_key": "sk-abc123..." } } }
// After
{ "providers": { "openai": { "api_key": "keychain://openai" } } }
// Env var — unchanged, never migrated
{ "providers": { "openai": { "api_key": "$OPENAI_API_KEY" } } }
```

## New package: `internal/keyring/keyring.go`

Wrapper with 3s timeouts on all ops (D-Bus can hang, same pattern as GitHub CLI `cli/cli/internal/keyring/keyring.go`):

```go
func Set(providerID, secret string) error
func Get(providerID string) (string, error)
func Delete(providerID string) error
func Available() bool                              // probe: write+read+delete test entry
func MockInit()                                    // call in init() of keychain test files only
func IsUnavailable(err error) bool                 // ErrUnsupportedPlatform || TimeoutError
```

Map `ErrUnsupportedPlatform` → `ErrUnavailable`. Wrap all calls in goroutine+`select`+`time.After(3s)`.

## Resolution order (`internal/config/resolve.go`)

Add **before** the `$` check in `shellVariableResolver.ResolveValue()` **and** `identityResolver.ResolveValue()`:

```go
if strings.HasPrefix(value, "keychain://") {
    return keyring.Get(strings.TrimPrefix(value, "keychain://"))
}
```

Order: `keychain://` → `$VAR`/`$(cmd)` → literal plaintext.

---

## Touch points (all in upstream `charmbracelet/crush`)

### `store.go:SetProviderAPIKey` (line 240) — **string case**

```go
ref := v // plaintext fallback
if keyring.Available() {
    if err := keyring.Set(providerID, v); err == nil {
        ref = "keychain://" + providerID
    }
}
if err := s.SetConfigField(scope, fmt.Sprintf("providers.%s.api_key", providerID), ref); err != nil {
    if ref != v { _ = keyring.Delete(providerID) } // rollback
    return fmt.Errorf("failed to save api key to config file: %w", err)
}
setKeyOrToken = func() { providerConfig.APIKey = ref }
```

### `store.go:SetProviderAPIKey` — **OAuth case** (line 251)

Same pattern. Store `AccessToken` in keychain. Keep `oauth` struct (refresh_token + expires_at) in config.

**Known gap**: `refresh_token` stays in plaintext config. A refresh token can generate new access tokens — it's a credential. If we want full protection, serialize the whole `oauth.Token` as JSON and store it in keychain under `providerID + "#oauth"`, then remove the `oauth` field from config. This adds complexity (serialization, partial keychain failure). Decision: start with access-token-only in keychain, file a follow-up for full OAuth protection.

```go
apiKeyValue := v.AccessToken
if keyring.Available() {
    if err := keyring.Set(providerID, v.AccessToken); err == nil {
        apiKeyValue = "keychain://" + providerID
    }
}
// SetConfigFields with apiKeyValue + oauth struct
```

### `store.go:RefreshOAuthToken` (line 306)

Three sub-paths, all need keychain update:

- **3a** Cross-session check (line 318-329): after `providerConfig.APIKey = newToken.AccessToken`, also `keyring.Set(providerID, newToken.AccessToken)`
- **3b** Actual refresh (line 346-362): same pattern as OAuth case above — `keychain://` ref + update keychain
- **3c** In-memory update (line 346): use same `apiKeyValue` var (either `keychain://` or plaintext)

### `store.go:ImportCopilot` (line 452)

Same pattern. Line 473: replace `token.AccessToken` with `keychain://copilot` when available.

### `load.go:configureProviders` (line 164) — **auto-migration**

After each `c.Providers.Set(...)` in both known-provider loop (line 339) and custom-provider loop (line 403):

```go
rawKey := p.APIKey
if rawKey != "" && !strings.HasPrefix(rawKey, "$") && !strings.HasPrefix(rawKey, "keychain://") && keyring.Available() {
    if err := keyring.Set(id, rawKey); err == nil {
        if err := store.SetConfigField(ScopeGlobal, fmt.Sprintf("providers.%s.api_key", id), "keychain://"+id); err != nil {
            _ = keyring.Delete(id) // rollback
        }
    }
}
```

`ScopeGlobal` because API keys are never at workspace scope (workspace config goes in git). `autoReloadDisabled` is already true during `configureProviders` (line 104), so migration writes don't trigger re-entry.

### `load.go:configureProviders` — **provider deletion**

Replace every `c.Providers.Del(string(p.ID))` with:

```go
c.Providers.Del(string(p.ID))
_ = keyring.Delete(string(p.ID))
```

### `load.go:APIKeyTemplate` (line 247)

No change. `APIKeyTemplate` stores the raw config value. After migration, the raw value becomes `keychain://` on next load, and re-resolution calls `keyring.Get()` — correct.

### `config.go:TestConnection` (line 736)

No change. Already uses `resolver.ResolveValue(c.APIKey)`.

### `ui/dialog/api_key_input.go`

Show "Stored in: System Keychain" vs config file path based on `keyring.Available()`.

---

## Edge cases

| Scenario | Behavior |
|---|---|
| `keychain://` ref but entry deleted | `ErrNotFound` → provider skipped (same as missing `$VAR`) |
| `keychain://` ref, headless Linux | `ErrUnavailable` → provider skipped with warning |
| Linux desktop over SSH | D-Bus works — keychain available |
| Two sessions auto-migrate simultaneously | Idempotent `keyring.Set`, read-modify-write config. Safe. |
| Keychain write succeeds, config write fails | Rollback: `keyring.Delete`. Return error. |
| User edits config back to literal | Next load: migration runs again. |
| `$VAR` in config | Skipped by migration (`HasPrefix("$")` check). |
| Config copied to another machine | `keychain://` refs in config point to keychain entries that don't exist on new machine. User must re-enter keys. Same as today if env vars aren't set. Tradeoff accepted. |

## Files changed

| File | Change |
|---|---|
| `internal/keyring/keyring.go` | **NEW** |
| `internal/config/resolve.go` | `keychain://` prefix in both resolvers |
| `internal/config/store.go` | `SetProviderAPIKey`, `RefreshOAuthToken`, `ImportCopilot` |
| `internal/config/load.go` | auto-migration + keychain cleanup on `Providers.Del` |
| `internal/ui/dialog/api_key_input.go` | storage backend label |

No changes: `backend/config.go` (delegate), `config.go` (struct + TestConnection unchanged), `workspace/app_workspace.go` (delegate).

## Testing

CI has no D-Bus → `Available()` = false → existing tests unchanged. Only keychain-specific test files call `MockInit()` in `init()`.

New files: `keyring/keyring_test.go`, `config/resolve_keychain_test.go`, `config/store_keychain_test.go`.
