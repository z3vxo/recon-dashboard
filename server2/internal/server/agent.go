package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/z3vxo/vantage/internal/database"
)

func agentMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.AgentSecret), nil
		})
		if err != nil || !token.Valid {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AgentToken_Handler(w http.ResponseWriter, r *http.Request) {
	var data LoginData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	ip := realIP(r)

	if cfg == nil || data.Username != cfg.Username || !CheckPassword(data.Password) {
		slog.Warn("agent token failed", "user", data.Username, "ip", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": data.Username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(cfg.AgentSecret))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
		return
	}

	slog.Info("agent token issued", "user", data.Username, "ip", ip)
	writeJSON(w, http.StatusOK, map[string]string{"token": signed})
}

func AgentData_Handler(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")

	qType := r.URL.Query().Get("type")
	if qType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type is required (summary or full)"})
		return
	}
	if qType != "summary" && qType != "full" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be summary or full"})
		return
	}

	limit := 50
	if v := r.URL.Query().Get("maxlimit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	status := r.URL.Query().Get("status")

	q := database.AgentQuery{
		Type:   qType,
		Limit:  limit,
		Offset: offset,
		Status: status,
	}

	data, err := database.AgentData(domain, q)
	if err != nil {
		slog.Error("agent data query failed", "domain", domain, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, data)
}
