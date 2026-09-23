-- Kueri modul Daftar Detail Dokumen Travel.
--
-- ============================================================================
-- BACA BAGIAN INI SEBELUM MENGUBAH SATU BARIS PUN DI BAWAHNYA.
-- ============================================================================
--
-- # Empat objek yang DIBACA sudah terverifikasi, dua yang DITULIS belum
--
-- Jalur baca layar ini terbaca lengkap dari export:
--
--   POOLDATA.V_LST_DOC_TRAVEL            Report Definition/BrowseLstDocTravel_RD-RD.xml
--   POOLDATA.V_LST_DOC_TRAVEL_COVERAGE   Activity/BrowseDocTravel-Act.xml
--   POOLDATA.M_DOCTRAVEL                 Database/DOCTRAVEL_CVG.prc  (sudah dipakai
--                                        modul Master Dokumen Travel yang berjalan)
--   POOLDATA.M_PLANTRAVEL                Activity/SetspreadingtoCoverage-Act.xml,
--                                        pembenaran peringatan: "ngambil data coverage
--                                        bukan dari coverage travel tapi dari
--                                        m_plantravel"
--
-- Jalur TULIS tidak. Tombol Simpan layar lama memanggil `CNMInsertDocumentTravel_act`
-- dan tombol Ubah memanggil `CNMSetDetailTravelDocument_act`
-- (`Section/BrowseDocumentTravel-Section.xml`), dan **tidak satu pun ada di export**
-- (`R-16`). Tidak ada pula procedure penggantinya.
--
-- Akibatnya TIGA nama objek di bawah adalah TURUNAN, bukan bacaan:
--
--   POOLDATA.LST_DOC_TRAVEL             diturunkan dari nama view V_LST_DOC_TRAVEL
--   POOLDATA.LST_DOC_TRAVEL_COVERAGE    diturunkan dari nama view-nya
--   POOLDATA.LST_DOC_TRAVEL_SEQ         diturunkan dari pola POOLDATA.DOCTRAVEL_SEQ
--
-- Ketiganya WAJIB dikonfirmasi DBA sebelum modul ini dinyatakan siap di satu entitas
-- mana pun. Prosedur konfirmasinya ada di
-- `migrations/0006_detail_dokumen_travel.up.sql`, dan berkas ini adalah SATU-SATUNYA
-- tempat ketiga nama itu muncul di seluruh modul — satu suntingan di sini memperbaiki
-- seluruhnya, dan tidak ada nama tabel yang tercecer di dalam kode Go.
--
-- Bila ternyata view-nya sendiri dapat ditulis, atau nama tabel dasarnya berbeda, yang
-- berubah hanyalah kelima pernyataan tulis di bawah. Bentuk datanya — kolom apa yang
-- diisi dan apa artinya — tidak ikut berubah, karena itu dibaca dari form dan kedua
-- view, bukan ditebak.
--
-- # Satu tabel satu penulis (P-1)
--
-- Layar Daftar Detail Dokumen Travel adalah satu-satunya penulis kedua tabel ini di
-- sistem lama. M_DOCTRAVEL dan M_PLANTRAVEL HANYA DIBACA di sini: yang pertama dimiliki
-- modul Master Dokumen Travel, yang kedua dimiliki GISFW (`D-03`). Tidak ada satu pun
-- INSERT, UPDATE, maupun DELETE terhadap keduanya di berkas ini, dan tidak boleh ada.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.


-- name: detail_list
--
-- Grid layar. Urutannya mengikuti `BrowseLstDocTravel_RD-RD.xml`: ID menaik
-- (pySortOrder 1), lalu DOCID menaik (pySortOrder 2).
--
-- Kelima kolomnya sama persis dengan pxResults Report Definition itu, tidak kurang dan
-- tidak lebih. Coverage TIDAK ikut dibaca di sini — gridnya tidak menampilkannya, dan
-- menariknya untuk seluruh baris berarti satu kueri yang hasilnya tidak pernah dilihat
-- siapa pun.
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

