package controlapi

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	loginRateLimit  = 5
	loginRateWindow = 15 * time.Minute
)

type sessionStatusResponse struct {
	SchemaVersion       int  `json:"schema_version"`
	GUIPasswordRequired bool `json:"gui_password_required"`
	PasswordConfigured  bool `json:"password_configured"`
	Authenticated       bool `json:"authenticated"`
}

type guiPasswordRequest struct {
	Password string `json:"password"`
}

type guiPasswordChangeRequest struct {
	Current string `json:"current"`
	Next    string `json:"next"`
}

func (s *Server) requireLANMutation(w http.ResponseWriter, r *http.Request) bool {
	if !s.lanMode() {
		writeError(w, http.StatusUnauthorized, "authentication_required", "open the Web GUI using an authenticated launcher link")
		return false
	}
	if !s.originAllowed(r.Header.Get("Origin")) {
		writeError(w, http.StatusForbidden, "origin_rejected", "mutation origin is not allowed")
		return false
	}
	return true
}

func (s *Server) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	configured := s.lanMode() && s.store.GUIPasswordConfigured()
	authenticated := s.webSessionID(r) != "" && (!s.lanMode() || configured)
	writeJSON(w, http.StatusOK, sessionStatusResponse{
		SchemaVersion:       SchemaVersion,
		GUIPasswordRequired: s.lanMode(),
		PasswordConfigured:  configured,
		Authenticated:       authenticated,
	})
}

func (s *Server) handleSessionSetup(w http.ResponseWriter, r *http.Request) {
	if !s.requireLANMutation(w, r) {
		return
	}
	var request guiPasswordRequest
	if err := decodeJSON(r, &request, 16<<10); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	err := s.store.CreateGUIPassword(request.Password)
	if errors.Is(err, errGUIPasswordInvalid) {
		writeError(w, http.StatusBadRequest, "invalid_request", "password must be 8 to 128 characters")
		return
	}
	if errors.Is(err, errGUIPasswordConfigured) {
		writeError(w, http.StatusConflict, "password_already_configured", "panel password is already set")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password_write_failed", "could not save panel password")
		return
	}
	s.grantWebSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSessionLogin(w http.ResponseWriter, r *http.Request) {
	if !s.requireLANMutation(w, r) {
		return
	}
	key := s.loginClientKey(r)
	if s.loginRateLimited(key) {
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", "too many failed login attempts, try again later")
		return
	}
	if !s.store.GUIPasswordConfigured() {
		writeError(w, http.StatusConflict, "password_not_configured", "set a panel password first")
		return
	}
	var request guiPasswordRequest
	if err := decodeJSON(r, &request, 16<<10); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ok, err := s.store.PasswordMatches(request.Password)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusConflict, "password_not_configured", "set a panel password first")
			return
		}
		writeError(w, http.StatusInternalServerError, "password_unusable", "panel password file is unusable")
		return
	}
	if !ok {
		s.recordLoginFailure(key)
		writeError(w, http.StatusUnauthorized, "invalid_password", "password is incorrect")
		return
	}
	s.clearLoginFailures(key)
	s.grantWebSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSessionLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("opensurge_session"); err == nil && cookie.Value != "" {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}
	clearWebSessionCookie(w, requestIsHTTPS(r))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSessionPassword(w http.ResponseWriter, r *http.Request) {
	if !s.lanMode() {
		writeError(w, http.StatusUnauthorized, "authentication_required", "open the Web GUI using an authenticated launcher link")
		return
	}
	if !s.store.GUIPasswordConfigured() {
		writeError(w, http.StatusConflict, "password_not_configured", "set a panel password first")
		return
	}
	var request guiPasswordChangeRequest
	if err := decodeJSON(r, &request, 16<<10); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ok, err := s.store.PasswordMatches(request.Current)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusConflict, "password_not_configured", "set a panel password first")
			return
		}
		writeError(w, http.StatusInternalServerError, "password_unusable", "panel password file is unusable")
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_password", "password is incorrect")
		return
	}
	if err := s.store.ReplaceGUIPassword(request.Next); err != nil {
		if errors.Is(err, errGUIPasswordInvalid) {
			writeError(w, http.StatusBadRequest, "invalid_request", "password must be 8 to 128 characters")
			return
		}
		writeError(w, http.StatusInternalServerError, "password_write_failed", "could not save panel password")
		return
	}
	s.mu.Lock()
	s.sessions = map[string]time.Time{}
	s.mu.Unlock()
	s.grantWebSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func clearWebSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "opensurge_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) loginClientKey(r *http.Request) string {
	host := r.Host
	if parsed, _, err := net.SplitHostPort(r.Host); err == nil {
		host = parsed
	}
	if isFnOSConnectHost(host) {
		if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
			ip, _, _ := strings.Cut(fwd, ",")
			if ip = strings.TrimSpace(ip); ip != "" {
				return ip
			}
		}
	}
	addr := r.RemoteAddr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}

func (s *Server) loginRateLimited(key string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := now.Add(-loginRateWindow)
	fails := s.loginFailures[key]
	kept := fails[:0]
	for _, at := range fails {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) == 0 {
		delete(s.loginFailures, key)
	} else {
		s.loginFailures[key] = kept
	}
	return len(kept) >= loginRateLimit
}

func (s *Server) recordLoginFailure(key string) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loginFailures[key] = append(s.loginFailures[key], now)
}

func (s *Server) clearLoginFailures(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.loginFailures, key)
}
