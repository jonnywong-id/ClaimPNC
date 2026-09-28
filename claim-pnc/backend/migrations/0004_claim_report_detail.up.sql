-- =====================================================================
-- DICABUT 2026-09-23 — JANGAN DIJALANKAN
-- =====================================================================
--
-- Work Owner menetapkan berkas laporan TIDAK ditulis ke tabel milik aplikasi ini,
-- melainkan ke POOLDATA.T_CLAIM_RECIVEDCLAIM — tabel bisnis yang sudah dipakai Pega
-- lewat Database/PROCINSERTDATARECIVEDKLAIM.prc.
--
-- Berkas ini dipertahankan sebagai REKAMAN rancangan yang pernah diambil, bukan
-- sebagai langkah pemasangan. Menjalankannya akan membuat tabel yang tidak dibaca
-- maupun ditulis satu baris kode pun.
--
-- Alasan lengkapnya di docs/keputusan-implementasi.md.
-- =====================================================================

-- 0004 — Isian form Input Receive Document (Oracle 19c)
--
-- MENAMBAH sepuluh kolom pada POOLDATA.CPNC_LAPORAN_KLAIM. Tidak satu pun tabel milik
-- sistem lama disentuh, dan tidak satu pun kolom yang sudah ada diubah atau dihapus.
--
-- ============================================================================
-- KENAPA IA BERKAS TERSENDIRI, BUKAN SUNTINGAN PADA 0003
-- ============================================================================
--
-- Karena 0003 sudah diserahkan sebagai permintaan perubahan skema (`D-63`), dan mungkin
-- sudah dijalankan DBA di salah satu portal. Menyuntingnya berarti dua orang memegang
-- berkas 0003 dengan isi yang berbeda — dan yang menjalankan versi lama tidak punya cara
-- mengetahuinya.
--
-- Berkas ini karena itu bersifat MENAMBAH dan aman dijalankan apa pun keadaannya, selama
-- 0003 sudah lebih dulu. Ia mengikuti `P-4`: menambah kolom boleh langsung, dan
-- seluruhnya NULLABLE sehingga versi aplikasi yang lama tetap berjalan terhadap skema ini.
--
-- ============================================================================
-- DARI MANA KESEPULUH KOLOM INI
-- ============================================================================
--
-- Dari form yang dirender flow action `InputReceiveDocument`
-- (`Flow/InputReceiveDocument.xml` → `Flow Action/InputReceiveDocument-FlowAction.xml`),
-- dibaca lewat `Section/ViewInputReceiveDocument_sec-Section.xml` dan disilangkan dengan
-- parameter `Database/PROCINSERTDATARECIVEDKLAIM.prc`:
--
--   properti form                       procedure lama           kolom di sini
--   ---------------------------------------------------------------------------------
--   .ReceiveDocument.ReceivedDate       TANGGALTERIMADOKUMEN     TGL_TERIMA_DOKUMEN
--   .ReceiveDocument.EmailPengirim      EMAILPENGIRIM            EMAIL_PELAPOR
--   .ReceiveDocument.TelpPengirim       TLPPENGIRIM              TLP_PELAPOR
--   .ReceiveDocument.Kurir              NAMAKURIRASM             NAMA_KURIR
--   .ReceiveDocument.Estimasi           (tidak ada)              NILAI_ESTIMASI
--   .ReceiveDocument.LokasiKejadian     LOKASIKEJADIAN           LOKASI_KEJADIAN
--   .ReceiveDocument.KronologisKejadian KRONOLOGIKEJADIAN        KRONOLOGIS
--   .ReceiveDocument.RincianKerusakan   RINCIANKERUSAKAN         RINCIAN_KERUSAKAN
--   .ReceiveDocument.NotRegistNote      KETERANGANBLMREGIST      KET_BELUM_REGISTRASI
--   .ReceiveDocument.NumberOfDocument   (tidak ada)              JUMLAH_DOKUMEN
--
-- Dua kolom terakhir pada daftar itu ADA di form tetapi TIDAK ada di procedure — di
-- sistem lama keduanya tinggal di objek kerja Pega, bukan di tabel bisnis. Keduanya tetap
-- dibuat: ia isian yang benar-benar diketik petugas, dan membuangnya berarti form baru
-- kehilangan sesuatu yang form lama punya.
--
-- Tiga isian form lain sengaja TIDAK dibuatkan kolom, dan alasannya di
-- internal/inboxlaporanklaim/detail.go: blok data pelapor beserta alamatnya (`.ReportHE.*`
-- — area Heavy Equipment yang `D-34` keluarkan dari lingkup), grid dokumen (`S-1` belum
-- ada), serta riwayat komunikasi dan progres (milik modul lain).
--
-- Seluruh kolom waktu menyimpan UTC. Konversi ke WIB hanya terjadi di aplikasi
-- (docs/Steering/08-TECHNICAL-STRATEGY.md §4.4) — tidak ada penambahan 7 jam di sini.
--
-- PERHATIAN: berkas ini BELUM dijalankan di lingkungan mana pun. Menjalankannya menuntut
-- permintaan perubahan skema tertulis, persetujuan Work Owner, dan pelaksanaan oleh DBA
-- (D-63). Sama seperti 0003, ia dijalankan di SETIAP portal entitas.

