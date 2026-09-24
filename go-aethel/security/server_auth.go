package security

// STATUS: DIAMANT VGT SUPREME

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	ServerAuthSchemaVersion  = 1
	MinOperatorPasswordRunes = 10
	MaxOperatorPasswordBytes = 256
	DefaultSessionTTL        = 12 * time.Hour
	MaxActiveSessions        = 64
	MaxFailedLoginBuckets    = 1024
	LoginRateLimitWindow     = 1 * time.Minute
	MaxFailedLoginsPerWindow = 5
	LoginLockoutDuration     = 5 * time.Minute
	ServerSessionCookieName  = "aethel_session"
	ServerCSRFHeaderName     = "X-Aethel-CSRF"
	SessionCookieName        = ServerSessionCookieName
	CSRFHeaderName           = ServerCSRFHeaderName

	argon2TimeCost   uint32 = 3
	argon2MemoryKiB  uint32 = 64 * 1024
	argon2Threads    uint8  = 4
	argon2KeyLength  uint32 = 32
	argon2SaltLength int    = 32
)

var (
	ErrAuthPasswordTooShort  = errors.New("Passwort muss mindestens 10 Zeichen lang sein")
	ErrAuthPasswordTooLong   = errors.New("Passwort überschreitet die maximal zulässige Länge")
	ErrAuthPasswordInvalid   = errors.New("Passwort enthält ungültige Steuerzeichen oder besteht nur aus Leerzeichen")
	ErrAuthInvalidCredential = errors.New("Ungültiges Passwort")
	ErrInvalidCredentials    = ErrAuthInvalidCredential
	ErrAuthRateLimited       = errors.New("Zu viele fehlgeschlagene Anmeldeversuche. Bitte warten.")
	ErrAuthNotConfigured     = errors.New("Server-Passwort wurde noch nicht eingerichtet")
	ErrAuthAlreadyConfigured = errors.New("Server-Passwort ist bereits eingerichtet")
)

