package main

import (
	"fmt"
	"net/http"
	"strings"

	portalhttp "claim-pnc/internal/portal/http"
)

// onlyPrimaryPortal menolak permintaan yang portal aktifnya bukan portal utama.
//
// # Kenapa penjaga ini ada
//
// Modul Registrasi Klaim terikat pada SATU koneksi basis data: seam penyimpanannya
// menerima `*sql.DB`, bukan pemilih repo per portal seperti modul inbox. Tanpa penjaga
// ini, petugas yang sedang membuka portal Syariah akan menulis klaimnya ke basis data
// entitas lain.
//
// Kegagalan seperti itu TIDAK menampakkan diri sebagai galat — layarnya tampil normal,
// angkanya masuk akal, dan yang salah hanya *milik siapa* data itu. `R-20` menyebutnya
// kebocoran data antar badan hukum, dan menandainya berdampak sangat tinggi.
//
// # Kenapa menolak, bukan mengalihkan ke portal utama
//
// Mengalihkan diam-diam berarti petugas mengira sedang bekerja di entitasnya sendiri
// padahal tidak. Penolakan yang menyebutkan sebabnya membuat batas itu terlihat, dan
// itulah yang `ADR-0030` tuntut: portal yang bukan haknya tidak dilayani, bukan diganti.
//
// Penjaga ini hilang dengan sendirinya begitu modulnya menerima pemilih repo per portal.
func onlyPrimaryPortal(
	primary string,
	writeError func(w http.ResponseWriter, r *http.Request, err error),
) func(http.Handler) http.Handler {
	want := strings.ToUpper(strings.TrimSpace(primary))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			active, existing := portalhttp.ActivePortalFrom(r.Context())
			if !existing {
				// Tidak seharusnya terjadi: middleware portal berjalan lebih dulu.
				// Menolak tetap lebih benar daripada melanjutkan tanpa entitas yang jelas.
				writeError(w, r, fmt.Errorf("portal aktif tidak terbaca pada permintaan registrasi"))
				return
			}
			if strings.ToUpper(strings.TrimSpace(active.Alias)) != want {
				writeError(w, r, fmt.Errorf(
					"modul Registrasi Klaim belum melayani portal %q; untuk sementara hanya portal %s",
					active.Alias, want))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
