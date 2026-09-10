# Status Report — Keychain PR #3771 Session (2026-09-10 09:15)

**Branch:** `feat/keychain-api-keys` @ `a9edb808` (3 commits, rebased on upstream/main `4a74b9e0`)
**PR:** charmbracelet/crush#3771 (draft) — **CI RED: 3 build checks fail (data race), 12/15 green**
**Scope of this session:** continue the keychain PR per self-review plan: coordinator tests, plaintext-fallback surfacing, PR body, rebase/push, CI verification.

---

## a) FULLY DONE

1. **Session-start verification** — repo state, branch, PR 15/15 green, upstream drift measured (2 commits: `4a74b9e0` sessions fix + CLA signature).
2. **3 carry-over questions resolved autonomously** (documented):
   - Q1 migration ambition → keep "no auto-migration" (status quo), documented as PR scope cut #1.
   - Q2 Discussion gate → no separate Discussion; #2477 reference in PR body suffices for a draft.
   - Q3 stray files → left untracked, never committed (`docs/proposals/`, `docs/reviews/`, `internal/stringext/plural_test.go`).
3. **Root-cause read of the 401 path** — `retryAfterUnauthorized` switch, `makeAuthRefreshCallback`, `refreshApiKeyTemplate`, `buildProvider` resolution point (coordinator.go:1202), load.go known-vs-custom provider loops.
4. **Discovery: custom providers never populate `APIKeyTemplate`** (load.go:431-502 custom loop) — keychain refs get no 401 recovery on custom providers, identical to pre-existing `$VAR` behavior. Documented as scope cut #4 in PR body.
5. **New test file `internal/agent/coordinator_keychain_test.go`** (167 lines, 3 tests):
   - `TestMakeAuthRefreshCallback` — 5-case table (keychain ref, `$VAR`, OAuth, AWS, static→nil).
   - `TestRetryAfterUnauthorizedReResolvesKeychainRef` — hermetic coordinator (known-provider path, mocked keyring), rotated secret re-resolved into provider, template survives.
   - `TestRetryAfterUnauthorizedKeychainEntryMissing` — deleted entry errors out, provider config unchanged.
   - All 3 pass locally (non-race mode); full agent package green.
6. **`keyring.Verify(providerID, secret)` helper + `TestVerifyDetectsFallback`** (4 assertions incl. unreachable-backend case).
7. **Plaintext-fallback warning in the API key dialog** (`api_key_input.go` `saveKeyAndContinue`): post-save `keyring.Verify` check → `util.ReportWarn` batched with the model-selection transition via `tea.Batch`. Documented client-side probe limitation inline.
8. **PR body amended** — added "Test isolation: the `testing.Testing()` gate" section, scope cut #4 (custom-provider parity), updated Evidence (new tests), dialog bullet updated. Two checkboxes preserved verbatim.
9. **Commit `a9edb808`** "feat: warn when keychain storage falls back to plaintext" (4 files, +208/−2), clean gofumpt, golangci-lint 0 issues on all touched packages.
10. **Rebase onto upstream/main + force-with-lease push** — clean, no conflicts.
11. **Local verification** — `go build ./...` pass; full `go test` pass excluding the pre-existing broken `internal/stringext` stray; keyring/config/agent/dialog/cmd packages green.
12. **CI diagnosis of the new failure** (see d).

## b) PARTIALLY DONE

