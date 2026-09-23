package pelaporanklaimhttp

import "github.com/go-chi/chi/v5"

// Mount mendaftarkan seluruh rute modul Pelaporan Klaim ke router yang diberikan.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di cmd/claimpnc
// ia dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga middleware sesi.
//
// # Kenapa "pelaporan-klaim" dan bukan "receive-document"
//
// Jalur URL adalah kontrak yang dibaca orang, dan yang dibaca orang di sistem lama pun
// bukan "receive document": menunya berbunyi "Inbox Laporan Klaim"
// (`Navigation/pyCaseWorkerNavigation-Navigation.xml:19864`). Nama kelas Pega
// `Work-ReceiveDocument` adalah nama internal, dan `D-19` menetapkan alias internal tidak
// dibawa ke sistem baru.
//
// # Kenapa aksi bisnis menjadi sub-sumber daya
//
// `POST /{nomor}/transfer` dan `POST /{nomor}/klaim`, bukan `PATCH /{nomor}` yang menyetel
// satu field. `10-API-STRATEGY.md` §2 menetapkannya, dan alasannya nyata di sini: kedua
// aksi punya invarian sendiri — transfer tidak boleh terjadi dua kali, penautan klaim
// tidak boleh terjadi dua kali — dan pembaruan field generik akan melewatkan keduanya
// sekaligus tidak meninggalkan jejak bahwa aksinya pernah dilakukan.
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya. Sistem lama membatasi layar
// ini pada tujuh peran lewat When rule `IsReceivePNC` — Administrators, CaseManager,
// PncAdmin, PncManagerAdmin, PncReceive, PNCReportClaimInternal, dan
// PNCReportClaimEksternal — dan yang terakhir menandakan PELAPOR LUAR juga membukanya.
//
// Penegakan "apakah peran pemanggil memiliki menu ini" adalah `TKT-F3-005`, yang
// bergantung pada tabel peran `TKT-F3-004` — dan tabel itu dapat dibangun tetapi belum
// dapat diisi, karena penugasan operator ke peran tidak ada di basis data maupun di export
// (`11-SECURITY.md` §3.1). Keadaan ini sama dengan seluruh rute lain hari ini.
//
// Taruhannya di sini lebih tinggi daripada di layar master: laporan memuat nama
// tertanggung, nomor polis, kronologi kejadian, dan alamat surel. Dicatat terbuka di
// docs/keputusan-implementasi.md, bukan disembunyikan.
func Mount(r chi.Router, h *Handler) {
	r.Route("/pelaporan-klaim", func(reports chi.Router) {
		reports.Get("/", h.List)
		reports.Post("/", h.Record)

		reports.Get("/{nomor}", h.Get)
		reports.Put("/{nomor}", h.Update)

		// Tidak ada DELETE, dan ketiadaannya disengaja: `ADR-0012` menetapkan penghapusan
		// lunak menyeluruh, dan laporan yang sudah tertaut klaim dirujuk klaimnya lewat
		// `ClaimData.RCV_ID` — rujukan yang dipakai 41 rule Pega. Rute yang tidak
		// didaftarkan dijawab chi dengan 405, sehingga penambahannya kelak menjadi
		// keputusan sadar, bukan kelalaian yang lolos review.

		reports.Post("/{nomor}/transfer", h.Transfer)
		reports.Post("/{nomor}/klaim", h.LinkClaim)
	})
}
