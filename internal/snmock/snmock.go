// Package snmock provides an in-memory stand-in for the Standard Notes API.
//
// It implements just enough of the API for gosn-v2 to sign in and sync: the
// login-params, login, session refresh and items endpoints. Items are held as
// the opaque encrypted blobs the client uploads, and the account is seeded with
// a real SN|ItemsKey encrypted with the master key derived from the mock
// credentials. Only the server is fake: sign-in, key derivation, encryption,
// decryption and the whole sync round trip run as they do against a real
// account, so tests exercise the same code paths that production does.
package snmock

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jonhadfield/gosn-v2/auth"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/crypto"
	"github.com/jonhadfield/gosn-v2/items"
	snsession "github.com/jonhadfield/gosn-v2/session"
)

const (
	// Email is the account the mock server accepts.
	Email = "sn-dotfiles@example.com"
	// Password is the password the mock server accepts.
	Password = "sn-dotfiles-test-password"

	accessToken  = "mock-access-token"
	refreshToken = "mock-refresh-token"
)

// Server is a mock Standard Notes API served over HTTP.
type Server struct {
	// URL is the base address to point a session at.
	URL string

	httpServer *httptest.Server

	masterKey      string
	serverPassword string
	keyParams      auth.KeyParams

	// itemsKeyUUID is the uuid of the SN|ItemsKey the account is seeded with.
	itemsKeyUUID string

	mu sync.Mutex
	// revision increases with every write, and doubles as the sync token so
	// that a client only retrieves what it has not already seen.
	revision int64
	stored   map[string]*storedItem
	syncs    int
}

type storedItem struct {
	item     items.EncryptedItem
	revision int64
}

// New starts a mock server with an empty account holding a single items key.
// Close must be called when the caller is done with it.
func New() (*Server, error) {
	nonceBytes, err := crypto.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("snmock: generating password nonce: %w", err)
	}

	nonce := hex.EncodeToString(nonceBytes)

	masterKey, serverPassword, err := crypto.GenerateMasterKeyAndServerPassword004(crypto.GenerateEncryptedPasswordInput{
		UserPassword:  Password,
		Identifier:    Email,
		PasswordNonce: nonce,
	})
	if err != nil {
		return nil, fmt.Errorf("snmock: deriving master key: %w", err)
	}

	s := &Server{
		masterKey:      masterKey,
		serverPassword: serverPassword,
		keyParams: auth.KeyParams{
			Created:     strconv.FormatInt(time.Now().UTC().UnixMilli(), 10),
			Identifier:  Email,
			Origination: "registration",
			PwNonce:     nonce,
			Version:     "004",
		},
		stored: make(map[string]*storedItem),
	}

	if err = s.seedItemsKey(); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(common.AuthParamsPath, s.handleAuthParams)
	mux.HandleFunc(common.SignInPath, s.handleSignIn)
	mux.HandleFunc(common.AuthRefreshPath, s.handleRefresh)
	mux.HandleFunc(common.SyncPath, s.handleSync)

	s.httpServer = httptest.NewServer(mux)
	s.URL = s.httpServer.URL

	return s, nil
}

// Close shuts the server down.
func (s *Server) Close() {
	s.httpServer.Close()
}

// seedItemsKey creates the SN|ItemsKey the account's items are encrypted with.
// A real account gets one at registration; gosn-v2 never creates one during a
// sync, so without it nothing can be encrypted.
func (s *Server) seedItemsKey() error {
	key, err := crypto.GenerateItemKey(64)
	if err != nil {
		return fmt.Errorf("snmock: generating items key: %w", err)
	}

	now := time.Now().UTC()

	sik := snsession.SessionItemsKey{
		UUID:               items.GenUUID(),
		ItemsKey:           key,
		Version:            "004",
		Default:            true,
		CreatedAt:          now.Format(common.TimeLayout),
		CreatedAtTimestamp: now.UnixMicro(),
	}

	encrypted, err := items.EncryptItemsKey(sik, &snsession.Session{
		MasterKey: s.masterKey,
		KeyParams: s.keyParams,
	}, true)
	if err != nil {
		return fmt.Errorf("snmock: encrypting items key: %w", err)
	}

	s.itemsKeyUUID = sik.UUID
	s.put(encrypted)

	return nil
}

