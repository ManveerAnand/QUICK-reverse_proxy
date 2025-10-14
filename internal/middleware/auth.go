package middleware

import (
    "net/http"
    "strings"
)

// AuthMiddleware is a middleware that checks for authentication tokens in the request headers.
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check for the Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Validate the token (this is a placeholder for actual validation logic)
        token := strings.TrimPrefix(authHeader, "Bearer ")
        if !isValidToken(token) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // If the token is valid, proceed to the next handler
        next.ServeHTTP(w, r)
    })
}

// isValidToken is a placeholder function for token validation logic.
func isValidToken(token string) bool {
    // Implement actual token validation logic here
    return token == "valid-token" // Example: replace with real validation
}