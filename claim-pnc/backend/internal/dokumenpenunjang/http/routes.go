package dokumenpenunjanghttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute dokumen penunjang.
//
// # Jalurnya bersarang di bawah KLAIM, bukan di bawah proteksi
//
//	GET  /klaim/{nomor}/dokumen-penunjang
//	POST /klaim/{nomor}/dokumen-penunjang
//
// Work Owner menetapkan dokumennya menempel *"Ke klaim, seperti Pega"*, dan bentuk datanya
// menuntut hal yang sama: `T_CLAIM_OPENPROTECTION` tidak punya satu pun kolom dokumen,
// sedangkan `GENERAL.T_STORAGE_IMAGE.NO_CLAIM` **NOT NULL**.
//
// Akibatnya rute ini melayani DUA layar dengan satu jalur — Input Req Protection dan Inbox
// Accept Open Protection keduanya memanggil jalur yang sama untuk klaim yang sama. Itu
// disengaja: dokumen yang diunggah dari satu layar harus terlihat dari layar lainnya, sebab
// keduanya membicarakan klaim yang sama.
//
// Bersarangnya dua tingkat, sesuai batas `09-API-STRATEGY.md` §2.
//
// # Tidak ada rute HAPUS, dan itu keputusan
//
// Pega tidak punya jalur hapus pada layar ini, dan `D-66` menetapkan penghapusan dinyatakan
// lewat penanda. `T_STORAGE_IMAGE` sudah punya `DELETE_DATE` untuk itu — tetapi SIAPA yang
// boleh menghapus dokumen klaim belum ditetapkan, dan `D-59` menghapus pemisahan tugas
// sehingga jawabannya tidak dapat disimpulkan sendiri.
//
// Rute yang tidak didaftarkan dijawab chi dengan 405, sehingga penambahannya kelak menjadi
// keputusan sadar — bukan sesuatu yang lolos review.
//
// # Kedua rutenya menuntut portal aktif
//
// Dokumen sebuah klaim dicari lewat basis data portalnya sendiri (`ADR-0030`). Permintaan
// tanpa header `X-Portal` DITOLAK middleware — tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// # Kewenangan belum berbutir
//
// Siapa yang boleh mengunggah dokumen sebuah klaim belum ditetapkan terpisah; yang berlaku
// adalah kewenangan layar yang memanggilnya. Ini sejalan `D-59` — satuan izin adalah MENU —
// dan sekaligus batasnya: pemanggil yang mengetahui jalur ini dapat memakainya tanpa membuka
// layar mana pun. Penegakan berbutir untuk 31 modul lain adalah `TKT-F3-005`.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/klaim/{nomor}/dokumen-penunjang", func(dokumen chi.Router) {
		dokumen.Use(portalhttp.ActivePortal(portalDeps))

		dokumen.Get("/", h.List)
		dokumen.Post("/", h.Upload)
	})
}
