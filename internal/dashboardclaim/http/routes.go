package dashboardclaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Dashboard Claim.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di cmd/claimpnc ia
// dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga middleware sesi.
//
// # Kenapa jalurnya "dashboard-claim"
//
// Itu judul layar rujukannya sendiri — `Harness/DashboardClaim_Harness-Harness.xml`, yang
// terdaftar di `Navigation/pyCaseWorkerNavigation-Navigation.xml` sebagai butir menu
// ber-`pyAction=showHarness`.
//
// # Tiga rute, dan TIDAK ADA yang menulis
//
//	GET /penyaring    isi dropdown dan daftar kartu — bentuk layar, bukan data
//	GET /ringkasan    keempat angka kartu
//	GET /{tile}       telusur satu kartu
//
// Layar ini memang tidak punya satu pun aksi tulis: di sistem lama ia hanya membaca, dan
// seluruh tabel yang dibacanya masih ditulis Pega selama masa paralel (`P-1`). Rute yang
// tidak didaftarkan dijawab chi dengan 405, sehingga penambahan rute tulis kelak menjadi
// keputusan sadar — bukan sesuatu yang lolos review.
//
// # Urutan pendaftaran mengikat
//
// `/penyaring` dan `/ringkasan` didaftarkan SEBELUM `/{tile}`. Keduanya cocok dengan pola
// `/{tile}`, dan chi memilih rute statis lebih dulu — tetapi menuliskannya dalam urutan ini
// membuat maksudnya terbaca, dan melindungi bila pola rutenya kelak berubah.
//
// Keduanya juga BUKAN nama tile yang sah, sehingga permintaan `/dashboard-claim/penyaring`
// tidak pernah dapat terbaca sebagai telusur tile bernama "penyaring".
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya — keadaan yang sama dengan
// seluruh rute lain hari ini. Penegakan "apakah peran pemanggil memiliki menu ini" adalah
// `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004`.
//
// Taruhannya di sini perlu dinyatakan: layar ini memperlihatkan RINGKASAN seluruh pekerjaan
// satu badan hukum — bukan satu baris klaim, melainkan gambaran menyeluruhnya. Sampai
// `TKT-F3-005` ada, siapa pun yang punya sesi dan menyebut portal dapat membacanya.
// Dicatat terbuka di docs/keputusan-implementasi.md.
//
// # Seluruh rutenya menuntut portal aktif
//
// Klaim ada di basis data setiap entitas (`ADR-0030`), sehingga setiap permintaan harus
// menyebut entitas mana yang dibacanya lewat header `X-Portal`. Permintaan yang tidak
// menyebutkannya DITOLAK oleh middleware — tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// Pada layar INI akibatnya khas: yang salah bukan satu baris melainkan seluruh angka
// ringkasannya, dan angka yang salah entitas tidak terlihat keliru dengan cara lain apa pun.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/dashboard-claim", func(dashboard chi.Router) {
		dashboard.Use(portalhttp.ActivePortal(portalDeps))

		dashboard.Get("/penyaring", h.Metadata)
		dashboard.Get("/ringkasan", h.Summary)

		// Tab kedua layar lama — "Inbox Tampungan PIC". Nama jalurnya statis dan BUKAN nama
		// tile yang sah, sehingga ia tidak pernah terbaca sebagai telusur tile.
		dashboard.Get("/tampungan", h.Holding)

		// Didaftarkan SEBELUM `/{tile}`: chi mencocokkan jalur statis lebih dulu, tetapi
		// menaruhnya di sini membuat urutannya terbaca tanpa perlu tahu aturan itu.
		dashboard.Get("/pic-teknik", h.TechnicalPIC)

		// Unduhan dipisahkan menjadi jalurnya sendiri, bukan parameter format pada daftar.
		// Keduanya berbeda sifat: yang satu dipaginasi dan dibaca layar, yang lain mengalir
		// sampai habis dan diterima sebagai berkas.
		// Rincian satu klaim — isi popup yang terbuka saat nomor klaim diklik.
		//
		// Didaftarkan SEBELUM "/{tile}", karena chi memilih rute menurut urutan dan
		// "/klaim/..." akan tertangkap pola satu segmen itu bila dipasang sesudahnya.
		dashboard.Get("/klaim/{klaim_id}", h.ClaimDetail)

		dashboard.Get("/{tile}", h.List)
		dashboard.Get("/{tile}/unduh", h.ExportTile)

		// SATU-SATUNYA rute yang menulis.
		//
		// KOREKSI (2026-10-08): keterangan sebelumnya menyebut "SATU KOLOM pada tabel kerja
		// Pega". Itu kurang lengkap — benar untuk skema DATAPEGA, tetapi satu transaksinya
		// menulis TIGA tabel:
		//
		//	DATAPEGA.PC_ASM_FW_GCNMFW_WORK   USERTEKNIS_1      PIC Teknik klaim
		//	POOLDATA.MST_USER_TEKNIK         COUNTER_QUOTA     pencacah beban PIC baru
		//	POOLDATA.PEGA_DASHBOARDPNC       PIC               tabel ringkasan dashboard
		//
		// Ketiganya ditulis `PNC_ReassignPNCTeknik` di layar lama, jadi tidak satu pun
		// tambahan kita. Yang penting untuk `P-1`: antrean tugas Pega
		// (DATAPEGA.PC_ASSIGN_WORKLIST) TIDAK disentuh sama sekali.
		//
		// Ketiganya punya probe di `claimpnc -periksa`. Probe ketiga baru ditambahkan pada
		// tanggal koreksi ini; sebelumnya pemeriksaan dapat hijau sementara tombolnya gagal.
		//
		// Rinciannya di pindahpic.sql.
		dashboard.Post("/transfer", h.Transfer)
	})
}
