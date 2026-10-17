# fnOS GUI Password Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In fnOS LAN mode, browsers must set then enter a panel password before receiving `opensurge_session`; `/enter` no longer grants a cookie from Host allowlist alone.

**Architecture:** Store a bcrypt hash in `{store}/gui-password`. New unauthenticated session APIs (status/setup/login) plus logout/change-password. `auth()` ignores browser sessions when the hash file is missing. React `/setup` and `/login` pages replace the auto-`/enter` cookie grant. macOS desktop (non-`lanMode`) is unchanged.

**Tech Stack:** Go `controlapi` (`golang.org/x/crypto/bcrypt`), existing HttpOnly session cookies, React + vitest.

**Spec:** `docs/superpowers/specs/2026-10-17-gui-password-design.md`

Do not commit unrelated dirty 0.1.2 branding/source-manage files. Do not bump version or push GHCR.

---

## File structure

| File | Responsibility |
| --- | --- |
| `internal/controlapi/gui_password.go` | Store JSON hash: create / replace / match / configured |
| `internal/controlapi/gui_password_test.go` | Store table tests |
| `internal/controlapi/session_password.go` | HTTP: status, setup, login, logout, change password, rate limit, enter redirect |
| `internal/controlapi/session_password_test.go` | HTTP table tests |
| `internal/controlapi/server.go` | Register routes; `auth()` session invalid when hash missing |
| `internal/controlapi/baseurl_test.go` | Rewrite `/enter` expectations |
| `web/src/api.ts` | session status/setup/login/logout/password; 401 event skip |
| `web/src/types.ts` | `SessionStatus` |
| `web/src/pages/SetupPage.tsx` | First-run set password |
| `web/src/pages/LoginPage.tsx` | Login form |
| `web/src/pages/auth-pages.test.tsx` | Page tests |
| `web/src/App.tsx` | Status gate; sidebar change/logout |
| `web/src/App.test.tsx` | Gate + sidebar tests |
| `web/src/styles.css` | Login card |
| `web/src/i18n.ts` / `i18n.en.ts` | Strings |
| fnOS + agent-wiki docs | `/enter` now means set/login, not free cookie |

---

## Chunk 1: Password store

### Task 1: Store bcrypt hash in `{store}/gui-password`

**Files:**
- Create: `internal/controlapi/gui_password.go`
- Test: `internal/controlapi/gui_password_test.go`

- [ ] **Step 1: Write failing store tests**

```go
package controlapi

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestCreateGUIPasswordWrites0600BcryptFile(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateGUIPassword("correct horse"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(store.Dir(), "gui-password"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o", info.Mode().Perm())
	}
	ok, err := store.PasswordMatches("correct horse")
	if err != nil || !ok {
		t.Fatalf("match = %v, %v", ok, err)
	}
	ok, err = store.PasswordMatches("wrong")
	if err != nil || ok {
		t.Fatalf("wrong password matched: %v %v", ok, err)
	}
}

func TestCreateGUIPasswordRejectsShortAndDuplicate(t *testing.T) {
	store := NewStore(t.TempDir())
	_ = store.Ensure()
	if err := store.CreateGUIPassword("short"); err == nil {
		t.Fatal("expected short password error")
	}
	if err := store.CreateGUIPassword("long-enough-password"); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateGUIPassword("another-long-password"); !errors.Is(err, errGUIPasswordConfigured) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateGUIPasswordConcurrentOnlyOneSucceeds(t *testing.T) {
	store := NewStore(t.TempDir())
	_ = store.Ensure()
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.CreateGUIPassword("long-enough-password")
		}()
	}
	wg.Wait()
	close(errs)
	var ok, dup int
	for err := range errs {
		if err == nil {
			ok++
		} else if errors.Is(err, errGUIPasswordConfigured) {
			dup++
		} else {
			t.Fatalf("unexpected %v", err)
		}
	}
	if ok != 1 || dup != 7 {
		t.Fatalf("ok=%d dup=%d", ok, dup)
	}
}

func TestReplaceGUIPasswordChangesHash(t *testing.T) {
	store := NewStore(t.TempDir())
	_ = store.Ensure()
	if err := store.CreateGUIPassword("old-password-ok"); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceGUIPassword("new-password-ok"); err != nil {
		t.Fatal(err)
	}
	ok, _ := store.PasswordMatches("new-password-ok")
	if !ok {
		t.Fatal("new password should match")
	}
	ok, _ = store.PasswordMatches("old-password-ok")
	if ok {
		t.Fatal("old password should not match")
	}
}

func TestPasswordMatchesMissingFile(t *testing.T) {
	store := NewStore(t.TempDir())
	_ = store.Ensure()
	ok, err := store.PasswordMatches("anything1")
	if ok || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestGUIPasswordRejectsTooLong(t *testing.T) {
	store := NewStore(t.TempDir())
	_ = store.Ensure()
	long := make([]rune, 129)
	for i := range long {
		long[i] = 'a'
	}
	if utf8.RuneCountInString(string(long)) != 129 {
		t.Fatal("fixture")
	}
	if err := store.CreateGUIPassword(string(long)); err == nil {
		t.Fatal("expected too long")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
PATH="/Users/renq/.goenv/versions/1.26.1/bin:$PATH" GOCACHE=/private/tmp/opensurge-gocache-126 GOTOOLCHAIN=local \
  go test ./internal/controlapi/ -count=1 -run 'TestCreateGUIPassword|TestReplaceGUIPassword|TestPasswordMatchesMissingFile|TestGUIPasswordRejectsTooLong'
```