// ItemsKeyUUID returns the uuid of the account's items key.
func (s *Server) ItemsKeyUUID() string {
	return s.itemsKeyUUID
}

// Items returns a copy of everything the account currently holds, including
// items that have been deleted.
func (s *Server) Items() items.EncryptedItems {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(items.EncryptedItems, 0, len(s.stored))
	for _, si := range s.stored {
		out = append(out, si.item)
	}

	return out
}

// LiveItemsOfType returns the undeleted items of the given content type.
func (s *Server) LiveItemsOfType(contentType string) items.EncryptedItems {
	var out items.EncryptedItems

	for _, i := range s.Items() {
		if i.ContentType == contentType && !i.Deleted {
			out = append(out, i)
		}
	}

	return out
}

// Syncs returns the number of sync requests the server has handled.
func (s *Server) Syncs() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.syncs
}

// put stores an item at a new revision. The caller must not hold the lock,
// except during construction.
func (s *Server) put(item items.EncryptedItem) items.EncryptedItem {
	s.revision++

	now := time.Now().UTC()
	item.UpdatedAt = now.Format(common.TimeLayout)
	item.UpdatedAtTimestamp = now.UnixMicro()

	if item.CreatedAt == "" {
		item.CreatedAt = item.UpdatedAt
	}

	if item.CreatedAtTimestamp == 0 {
		item.CreatedAtTimestamp = item.UpdatedAtTimestamp
	}

	s.stored[item.UUID] = &storedItem{item: item, revision: s.revision}

	return item
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (s *Server) handleAuthParams(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if !emailMatches(req.Email) {
		// A real server returns params for unknown accounts too, to avoid
		// disclosing which addresses are registered.
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{
				"identifier": req.Email,
				"pw_nonce":   "unknown-account-nonce",
				"version":    "004",
			},
		})

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"identifier": Email,
			"pw_nonce":   s.keyParams.PwNonce,
			"version":    s.keyParams.Version,
		},
	})
}

func (s *Server) handleSignIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if !emailMatches(req.Email) || req.Password != s.serverPassword {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"data": map[string]any{
				"error": map[string]any{
					"tag":     "invalid-auth",
					"message": "Invalid email or password",
				},
			},
		})

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"meta": map[string]any{},
		"data": map[string]any{
			"session":    s.sessionPayload(),
			"key_params": s.keyParams,
			"user": map[string]any{
				"uuid":            "mock-user-uuid",
				"email":           Email,
				"protocolVersion": "004",
			},
		},
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"meta": map[string]any{},
		"data": map[string]any{"session": s.sessionPayload()},
	})
}

func (s *Server) sessionPayload() map[string]any {
	now := time.Now().UTC()

	return map[string]any{
		"access_token":       accessToken,
		"refresh_token":      refreshToken,
		"access_expiration":  now.Add(24 * time.Hour).UnixMilli(),
		"refresh_expiration": now.Add(30 * 24 * time.Hour).UnixMilli(),
		"readonly_access":    false,
	}
}

