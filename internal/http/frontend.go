package http

import "net/http"

func StartFrontend() {
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", http.StripPrefix("/", fs))
}
