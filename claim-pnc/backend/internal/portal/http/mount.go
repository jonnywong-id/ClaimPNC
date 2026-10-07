package portalhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Route adalah satu rute GET.
type Route struct {
	Path    string
	Handler http.HandlerFunc
}

// MountGets mendaftarkan rute GET modul di balik pemeriksaan portal aktif. Permintaan tanpa
// portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan (`R-20`).
func MountGets(r chi.Router, deps ActivePortalDeps, routes ...Route) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(ActivePortal(deps))
		for _, route := range routes {
			perPortal.Get(route.Path, route.Handler)
		}
	})
}