Expected: FAIL compile (`CreateGUIPassword` undefined).

- [ ] **Step 3: Implement store**

`gui_password.go`: JSON record `schema_version=1`, `algo=bcrypt`, `hash`, `updated_at` RFC3339 UTC. `bcrypt.DefaultCost`. Length 8–128 runes (`utf8.RuneCountInString`). `CreateGUIPassword` under `s.mu`: if file exists → `errGUIPasswordConfigured`; else hash + `writeAtomic(..., 0o600)`. `ReplaceGUIPassword` overwrites. `PasswordMatches` reads file, `bcrypt.CompareHashAndPassword`. `GUIPasswordConfigured` true iff file exists and parses with `algo=bcrypt`.

Promote `golang.org/x/crypto` to a direct module dependency.

- [ ] **Step 4: Run tests to verify they pass**

Same `go test` command. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/controlapi/gui_password.go internal/controlapi/gui_password_test.go go.mod go.sum
git commit -m "feat(controlapi): store bcrypt GUI password hash"
```

---

## Chunk 2: Session HTTP and auth

### Task 2: Session status / setup / login

**Files:**
- Create: `internal/controlapi/session_password.go`
- Test: `internal/controlapi/session_password_test.go`
- Modify: `internal/controlapi/server.go` (`Handler` routes ~302; `auth` ~602)

Helper in tests (LAN server like `baseurl_test.go`):

```go
func newLANControl(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	server, err := New(Options{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Addr:       "0.0.0.0:61767",
		BaseURL:    "http://192.168.1.20:61767",
		StoreDir:   filepath.Join(dir, "store"),
		Runner:     fakeRunner{},
		Static:     http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func lanPOST(server *Server, path, origin, body, remote string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Host = "192.168.1.20:61767"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	req.RemoteAddr = remote
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}
```

Cover: status before/after setup; setup 204 + cookie + 0600 file; second setup 409; login success; wrong password 401 `invalid_password`; login before setup 409; missing Origin 403; non-lanMode setup 401; password not in error body.

Register **without** `auth()`:

- `GET /api/v1/session/status`
- `POST /api/v1/session/setup`
- `POST /api/v1/session/login`
- `POST /api/v1/session/logout`

Register **with** `auth()`:

- `PUT /api/v1/session/password`

Setup/login POST: `originAllowed`; empty Origin rejected.

- [ ] **Step 1: Write failing HTTP tests** (file above)
- [ ] **Step 2: Run** `go test ./internal/controlapi/ -count=1 -run TestSession` — FAIL
- [ ] **Step 3: Implement handlers + routes**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit** `feat(controlapi): add GUI password setup and login APIs`

### Task 3: `/enter` redirects instead of minting a cookie

**Files:**
- Modify: `internal/controlapi/server.go` `handleEnter` (~652)
- Modify: `internal/controlapi/baseurl_test.go` `TestEnterIssuesSessionWhenBaseURLSet`
- Test: append cases in `session_password_test.go`

New `/enter` behavior (lanMode):

- valid session AND password configured → renew that cookie, 302 `/{path}`
- password not configured → 302 `/setup`, no cookie
- otherwise → 302 `/login`, no cookie

Replace `TestEnterIssuesSessionWhenBaseURLSet` with tests for the three cases.

- [ ] Write tests → fail → implement → pass → commit `fix(controlapi): stop granting session on GET /enter`

### Task 4: auth, logout, change password, rate limit, bootstrap bypass

**Files:**
- Modify: `internal/controlapi/server.go` `auth()`
- Modify: `internal/controlapi/session_password.go`

Tests:

1. After setup, cookie GET `/api/v1/overview` (may 500 if no config — instead hit a tiny `auth()` wrapper like `TestAuthenticatedWebSessionSlidesIdleExpiry`, or GET `/api/v1/session/status` authenticated flag). Prefer `auth()` wrapper + dummy handler for isolation.
2. Delete `gui-password` file → same cookie → 401.
3. Change password → other sessions 401; response Set-Cookie works.
4. Logout clears cookie; subsequent API 401.
5. Six wrong logins: 5×401 then 429; success clears counter.
6. Bearer token still passes `auth()` with password configured and no cookie.
7. `/bootstrap?code=` still sets cookie after password is configured.
8. FN Connect host uses `X-Forwarded-For` for rate-limit key (two IPs do not share the 5-failure budget).

`auth()` addition at start of session check:

```
if s.lanMode() && !s.store.GUIPasswordConfigured() {
  sessionOK = false  // ignore cookie
}
```

Still allow Bearer.

Rate limit: map `string`→`[]time.Time` under `s.mu`, 5 fails / 15 min. Key = host of `RemoteAddr`, except `isFnOSConnectHost(host)` and `X-Forwarded-For` present → leftmost IP.

- [ ] Tests → fail → implement → pass → commit `feat(controlapi): enforce GUI password on browser sessions`

---

## Chunk 3: Web GUI

### Task 5: API client and types

**Files:**
- Modify: `web/src/types.ts`, `web/src/api.ts`, `web/src/api.test.ts`

```ts
export interface SessionStatus {
  schema_version: number
  gui_password_required: boolean
  password_configured: boolean
  authenticated: boolean
}
```

`api.sessionStatus()`, `setupGUIPassword(password)`, `loginGUI(password)`, `logoutGUI()`, `changeGUIPassword(current, next)`.

`controlResponse`: do **not** dispatch `authenticationRequiredEvent` when `path` starts with `/api/v1/session/login` or `/api/v1/session/setup` (and not for `/api/v1/session/status`).

204 handlers: `controlResponse` then return void (like `deleteSource`).

- [ ] Tests in `api.test.ts` for 401-on-login not firing the event
- [ ] Implement → pass → commit `feat(web): add GUI session API client`

### Task 6: Setup and login pages

**Files:**
- Create: `web/src/pages/SetupPage.tsx`, `web/src/pages/LoginPage.tsx`, `web/src/pages/auth-pages.test.tsx`
- Modify: `web/src/styles.css`, `web/src/i18n.ts` (expired LAN copy), `web/src/i18n.en.ts`

Setup: icon + title, password + confirm, min 8, mismatch inline, submit `setupGUIPassword`, `onSuccess()`.
Login: one password field, `loginGUI`, show `invalid_password` / `too_many_attempts`.
No forgot-password link.

`.auth-shell` full-viewport centered card; reuse `.primary`. Light-theme rules next to other `html[data-theme="light"]` blocks.

- [ ] Tests with mocked `api` → fail → implement → pass → commit `feat(web): add setup and login pages`

### Task 7: App gate and sidebar

**Files:**
- Modify: `web/src/App.tsx`, `web/src/App.test.tsx`

On mount (not `isDesktop()`): `api.sessionStatus()`.

- `gui_password_required && !password_configured` → `<SetupPage onSuccess={reload status + refresh overview} />`
- `gui_password_required && !authenticated` → `<LoginPage onSuccess={...} />`
- else existing shell

Remove `redirectToEnterOnce()` auto cookie grant. fnOS 401 → `location.replace` `/login` or `/setup` from last status (or re-fetch status). Desktop keeps session-expired panel; LAN copy/link uses `/login`.

Sidebar when `gui_password_required` (from status, not only edition): change password form + logout in quick settings.

App tests: mock `sessionStatus` `{ gui_password_required: false, ...}` in `beforeEach` so existing tests keep working. Add cases for setup/login gate and that 401 does not replace `/enter`. fnOS sidebar shows 修改面板密码 / 退出登录.

- [ ] Tests → implement → pass → commit `feat(web): gate LAN GUI on panel password`

---

## Chunk 4: Docs

### Task 8: User and agent docs

**Files:**
- `docs/fnos-port/FPK-USER-GUIDE.md` §5
- `docs/fnos-port/DEPLOY.md`
- `docs/fnos-port/DESIGN.md` `/enter` paragraph
- `docker/entrypoint.sh` GUI line
- `docs/agent-wiki/sources/control-api.md`
- `docs/agent-wiki/wiki/concepts/gui-control-plane.md`

State: Host check remains; browser must set then enter password; reset = delete `{store}/gui-password`; token/bootstrap still for ops.

- [ ] Edit docs → commit `docs: describe fnOS GUI password login`

### Task 9: Full verification

```bash
PATH="/Users/renq/.goenv/versions/1.26.1/bin:$PATH" GOCACHE=/private/tmp/opensurge-gocache-126 GOTOOLCHAIN=local make test
cd web && pnpm exec vitest run src/App.test.tsx src/api.test.ts src/pages/auth-pages.test.tsx src/product.test.ts
```

Do not run `make lab-test`. Rebuild `web` + `internal/webui/dist` only if asked to ship an image.

---

## Execution notes

- TDD each task: failing test, then impl, then commit.
- `Origin` on setup/login tests must be `http://192.168.1.20:61767`.
- bcrypt tests are slow-ish; keep cost at `DefaultCost`.
- i18n: add English in `i18n.en.ts`; run `pnpm check:i18n` before web build.
