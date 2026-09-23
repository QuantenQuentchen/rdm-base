# The Vanguard Awards — frontend (TypeScript, cookie-session auth)

Vue 3 + TypeScript, `<script setup>` throughout. No dependencies beyond Vue —
the router, stores, HTTP client and mock backend are all local modules.

This version matches a backend that authenticates via an **HttpOnly session
cookie**, not a Bearer token: the client never holds, reads, or attaches a
credential itself. `GET /login` is a real browser navigation, not a fetch —
your Go server owns the entire Discord round trip and sets the cookie itself.

## Dropping this into a fresh project

Same as before: copy `src/` and `index.html` over the generated ones, copy
`env.d.ts` into the project root, then **delete** `src/assets/` (its
`main.css`/`base.css` fight the dark theme) and any scaffolded
`src/components/*`, `src/router/`, `src/stores/counter.ts` you didn't ask for.

## How auth actually flows now

1. Not authenticated → landing on `/` (any hash, or none) resolves to
   `/awards`, which requires auth → `App.vue` renders `SignInView`. This is
   the whole "`/` is the login page when logged out" behavior — no special
   case needed, it falls out of the route guard.
2. "Continue with Discord" is `window.location.assign(...)` to your backend's
   `/login` — a full top-level navigation, not a fetch. Discord needs to
   redirect the actual browser window, and your backend needs to set a
   cookie on its own origin as part of that chain.
3. Your backend does the entire OAuth exchange server-side (as it already
   does), sets the `session_token` cookie, and **should redirect the browser
   back to the frontend's root** when it's done — see the backend note below,
   this needs a one-line change from what you've shown me.
4. Frontend loads at `/`, calls `GET /auth/session` (needs to be added — see
   below) with `credentials: 'include'`. Cookie's already there from step 3;
   the browser attaches it automatically on this cross-origin (different
   port) request because it's same-*site* even though it's a different
   *origin*.
5. Session store gets `{ user }` back, renders the app.
6. Sign out: `POST /logout`. Your backend already clears the cookie; the
   frontend just drops its local copy of the user.

There is **no `/auth/callback` route on the frontend anymore** — nothing
here parses an OAuth `code`/`state`, because your backend never sends the
browser back with one. If your handler currently ends with
`w.WriteHeader(http.StatusOK)` after setting the cookie, the person is stuck
looking at a raw response instead of your app; it needs to redirect instead
(see below).

## Debug mode — unchanged mechanism, no token anymore

Same three flags (`bypassAuth`, `useMockApi`, `forceAdmin`), same resolution
order (localStorage override → `VITE_` env var → on-in-dev default), same
Debug pill in the top bar. The only functional difference: `bypassAuth` used
to fabricate a token; now it just sets local session state directly, and if
`useMockApi` is also on, it tells the mock server it's signed in too (they're
independent flags tracking independent "am I logged in" state, since the
mock server has no way to see what the store did).

```
VITE_API_BASE_URL=http://localhost:8080     # full origin — see below
VITE_DEBUG_BYPASS_AUTH=false
VITE_DEBUG_MOCK_API=false
VITE_DEBUG_FORCE_ADMIN=false
```

`VITE_API_BASE_URL` now needs to be the backend's **full origin**, not a
same-origin proxy path — the frontend dev server and the Go backend are on
different ports, so every call is genuinely cross-origin.

## Backend changes this needs (yours to make)

I can't edit your Go project directly, but here's exactly what the frontend
now expects, matched against the code you showed me.

**1. Redirect instead of `200 OK` after the Discord callback.** Right now
`discordCallback` ends with `w.WriteHeader(http.StatusOK)`. Since Discord's
redirect is a top-level browser navigation, that response renders as a blank
page showing nothing useful. Redirect to the frontend instead:

```go
frontendURL := mustEnv("FRONTEND_URL") // e.g. "http://localhost:5173"

// replace the final w.WriteHeader(http.StatusOK) with:
http.Redirect(w, r, frontendURL, http.StatusFound)
```

**2. A new `GET /auth/session` handler.** This doesn't exist yet — it's what
the frontend calls on every page load to find out who's signed in. Reuses
`checkAndGetIDAuth` and whatever builds `UserReply` for the admin endpoints:

```go
type sessionReply struct {
	User UserReply `json:"user"`
}

func (a *AuthStruct) getSession(w http.ResponseWriter, r *http.Request) {
	id, err := a.checkAndGetIDAuth(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	usr, err := a.db.getUserByID(id)
	if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	if usr == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	reply, err := a.generateUserReply(r.Context(), usr.DiscordID)
	if err != nil {
		http.Error(w, "failed to build user reply", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessionReply{User: reply})
}

// mux.HandleFunc("GET /auth/session", auth.getSession)
```

