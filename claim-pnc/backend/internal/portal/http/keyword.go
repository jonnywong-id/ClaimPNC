package portalhttp

import (
	"net/http"
	"strings"

	"claim-pnc/internal/portal"
)

// ActiveKeyword membaca portal aktif dan kata kunci pencarian `cari` sebuah permintaan daftar.
//
// Permintaan tanpa portal dijawab lewat fail dengan portal.ErrNotStated. Kata kunci yang lebih
// panjang dari maxLength dijawab lewat tooLong — 400, bukan daftar kosong: daftar kosong akan
// terbaca sebagai "memang tidak ada datanya", dan pengguna tidak punya cara membedakan
// keduanya (`10-API-STRATEGY.md` §4). Nilai ketiga false bila jawabannya sudah ditulis.
func ActiveKeyword(w http.ResponseWriter, r *http.Request, maxLength int,
	fail func(http.ResponseWriter, *http.Request, error),
	tooLong func(http.ResponseWriter, *http.Request)) (portal.Portal, string, bool) {
	active, exists := ActivePortalFrom(r.Context())
	if !exists {
		fail(w, r, portal.ErrNotStated)
		return portal.Portal{}, "", false
	}
	keyword := strings.TrimSpace(r.URL.Query().Get("cari"))
	if len(keyword) > maxLength {
		tooLong(w, r)
		return portal.Portal{}, "", false
	}
	return active, keyword, true
}