-- name: detail_coverage_list
--
-- Pembatasan plan dan jaminan milik SATU baris detail.
--
-- Urutannya menurut nama plan lalu nama jaminan — bukan menurut ID. Yang dibaca petugas
-- di grid adalah namanya, dan grid yang berpindah urutan setiap kali disimpan akan
-- terbaca sebagai barisnya berubah padahal tidak.
--
-- Penyaringnya ID, bukan TRAVELDOCID. Kolom TRAVELDOCID tidak ada pada view ini — dan
-- tidak ada pada objek POOLDATA mana pun. Definisi view-nya meratakan
-- `LST_DOC_TRAVEL.JSON_DATA.COVERAGELIST[*]` dan membawa serta `a.ID` sebagai kunci
-- induknya, sehingga ID inilah rujukan ke baris detailnya.
SELECT ID,
       PLANID,
       PLANNAME,
       COVERAGEID,
       COVERAGENAME
  FROM POOLDATA.V_LST_DOC_TRAVEL_COVERAGE
 WHERE ID = :1
 ORDER BY PLANNAME, COVERAGENAME

-- name: detail_next_sequence
--
-- Nomor urut ID, dipakai baris detail MAUPUN baris coverage.
--
-- Satu urutan untuk dua tabel, bukan dua urutan. Sebabnya bukan kemalasan: setiap nama
-- objek yang belum terverifikasi adalah satu hal lagi yang dapat salah dan satu hal lagi
-- yang harus diperiksa DBA. Keduanya tabel yang berbeda, sehingga nomor yang sama tidak
-- pernah bertabrakan, dan lubang di deret salah satunya tidak merusak apa pun —
-- `DOCTRAVEL_CVG.prc` pun meninggalkan lubang setiap kali INSERT-nya gagal.
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
-- ID tidak pernah berubah — ia dirujuk baris coverage lewat TRAVELDOCID, dan mengubahnya
-- akan memutus keduanya. Ia hanya dipakai sebagai penyaring WHERE.
UPDATE POOLDATA.LST_DOC_TRAVEL
   SET DOCID = :1,
       DOCUMENTNAME = :2,
       STSWAJIB = :3,
       MINUNGGAH = :4
 WHERE ID = :5

-- name: detail_coverage_delete_all
--
-- Seluruh baris coverage milik satu detail dibuang sebelum susunan barunya disisipkan.
--
-- # Kenapa DELETE di sini tidak melanggar `D-66`
--
-- `D-66` melarang penghapusan fisik DATA BERNILAI BISNIS. Baris ini bukan itu: ia baris
-- penghubung yang menyatakan "aturan dokumen X berlaku pada plan P dan jaminan J", dan
-- satu-satunya isinya adalah kedua rujukan itu. Membuangnya tidak menghilangkan
-- keterangan apa pun yang tidak dapat disusun ulang dari susunan barunya.
--
-- Perlakuan yang sama sudah dipakai modul Master COL Simas Online untuk pemetaan
-- bisnisnya, dan di sana pun alasannya sama.
--
-- Yang TIDAK dilakukan: menghapus baris induknya. Aturan dokumen itu sendiri tetap tidak
-- dapat dihapus lewat jalur mana pun di modul ini.
DELETE FROM POOLDATA.LST_DOC_TRAVEL_COVERAGE
 WHERE TRAVELDOCID = :1

-- name: detail_coverage_insert
--
-- DOCUMENTNAME dan STSWAJIB ikut diisi, meski keduanya sudah ada di baris induknya.
--
-- Itu BUKAN kelalaian normalisasi melainkan bentuk tabel yang sudah ada:
-- `Activity/BrowseDocTravel-Act.xml` membaca kedua kolom itu dari baris
-- V_LST_DOC_TRAVEL_COVERAGE, bukan dari induknya, dan jalur registrasi klaim di
-- `Activity/TravelDocument_act-Act.xml` bergantung padanya. Mengosongkannya akan
-- membuat dokumen wajib berhenti terbaca pada klaim yang dibatasi per jaminan — cacat
-- yang tidak terlihat di layar master ini sama sekali.
INSERT INTO POOLDATA.LST_DOC_TRAVEL_COVERAGE
       (ID, TRAVELDOCID, DOCID, DOCUMENTNAME, STSWAJIB, PLANID, PLANNAME, COVERAGEID, COVERAGENAME)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9)

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