type sealedServerAuthRecord struct {
	Version      int       `json:"version"`
	Salt         []byte    `json:"salt"`
	PasswordHash []byte    `json:"password_hash"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ServerSession struct {
	TokenHash  [32]byte
	CSRFToken  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSeenAt time.Time
	RemoteIP   string
	UserAgent  string
}

type loginAttemptBucket struct {
	failures    []time.Time
	lockedUntil time.Time
	lastSeen    time.Time
}

// ServerAuthManager enforces operator authentication and session governance
// when Aethel runs in headless/server mode.
type ServerAuthManager struct {
	mu          sync.Mutex
	storePath   string
	requireAuth bool
	record      *sealedServerAuthRecord
	sessions    map[[32]byte]*ServerSession
	failedByIP  map[string]*loginAttemptBucket
	nowFn       func() time.Time
}

// NewServerAuthManager initializes the authentication store and loads any
// existing sealed operator password record.
func NewServerAuthManager(storePath string, requireAuth bool) (*ServerAuthManager, error) {
	mgr := &ServerAuthManager{
		storePath:   storePath,
		requireAuth: requireAuth,
		sessions:    make(map[[32]byte]*ServerSession),
		failedByIP:  make(map[string]*loginAttemptBucket),
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
	if storePath != "" {
		if err := mgr.loadFromDisk(); err != nil {
			return nil, err
		}
	}
	return mgr, nil
}

func (m *ServerAuthManager) loadFromDisk() error {
	raw, err := ReadAuthorityFile(m.storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read server auth authority store: %w", err)
	}
	var rec sealedServerAuthRecord
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rec); err != nil {
		return fmt.Errorf("decode server auth store: %w", err)
	}
	if rec.Version != ServerAuthSchemaVersion || len(rec.Salt) != argon2SaltLength || len(rec.PasswordHash) != int(argon2KeyLength) {
		return errors.New("invalid server auth record invariants")
	}
	m.record = &rec
	return nil
}

// IsAuthRequired returns true when HTTP requests must be authenticated.
func (m *ServerAuthManager) IsAuthRequired() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requireAuth
}

// IsConfigured returns true if an operator password has been provisioned.
func (m *ServerAuthManager) IsConfigured() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.record != nil
}

func validateOperatorPasswordPolicy(password string) error {
	if !utf8.ValidString(password) || strings.ContainsRune(password, '\x00') {
		return ErrAuthPasswordInvalid
	}
	if strings.TrimSpace(password) == "" {
		return ErrAuthPasswordInvalid
	}
	if len(password) > MaxOperatorPasswordBytes {
		return ErrAuthPasswordTooLong
	}
	if utf8.RuneCountInString(password) < MinOperatorPasswordRunes {
		return ErrAuthPasswordTooShort
	}
	return nil
}

// SetPassword hashes the operator password with Argon2id, seals the record
// with AES-256-GCM on disk, and invalidates all active sessions.
func (m *ServerAuthManager) SetPassword(password string) error {
	if err := validateOperatorPasswordPolicy(password); err != nil {
		return err
	}
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argon2TimeCost, argon2MemoryKiB, argon2Threads, argon2KeyLength)

	m.mu.Lock()
	defer m.mu.Unlock()

	rec := sealedServerAuthRecord{
		Version:      ServerAuthSchemaVersion,
		Salt:         salt,
		PasswordHash: hash,
		UpdatedAt:    m.nowFn(),
	}
	if m.storePath != "" {
		if dir := filepath.Dir(m.storePath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return fmt.Errorf("create server auth directory: %w", err)
			}
		}
		encoded, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal server auth record: %w", err)
		}
		if err := WriteSealedFile(m.storePath, encoded); err != nil {
			return fmt.Errorf("persist server auth record: %w", err)
		}
	}
	m.record = &rec
	m.sessions = make(map[[32]byte]*ServerSession)
	return nil
}

// VerifyPassword checks the candidate password against the sealed Argon2id hash
// in constant time.
func (m *ServerAuthManager) VerifyPassword(password string) bool {
	if len(password) == 0 || len(password) > MaxOperatorPasswordBytes {
		return false
	}
	m.mu.Lock()
	rec := m.record
	m.mu.Unlock()
	if rec == nil || len(rec.Salt) != argon2SaltLength || len(rec.PasswordHash) != int(argon2KeyLength) {
		return false
	}
	candidate := argon2.IDKey([]byte(password), rec.Salt, argon2TimeCost, argon2MemoryKiB, argon2Threads, argon2KeyLength)
	return subtle.ConstantTimeCompare(candidate, rec.PasswordHash) == 1
}

// Authenticate verifies the operator password, enforces per-IP rate limits, and
// issues a new cryptographically random session token and CSRF token.
func (m *ServerAuthManager) Authenticate(password, remoteAddr, userAgent string) (sessionToken string, csrfToken string, err error) {
	m.mu.Lock()
	if m.record == nil {
		m.mu.Unlock()
		return "", "", ErrAuthNotConfigured
	}
	if err := m.checkRateLimitLocked(remoteAddr); err != nil {
		m.mu.Unlock()
		return "", "", err
	}
	recCopy := *m.record
	m.mu.Unlock()

	var valid bool
	if len(password) > 0 && len(password) <= MaxOperatorPasswordBytes {
		candidate := argon2.IDKey([]byte(password), recCopy.Salt, argon2TimeCost, argon2MemoryKiB, argon2Threads, argon2KeyLength)
		valid = subtle.ConstantTimeCompare(candidate, recCopy.PasswordHash) == 1
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkRateLimitLocked(remoteAddr); err != nil {
		return "", "", err
	}
	if !valid {
		m.recordFailureLocked(remoteAddr)
		return "", "", ErrAuthInvalidCredential
	}

	delete(m.failedByIP, normalizeRemoteIP(remoteAddr))

	rawToken, err := generateRandomHex(32)
	if err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	csrf, err := generateRandomHex(32)
	if err != nil {
		return "", "", fmt.Errorf("generate csrf token: %w", err)
	}

	now := m.nowFn()
	expires := now.Add(DefaultSessionTTL)
	tokenHash := sha256.Sum256([]byte(rawToken))

	m.pruneSessionsLocked(now)
	m.sessions[tokenHash] = &ServerSession{
		TokenHash:  tokenHash,
		CSRFToken:  csrf,
		CreatedAt:  now,
		ExpiresAt:  expires,
		LastSeenAt: now,
		RemoteIP:   normalizeRemoteIP(remoteAddr),
		UserAgent:  truncateUserAgent(userAgent),
	}
	return rawToken, csrf, nil
}

// ValidateRequest checks whether an incoming HTTP request carries a valid
// session cookie or Bearer token, and enforces CSRF token matching on mutating
// requests.
func (m *ServerAuthManager) ValidateRequest(r *http.Request) (*ServerSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.requireAuth {
		return &ServerSession{}, true
	}
	if m.record == nil || r == nil {
		return nil, false
	}

	token, fromCookie := extractRequestToken(r)
	if token == "" {
		return nil, false
	}
	now := m.nowFn()
	candidateHash := sha256.Sum256([]byte(token))

	sess, exists := m.sessions[candidateHash]
	if !exists || sess == nil {
		return nil, false
	}
	if !now.Before(sess.ExpiresAt) {
		delete(m.sessions, candidateHash)
		return nil, false
	}
	if subtle.ConstantTimeCompare(sess.TokenHash[:], candidateHash[:]) != 1 {
		return nil, false
	}

	// Mutating requests authenticated via cookie must present the matching X-Aethel-CSRF header.
	if fromCookie && isMutatingMethod(r.Method) {
		csrfHeader := strings.TrimSpace(r.Header.Get(ServerCSRFHeaderName))
		if csrfHeader == "" || subtle.ConstantTimeCompare([]byte(csrfHeader), []byte(sess.CSRFToken)) != 1 {
			return nil, false
		}
	}

	sess.LastSeenAt = now
	copySess := *sess
	return &copySess, true
}

// SessionStatus inspects the request without requiring CSRF headers so GET /v1/auth/status
// can report current authentication state and return the active CSRF token.
func (m *ServerAuthManager) SessionStatus(r *http.Request) (authRequired, configured, authenticated bool, csrfToken string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	authRequired = m.requireAuth
	configured = m.record != nil
	if !m.requireAuth {
		return false, configured, true, ""
	}
	if m.record == nil || r == nil {
		return true, false, false, ""
	}
	token, _ := extractRequestToken(r)
	if token == "" {
		return true, true, false, ""
	}
	now := m.nowFn()
	candidateHash := sha256.Sum256([]byte(token))
	sess, exists := m.sessions[candidateHash]
	if !exists || sess == nil || !now.Before(sess.ExpiresAt) {
		if exists {
			delete(m.sessions, candidateHash)
		}
		return true, true, false, ""
	}
	sess.LastSeenAt = now
	return true, true, true, sess.CSRFToken
}

// WriteSessionCookie writes the hardened HttpOnly SameSite=Strict session cookie.
func (m *ServerAuthManager) WriteSessionCookie(w http.ResponseWriter, r *http.Request, sessionToken string) {
	isHTTPS := r != nil && (r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"))
	http.SetCookie(w, &http.Cookie{
		Name:     ServerSessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   int(DefaultSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
	})
}

// Logout revokes the session associated with r and clears the session cookie.
func (m *ServerAuthManager) Logout(w http.ResponseWriter, r *http.Request) {
	if r != nil {
		if token, _ := extractRequestToken(r); token != "" {
			hash := sha256.Sum256([]byte(token))
			m.mu.Lock()
			delete(m.sessions, hash)
			m.mu.Unlock()
		}
	}
	if w != nil {
		isHTTPS := r != nil && (r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"))
		http.SetCookie(w, &http.Cookie{
			Name:     ServerSessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   isHTTPS,
			SameSite: http.SameSiteStrictMode,
		})
	}
}

func (m *ServerAuthManager) checkRateLimitLocked(remoteAddr string) error {
	ip := normalizeRemoteIP(remoteAddr)
	now := m.nowFn()
	bucket, exists := m.failedByIP[ip]
	if !exists {
		return nil
	}
	if now.Before(bucket.lockedUntil) {
		return ErrAuthRateLimited
	}
	cutoff := now.Add(-LoginRateLimitWindow)
	filtered := bucket.failures[:0]
	for _, ts := range bucket.failures {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	bucket.failures = filtered
	if len(bucket.failures) == 0 && !now.Before(bucket.lockedUntil) {
		delete(m.failedByIP, ip)
	}
	return nil
}

func (m *ServerAuthManager) recordFailureLocked(remoteAddr string) {
	ip := normalizeRemoteIP(remoteAddr)
	now := m.nowFn()
	m.pruneAttemptBucketsLocked(now)

	bucket, exists := m.failedByIP[ip]
	if !exists {
		bucket = &loginAttemptBucket{}
		m.failedByIP[ip] = bucket
	}
	cutoff := now.Add(-LoginRateLimitWindow)
	filtered := bucket.failures[:0]
	for _, ts := range bucket.failures {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	filtered = append(filtered, now)
	bucket.failures = filtered
	bucket.lastSeen = now
	if len(bucket.failures) >= MaxFailedLoginsPerWindow {
		bucket.lockedUntil = now.Add(LoginLockoutDuration)
	}
}

func (m *ServerAuthManager) pruneAttemptBucketsLocked(now time.Time) {
	if len(m.failedByIP) < MaxFailedLoginBuckets {
		return
	}
	var oldestIP string
	var oldestTime time.Time
	cutoff := now.Add(-LoginRateLimitWindow)
	for ip, b := range m.failedByIP {
		if now.After(b.lockedUntil) && b.lastSeen.Before(cutoff) {
			delete(m.failedByIP, ip)
			continue
		}
		if oldestIP == "" || b.lastSeen.Before(oldestTime) {
			oldestIP = ip
			oldestTime = b.lastSeen
		}
	}
	if len(m.failedByIP) >= MaxFailedLoginBuckets && oldestIP != "" {
		delete(m.failedByIP, oldestIP)
	}
}

func (m *ServerAuthManager) pruneSessionsLocked(now time.Time) {
	for k, s := range m.sessions {
		if !now.Before(s.ExpiresAt) {
			delete(m.sessions, k)
		}
	}
	for len(m.sessions) >= MaxActiveSessions {
		var oldestKey [32]byte
		var oldestTime time.Time
		first := true
		for k, s := range m.sessions {
			if first || s.LastSeenAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = s.LastSeenAt
				first = false
			}
		}
		if first {
			break
		}
		delete(m.sessions, oldestKey)
	}
}

func extractRequestToken(r *http.Request) (token string, fromCookie bool) {
	if cookie, err := r.Cookie(ServerSessionCookieName); err == nil {
		val := strings.TrimSpace(cookie.Value)
		if len(val) == 64 {
			return val, true
		}
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		val := strings.TrimSpace(authHeader[7:])
		if len(val) == 64 {
			return val, false
		}
	}
	return "", false
}

func isMutatingMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}

func normalizeRemoteIP(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func truncateUserAgent(ua string) string {
	ua = strings.TrimSpace(ua)
	if len(ua) > 256 {
		return ua[:256]
	}
	return ua
}

func generateRandomHex(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
