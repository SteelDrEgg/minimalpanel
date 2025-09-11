package netx

import "net/http"

// StartFrontend Frontend binding
func StartFrontend() {
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", http.StripPrefix("/", fs))
}
