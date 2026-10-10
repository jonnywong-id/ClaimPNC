package inboxcompliancehttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Compliance.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Antrean kepatuhan satu badan hukum
// bukan antrean badan hukum lain, dan baris yang muncul di sana memuat nomor polis serta
// nama tertanggung.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan antrean satu badan hukum kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Rute keterangan layar (`/tab`) ikut di balik pemeriksaan itu meski isinya sama di seluruh
// entitas. Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat alias
// portal, dan mengembalikan alias portal utama untuk permintaan yang tidak menyebut portal
// akan membuat layar mengira ia sudah berada di portal yang benar.
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran, dan di modul ini
// ketiadaannya lebih berat akibatnya daripada di modul master mana pun:
//
// Antrean ini adalah antrean PEMERIKSAAN KEPATUHAN. Di sistem lama ia hanya dapat dibuka
// peran `GCNMFW:PncComplience`, dan barisnya memuat nomor polis beserta nama tertanggung
// setiap klaim yang sedang diperiksa. Tanpa pemeriksaan peran, setiap pengguna yang sudah
// masuk dapat membukanya.
//
// Penegakannya adalah `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004` — dan
// tabel itu dapat dibangun tetapi belum dapat diisi, karena penugasan operator ke peran
// tidak ada di basis data maupun di export (`11-SECURITY.md` §3.1). Ia dicatat di sini
// sebagai utang yang disadari, bukan sebagai hal yang terlewat.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`,
// `/api/inbox-admin`). `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan
// memperkenalkannya di modul ini saja akan membuat dua gaya jalur hidup berdampingan.
// Penyeragamannya dicatat sebagai utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-compliance/tab", h.Metadata)
		perPortal.Get("/inbox-compliance", h.List)

		// Satu-satunya rute yang MENGUBAH data di modul ini. Ia menulis ke
		// `POOLDATA.T_CLAIM_COMPLIANCE_H`, tabel baru yang tidak dikenal Pega — bukan ke
		// tabel warisan mana pun, sehingga `P-1` tidak dilanggar.
		//
		// Ketiadaan pemeriksaan peran paling berat akibatnya di sini: rute ini menerbitkan
		// baris yang dibaca pemeriksa Post Audit, dan setiap pengguna yang sudah masuk
		// dapat memanggilnya. Itu `TKT-F3-005`, dan sampai ia ada, jejaknya hanya berupa
		// baris log yang menyebut pengirimnya.
		perPortal.Post("/inbox-compliance/post-audit", h.SendPostAudit)

		// Form Compliance Checker — yang terbuka ketika petugas menekan Nomor Case.
		//
		// # Kenapa didaftarkan SETELAH kedua rute di atas
		//
		// Karena `/{referensi}` cocok dengan apa saja, termasuk `tab` dan `post-audit`.
		// chi memang mengutamakan jalur harfiah di atas parameter, sehingga urutannya
		// sebenarnya tidak menentukan — tetapi menaruhnya di bawah membuat urutan baca
		// kode sama dengan urutan kekhususan jalurnya, dan itu yang menolong orang
		// berikutnya yang menambahkan rute.
		//
		// `{referensi}` adalah `PZINSKEY`, yang memuat spasi dan tanda hubung. Layar
		// WAJIB mengkodekannya; chi mendekodekannya kembali.
		perPortal.Get("/inbox-compliance/{referensi}", h.OpenChecker)

		// Rute KEDUA yang mengubah data di modul ini.
		//
		// Ia menulis ke `POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE`, tabel baru milik aplikasi
		// ini — bukan ke tabel klaim, yang masih dimiliki Pega (`P-1`). Pada pilihan
		// Bayar/PostAudit ia juga menerbitkan baris Post Audit, sehingga akibatnya sama
		// dengan rute post-audit di atas ditambah tersimpannya keputusan.
		//
		// Ketiadaan pemeriksaan peran paling berat akibatnya di SINI, lebih berat
		// daripada di rute mana pun lain di modul ini: pilihan `0` berarti Fraud/Tolak,
		// yakni menolak klaim. Setiap pengguna yang sudah masuk dapat memanggilnya.
		// Itu `TKT-F3-005`, dan sampai ia ada, jejaknya hanya berupa baris log yang
		// menyebut pemutusnya.
		perPortal.Post("/inbox-compliance/{referensi}/keputusan", h.SubmitDecision)

		// Menerbitkan tautan baru untuk satu dokumen klaim.
		//
		// `POST`, bukan `GET`, walau ia terbaca seperti pembacaan — karena ia MENERBITKAN
		// sesuatu: satu tautan bertanda tangan berumur terbatas, yang dapat dibuka siapa
		// pun yang memegangnya selama masa berlakunya. Menjadikannya `GET` mengundangnya
		// masuk riwayat peramban, prefetch, dan cache perantara.
		//
		// Dokumennya dicari ULANG di server untuk memastikan ia memang milik klaim pada
		// jalur — lihat OpenDocument. Itu satu-satunya pemeriksaan kepemilikan yang ada,
		// karena layanan penyimpanan di seberang tidak melakukannya.
		perPortal.Post(
			"/inbox-compliance/{referensi}/dokumen/{dokumen}/tautan", h.OpenDocument)

		// Mengunggah satu lampiran. `multipart/form-data` — lihat UploadDocument.
		perPortal.Post("/inbox-compliance/{referensi}/dokumen", h.UploadDocument)

		// Menghapus satu lampiran.
		//
		// Penghapusannya FISIK pada `DATA_ATTACHFILE`, atas persetujuan Work Owner
		// (2026-10-07) — `D-66` tidak diberlakukan di tabel itu. Satu baris riwayat
		// ditulis lebih dulu, dan itu satu-satunya jejak yang tersisa.
		perPortal.Delete(
			"/inbox-compliance/{referensi}/dokumen/{dokumen}", h.DeleteDocument)

		// Menerbitkan Surat Penolakan — tombol "Generate PDF".
		//
		// Namanya BUKAN "unduh", dan itu bukan pilihan kata: ia tidak mengirim berkas ke
		// peramban melainkan MELAMPIRKAN surat ke klaim, persis seperti Pega
		// (`DownloadPDFReject` langkah 13-25). Jawabannya adalah baris dokumen yang baru
		// tercatat; layar membukanya lewat jalur tautan di atas, sama seperti dokumen
		// lain.
		//
		// Menekannya dua kali MENGGANTI suratnya, tidak menumpuk — lihat
		// GenerateRejectLetter.
		perPortal.Post(
			"/inbox-compliance/{referensi}/surat-penolakan", h.GenerateRejectLetter)

		// Melayani tab **Dokumen** (lini Travel).
		//
		// Keduanya dibaca SAAT DITEKAN, bukan ikut pada pembukaan form: sebagian besar
		// kategori kosong, dan memuat seluruh lampiran setiap kali form dibuka berarti
		// membaca banyak yang tidak pernah dilihat.
		//
		// `GET` di sini pembacaan murni — berbeda dari penerbitan tautan yang `POST`
		// karena ia menerbitkan sesuatu yang berumur terbatas.
		perPortal.Get(
			"/inbox-compliance/{referensi}/dokumen", h.ListDocumentsInCategory)

		// `POST`, bukan `PATCH`.
		//
		// PATCH memang lebih tepat secara makna — yang berubah satu isian dari dokumen
		// yang sudah ada. Tetapi klien HTTP bersama belum mengenalnya, dan melebarkannya
		// menyentuh kode yang dipakai SELURUH modul demi satu rute di sini.
		//
		// POST atas sub-sumber daya aksi adalah konvensi modul ini sendiri
		// (`10-API-STRATEGY.md` §2, dan lihat `/keputusan` serta `/surat-penolakan`),
		// sehingga ia bukan penyimpangan melainkan bentuk yang sudah berlaku.
		perPortal.Post(
			"/inbox-compliance/{referensi}/dokumen/{dokumen}/kategori",
			h.ChangeDocumentCategory)
	})
}
