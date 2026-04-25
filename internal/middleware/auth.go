package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

func RequireUID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Header.Get("X-Slot-UID")
		if uid == "" || !isValidUID(uid) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing or invalid X-Slot-UID header"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUID(r *http.Request) string {
	return r.Header.Get("X-Slot-UID")
}

func isValidUID(uid string) bool {
	// Basic UUID v4 format check
	parts := strings.Split(uid, "-")
	if len(parts) != 5 {
		return false
	}
	lengths := []int{8, 4, 4, 4, 12}
	for i, l := range lengths {
		if len(parts[i]) != l {
			return false
		}
	}
	return true
}
