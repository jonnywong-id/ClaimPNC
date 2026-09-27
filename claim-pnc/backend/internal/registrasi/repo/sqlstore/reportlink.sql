-- Tautan balik dari klaim ke berkas Receive Document asalnya.
--
-- ============================================================================
-- KENAPA DUA KOLOM, DAN KENAPA URUTANNYA PENTING
-- ============================================================================
--
-- Posisi berkas pada Inbox Laporan Klaim dihitung dari dua kolom
-- (`inboxlaporanklaim.sql`, cabang kedua):
--
--   NOKLAIM kosong, TRANSFERASM kosong  -> Not Transferred
--   NOKLAIM kosong, TRANSFERASM terisi  -> Not Registered
--   NOKLAIM terisi, TRANSFERASM terisi  -> Outstanding
--
-- Karena itu TRANSFERASM diisi lebih dulu (saat klaim dibuka) dan NOKLAIM menyusul (saat
-- nomor terbit di ujung Input Register). Membalik urutannya menghasilkan kombinasi
-- "NOKLAIM terisi, TRANSFERASM kosong" — yang kuerinya kembalikan sebagai NULL, sehingga
-- berkasnya lenyap dari SELURUH tab.
--
-- Kombinasi itu bukan kemungkinan teoretis: 1.127 baris warisan berada di dalamnya
-- (terverifikasi 2026-09-24, dari 1.470 ber-NOKLAIM dan 343 ber-TRANSFERASM). Jangan
-- menambahinya.
--
-- ============================================================================
-- PAGAR P-1 — BUKAN KENYAMANAN, MELAINKAN BATAS KEPEMILIKAN
-- ============================================================================
--
-- Selama masa paralel, tepat satu sistem menulis sebuah baris (`ADR-0004`). Berkas milik
-- Pega hanya boleh dibaca, dan pemisahnya adalah AWALAN KUNCI: baris terbitan aplikasi
-- ini ber-CLAIMID `RCVN%`.
--
-- Kedua UPDATE di bawah karena itu menyaring `CLAIMID LIKE 'RCVN%'`. Layar memang sudah
-- mematikan tombolnya pada berkas Pega, tetapi layar bukan tempat menegakkan kepemilikan
-- data — satu permintaan yang dibuat di luar layar akan melewatinya.
--
-- Baris yang tidak cocok menghasilkan 0 baris terpengaruh, dan pemanggilnya menjadikan
-- itu galat — bukan diam-diam dianggap berhasil.

-- name: laporan_tandai_diserahkan
-- Mengisi TRANSFERASM hanya bila masih kosong.
--
-- Syarat `TRANSFERASM IS NULL` membuat pemanggilan ulang tidak menggeser tanggal
-- penyerahan yang sudah tercatat. Pemanggil membedakan "tidak ada baris yang cocok" dari
-- "sudah terisi" lewat pemeriksaan terpisah, bukan dari jumlah baris terpengaruh.
UPDATE POOLDATA.T_CLAIM_RECIVEDCLAIM
   SET TRANSFERASM = :1
 WHERE CLAIMID = :2
   AND CLAIMID LIKE 'RCVN%'
   AND TRANSFERASM IS NULL

-- name: laporan_pasang_nomor_klaim
-- Mengisi NOKLAIM.
--
-- Tanpa syarat "masih kosong": nomor klaim terbit sekali dan tidak berubah, sehingga
-- penulisan ulang dengan nilai yang sama tidak merusak apa pun. Yang dijaga justru
-- sebaliknya — TRANSFERASM wajib sudah terisi, supaya kombinasi tanpa tab tidak lahir.
UPDATE POOLDATA.T_CLAIM_RECIVEDCLAIM
   SET NOKLAIM = :1
 WHERE CLAIMID = :2
   AND CLAIMID LIKE 'RCVN%'
   AND TRANSFERASM IS NOT NULL

-- name: laporan_keadaan
-- Keadaan sebuah berkas, dipakai untuk menjelaskan kegagalan.
--
-- Ia menjawab tiga pertanyaan sekaligus: barisnya ada atau tidak, miliknya siapa, dan
-- kedua kolom penentu posisinya sudah terisi atau belum. Tanpa ini, satu UPDATE yang
-- mengenai 0 baris tidak dapat dibedakan sebabnya.
SELECT CASE WHEN CLAIMID LIKE 'RCVN%' THEN 1 ELSE 0 END,
       CASE WHEN TRANSFERASM IS NULL THEN 0 ELSE 1 END,
       CASE WHEN NOKLAIM IS NULL THEN 0 ELSE 1 END
  FROM POOLDATA.T_CLAIM_RECIVEDCLAIM
 WHERE CLAIMID = :1

-- name: laporan_isi
-- Isi berkas laporan yang dibawa ke klaim saat ia dibuka.
--
-- Kolomnya dipilih mengikuti `Activity/CreateRegisterKlaimPNC_act.xml` langkah 14 — satu
-- satunya tempat di sistem lama yang menyalin isi berkas RCV ke klaim:
--
--   DOL                  -> ClaimData.DateOfLoss
--   TANGGALTERIMADOKUMEN -> ClaimData.ReportDate          (lihat catatan nama di bawah)
--   NAMAPELAPOR          -> ClaimData.ReporterName        (Pega: ReceiveDocument.Sender)
--   TLPPENGIRIM          -> ClaimData.ReporterTelp
--   EMAILPENGIRIM        -> ClaimData.Email
--   LOKASIKEJADIAN       -> ClaimData.Location
--   KRONOLOGIKEJADIAN    -> ClaimData.ReportDescription
--   ESTIMATIONVALUE      -> ClaimData.ClaimEstimate
--   NOPOLIS              -> diperiksa, tidak disalin
--
-- # Kenapa TANPA pagar RCVN
--
-- Berbeda dari kedua UPDATE di atas, ini PEMBACAAN. Berkas milik Pega boleh dibaca — yang
-- dilarang `P-1` adalah menulisinya. Memagari pembacaan akan membuat klaim yang dibuat
-- dari berkas warisan lahir kosong tanpa sebab yang terlihat.
--
-- ESTIMATIONVALUE bertipe angka dalam RUPIAH pada tabel warisan; pemanggil mengalikan 100
-- untuk mendapatkan SEN (`ADR-0016`). Lihat catatan satuan uang pada claim.sql.
--
-- # TANGGALTERIMADOKUMEN: namanya salah, tipenya teks, dan isinya berbeda per pemilik
--
-- Kolom ini `VARCHAR2`, dan Pega mengisinya dengan **ReferenceId — bukan tanggal**
-- (`Rcv_ProcInsertRecivedDocument`). Baris milik aplikasi ini selalu menuliskannya dalam
-- bentuk ISO `YYYY-MM-DD`.
--
-- Karena itu ia diambil APA ADANYA sebagai teks dan ditafsirkan di Go, bukan lewat
-- `TO_DATE` seperti pada daftar berkas. `TO_DATE` di sini akan menggagalkan seluruh
-- pembuatan klaim dengan ORA-01861 begitu berkas warisan diregistrasi — pembacaan yang
-- memang diizinkan. Teks yang bukan tanggal diperlakukan sebagai tidak ada.
SELECT DOL, TANGGALTERIMADOKUMEN, NAMAPELAPOR, TLPPENGIRIM, EMAILPENGIRIM,
       LOKASIKEJADIAN, KRONOLOGIKEJADIAN, ESTIMATIONVALUE, NOPOLIS
  FROM POOLDATA.T_CLAIM_RECIVEDCLAIM
 WHERE CLAIMID = :1
