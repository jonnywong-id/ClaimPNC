-- Kueri modul Daftar Detail Dokumen Travel.
--
-- ============================================================================
-- BACA BAGIAN INI SEBELUM MENGUBAH SATU BARIS PUN DI BAWAHNYA.
-- ============================================================================
--
-- # Dua objek yang DIBACA sudah terverifikasi, dua yang DITULIS belum
--
-- Jalur baca layar ini terbaca lengkap dari export:
--
--   POOLDATA.V_LST_DOC_TRAVEL   Report Definition/BrowseLstDocTravel_RD-RD.xml
--   POOLDATA.M_DOCTRAVEL        Database/DOCTRAVEL_CVG.prc  (sudah dipakai modul
--                               Master Dokumen Travel yang berjalan)
--
-- Jalur TULIS tidak. Tombol Simpan layar lama memanggil `CNMInsertDocumentTravel_act`
-- dan tombol Ubah memanggil `CNMSetDetailTravelDocument_act`
-- (`Section/BrowseDocumentTravel-Section.xml`), dan **tidak satu pun ada di export**
-- (`R-16`). Tidak ada pula procedure penggantinya.
--
-- Akibatnya DUA nama objek di bawah adalah TURUNAN, bukan bacaan:
--
--   POOLDATA.LST_DOC_TRAVEL       diturunkan dari nama view V_LST_DOC_TRAVEL
--   POOLDATA.LST_DOC_TRAVEL_SEQ   diturunkan dari pola POOLDATA.DOCTRAVEL_SEQ
--
-- Keduanya WAJIB dikonfirmasi DBA sebelum modul ini dinyatakan siap di satu entitas mana
-- pun. Prosedur konfirmasinya ada di `migrations/0006_detail_dokumen_travel.up.sql`, dan
-- berkas ini adalah SATU-SATUNYA tempat kedua nama itu muncul di seluruh modul — satu
-- suntingan di sini memperbaiki seluruhnya, dan tidak ada nama tabel yang tercecer di
-- dalam kode Go.
--
-- Bila ternyata view-nya sendiri dapat ditulis, atau nama tabel dasarnya berbeda, yang
-- berubah hanyalah ketiga pernyataan tulis di bawah. Bentuk datanya — kolom apa yang
-- diisi dan apa artinya — tidak ikut berubah, karena itu dibaca dari form dan view, bukan
-- ditebak.
--
-- # V_LST_DOC_TRAVEL_COVERAGE dan M_PLANTRAVEL TIDAK disentuh modul ini
--
-- Dicatat karena keduanya sempat ada di berkas ini lalu dicabut.
--
-- `Section/BrowseDocumentTravel-Section.xml:3731` memuat grid berulang berkelas
-- `ASM-FW-GCNMFW-Int-V_LST_DOC_TRAVEL_COVERAGE` tanpa kondisi yang menyembunyikannya,
-- sehingga dibaca dari XML saja ia tampak bagian layar. **Di aplikasi Pega yang berjalan
-- grid itu tidak ada** — Work Owner memeriksa layarnya langsung dan menetapkannya
-- 2026-10-03.
--
-- V_LST_DOC_TRAVEL_COVERAGE tetap dibaca jalur registrasi klaim lewat
-- `Activity/BrowseDocTravel-Act.xml`; yang berubah hanyalah bahwa layar master ini bukan
-- penulisnya. Menambahkan kembali pernyataan tulis ke sana menuntut keputusan Work Owner
-- lebih dulu, bukan sekadar menyalin dari riwayat berkas ini.
--
-- # Satu tabel satu penulis (P-1)
--
-- Layar Daftar Detail Dokumen Travel adalah satu-satunya penulis LST_DOC_TRAVEL di
-- sistem lama. M_DOCTRAVEL HANYA DIBACA di sini: ia dimiliki modul Master Dokumen
-- Travel. Tidak ada satu pun INSERT, UPDATE, maupun DELETE terhadapnya di berkas ini,
-- dan tidak boleh ada.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.