**3. CORS.** Every one of these calls is cross-origin (different port), so
without CORS headers the browser will send the request and the cookie, but
refuse to let `fetch()` read the response at all — including preflight
`OPTIONS` for the `PUT`/`POST` calls, since a JSON body isn't a "simple"
request. This is mandatory, not optional, for any of this to work:

```go
func withCORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// http.ListenAndServe(":8080", withCORS("http://localhost:5173", mux))
```

Never use `*` for `Access-Control-Allow-Origin` here — browsers reject the
wildcard combined with credentials, and you want exactly this either way.

**4. The `session_token` cookie's `MaxAge: 300` is 5 minutes** — copy-pasted
from the login-attempt cookie above it, where that's correct. For the actual
session you almost certainly want this much longer, and ideally matching
whatever TTL `createSession` actually gives the session in your DB:

```go
MaxAge: 60 * 60 * 24 * 14, // 2 weeks — match your session's real TTL
```

**Already fine, no action needed:** `hmac.Equal` for the state check, PKCE
verifier, `HttpOnly`/`SameSite=Lax` on both cookies, the `discord_login_attempt`
short TTL. One thing from earlier still worth a look when you get to it: the
`log.Fatal` inside `LoginAttemptStore.RegisterAttempt` kills the whole
process on an attemptID collision instead of just failing that request —
low-probability, but worth swapping for a returned error at some point.

## Endpoint paths — some of these are guesses

`src/config.ts`'s `endpoints` object is the single source of truth for every
path the frontend calls. I only have confirmed mount paths for `/login` and
the three `/api/admin/...` routes; `logout`, `session`, `adminCheck`,
`categories`, `mySuggestions`, and the per-category suggestions path are my
best read from earlier in this conversation, not your actual
`mux.HandleFunc` lines. Each is commented `// confirmed` or `// ASSUMED` in
that file — check the assumed ones against your router and fix in that one
place if they're wrong.

## Ids: numbers or strings, your choice

`CategoryUpsertRequest{ ID *int64, ... }` in your code implies category
(and likely suggestion) ids are `int64`, not strings — different from
`UserReply.ID`, which is explicitly `string`. Rather than guess wrong and
force a rewrite, every id in `src/types.ts` is typed:

```ts
export type EntityId = string | number
```

Internally, anything keyed by an id (the suggestions-by-category map, the
saving/error trackers in the awards store) always converts to a string first
via a `keyOf()`/`String()` call, so it works identically whichever type your
JSON actually sends. You shouldn't need to touch this even if I guessed
wrong — just don't be surprised to see `id: string | number` instead of a
single concrete type.

## Admin panel

Three tabs' worth of real functionality now, not a stub:

- **Suggestions** — read-only table via `GET /api/admin/viewSuggestions`,
  showing category, text, submitter (avatar + name from the rich
  `NominationSuggestionRich` shape), and last-updated time.
- **Categories** — full CRUD against your three endpoints. A table lists
  every category (including locked ones); Edit opens an inline form
  pre-filled from that row, New opens it empty; both submit through
  `POST /api/admin/upsert/Categories` with `{ id: null | EntityId, category }`;
  Delete confirms via `window.confirm` then calls
  `POST /api/admin/delete/Categories`.

Styling is deliberately plain and dense — grays, a blue accent, tabular
numbers — distinct from the awards page's ceremony look, so it reads as a
tool rather than part of the show.

`checkIsAdmin()` (`src/api/admin.ts`) now actually calls
`GET /auth/admin` and trusts the result — I dropped the earlier stub gate
since you've genuinely implemented `isAdmin` server-side. It still never
throws: a failed or unreachable check just means "not an admin," since the
backend is the real gate and this only decides what the UI shows.

## Verified

`vue-tsc --noEmit` (strict, clean), `vite build`, and jsdom-mounted tests
covering: default dev boot (`bypassAuth` + `useMockApi` both on), the actual
sign-in-button path against the mock (start signed out → sign in → load →
sign out), full category create/edit/delete round-trip through
`adminApi`, and that a plain-text `http.Error` body (which is what every one
of your Go handlers actually sends) surfaces as the real message instead of
a generic fallback — that was a genuine bug in the previous version's error
parsing, fixed as part of this pass.

## Layout

```
index.html
env.d.ts
src/
  main.ts        config.ts        types.ts        router.ts        App.vue
  styles/theme.css
  api/
    client.ts    auth.ts    awards.ts    admin.ts
  mock/
    server.ts    fixtures.ts
  stores/
    session.ts   awards.ts
  layout/
    AppShell.vue  TopBar.vue  SideNav.vue
  views/
    AwardsView.vue  AdminView.vue  SignInView.vue
  components/
    CategoryPanel.vue  SuggestionEditor.vue
    UiButton.vue  StatusNote.vue  DebugMenu.vue
```

Desktop-only, 960px floor. `AuthCallbackView.vue` is gone — delete it from
any earlier copy you have lying around, it's dead code now.
