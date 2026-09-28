package registrasihttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Mount mendaftarkan rute modul registrasi.
//
// Seluruh rutenya TERLINDUNGI: tidak satu pun bagian alur klaim boleh dijangkau tanpa
// sesi. Pemasangannya dilakukan pemanggil di dalam grup yang sudah memakai middleware
// autentikasi — sama seperti modul portal.
//
// # Yang BELUM ada di sini, dan konsekuensinya
//
// Pemeriksaan kewenangan per menu (`ADR-0023`, `TKT-F3-005`) belum dikerjakan karena
// tabel peran `TKT-F3-004` masih terhalang artefak. Akibatnya setiap pengguna yang punya
// sesi dapat memanggil seluruh rute di bawah ini. Ini utang yang disadari, bukan
// kelalaian: `ADR-0023` menuntut pemeriksaan di SETIAP endpoint, dan tempatnya sudah
// disiapkan — satu middleware di grup ini.
func Mount(r chi.Router, h *Handler) {
	r.Route("/registrasi", func(sub chi.Router) {
		// Definisi alur dibaca layar untuk menggambar jalur tahap. Ia tidak memuat data
		// klaim mana pun.
		sub.Get("/alur", h.Flow)

		sub.Get("/inbox", h.Inbox)

		sub.Post("/klaim", h.Start)
		sub.Get("/klaim/{klaimID}", func(w http.ResponseWriter, r *http.Request) {
			h.ViewClaim(w, r, chi.URLParam(r, "klaimID"))
		})

		// Tahap Input Register punya jalurnya sendiri karena ia membawa isian dan
		// gerbang validasi. Tahap lain ditutup lewat /tugas/{id}/selesai.
		sub.Post("/register", h.SaveRegister)

		// Tombol Save: menyimpan isian Input Register tanpa menutup tahapnya.
		sub.Post("/register/simpan", h.SaveDraft)
		sub.Post("/estimasi", h.CompleteEstimate)
		sub.Post("/estimasi/simpan", h.SaveEstimate)
		sub.Get("/mata-uang", h.Currencies)
		sub.Get("/klaim/{klaimID}/pilihan-item", func(w http.ResponseWriter, r *http.Request) {
			h.ItemOptions(w, r, chi.URLParam(r, "klaimID"))
		})

		// Tab pendamping tahap Input Estimasi. Ketiganya hanya membaca tabel warisan.
		sub.Get("/klaim/{klaimID}/survey", func(w http.ResponseWriter, r *http.Request) {
			h.Surveys(w, r, chi.URLParam(r, "klaimID"))
		})
		sub.Get("/klaim/{klaimID}/dokumen", func(w http.ResponseWriter, r *http.Request) {
			h.Documents(w, r, chi.URLParam(r, "klaimID"))
		})
		// Tombol Download Claim Face Sheet: membentuk PDF, mencatat revisi, mengunci estimasi.
		sub.Post("/klaim/{klaimID}/cfs", func(w http.ResponseWriter, r *http.Request) {
			h.FaceSheet(w, r, chi.URLParam(r, "klaimID"))
		})
		// Tombol Print PLA: menerbitkan PLA koasuransi dan mengunduh dokumennya.
		sub.Post("/klaim/{klaimID}/pla/daftar", func(w http.ResponseWriter, r *http.Request) {
			h.ListPLA(w, r, chi.URLParam(r, "klaimID"))
		})
		sub.Post("/klaim/{klaimID}/pla/catatan", func(w http.ResponseWriter, r *http.Request) {
			h.SavePLANotes(w, r, chi.URLParam(r, "klaimID"))
		})
		sub.Post("/klaim/{klaimID}/pla", func(w http.ResponseWriter, r *http.Request) {
			h.PLA(w, r, chi.URLParam(r, "klaimID"))
		})
		// Tombol Tambah pada grid Adjustment (tab Adjustment & Akseptasi, layar InputSurveyor).
		sub.Post("/klaim/{klaimID}/adjustment/hitung", func(w http.ResponseWriter, r *http.Request) {
			h.PreviewSettlement(w, r, chi.URLParam(r, "klaimID"))
		})
		sub.Post("/klaim/{klaimID}/adjustment", func(w http.ResponseWriter, r *http.Request) {
			h.AddSettlement(w, r, chi.URLParam(r, "klaimID"))
		})
		// Transfer Komite pada baris Adjustment, dan putusan anggota komite.
		sub.Post("/klaim/{klaimID}/adjustment/komite", func(w http.ResponseWriter, r *http.Request) {
			h.TransferCommittee(w, r, chi.URLParam(r, "klaimID"))
		})
		sub.Get("/komite", h.PendingCommittees)
		sub.Get("/komite/{komiteID}", func(w http.ResponseWriter, r *http.Request) {
			h.Committee(w, r, chi.URLParam(r, "komiteID"))
		})
		sub.Post("/komite/{komiteID}/putusan", func(w http.ResponseWriter, r *http.Request) {
			h.DecideCommittee(w, r, chi.URLParam(r, "komiteID"))
		})
		// Penerima Klaim (InputReceiver): isian No Rekening membaca Master Rekening, tombol
		// Simpan menyimpan penerima baru atau yang diubah.
		sub.Get("/rekening/{nomor}", func(w http.ResponseWriter, r *http.Request) {
			h.FindAccount(w, r, chi.URLParam(r, "nomor"))
		})
		sub.Post("/klaim/{klaimID}/penerima", func(w http.ResponseWriter, r *http.Request) {
			h.SaveReceiver(w, r, chi.URLParam(r, "klaimID"))
		})
		sub.Get("/klaim/{klaimID}/progres", func(w http.ResponseWriter, r *http.Request) {
			h.Progress(w, r, chi.URLParam(r, "klaimID"))
		})

		// Daftar pilihan wilayah kejadian bertingkat: negara, provinsi, kota,
		// kabupaten, kelurahan. Hanya membaca master.
		sub.Get("/wilayah/{tingkat}", func(w http.ResponseWriter, r *http.Request) {
			h.AreaOptions(w, r, chi.URLParam(r, "tingkat"))
		})

		sub.Post("/tugas/{tugasID}/ambil", func(w http.ResponseWriter, r *http.Request) {
			h.ClaimTask(w, r, chi.URLParam(r, "tugasID"))
		})
		sub.Post("/tugas/{tugasID}/selesai", func(w http.ResponseWriter, r *http.Request) {
			h.CompleteTask(w, r, chi.URLParam(r, "tugasID"))
		})
	})
}