type syncRequest struct {
	Items       items.EncryptedItems `json:"items"`
	SyncToken   string               `json:"sync_token"`
	CursorToken string               `json:"cursor_token"`
	Limit       int                  `json:"limit"`
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+accessToken {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid access token"})
		return
	}

	var req syncRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncs++

	seen := parseSyncToken(req.SyncToken)

	// Apply the client's writes first, so its own changes are reported back as
	// saved rather than retrieved.
	saved := make(items.EncryptedItems, 0, len(req.Items))
	pushed := make(map[string]bool, len(req.Items))

	for _, in := range req.Items {
		if in.UUID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "item without uuid"})
			return
		}

		if existing, ok := s.stored[in.UUID]; ok && in.CreatedAt == "" {
			in.CreatedAt = existing.item.CreatedAt
			in.CreatedAtTimestamp = existing.item.CreatedAtTimestamp
		}

		if in.Deleted {
			// A deleted item keeps its identity but loses its payload, which
			// is what the real API returns.
			in.Content = ""
			in.EncItemKey = ""
			in.ItemsKeyID = ""
		}

		pushed[in.UUID] = true
		saved = append(saved, s.put(in))
	}

	// Anything written since the client's last sync token, other than what it
	// just pushed, is new to it.
	retrieved := items.EncryptedItems{}

	for _, si := range s.stored {
		if si.revision > seen && !pushed[si.item.UUID] {
			retrieved = append(retrieved, si.item)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"retrieved_items": retrieved,
			"saved_items":     saved,
			"unsaved":         items.EncryptedItems{},
			"conflicts":       []any{},
			"sync_token":      strconv.FormatInt(s.revision, 10),
			"cursor_token":    "",
		},
	})
}

func parseSyncToken(token string) int64 {
	if token == "" {
		return 0
	}

	seen, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return 0
	}

	return seen
}

// emailMatches compares an email from a request against the mock account,
// allowing for the path escaping the client applies.
func emailMatches(in string) bool {
	if in == Email {
		return true
	}

	unescaped, err := url.PathUnescape(in)

	return err == nil && unescaped == Email
}

// Session signs in to the mock server and returns a cache session for appName
// with its cache database path set, ready to hand to the sn-dotfiles package.
func (s *Server) Session(appName string, debug bool) (*cache.Session, error) {
	return newCacheSession(Email, Password, s.URL, appName, debug)
}

// UseEnv points SN_EMAIL, SN_PASSWORD and SN_SERVER at the mock server, for
// code that takes its credentials from the environment as the CLI does.
func (s *Server) UseEnv() error {
	for k, v := range map[string]string{
		"SN_EMAIL":    Email,
		"SN_PASSWORD": Password,
		"SN_SERVER":   s.URL,
	} {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("snmock: setting %s: %w", k, err)
		}
	}

	return nil
}

// NewSession returns a cache session for tests to run against, along with the
// mock server backing it. It uses the account in SN_EMAIL/SN_PASSWORD/SN_SERVER
// when one is configured, in which case the returned server is nil. The cleanup
// function shuts the mock server down, and does nothing for a real account.
func NewSession(appName string, debug bool) (sess *cache.Session, srv *Server, cleanup func(), err error) {
	email, password, server := os.Getenv("SN_EMAIL"), os.Getenv("SN_PASSWORD"), os.Getenv("SN_SERVER")

	if email != "" && password != "" {
		sess, err = newCacheSession(email, password, server, appName, debug)

		return sess, nil, func() {}, err
	}

	srv, err = New()
	if err != nil {
		return nil, nil, func() {}, err
	}

	sess, err = srv.Session(appName, debug)
	if err != nil {
		srv.Close()

		return nil, nil, func() {}, err
	}

	return sess, srv, srv.Close, nil
}

func newCacheSession(email, password, server, appName string, debug bool) (*cache.Session, error) {
	in, err := auth.CliSignIn(email, password, server, debug)
	if err != nil {
		return nil, fmt.Errorf("snmock: signing in: %w", err)
	}

	if server == "" {
		server = common.APIServer
	}

	// CliSignIn does not carry the server address through to the session, so
	// set it from the address we signed in to.
	sess := &cache.Session{
		Session: &snsession.Session{
			Debug:             debug,
			Server:            server,
			Token:             in.Token,
			MasterKey:         in.MasterKey,
			KeyParams:         in.KeyParams,
			AccessToken:       in.AccessToken,
			AccessExpiration:  in.AccessExpiration,
			RefreshToken:      in.RefreshToken,
			RefreshExpiration: in.RefreshExpiration,
		},
	}

	path, err := cache.GenCacheDBPath(*sess, "", appName)
	if err != nil {
		return nil, fmt.Errorf("snmock: generating cache db path: %w", err)
	}

	sess.CacheDBPath = path

	return sess, nil
}