-- name: plan_list
--
-- Pilihan isian Nama Plan.
--
-- DISTINCT, dan itu bukan kerapian: satu plan muncul berkali-kali di M_PLANTRAVEL —
-- sekali untuk setiap jaminan yang dimilikinya. Tanpa DISTINCT, daftar plan akan memuat
-- nama yang sama berulang sebanyak jaminannya.
--
-- HANYA SELECT. Objek ini dimiliki GISFW (`D-03`).
--
-- Dibaca dari view POOLDATA.PLANTRAVEL, bukan dari tabel M_PLANTRAVEL. Tabel itu hanya
-- punya ID, OLDID, dan JSONDATA — nama plannya ada di dalam JSON-nya. View inilah yang
-- sudah memaparkannya sebagai kolom, dan inilah yang dibaca Pega (keputusan Work Owner
-- 2026-09-22: langsung ke basis data, bukan lewat JSON).
--
-- Kunci plannya bernama ID pada view ini, bukan PLANID. Aliasnya dijaga tetap PLANID
-- supaya bentuk hasilnya tidak berubah bagi pemanggil.
SELECT DISTINCT ID AS PLANID,
       PLANNAME
  FROM POOLDATA.PLANTRAVEL
 ORDER BY PLANNAME

-- name: coverage_list
--
-- Pilihan isian Nama Jaminan beserta plan pemiliknya.
--
-- Tidak disaring per plan, berbeda dari `SearchCoverageTravel_RD` yang menerima
-- parameter `plan`. Alasannya bentuk layarnya berbeda: grid di form dapat memuat banyak
-- baris dengan plan berbeda-beda, dan menyaring di server berarti satu permintaan per
-- baris grid setiap kali plannya berganti. Penyaringannya dikerjakan layar atas daftar
-- yang sudah di tangan.
--
-- HANYA SELECT. Objek ini dimiliki GISFW (`D-03`).
--
-- Dibaca dari view POOLDATA.COVERAGETRAVEL, dengan alasan yang sama seperti plan_list di
-- atas: nama jaminan hidup di dalam JSON pada tabelnya, dan view inilah yang memaparkannya
-- sebagai kolom.
--
-- INDCOVERAGENAME, bukan ENGCOVERAGENAME. Keduanya ada berdampingan di view itu, dan yang
-- dipakai Pega untuk nama yang dilihat petugas adalah yang Indonesia —
-- `Activity/InsertObjekTravel-Act.xml` menyalin `.INDCoverageName` ke nama item objek.
-- Memilih yang salah tidak menimbulkan galat apa pun; daftarnya hanya berbahasa Inggris.
SELECT ID AS COVERAGEID,
       INDCOVERAGENAME AS COVERAGENAME,
       PLANID
  FROM POOLDATA.COVERAGETRAVEL
 ORDER BY PLANID, INDCOVERAGENAME

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

-- name: detail_coverage_check_table
SELECT ID,
       PLANID,
       PLANNAME,
       COVERAGEID,
       COVERAGENAME
  FROM POOLDATA.V_LST_DOC_TRAVEL_COVERAGE
 WHERE 1 = 0

-- name: document_check_table
SELECT DOCID,
       NAMADOKUMEN
  FROM POOLDATA.M_DOCTRAVEL
 WHERE 1 = 0

-- name: plan_check_table
SELECT ID AS PLANID,
       PLANNAME
  FROM POOLDATA.PLANTRAVEL
 WHERE 1 = 0