1. **CI green on the new push** — 12/15 pass (lint ×3, CodeQL ×2, govulncheck, grype, CLA, dependency-review), but **build ×3 fail** on `go test -race -failfast ./...` with a data race in `TestRetryAfterUnauthorizedReResolvesKeychainRef`. Root cause fully identified; fix not yet written.
2. **Race fix design** (in progress when report requested):
   - Confirmed: race is in `zalando/go-keyring` v0.2.8 `mockProvider` — internal map has **no mutex** (keyring_mock.go:20/30).
   - Confirmed racer: `buildAgent` spawns async readiness goroutines (`readyWg`: prompt build + `buildTools` → `agentTool` → `buildAgentModels` → `buildProvider` → `Resolve` → `keyring.Get`) which read the mock map while the test goroutine writes via `keyring.Set("openai", "key-v2")`.
   - Confirmed: `UpdateModels`→`buildTools` path in the retry is synchronous (agentTool's `buildAgent` inside readiness is the async one).
   - Lead fix option: `require.NoError(t, coord.readyWg.Wait())` after `buildAgent`, before the mock mutation (same pattern as coordinator_readiness_test.go). Alternative: package mutex in our `call()` wrapper when mock active; upstream patch to go-keyring (slow, out of our control).
3. **Post-fix tasks queued but untouched**: amend PR body Evidence after race fix, re-push, re-poll CI.

## c) NOT STARTED

1. `readyWg.Wait()` race fix + local `-race` run of the agent package (I never ran `-race` locally — CI caught it).
2. Upstream sync-loop re-check for NEW upstream commits since this session's rebase.
3. Migration design for existing plaintext keys (deferred by documented decision — needs user's answer to previous session's Q1).
4. Full `oauth.Token` in keyring (scope cut #2 follow-up).
5. Custom-provider `APIKeyTemplate` parity fix (scope cut #4 — separate PR candidate).
6. Split-mode logout no-op (scope cut #3 — server-side deletion design).

## d) TOTALLY FUCKED UP

1. **CI went red on a PR that was 15/15 green.** My new coordinator test has a data race under `-race` — the one Go flag I did not run locally (`go test` without `-race` is green everywhere). This is the top-priority fix; the PR is unreviewable in this state.
2. **Contributing factor:** CI's `-race -failfast` combination is stricter than anything I ran locally; "all local tests green" was accepted as sufficient without matching CI's flags. That gap is on me, not the CI.
3. **Minor:** first version of the keychain coordinator test used `disable_default_providers: true`, which silently routes through the custom-provider loop and skips `APIKeyTemplate` — two tests failed for the wrong reason before I understood the two-loop structure of load.go. Cost: one extra iteration. (Redeeming: the failure exposed a real parity gap now documented in the PR.)

## e) WHAT WE SHOULD IMPROVE

1. **Always run `go test -race` on new concurrency-adjacent tests** — at minimum on the packages touched, matching CI flags exactly.
2. **Read the full provider-load loop before writing hermetic configs** — the `disable_default_providers` + known/custom two-loop structure silently changes which fields get populated.
3. **Verify test-global hygiene earlier** — MockInit/readyWg/availability-cache globals interact; a shared "keychain test kit" note in the keyring package docs would prevent the next session from rediscovering this.
4. **Memory gap still open:** the `GOTOOLCHAIN=auto` requirement (go.mod ≥1.27 vs local 1.26.7, LSP/gopls permanently broken) was flagged by the last self-review for persistence and is still not written anywhere durable.
5. **Consider upstreaming a mutex fix to go-keyring's mock** — every consumer of their mock with concurrent tests hits this.

## f) NEXT 50 (prioritized, grouped)

**Immediate — unblock CI (1-5)**
1. Add `coord.readyWg.Wait()` (or equivalent barrier) after `buildAgent` in the keychain test harness.
2. Run `go test -race ./internal/agent/ -run TestRetryAfterUnauthorized -count=1` locally until clean.
3. Run `-race` across all touched packages locally.
4. Commit fix (`fix: serialize keychain mock access in coordinator tests` or similar), push.
5. Re-poll `gh pr checks 3771` until 15/15 green.

**Race-hardening (6-10)**
6. Decide: barrier in tests vs mutex in `keyring.call()` under mock — pick one, delete the other.
7. Grep all keyring test files for other unguarded concurrent-mock accesses.
8. Check `TestRetryAfterUnauthorizedKeychainEntryMissing` for the same race (Delete vs readiness goroutine).
9. Confirm no goroutine from `UpdateModels` lingers post-return (read `agentTool` tool closure for lazy builds).
10. File upstream issue (or PR) at zalando/go-keyring for the unsynchronized mock.

**PR polish (11-16)**
11. Update PR body Evidence with race-fix note.
12. Re-read full PR diff end-to-end as a reviewer.
13. Verify PR body renders correctly on GitHub (checkboxes, code fences).
14. Consider trimming PR description length for reviewer fatigue.
15. Add PR comment on the custom-provider parity gap inviting maintainer guidance.
16. Cross-link PR from fork issue #3 with final state summary.

**Upstream tracking (17-20)**
17. Re-fetch upstream; note if new commits land during CI window.
18. Watch #2477 for maintainer comments on our PR reference.
19. Check if #2477 gets a Discussion opened by maintainers.
20. Set a personal reminder to rebase + re-push if upstream moves again.

**Memory/docs hygiene (21-24)**
21. Persist `GOTOOLCHAIN=auto` gotcha to project AGENTS.md or memory.
22. Persist "CI runs -race -failfast" fact to AGENTS.md.
23. Persist known/custom-provider two-loop structure insight to AGENTS.md.
24. Persist go-keyring mock thread-unsafety to AGENTS.md.

**Feature follow-ups from PR scope cuts (25-30)**
25. Design plaintext→keychain migration (needs user Q1 answer: opt-in command vs first-run prompt vs auto).
26. Prototype migration behind a flag, gated by `testing.Testing()` like the rest.
27. Move full `oauth.Token` into keyring (refresh token out of config).
28. Server-side logout deletion for split mode.
29. Custom-provider `APIKeyTemplate` population (parity PR).
30. Evaluate surfacing fallback warnings in the OAuth login dialog too.

**Quality gates (31-36)**
31. Full `go test -race ./...` clean baseline (minus stringext stray).
32. `go vet ./...` clean.
33. golangci-lint full-project run.
34. `task fmt` / gofumpt on all touched files.
35. Verify golden files unaffected (`go test ./... -update` dry check).
36. Review test coverage of store.go `refreshedSecretValue` paths.

**Repo hygiene (37-40)**
37. Decide fate of `internal/stringext/plural_test.go` stray (breaks local full-suite; needs user input).
38. Decide fate of `docs/proposals/` stray.
39. Decide whether `docs/reviews/.../brutal-self-review.html` should stay untracked or move to a gist.
40. Clean `/tmp/pr_body.md`, `/tmp/pr_body_new.md` scratch files.

**Strategic (41-46)**
41. Ping maintainers once CI is green (draft PRs get low attention without a ping).
42. Prepare a short changelog entry for the feature.
43. Draft the Discussion post in case maintainers ask for one.
44. Evaluate `go-keyring` alternatives (native bindings, 99designs/keyring) for the long term.
45. Document `keychain://` user-facing in README/config docs (PR currently only documents in code).
46. Add a `crush doctor`-style keyring diagnostics command (probe + service name).

**Testing depth (47-50)**
47. Table-test `storeSecret` fallback branches (unavailable vs Set-fail vs success).
48. Test dialog fallback warning via UI-level test if dialog test harness exists.
49. Property-style test: config file never contains a raw secret after save (golden).
50. Test `logout` keyring deletion paths incl. split-mode no-op documentation.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Race-fix approach approval:** barrier-wait in the tests (`readyWg.Wait()`), a mutex inside our `keyring.call()` wrapper (production code touched for test stability), or leave CI red and file upstream first? I recommend the barrier; do you agree, or do you want the wrapper mutex for defense-in-depth?
2. **Migration ambition (still unanswered from last session):** when plaintext keys exist at load, should Crush (a) leave them forever until manual re-auth, (b) offer a one-time opt-in `crush migrate-keys` command, or (c) prompt on first TUI start?
3. **The two stray files:** `internal/stringext/plural_test.go` (references nonexistent `Plural`, breaks every local full-suite run) and `docs/proposals/` — delete, fix, or leave untouched? They are not mine and I will not touch them without your call.

---

*Session window: ~07:45–09:15 UTC. Commits this session: `a9edb808`. CI state at writing: 12/15 green, 3 build failures (data race), fix designed but not implemented.*