-- name: detail_list
--
-- Grid layar. Urutannya mengikuti `BrowseLstDocTravel_RD-RD.xml`: ID menaik
-- (pySortOrder 1), lalu DOCID menaik (pySortOrder 2).
--
-- Kelima kolomnya sama persis dengan pxResults Report Definition itu, tidak kurang dan
-- tidak lebih.
--
-- Batas `pyMaxRecords=500` milik Pega tidak ditiru. Ia bukan aturan bisnis melainkan
-- pemotongan senyap (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2), dan menirunya berarti
-- menyembunyikan baris yang benar-benar ada dari petugas yang sedang menyuntingnya.
SELECT ID,
       DOCID,
       DOCUMENTNAME,
       STSWAJIB,
       MINUNGGAH
  FROM POOLDATA.V_LST_DOC_TRAVEL
 ORDER BY ID, DOCID

-- name: detail_get
SELECT ID,
       DOCID,
       DOCUMENTNAME,
       STSWAJIB,
       MINUNGGAH
  FROM POOLDATA.V_LST_DOC_TRAVEL
 WHERE ID = :1

-- name: detail_next_sequence
--
-- Nomor urut ID baris detail.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`.
--
-- Pemformatan nomornya menjadi ID dikerjakan di Go, bukan dengan LPAD di sini: LPAD dan
-- TO_CHAR termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat
-- kueri pada dialek Oracle.
SELECT POOLDATA.LST_DOC_TRAVEL_SEQ.NEXTVAL
  FROM DUAL

-- name: detail_insert
--
-- STSWAJIB diisi 1 atau 0, bukan 'Ya' atau 'Tidak'. Buktinya di
-- `Activity/BrowseDocTravel-Act.xml`, yang precondition langkahnya berbunyi
-- `.STSWAJIB==1` dan `.STSWAJIB==0` — teks "Ya"/"Tidak" hanya dibentuk untuk
-- ditampilkan, tidak pernah disimpan.
INSERT INTO POOLDATA.LST_DOC_TRAVEL (ID, DOCID, DOCUMENTNAME, STSWAJIB, MINUNGGAH)
VALUES (:1, :2, :3, :4, :5)

-- name: detail_update
--
-- ID tidak pernah berubah — ia dirujuk baris V_LST_DOC_TRAVEL_COVERAGE lewat TRAVELDOCID
-- yang dibaca jalur registrasi klaim, dan mengubahnya akan memutus keduanya. Ia hanya
-- dipakai sebagai penyaring WHERE.
UPDATE POOLDATA.LST_DOC_TRAVEL
   SET DOCID = :1,
       DOCUMENTNAME = :2,
       STSWAJIB = :3,
       MINUNGGAH = :4
 WHERE ID = :5

-- name: document_list
--
-- Pilihan isian ID Dokumen, dari master induknya.
--
-- Kueri ini SAMA PERSIS dengan `travel_document_list` milik modul Master Dokumen Travel,
-- dan itu disengaja: keduanya membaca tabel yang sama dengan urutan yang sama
-- (`BrowseMstDocTravel_RD-RD.xml`). Modul ini punya salinannya sendiri alih-alih
-- mengimpor repo modul lain, karena modul tidak saling mengimpor — yang dibagi adalah
-- tabelnya, bukan kodenya.
--
-- HANYA SELECT. Tabel ini dimiliki modul Master Dokumen Travel (`P-1`).
SELECT DOCID,
       NAMADOKUMEN
  FROM POOLDATA.M_DOCTRAVEL
 ORDER BY DOCID

-- name: detail_check_table
--
-- Memastikan view-nya ada dan kelima kolomnya dapat dibaca akun aplikasi, tanpa
-- mengambil satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan
-- yang tampak mirip tetapi perbaikannya berbeda jauh: objeknya tidak ada di basis data
-- entitas itu, versus akun aplikasi tidak punya hak baca atasnya.
SELECT ID,
       DOCID,
       DOCUMENTNAME,
       STSWAJIB,
       MINUNGGAH
  FROM POOLDATA.V_LST_DOC_TRAVEL
 WHERE 1 = 0

-- name: document_check_table
SELECT DOCID,
       NAMADOKUMEN
  FROM POOLDATA.M_DOCTRAVEL
 WHERE 1 = 0
