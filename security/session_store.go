package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

// Session represents a user session.
type Session struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Values map[string]any `json:"values"`
	Dirty  bool           `json:"-"`
}

// NewSession creates a new session.
func NewSession(id string) *Session {
	return &Session{
		ID:     id,
		Values: make(map[string]any),
	}
}

// Get retrieves a value from the session.
func (s *Session) Get(key string) (any, bool) {
	val, ok := s.Values[key]
	return val, ok
}

// Set stores a value in the session and marks it dirty.
func (s *Session) Set(key string, val any) {
	s.Values[key] = val
	s.Dirty = true
}

// Delete removes a value from the session and marks it dirty.
func (s *Session) Delete(key string) {
	delete(s.Values, key)
	s.Dirty = true
}

// Clear clears all values from the session and marks it dirty.
func (s *Session) Clear() {
	s.Values = make(map[string]any)
	s.Dirty = true
}

// SessionStore defines the interface for session storage and retrieval.
type SessionStore interface {
	Get(r *http.Request, name string) (*Session, error)
	Save(r *http.Request, w http.ResponseWriter, session *Session) error
}

// CookieSessionStore stores session data inside signed cookies.
type CookieSessionStore struct {
	secret []byte
}

// NewCookieSessionStore creates a new CookieSessionStore.
func NewCookieSessionStore(secret string) *CookieSessionStore {
	return &CookieSessionStore{
		secret: []byte(secret),
	}
}

func (s *CookieSessionStore) sign(value string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(value))
	sig := mac.Sum(nil)
	return value + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (s *CookieSessionStore) verify(cookieValue string) (string, error) {
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid cookie format")
	}
	payload := parts[0]
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sig, expectedSig) {
		return "", errors.New("signature mismatch")
	}
	return payload, nil
}

// Get retrieves the session from the cookie.
func (s *CookieSessionStore) Get(r *http.Request, name string) (*Session, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		sess := NewSession(generateSessionID())
		sess.Name = name
		return sess, nil
	}

	payload, err := s.verify(cookie.Value)
	if err != nil {
		sess := NewSession(generateSessionID())
		sess.Name = name
		return sess, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		sess := NewSession(generateSessionID())
		sess.Name = name
		return sess, nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		sess := NewSession(generateSessionID())
		sess.Name = name
		return sess, nil
	}
	session.Name = name
	return &session, nil
}

// Save signs and encodes the session, writing it back as a cookie.
func (s *CookieSessionStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	if !session.Dirty {
		return nil
	}

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	payload := base64.RawURLEncoding.EncodeToString(data)
	signed := s.sign(payload)

	cookie := &http.Cookie{
		Name:     session.Name,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
	}
	http.SetCookie(w, cookie)
	session.Dirty = false
	return nil
}

// InMemorySessionStore stores sessions in-memory.
type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string
}

// NewInMemorySessionStore creates a new InMemorySessionStore.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]string),
	}
}

// Get retrieves the session from the in-memory map.
func (s *InMemorySessionStore) Get(r *http.Request, name string) (*Session, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		sess := NewSession(generateSessionID())
		sess.Name = name
		return sess, nil
	}

	s.mu.RLock()
	jsonVal, exists := s.sessions[cookie.Value]
	s.mu.RUnlock()

	if !exists {
		sess := NewSession(generateSessionID())
		sess.Name = name
		return sess, nil
	}

	var values map[string]any
	if err := json.Unmarshal([]byte(jsonVal), &values); err != nil {
		values = make(map[string]any)
	}

	sess := &Session{
		ID:     cookie.Value,
		Name:   name,
		Values: values,
	}
	return sess, nil
}

// Save stores the session values in memory and sets a session ID cookie.
func (s *InMemorySessionStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	if !session.Dirty {
		return nil
	}

	data, err := json.Marshal(session.Values)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.sessions[session.ID] = string(data)
	s.mu.Unlock()

	cookie := &http.Cookie{
		Name:     session.Name,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
	}
	http.SetCookie(w, cookie)
	session.Dirty = false
	return nil
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
