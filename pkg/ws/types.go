package ws

import "net/http"

// --- AUTHENTICATION FUNCTION TYPE ---
// Used to extract and validate user credentials on websocket connect.
type AuthFunc func(r *http.Request) (string, error)