package api

import (
	"context"
	"net/http"
)

// This middleware is a placeholder to be replaced with an actual OIDC/JWT implementation.
// It currently allows all requests to pass through.
func NewAuthMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In a real implementation, you would:
			// 1. Get the token from the request header.
			// 2. Validate the token and its claims.
			// 3. Put the user's information on the request context.

			// For now, we'll just let all requests through.
			// This can be used for local development and testing.

			// If you want to put a placeholder user on the context, you can do this:
			ctx := context.WithValue(r.Context(), "user", "placeholder-user")

			// Call the next handler in the chain.
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
