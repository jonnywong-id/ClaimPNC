// Package inboxlaporanklaim adalah inti modul Inbox Laporan Klaim.
//
// # Layar apa ini
//
// Menu `MENU_ID 64` "Inbox Laporan Klaim" pada POOLDATA.M_MENU_APLIKASI_PNC, yang
// menunjuk harness `InboxRCVApp_Harness`. Judul yang dibaca pengguna di sistem lama
// adalah **"Inbox Reporting Claim"**.
//
// Isinya adalah daftar **laporan klaim yang masuk** — kejadian yang dilaporkan lewat
// surel, kurir, atau aplikasi, sebelum ia menjadi klaim ber-nomor. Satu baris di sini
// adalah satu berkas laporan yang menunggu diproses petugas: dicek, diregistrasi menjadi
// klaim, lalu berjalan ke akseptasi — atau ditolak.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxRCVApp_Harness-Harness.xml        pembungkus layar; isinya satu section
//	Section/ViewStatusReceiveDocument-Section.xml   9 grid, judul, tombol, saringan, bagan
//	Activity/SetListRCV_Act-Act.xml                 pemilih kueri, saringan, dan paginasi
//	Activity/GetClaimRCVList_Act-Act.xml            pemuat daftar ke halaman klipboard
//	Activity/CreateNewCaseRCV-Act.xml               tombol "Buat Baru"
//	Activity/ExportNotTransferRCV-Act.xml           tombol "Export Data"
//	RDB List/ViewAllCase-SQL.xml                    tab "All data" + label Position
//	RDB List/ViewTableBrowseRCVInProcess-SQL.xml    tab "Data hasn't been transferred"
//	RDB List/ViewTableBrowseClaimNotRegister-SQL.xml tab "Unregistered data"
//	RDB List/ViewTableBrowseClaimRegister-SQL.xml   tab "Outstanding Data"
//	RDB List/ViewTableBrowseRCVAcc-SQL.xml          tab "Data has been accepted"
//	RDB List/ViewTableBrowseRCVReject-SQL.xml       tab "Data rejected"
//	RDB List/ViewRejectKomunikasiUser-SQL.xml       tiga tab komunikasi
//	RDB List/BrowseClaimRCV_Aksep-SQL.xml           delapan pencacah di atas layar
//	Database/PROCINSERTDATARECIVEDKLAIM.prc         isi baris laporan yang disimpan
//
// # Dari mana daftar ditarik, dan ke mana berkas baru ditulis
//
// Sistem lama menyimpan satu laporan di DUA tempat sekaligus — kepala berkas di
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` milik engine Pega, rincian di
// `POOLDATA.T_CLAIM_RECIVEDCLAIM`. Modul ini pernah membaca yang pertama secara langsung.
//
// **Sejak 2026-09-23 (Work Owner), daftar ditarik dari `POOLDATA.T_CLAIMLIST_ADMIN`** —
// tabel rata yang diisi proses lain, dan yang **hanya dibaca** aplikasi ini.
//
// Penulisan tidak berubah: berkas yang dibuat aplikasi ini tetap ditulis ke tabel miliknya
// sendiri, `POOLDATA.CPNC_LAPORAN_KLAIM` (migrasi 0003 dan 0004). Work Owner menetapkan
// **proses pengisi T_CLAIMLIST_ADMIN diperluas agar ikut membaca tabel itu**, sehingga
// berkas terbitan aplikasi ini masuk ke daftar lewat jalur yang sama dengan berkas Pega.
//
//	membaca daftar   POOLDATA.T_CLAIMLIST_ADMIN     diisi proses lain
//	menulis berkas   POOLDATA.CPNC_LAPORAN_KLAIM    hanya aplikasi ini
//
// Itulah bentuk nyata `ADR-0004` penulis tunggal per tabel: tidak ada satu tabel pun yang
// ditulis dua sistem.
//
// Asal setiap baris terbaca dari NOMORNYA (lihat ReportNumberPrefix) — bukan dari tabel
// asalnya, karena tabelnya kini satu. Awalan `RCVN.` ditetapkan `D-71` justru untuk itu.
//
// # Dua akibat yang diterima secara sadar
//
//  1. **Berkas yang baru dibuat belum muncul di daftar** sampai proses pengisi berjalan.
//     Ia tetap dapat dibuka langsung sesudah dibuat — pembacaan satu berkas menempuh
//     jalur tersendiri ke CPNC_LAPORAN_KLAIM, lihat claim_report_get_own_body.
//
//  2. **Tiga kolom grid menjadi kosong**: "Reference no", "Alasan", dan "Subject Email".
//     Ketiganya tidak punya padanan di T_CLAIMLIST_ADMIN, dan sengaja TIDAK dipetakan ke
//     kolom lain yang kebetulan mirip.
//
// # STS_AKTIF
//
// T_CLAIMLIST_ADMIN memuat kolom STS_AKTIF: '1' klaim masih aktif, '0' sudah tidak aktif
// dan TIDAK ditampilkan lagi (Work Owner, 2026-09-23).
//
// Yang dikecualikan hanya yang bernilai '0' secara tegas, bukan "yang bukan 1". Kolomnya
// nullable, dan baris ber-NULL berarti penandanya tidak ditetapkan — menyembunyikannya
// berarti menghilangkan pekerjaan dari layar tanpa seorang pun tahu.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxlaporanklaim/          aturan modul + seam          ← paket ini
//	inboxlaporanklaim/usecase/  orkestrasi: daftar, ringkas, buat, ekspor
//	inboxlaporanklaim/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxlaporanklaim/http/     lapisan transport modul ini  — handler, dto, rute
//
// # Yang BUKAN urusan paket ini
//
// Apa yang terjadi SETELAH laporan dibuka. Menekan satu baris di sistem lama membuka
// penugasan `ReceiveDocument_Flow` lewat harness `ViewReceiveDocument` — layar tersendiri
// dengan menunya sendiri, dan lingkup `B-14`. Modul ini mengetahui KEBERADAAN penugasan
// itu (ia menyusun rujukannya, lihat ClaimReport.AssignmentRef) tetapi tidak mengetahui
// isinya.
//
// Juga bukan urusan paket ini: registrasi klaim (`B-2`, modul registrasi), akseptasi
// (`B-10`), dan isi percakapan pada POOLDATA.M_KOMUNIKASI_PNC — yang dibaca di sini
// hanyalah pesan terakhirnya, persis sebanyak yang ditampilkan kolom "Last message".
package inboxlaporanklaim
