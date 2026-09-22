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
// # Dua tabel, dan kenapa keduanya perlu dibaca
//
// Sistem lama menyimpan satu laporan di DUA tempat sekaligus:
//
//	DATAPEGA.PC_ASM_FW_GCNMFW_WORK    kepala berkas — milik ENGINE Pega
//	POOLDATA.T_CLAIM_RECIVEDCLAIM     rincian laporan — tabel bisnis
//
// Work Owner menetapkan 2026-09-19: **penulisan tidak lagi masuk ke
// PC_ASM_FW_GCNMFW_WORK.** Laporan yang dibuat aplikasi ini karena itu tinggal di tabel
// miliknya sendiri, `POOLDATA.CPNC_LAPORAN_KLAIM` (migrasi 0003), sementara laporan lama
// tetap dibaca dari tabel Pega.
//
// Itulah bentuk nyata `ADR-0004` penulis tunggal per tabel: selama masa paralel, Pega
// tetap satu-satunya yang menulis tabelnya sendiri dan aplikasi ini hanya MEMBACA-nya;
// aplikasi ini satu-satunya yang menulis tabelnya sendiri. Daftar yang dilihat pengguna
// adalah gabungan keduanya, dan asal setiap baris terbaca langsung dari nomornya
// (lihat ReportNumberPrefix) tanpa perlu tabel pemetaan — prinsip yang sama dipakai
// `D-71` untuk nomor klaim.
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
