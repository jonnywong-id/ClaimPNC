-- 0002 — Master Status Klaim: label pindah dari JSON ke kolom (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Berbeda dari migrasi 0001 yang hanya MENAMBAH dua tabel baru, berkas ini MENGUBAH
-- objek milik sistem lama yang sedang melayani produksi:
--
--   * POOLDATA.M_STS_CLAIM  — kolom LSC_NOTE diisi (kolomnya sudah ada, masih kosong)
--   * POOLDATA.V_STS_CLAIM  — view-nya didefinisikan ulang
--
-- View itu dibaca 23 rule Pega (pencarian klaim, laporan TAT, laporan KPI). Bila
-- definisinya salah, yang rusak bukan layar master ini melainkan laporan yang dibaca
-- manajemen — dan rusaknya tidak menimbulkan galat, hanya kolom status yang kosong.
--
-- Menjalankannya menuntut permintaan perubahan skema tertulis, persetujuan Work Owner,
-- pelaksanaan oleh DBA, dan pengujian dengan MENJALANKAN PEGA DAN GO BERSAMAAN terhadap
-- skema hasil perubahan (D-63). Akun aplikasi tidak memiliki hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
--
-- ## Keadaan skema yang sebenarnya, diverifikasi 2026-09-17
--
-- Berkas ini mula-mula ditulis dengan menebak isi skema dari source procedure. Tebakan
-- itu SALAH di dua tempat, dan keduanya akan membuat migrasi gagal di langkah pertama.
-- Yang di bawah dibaca langsung dari katalog basis data:
--
--   POOLDATA.M_STS_CLAIM
--     LSC_ID      CHAR(4)        NOT NULL   <- kunci utama, constraint M_STS_CLAIM_PK
--     JSONDATA    CLOB           NULL       <- terisi pada seluruh 32 baris
--     OLD_LSC_ID  CHAR(4)        NULL       <- terisi pada 11 baris
--     LSC_NOTE    VARCHAR2(100)  NULL       <- SUDAH ADA, tetapi KOSONG pada 32 baris
--
--   POOLDATA.V_STS_CLAIM, definisi sekarang:
--     SELECT lsc_id, old_lsc_id, json_value (jsondata, '$.LSC_NOTE') FROM m_sts_claim
--
-- Tiga akibatnya:
--
--   1. TIDAK ADA ALTER TABLE di berkas ini. Kedua kolom sudah ada; menambahkannya akan
--      gagal dengan ORA-01430 dan menghentikan seluruh migrasi di baris pertama.
--   2. Kunci JSON-nya pasti: '$.LSC_NOTE'. Tidak ada lagi yang perlu ditebak.
--   3. Kolom LSC_NOTE pada tabel sudah ada tetapi TIDAK PERNAH DIISI siapa pun —
--      seseorang menyiapkannya lalu berhenti di situ. Migrasi ini yang mengisinya.
--
--
-- ## Kenapa perubahan ini diminta
--
-- Sistem lama menyimpan label status di dalam dokumen JSON (M_STS_CLAIM.JSONDATA), lalu
-- membongkarnya kembali lewat view. Yang menulisnya adalah procedure
-- POOLDATA.PEGA_M_STS_CLAIM.
--
-- Tiga keputusan yang sudah disetujui menutup jalan itu:
--
--   D-02  Logika stored procedure naik ke Go; aplikasi tidak memanggil procedure.
--   D-68  Kepemilikan transaksi pindah ke Go. Kontrak galat procedure lama tidak
--         dibawa: parameter keluarannya bernama ErrMsg tetapi pada jalur BERHASIL ia
--         berisi kalimat "Data Sudah Disimpan dengan ID : 1167", sehingga pemanggil
--         tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
--   —     Work Owner 2026-09-17: Go menjadi penulis tunggal M_STS_CLAIM, tidak lagi
--         menulis JSONDATA, dan isi JSON yang ada dipindahkan ke kolom.
--
-- P-1 tetap dipatuhi — satu tabel satu penulis. Layar Master Status Klaim adalah
-- SATU-SATUNYA penulis tabel ini di sistem lama (RDB List/UpdateStsClaim-SQL.xml),
-- sehingga memindahkan layarnya memindahkan kepemilikan tabelnya secara utuh. Pega
-- berubah menjadi pembaca saja.
--
--
-- ## Yang HARUS dilakukan DBA sebelum menjalankan
--
-- 1. SIMPAN DEFINISI VIEW YANG SEKARANG, beserta daftar kolomnya. Migrasi turun
--    membutuhkannya, dan ALL_VIEWS.TEXT tidak menyimpan daftar kolom view:
--
--        SELECT DBMS_METADATA.GET_DDL('VIEW', 'V_STS_CLAIM', 'POOLDATA') FROM DUAL;
--
--    Simpan hasilnya ke berkas dan lampirkan pada permintaan perubahan skema.
--
-- 2. PASTIKAN TIDAK ADA LABEL GANDA. Langkah 4 membuat indeks unik dan akan GAGAL bila
--    ada dua status berlabel sama:
--
--        SELECT UPPER(TRIM(LSC_NOTE)) AS label, COUNT(*) AS jumlah
--          FROM POOLDATA.V_STS_CLAIM
--         GROUP BY UPPER(TRIM(LSC_NOTE))
--        HAVING COUNT(*) > 1;
--
--    Yang diharapkan nol baris. Pemeriksaan 2026-09-17 atas 32 baris: nol.
--
-- 3. PASTIKAN TIDAK ADA LABEL YANG LEBIH PANJANG DARI 100 KARAKTER. Kolom tujuannya
--    VARCHAR2(100), sedangkan JSON_VALUE mengembalikan sampai 4000:
--
--        SELECT LSC_ID FROM POOLDATA.V_STS_CLAIM WHERE LENGTH(LSC_NOTE) > 100;
--
--    Yang diharapkan nol baris. Label terpanjang saat diperiksa: 27 karakter.


-- ---------------------------------------------------------------------------
-- Langkah 1 — pindahkan label dari JSON ke kolom.
--
-- Kunci '$.LSC_NOTE' diambil dari definisi view yang sekarang, bukan ditebak — ia
-- SATU-SATUNYA sumber yang tahu bentuk dokumen JSON-nya.
--
-- Backward-compatible: selama langkah 2 belum jalan, view masih membaca JSONDATA dan
-- Pega tidak terganggu sedetik pun (P-4).
-- ---------------------------------------------------------------------------

UPDATE POOLDATA.M_STS_CLAIM
   SET LSC_NOTE = JSON_VALUE(JSONDATA, '$.LSC_NOTE')
 WHERE JSONDATA IS NOT NULL;

COMMIT;


-- ---------------------------------------------------------------------------
-- Langkah 2 — VERIFIKASI. Jangan lanjut bila salah satu jawabannya tidak seperti yang
-- disebutkan. Setelah langkah 3 berjalan, kesalahan di sini tidak dapat lagi dideteksi
-- dengan membandingkan ke view.
-- ---------------------------------------------------------------------------

-- 2a. Berapa baris yang labelnya masih kosong?  DIHARAPKAN: 0
--
--     SELECT COUNT(*) FROM POOLDATA.M_STS_CLAIM WHERE LSC_NOTE IS NULL;

-- 2b. Adakah baris yang kolomnya berbeda dari view?  DIHARAPKAN: 0
--
--     SELECT COUNT(*)
--       FROM POOLDATA.M_STS_CLAIM m
--       JOIN POOLDATA.V_STS_CLAIM v ON v.LSC_ID = m.LSC_ID
--      WHERE NVL(TRIM(m.LSC_NOTE), '~') <> NVL(TRIM(v.LSC_NOTE), '~');
--
--     NVL dipakai di sini dengan sengaja meski COALESCE yang portabel: berkas ini
--     dijalankan DBA langsung di Oracle dan tidak pernah ikut ke PostgreSQL.

-- 2c. Berapa jumlah barisnya?  DIHARAPKAN: 32 per 2026-09-17
--
--     SELECT COUNT(*) FROM POOLDATA.M_STS_CLAIM;
--
--     CATATAN: berkas Database/v_sts_claim.csv yang diekspor Work Owner memuat 33
--     baris — termasuk 1165 "Rejected Chasier", yang TIDAK ADA di basis data ini.
--     Selisih itu belum dijelaskan dan tidak diperbaiki migrasi ini. Lihat catatan
--     pengembangan; ia pertanyaan untuk Work Owner, bukan sesuatu yang layak ditambal
--     diam-diam oleh DBA.


-- ---------------------------------------------------------------------------
-- Langkah 3 — definisikan ulang view supaya membaca kolom, bukan JSON.
--
-- Inilah pernyataan yang menyentuh Pega. Sesudah ini, apa yang ditulis aplikasi Go
-- langsung terlihat oleh 23 rule yang membaca view ini.
--
-- Tiga hal yang WAJIB dipertahankan, dan semuanya sudah diperiksa terhadap view yang
-- sekarang:
--
--   * URUTAN KOLOM: LSC_ID, OLD_LSC_ID, LSC_NOTE. Perhatikan LSC_NOTE di urutan
--     KETIGA, bukan kedua. Menukarnya akan mengubah hasil setiap pembacaan posisional.
--   * DAFTAR NAMA KOLOM ditulis eksplisit. View yang sekarang memakainya — tanpa itu,
--     kolom ketiga akan bernama "JSON_VALUE(JSONDATA,'$.LSC_NOTE')" dan seluruh rule
--     Pega kehilangan kolom LSC_NOTE.
--   * CREATE OR REPLACE, bukan DROP lalu CREATE. Yang pertama mempertahankan seluruh
--     grant; yang kedua menghapusnya dan membuat Pega kehilangan hak baca.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE VIEW POOLDATA.V_STS_CLAIM (LSC_ID, OLD_LSC_ID, LSC_NOTE) AS
SELECT LSC_ID,
       OLD_LSC_ID,
       LSC_NOTE
  FROM POOLDATA.M_STS_CLAIM;


-- ---------------------------------------------------------------------------
-- Langkah 4 — indeks unik atas label.
--
-- Keunikan label adalah KEBUTUHAN BARU yang diputuskan Work Owner 2026-09-17; sistem
-- lama tidak memvalidasi apa pun pada layar ini, dan satu-satunya constraint yang ada
-- di tabel sampai hari ini adalah kunci utama M_STS_CLAIM_PK.
--
-- Indeksnya atas UPPER(TRIM(...)), bukan atas kolom apa adanya, supaya "Paid" dan
-- "PAID  " dikenali sebagai label yang sama. Ekspresi ini HARUS sama persis dengan
-- masterstatus.LabelKey di kode Go — bila keduanya berbeda, aplikasi akan menerima
-- label yang kemudian ditolak basis data.
--
-- Namanya dipakai kode Go untuk menerjemahkan galat bentrok menjadi pesan yang dapat
-- dibaca pengguna (konstanta NamaIndeksLabel di
-- internal/masterstatus/repo/sqlstore/masterstatus.go). Mengganti namanya di sini tanpa
-- mengganti konstanta itu akan membuat bentrok label muncul sebagai galat 500.
-- ---------------------------------------------------------------------------

CREATE UNIQUE INDEX POOLDATA.UX_M_STS_CLAIM_LABEL
    ON POOLDATA.M_STS_CLAIM (UPPER(TRIM(LSC_NOTE)));


-- ---------------------------------------------------------------------------
-- Langkah 5 — hak akses untuk akun aplikasi.
--
-- Sampai sekarang akun aplikasi hanya perlu MEMBACA tabel warisan. Modul ini yang
-- pertama kali menulis ke salah satunya, dan haknya diberikan sesempit mungkin:
-- SELECT, INSERT, dan UPDATE. TANPA DELETE — tidak ada satu pun jalur di aplikasi yang
-- menghapus baris master, dan hak yang tidak diberikan tidak dapat disalahgunakan kode
-- yang ditulis kemudian.
--
-- Hak baca M_STS_CLAIM sudah terbukti ada (mode periksa berhasil membacanya pada
-- 2026-09-17); yang belum tentu ada adalah hak tulis dan hak atas urutan.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya.
-- ---------------------------------------------------------------------------

-- GRANT SELECT, INSERT, UPDATE ON POOLDATA.M_STS_CLAIM TO <AKUN_APLIKASI>;
-- GRANT SELECT ON POOLDATA.M_SITE_DATABASE TO <AKUN_APLIKASI>;
-- GRANT SELECT ON POOLDATA.M_STS_CLAIM_SEQ TO <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 6 — JSONDATA setelah migrasi ini.
--
-- Kolomnya SENGAJA TIDAK DIHAPUS dan tidak dikosongkan. Ia dibiarkan berisi nilai
-- terakhir yang ditulis Pega, sebagai bahan pembanding bila ada yang meragukan hasil
-- perpindahan ini, dan sebagai satu-satunya bahan untuk migrasi turun.
--
-- Akibat yang harus disadari: sejak aplikasi Go menulis, JSONDATA menjadi USANG dan
-- akan berbeda dari kolom. Siapa pun yang membacanya akan mendapat nilai lama. Tidak
-- ada rule Pega yang membacanya langsung — satu-satunya yang menyentuhnya adalah
-- procedure PEGA_M_STS_CLAIM, yang sejak sekarang tidak dipanggil siapa pun.
--
-- Kapan JSONDATA boleh dibuang adalah keputusan tersendiri, dan sebaiknya diambil
-- setelah masa pengamatan berjalan — bukan di berkas ini.
--
--
-- ## Satu batas yang perlu diketahui sebelum menyetujui
--
-- LSC_ID bertipe CHAR(4), dan kode dibentuk kode_situs || lpad(urutan, 3, '0').
-- POOLDATA.M_STS_CLAIM_SEQ berada di 193 pada 2026-09-17, sementara kode tertinggi yang
-- terpakai baru 1166. Saat urutan mencapai 1000, kodenya menjadi lima karakter dan
-- penyisipan akan DITOLAK dengan ORA-12899.
--
-- Itu sekitar 806 penambahan lagi — tidak mendesak, tetapi keputusan memperlebar kolom
-- sebaiknya diambil sebelum penyisipan pertama yang gagal, bukan sesudahnya.