ALTER TABLE CPNC_LAPORAN_KLAIM ADD (
    TGL_TERIMA_DOKUMEN   DATE,
    EMAIL_PELAPOR        VARCHAR2(200),
    TLP_PELAPOR          VARCHAR2(64),
    NAMA_KURIR           VARCHAR2(255),
    NILAI_ESTIMASI       NUMBER(19),
    LOKASI_KEJADIAN      VARCHAR2(500),
    KRONOLOGIS           VARCHAR2(4000),
    RINCIAN_KERUSAKAN    VARCHAR2(4000),
    KET_BELUM_REGISTRASI VARCHAR2(1000),
    JUMLAH_DOKUMEN       NUMBER(10)
);

-- NILAI_ESTIMASI disimpan dalam SEN, bukan rupiah — `ADR-0016` menuntut nilai uang
-- presisi penuh, dan pembulatan hanya saat ditampilkan. Tipe bilangan bulat dipilih
-- karena pecahan biner tidak punya wakil tepat untuk 0,1.
--
-- Ia BUKAN nilai klaim. Nilai klaim lahir di `B-5` setelah registrasi; yang di sini
-- adalah angka yang disebut pelapor saat berkasnya masuk, dan tidak dipakai menghitung
-- apa pun.
COMMENT ON COLUMN CPNC_LAPORAN_KLAIM.NILAI_ESTIMASI IS 'Estimasi kerugian dalam SEN (ADR-0016); bukan nilai klaim';

-- JUMLAH_DOKUMEN hanya ANGKA. Rincian per dokumen terikat page list
-- .ReceiveDocument.DocumentList di form lama, dan menuntut penyimpanan dokumen (S-1, D-16)
-- yang belum ada.
COMMENT ON COLUMN CPNC_LAPORAN_KLAIM.JUMLAH_DOKUMEN IS 'Total jumlah dokumen; rincian per dokumen menunggu modul S-1';

-- TGL_TERIMA_DOKUMEN bertipe DATE, bukan TIMESTAMP: yang dicatat petugas adalah TANGGAL
-- dokumen diterima, dan jamnya tidak pernah diisi maupun ditampilkan di form lama.
COMMENT ON COLUMN CPNC_LAPORAN_KLAIM.TGL_TERIMA_DOKUMEN IS 'Tanggal Terima Dokumen; jam tidak dicatat';

-- Index pencarian nomor polis. Berkas laporan dicari petugas lewat nomor polis jauh lebih
-- sering daripada lewat nomor registernya — nomor register baru diketahui SETELAH
-- berkasnya masuk ke sistem.
CREATE INDEX IX_CPNC_LAPORAN_POLIS ON CPNC_LAPORAN_KLAIM (NO_POLIS);
