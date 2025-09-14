package auth

import (
	"github.com/zishang520/socket.io/servers/socket/v3"
	"net/http"
	"strings"
)

// RequireAuth is a middleware that checks authentication for protected routes
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, authenticated := IsAuthenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/pages/login.html", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// RequireAuthSocketIO is a middleware that checks authentication for protected Socket.IO endpoints
func RequireAuthSocketIO(client *socket.Socket, next func(*socket.ExtendedError)) {
	// Safely get cookies from headers
	cookieHeader := client.Handshake().Headers["Cookie"]
	if cookieHeader == nil {
		next(socket.NewExtendedError("Unauthorized", "No cookies provided"))
		return
	}

	cookieSlice, ok := cookieHeader.([]string)
	if !ok || len(cookieSlice) == 0 {
		next(socket.NewExtendedError("Unauthorized", "Invalid cookie format"))
		return
	}

	cookies := cookieSlice[0]
	cookie := func() string {
		parts := strings.Split(cookies, ";")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, CookieName+"=") {
				return strings.TrimPrefix(p, CookieName+"=")
			}
		}
		return ""
	}()

	if _, ok := ValidateSession(cookie); ok {
		next(nil)
	} else {
		next(socket.NewExtendedError("Unauthorized", "Invalid session"))
	}
}
