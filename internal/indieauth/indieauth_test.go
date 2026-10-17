package indieauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gorilla/sessions"
)

const testIdentity = "https://identity.example/"

func newTestAuth(t *testing.T, verificationStatus int) (*IndieAuth, *atomic.Int32) {
	t.Helper()
	requests := new(atomic.Int32)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("unexpected verification request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		for key, expected := range map[string]string{
			"code": "test-code", "client_id": "https://app.example",
			"redirect_uri": "https://app.example" + DefaultRedirectPath,
		} {
			if r.Form.Get(key) != expected {
				t.Errorf("verification %s = %q, want %q", key, r.Form.Get(key), expected)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(verificationStatus)
		if err := json.NewEncoder(w).Encode(verifyResp{Me: testIdentity}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(endpoint.Close)
	store := sessions.NewCookieStore([]byte("test-only-cookie-authentication-key"))
	ia, err := New(store, testIdentity, endpoint.URL)
	if err != nil {
		t.Fatal(err)
	}
	return ia, requests
}

func forwardedRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "http://internal"+target, nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "app.example")
	r.Header.Set("X-Forwarded-Uri", target)
	return r
}

func beginLogin(t *testing.T, ia *IndieAuth, target string) string {
	t.Helper()
	w := httptest.NewRecorder()
	ia.Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("unauthenticated request reached the protected handler")
	})).ServeHTTP(w, forwardedRequest(target))
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("login status = %d", w.Code)
	}
	location, err := w.Result().Location()
	if err != nil {
		t.Fatal(err)
	}
	if location.Scheme+"://"+location.Host != ia.authEndpoint {
		t.Fatalf("login redirected to an unexpected endpoint: %s", location)
	}
	query := location.Query()
	if query.Get("me") != testIdentity || query.Get("client_id") != "https://app.example" ||
		query.Get("redirect_uri") != "https://app.example"+DefaultRedirectPath || query.Get("state") == "" {
		t.Fatalf("invalid authorization parameters: %v", query)
	}
	return query.Get("state")
}

func callback(ia *IndieAuth, state string) *httptest.ResponseRecorder {
	query := url.Values{"me": {testIdentity}, "code": {"test-code"}, "state": {state}}
	w := httptest.NewRecorder()
	ia.RedirectHandler(w, forwardedRequest(DefaultRedirectPath+"?"+query.Encode()))
	return w
}

func requireInvalidState(t *testing.T, ia *IndieAuth, state string) {
	t.Helper()
	w := httptest.NewRecorder()
	defer func() {
		// The existing handler rejects invalid states with a panic. Preserve that
		// behavior here rather than introducing a new HTTP error policy in tests.
		if recovered := recover(); recovered == nil || fmt.Sprint(recovered) != "invalid state" {
			t.Errorf("invalid state rejection = %v", recovered)
		}
		if len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
			t.Error("invalid state created a session or redirected the user")
		}
	}()
	query := url.Values{"me": {testIdentity}, "code": {"test-code"}, "state": {state}}
	ia.RedirectHandler(w, forwardedRequest(DefaultRedirectPath+"?"+query.Encode()))
}

func TestLoginPreservesRedirectCreatesSessionAndConsumesState(t *testing.T) {
	ia, requests := newTestAuth(t, http.StatusOK)
	target := "/private/report?filter=a%2Fb&sort=desc"
	state := beginLogin(t, ia, target)
	w := callback(ia, state)
	if w.Code != http.StatusTemporaryRedirect || w.Header().Get("Location") != "https://app.example"+target {
		t.Fatalf("original redirect lost: status=%d location=%q", w.Code, w.Header().Get("Location"))
	}
	if requests.Load() != 1 {
		t.Fatalf("verification requests = %d", requests.Load())
	}
	authenticated := forwardedRequest(target)
	for _, cookie := range w.Result().Cookies() {
		authenticated.AddCookie(cookie)
	}
	protected := httptest.NewRecorder()
	ia.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(protected, authenticated)
	if protected.Code != http.StatusNoContent {
		t.Fatalf("session did not unlock protected handler: %d", protected.Code)
	}
	// A replay must fail before a second authorization request or new session.
	requireInvalidState(t, ia, state)
	if requests.Load() != 1 {
		t.Fatal("replayed state reached the authorization endpoint")
	}
}

func TestUnknownAndMissingLoginStatesAreRejected(t *testing.T) {
	for _, state := range []string{"", "unknown-state"} {
		t.Run(state, func(t *testing.T) {
			ia, requests := newTestAuth(t, http.StatusOK)
			requireInvalidState(t, ia, state)
			if requests.Load() != 0 {
				t.Fatal("invalid state reached the authorization endpoint")
			}
		})
	}
}

func TestConcurrentLoginStatesPreserveTheirOwnRedirects(t *testing.T) {
	ia, _ := newTestAuth(t, http.StatusOK)
	first := beginLogin(t, ia, "/first")
	second := beginLogin(t, ia, "/second")
	if first == second {
		t.Fatal("independent login attempts reused a state")
	}
	for _, attempt := range []struct{ state, path string }{{second, "/second"}, {first, "/first"}} {
		if w := callback(ia, attempt.state); w.Header().Get("Location") != "https://app.example"+attempt.path {
			t.Fatalf("login state redirected to %q", w.Header().Get("Location"))
		}
	}
}

func TestForbiddenVerificationDoesNotCreateSessionAndConsumesState(t *testing.T) {
	ia, requests := newTestAuth(t, http.StatusForbidden)
	state := beginLogin(t, ia, "/private")
	w := callback(ia, state)
	if w.Code != http.StatusForbidden || len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
		t.Fatal("failed verification authenticated or redirected the user")
	}
	requireInvalidState(t, ia, state)
	if requests.Load() != 1 {
		t.Fatal("failed login state could be replayed")
	}
}

func TestLoginStateCacheEvictsOldestPendingAttempt(t *testing.T) {
	ia, requests := newTestAuth(t, http.StatusOK)
	states := make([]string, 65)
	for i := range states {
		states[i] = beginLogin(t, ia, fmt.Sprintf("/pending/%d", i))
	}
	// The 64-entry cache bounds pending state storage; an evicted state must
	// never reach verification or create an authenticated session.
	requireInvalidState(t, ia, states[0])
	if requests.Load() != 0 {
		t.Fatal("evicted state reached the authorization endpoint")
	}
	for _, i := range []int{1, 64} {
		w := callback(ia, states[i])
		if w.Code != http.StatusTemporaryRedirect || w.Header().Get("Location") != fmt.Sprintf("https://app.example/pending/%d", i) {
			t.Fatalf("retained state %d did not preserve its redirect", i)
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("verification requests = %d, want 2", requests.Load())
	}
}

func TestNonOKVerificationDoesNotAuthenticate(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			ia, requests := newTestAuth(t, status)
			state := beginLogin(t, ia, "/private")
			query := url.Values{"me": {testIdentity}, "code": {"test-code"}, "state": {state}}
			w := httptest.NewRecorder()
			var failure any
			func() {
				defer func() { failure = recover() }()
				ia.RedirectHandler(w, forwardedRequest(DefaultRedirectPath+"?"+query.Encode()))
			}()
			if failure == nil || len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
				t.Error("failed verification was accepted instead of failing closed")
			}
			requireInvalidState(t, ia, state)
			if requests.Load() != 1 {
				t.Fatal("failed verification state could be replayed")
			}
		})
	}
}
