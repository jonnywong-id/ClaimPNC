-- 0005 — Daftar Tipe Dokumen: isi pindah dari JSON ke kolom (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Berkas ini MENGUBAH objek milik sistem lama yang sedang melayani produksi:
--
--   * POOLDATA.LST_DOC_TYPE     — empat kolom ditambahkan dan diisi
--   * POOLDATA.V_LST_DOC_TYPE   — view-nya didefinisikan ulang
--
-- Menjalankannya menuntut permintaan perubahan skema tertulis, persetujuan Work Owner,
-- pelaksanaan oleh DBA, dan pengujian dengan MENJALANKAN PEGA DAN GO BERSAMAAN terhadap
-- skema hasil perubahan (`D-63`). Akun aplikasi tidak memiliki hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
-- Ia juga harus dijalankan di BASIS DATA SETIAP ENTITAS, bukan hanya di portal utama.
-- Tabelnya per entitas — `Database/PEGA_LST_DOC_TYPE.prc:12` membentuk ID dari kode situs
-- milik basis data tempat ia berjalan — sehingga entitas yang terlewat akan membuat
-- layarnya gagal justru pada portal itu saja, dan gejalanya akan tampak seperti cacat
-- aplikasi.
--
--
-- ## PERINGATAN — berkas ini ditulis TANPA melihat katalog
--
-- Peringatan yang sama dengan migrasi 0004, dan sebabnya sama: pembacaan katalog belum
-- dilakukan untuk tabel ini. Migrasi 0002 mula-mula ditulis dengan menebak isi skema dari
-- source procedure, dan tebakan itu ternyata SALAH di dua tempat — kedua kolom yang hendak
-- ditambahkannya ternyata sudah ada, sehingga ALTER TABLE-nya akan gagal dengan ORA-01430
-- dan menghentikan seluruh migrasi di baris pertama.
--
-- Yang diketahui tentang tabel ini hanya:
--
--   * Dari `Database/PEGA_LST_DOC_TYPE.prc:23`, tabelnya punya sekurang-kurangnya dua
--     kolom — ID dan JSON_DATA.
--   * Dari `Report Definition/BrowseLstDocType_RD-RD.xml`, view V_LST_DOC_TYPE
--     mengeluarkan ENAM kolom: ID, OLD_ID, TYPE_DOCUMENT, STS_PROSES, USER_EDIT, TGL_EDIT.
--   * Dari `RDB List/BrowseRegisterCvg-SQL.xml:82` dan sembilan rule sejenis, yang
--     dibaca pihak lain dari view itu hanyalah ID dan TYPE_DOCUMENT.
--   * Dari `Activity/SearchDataArchiveFilling-Act.xml`, STS_PROSES dibaca layar Arsip
--     Dokumen dan dialias-namakan "NoteKasir" — ia CATATAN TEKS BEBAS, bukan penanda
--     aktif. Jangan mengubahnya menjadi kolom berdomain tertutup.
--
-- Nama kolom di bawah karena itu mengikuti nama kolom view, dan kunci JSON-nya diturunkan
-- dari nama property Pega yang sama. SELURUHNYA HARUS DIVERIFIKASI DBA terhadap katalog
-- sebelum dijalankan — lihat langkah 0.
--
--
-- ## Kenapa perubahan ini diminta
--
-- Sistem lama menyimpan seluruh baris sebagai satu dokumen JSON
-- (`LST_DOC_TYPE.JSON_DATA`), lalu membongkarnya kembali lewat view. Yang menulisnya
-- adalah procedure POOLDATA.PEGA_LST_DOC_TYPE.
--
-- Tiga keputusan yang sudah disetujui menutup jalan itu:
--
--   D-02  Logika stored procedure naik ke Go; aplikasi tidak memanggil procedure.
--   D-68  Kepemilikan transaksi pindah ke Go. Kontrak galat procedure lama tidak dibawa:
--         parameter keluarannya bernama ErrMsg tetapi pada jalur BERHASIL ia berisi
--         kalimat "Data Sudah Disimpan dengan ID : 10001" (`:22`, `:35`), sehingga
--         pemanggil tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
--   —     Work Owner 2026-09-21: penyimpanan tidak lagi memakai JSON; nilai disimpan
--         langsung ke kolom, di tempat yang sama seperti Pega.
--
-- Perlakuannya sama persis dengan yang sudah dijalankan migrasi 0002 untuk M_STS_CLAIM dan
-- migrasi 0004 untuk M_CAUSE_OF_LOSS — termasuk keputusan membiarkan kolom JSON lamanya
-- tetap ada.
--
-- P-1 tetap dipatuhi — satu tabel satu penulis. Layar Daftar Tipe Dokumen adalah
-- SATU-SATUNYA penulis tabel ini di sistem lama (`RDB List/UpdateLstDocType-SQL.xml`,
-- pemanggil tunggal PEGA_LST_DOC_TYPE), sehingga memindahkan layarnya memindahkan
-- kepemilikan tabelnya secara utuh. Pega berubah menjadi pembaca saja.
--
--
-- ## Yang HARUS dilakukan DBA sebelum menjalankan — LANGKAH 0
--
-- 1. SIMPAN DEFINISI VIEW YANG SEKARANG, beserta daftar kolomnya. Migrasi turun
--    membutuhkannya, dan ALL_VIEWS.TEXT tidak menyimpan daftar kolom view:
--
--        SELECT DBMS_METADATA.GET_DDL('VIEW', 'V_LST_DOC_TYPE', 'POOLDATA') FROM DUAL;
--
--    Simpan hasilnya ke berkas dan lampirkan pada permintaan perubahan skema. Definisi itu
--    juga yang MEMASTIKAN kunci JSON yang dipakai langkah 1 di bawah, DAN yang menjawab
--    satu pertanyaan yang belum terjawab dari export: apakah OLD_ID kolom sungguhan pada
--    tabel dasarnya, atau nilai yang dibongkar dari JSON seperti halnya OLD_M_COL_ID pada
--    migrasi 0004. Jawabannya menentukan langkah 5.
--
-- 2. BACA DAFTAR KOLOM TABEL DASARNYA. Bila salah satu kolom di langkah 1 ternyata SUDAH
--    ADA — seperti yang terjadi pada LSC_NOTE di migrasi 0002 — hapus ALTER TABLE yang
--    bersangkutan, jangan dijalankan:
--
--        SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, NULLABLE
--          FROM ALL_TAB_COLUMNS
--         WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'LST_DOC_TYPE'
--         ORDER BY COLUMN_ID;
--
--    Sekalian catat LEBAR ID. Kode dibentuk `kode_situs || lpad(urutan, 4, '0')`, sehingga
--    penyisipan ke-10000 menghasilkan kode ENAM karakter bila kode situsnya dua karakter.
--    Bila kolomnya sempit, penyisipan itu akan ditolak ORA-12899 — dan keputusan
--    memperlebar kolom sebaiknya diambil sebelum, bukan sesudah, penyisipan pertama yang
--    gagal.
--
-- 3. PERIKSA POSISI URUTAN. Kode yang diterbitkan aplikasi harus MELANJUTKAN deret yang
--    sudah ada, tidak menabraknya. Perhatikan namanya: SET_LST_DOC_TYPE, bukan
--    LST_DOC_TYPE_SEQ.
--
--        SELECT LAST_NUMBER FROM ALL_SEQUENCES
--         WHERE SEQUENCE_OWNER = 'POOLDATA' AND SEQUENCE_NAME = 'SET_LST_DOC_TYPE';
--
--        SELECT MAX(ID) FROM POOLDATA.LST_DOC_TYPE;
--
-- 4. PERIKSA PANJANG NILAI YANG ADA. Langkah 1 akan gagal bila ada nilai yang lebih
--    panjang dari lebar kolom tujuannya:
--
--        SELECT COUNT(*) FROM POOLDATA.LST_DOC_TYPE
--         WHERE LENGTH(JSON_VALUE(JSON_DATA, '$.TYPE_DOCUMENT')) > 200
--            OR LENGTH(JSON_VALUE(JSON_DATA, '$.STS_PROSES'))    > 200
--            OR LENGTH(JSON_VALUE(JSON_DATA, '$.USER_EDIT'))     > 100;
--
--    Yang diharapkan nol baris. Bila tidak nol, lebarkan kolomnya di langkah 1.
--
--    Lebar 200 di bawah adalah PILIHAN, bukan bacaan: layar Pega tidak memasang
--    `pyMaxLength` pada satu pun isian, dan aplikasi baru pun tidak membatasinya
--    (keputusan Work Owner 2026-09-21, layar ini tanpa validasi). Angkanya dipilih longgar
--    supaya isian yang sah tidak tertolak, dan sempit supaya salah tempel sepanjang satu
--    paragraf tidak diam-diam tersimpan.
--
-- 5. PERIKSA BENTUK TGL_EDIT PADA VIEW. Bila view sekarang mengeluarkannya sebagai TEKS
--    hasil TO_CHAR dari JSON, mengubahnya menjadi kolom DATE akan mengubah tipe kolom yang
--    dibaca Pega. Pembaca TGL_EDIT dari export hanya satu — Report Definition yang mengisi
--    grid layar ini sendiri — sehingga risikonya kecil, tetapi ia HARUS diperiksa, bukan
--    diandaikan:
--
--        SELECT JSON_VALUE(JSON_DATA, '$.TGL_EDIT') FROM POOLDATA.LST_DOC_TYPE
--         WHERE ROWNUM <= 5;


-- ---------------------------------------------------------------------------
-- Langkah 1 — tambahkan kolom, lalu pindahkan isi JSON ke dalamnya.
--
-- JANGAN JALANKAN ALTER TABLE bila langkah 0 butir 2 menunjukkan kolomnya sudah ada.
--
-- Backward-compatible: selama langkah 3 belum jalan, view masih membaca JSON_DATA dan Pega
-- tidak terganggu sedetik pun (`P-4`).
--
-- OLD_ID TIDAK ditambahkan di sini. Bila langkah 0 butir 1 menunjukkan ia kolom sungguhan,
-- ia memang sudah ada dan tidak perlu disentuh; bila ia nilai dari JSON, ia jejak sejarah
-- yang tidak dikelola siapa pun — aplikasi baru tidak pernah menulisnya, dan layar Pega
-- pun tidak menampilkannya. Sesuaikan langkah 5 menurut jawabannya.
-- ---------------------------------------------------------------------------

ALTER TABLE POOLDATA.LST_DOC_TYPE ADD (TYPE_DOCUMENT VARCHAR2(200));
ALTER TABLE POOLDATA.LST_DOC_TYPE ADD (STS_PROSES    VARCHAR2(200));
ALTER TABLE POOLDATA.LST_DOC_TYPE ADD (USER_EDIT     VARCHAR2(100));
ALTER TABLE POOLDATA.LST_DOC_TYPE ADD (TGL_EDIT      DATE);

UPDATE POOLDATA.LST_DOC_TYPE
   SET TYPE_DOCUMENT = JSON_VALUE(JSON_DATA, '$.TYPE_DOCUMENT'),
       STS_PROSES    = JSON_VALUE(JSON_DATA, '$.STS_PROSES'),
       USER_EDIT     = JSON_VALUE(JSON_DATA, '$.USER_EDIT')
 WHERE JSON_DATA IS NOT NULL;

COMMIT;

-- TGL_EDIT dipindahkan TERPISAH, dan itu disengaja.
--
-- Nilai di dalam JSON adalah teks, dan bentuknya belum diketahui — ia ditulis
-- `@getCurrentTimeStamp()` Pega, yang menghasilkan cap waktu bergaya Pega
-- ('20260921T030000.000 GMT'), bukan tanggal Oracle. Menjalankannya bersama UPDATE di atas
-- akan membuat SATU baris berformat tak terduga menggagalkan SELURUH pemindahan.
--
-- JALANKAN HANYA SETELAH langkah 0 butir 5 memperlihatkan bentuknya, dan SESUAIKAN topeng
-- formatnya. Yang di bawah mengikuti bentuk cap waktu Pega:
--
--     UPDATE POOLDATA.LST_DOC_TYPE
--        SET TGL_EDIT = TO_DATE(SUBSTR(JSON_VALUE(JSON_DATA, '$.TGL_EDIT'), 1, 15),
--                               'YYYYMMDD"T"HH24MISS')
--      WHERE JSON_DATA IS NOT NULL
--        AND JSON_VALUE(JSON_DATA, '$.TGL_EDIT') IS NOT NULL;
--
--     COMMIT;
--
-- Baris yang gagal diurai DIBIARKAN NULL. TGL_EDIT hanya jejak simpan — tidak ada satu pun
-- aturan bisnis yang membacanya — sehingga kehilangan jejak beberapa baris lama jauh lebih
-- ringan daripada menghentikan migrasi karenanya.


-- ---------------------------------------------------------------------------
-- Langkah 2 — VERIFIKASI. Jangan lanjut bila salah satu jawabannya tidak seperti yang
-- disebutkan. Setelah langkah 3 berjalan, kesalahan di sini tidak dapat lagi dideteksi
-- dengan membandingkan ke view.
-- ---------------------------------------------------------------------------

-- 2a. Berapa baris yang TYPE_DOCUMENT-nya masih kosong padahal JSON-nya terisi?
--     DIHARAPKAN: 0
--
--     SELECT COUNT(*) FROM POOLDATA.LST_DOC_TYPE
--      WHERE JSON_DATA IS NOT NULL AND TYPE_DOCUMENT IS NULL;

-- 2b. Adakah baris yang kolomnya berbeda dari view?  DIHARAPKAN: 0
--
--     SELECT COUNT(*)
--       FROM POOLDATA.LST_DOC_TYPE t
--       JOIN POOLDATA.V_LST_DOC_TYPE v ON v.ID = t.ID
--      WHERE NVL(TRIM(t.TYPE_DOCUMENT), '~') <> NVL(TRIM(v.TYPE_DOCUMENT), '~')
--         OR NVL(TRIM(t.STS_PROSES),    '~') <> NVL(TRIM(v.STS_PROSES),    '~');
--
--     NVL dipakai di sini dengan sengaja meski COALESCE yang portabel: berkas ini
--     dijalankan DBA langsung di Oracle dan tidak pernah ikut ke PostgreSQL.

-- 2c. Berapa baris yang STS_PROSES-nya kosong?  Angka ini TIDAK harus nol.
--
--     SELECT COUNT(*) FROM POOLDATA.LST_DOC_TYPE WHERE TRIM(STS_PROSES) IS NULL;
--
--     Ia catatan teks bebas tanpa kewajiban isi, dan layar Pega tidak pernah mewajibkannya.
--     Laporkan angkanya ke Work Owner sebagai gambaran seberapa sering kolom itu benar-
--     benar dipakai; itu bahan untuk memutuskan apakah kelak ia diberi arti yang tegas.


-- ---------------------------------------------------------------------------
-- Langkah 3 — definisikan ulang view supaya membaca kolom, bukan JSON.
--
-- Inilah pernyataan yang menyentuh Pega. Sesudah ini, apa yang ditulis aplikasi Go
-- langsung terlihat oleh sekurang-kurangnya sepuluh rule yang membaca view ini.
--
-- Tiga hal yang WAJIB dipertahankan:
--
--   * URUTAN KOLOM mengikuti definisi yang disimpan pada langkah 0 butir 1. Yang di bawah
--     mengikuti urutan yang terbaca di `Report Definition/BrowseLstDocType_RD-RD.xml`.
--     Menukarnya akan mengubah hasil setiap pembacaan posisional.
--   * DAFTAR NAMA KOLOM ditulis eksplisit; tanpa itu nama kolom hasil ekspresi akan
--     berubah dan rule Pega kehilangan kolom yang dicarinya.
--   * CREATE OR REPLACE, bukan DROP lalu CREATE. Yang pertama mempertahankan seluruh grant;
--     yang kedua menghapusnya dan membuat Pega kehilangan hak baca.
--
-- BILA OLD_ID ternyata BUKAN kolom pada tabel dasar (lihat langkah 0 butir 1), ganti
-- barisnya menjadi `JSON_VALUE(JSON_DATA, '$.OLD_ID')` — dan sadari akibatnya: ia berhenti
-- mutakhir sejak aplikasi Go menulis. Itu dapat diterima karena OLD_ID adalah jejak sejarah
-- yang tidak pernah diisi baris baru, baik oleh Pega maupun oleh aplikasi ini.
--
-- ## Akibat yang HARUS disetujui sebelum langkah ini dijalankan
--
-- JSON_DATA menjadi USANG sejak aplikasi Go menulis. Tidak ada pembaca yang terganggu
-- karenanya — berbeda dari migrasi 0004, di mana satu layar Pega mem-parse kolom JSON-nya
-- secara langsung. Di sini seluruh pembaca membaca KOLOM lewat view, dan kolom itu justru
-- menjadi mutakhir oleh langkah ini.
--
-- Yang tetap berlaku: langkah 3 adalah TITIK CUTOVER layar Daftar Tipe Dokumen. Jangan
-- jalankan sebelum modulnya lulus gerbang 2.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE VIEW POOLDATA.V_LST_DOC_TYPE
    (ID, OLD_ID, TYPE_DOCUMENT, STS_PROSES, USER_EDIT, TGL_EDIT) AS
SELECT ID,
       OLD_ID,
       TYPE_DOCUMENT,
       STS_PROSES,
       USER_EDIT,
       TGL_EDIT
  FROM POOLDATA.LST_DOC_TYPE;


-- ---------------------------------------------------------------------------
-- Langkah 4 — hak akses untuk akun aplikasi.
--
-- Diberikan sesempit mungkin: SELECT, INSERT, dan UPDATE. TANPA DELETE — tidak ada satu pun
-- jalur di aplikasi yang menghapus baris master (`D-66`), dan hak yang tidak diberikan
-- tidak dapat disalahgunakan kode yang ditulis kemudian.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya, dan jalankan di SETIAP basis data
-- entitas.
-- ---------------------------------------------------------------------------

-- GRANT SELECT, INSERT, UPDATE ON POOLDATA.LST_DOC_TYPE     TO <AKUN_APLIKASI>;
-- GRANT SELECT                  ON POOLDATA.M_SITE_DATABASE TO <AKUN_APLIKASI>;
-- GRANT SELECT                  ON POOLDATA.SET_LST_DOC_TYPE TO <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 5 — JSON_DATA setelah migrasi ini.
--
-- Kolomnya SENGAJA TIDAK DIHAPUS dan tidak dikosongkan. Ia dibiarkan berisi nilai terakhir
-- yang ditulis Pega, sebagai bahan pembanding bila ada yang meragukan hasil perpindahan
-- ini, dan sebagai satu-satunya bahan untuk migrasi turun.
--
-- Kapan JSON_DATA boleh dibuang adalah keputusan tersendiri, dan sebaiknya diambil setelah
-- masa pengamatan berjalan — bukan di berkas ini. Perlakuannya sama dengan JSONDATA pada
-- migrasi 0002 dan JSON_DATA pada migrasi 0004.
