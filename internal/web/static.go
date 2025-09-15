package web

import (
	"minimalpanel/internal/conf"
	"net/http"
	"path/filepath"
)

func StartPages(mux *http.ServeMux) {
	mux.Handle(
		"/pages/", http.StripPrefix(
			"/pages",
			http.FileServer(
				http.Dir(
					filepath.Join(conf.GetWeb().RootPath, "pages"),
				),
			),
		),
	)
}

func StartAssets(mux *http.ServeMux) {
	mux.Handle(
		"/assets/", http.StripPrefix(
			"/assets",
			http.FileServer(
				http.Dir(
					filepath.Join(conf.GetWeb().RootPath, "assets"),
				),
			),
		),
	)
}
