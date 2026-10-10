-- Kueri AKSI TULIS modul Monitoring SLINK OJK.
--
-- ============================================================================
-- BERKAS INI MENULIS — DAN ITU KEPUTUSAN YANG DISEBUT NAMANYA
-- ============================================================================
--
-- `POOLDATA.T_CLAIM_SLIK_OJK` juga diisi sistem lama pada jalur AKSEPTASI
-- (`InsertAdjustmentList`, `InsertAdjustmentListKredit`). Selama masa paralel, `P-1`
-- menetapkan satu tabel hanya boleh ditulis satu sistem.
--
-- Work Owner memutuskan pada 2026-09-26 bahwa ketiga tombol tulis layar ini tetap
-- dibangun, sesudah konsekuensinya disampaikan. Berkas ini karena itu MEMANG memuat
-- INSERT dan UPDATE; larangan menulis yang diuji di query_test.go berlaku untuk
-- `monitoringslinkojk.sql` saja, bukan untuk berkas ini.
--
-- ============================================================================
-- ASAL SETIAP PERNYATAAN
-- ============================================================================
--
--   report_count      RDB List/GetCountTClaimSlikOJK-SQL.xml
--   report_insert     RDB List/InsertDataSlikOJKF06-SQL.xml
--   source_rows       RDB List/GetDataSlinkAllFOG-SQL.xml
--   submission_next   RDB List/QuerySLINKIndividu-SQL.xml   (pyBrowseSQL)
--   submission_insert RDB List/QuerySLINKIndividu-SQL.xml   (pySaveSQL)
--   submission_done   RDB List/UpdateTransactionClaimSlinkIndividu-SQL.xml

-- Pencacah yang menentukan `operasidata` — 'U' bila > 0, 'C' bila 0.
--
-- Aliasnya di Pega `"CityID"`, salah satu alias paling menyesatkan di area ini: ia
-- menampung JUMLAH BARIS, bukan kode kota. Namanya tidak dibawa.

-- name: report_count
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_SLIK_OJK
 WHERE CLAIMID = :1

-- Menyusun satu baris laporan.
--
-- Kedua puluh delapan kolomnya disalin APA ADANYA dari `InsertDataSlikOJKF06`, dalam
-- urutan yang sama, supaya keduanya dapat dibaca berdampingan saat di-review.
--
-- Ia MENAMBAH, tidak menimpa. Itu perilaku sistem lama: baris berulang ditandai
-- `operasidata = 'U'`, bukan menggantikan baris sebelumnya — sehingga riwayat pelaporan
-- satu klaim tetap utuh.

-- name: report_insert
INSERT INTO POOLDATA.T_CLAIM_SLIK_OJK (
       NOREKFASILITAS, NOCIFDEBITUR, KODEJENISFASILITAS, SUMBERDANA,
       TANGGALMULAI, TANGGALAKHIR, SUKUBUNGA, KODEVALUTA,
       NOMINAL, NILAIMATAUANGASAL, KODEKOLEKTABILITAS, TANGGALMACET,
       KODESEBABMACET, TUNGGAKAN, JUMLAHHARITUNGGAKAN, TANGGALKONDISI,
       KODEKONDISI, KODEKANTORCABANG, OPERASIDATA, KETERANGAN,
       BULANLAPOR, CLAIMID, CONTRACTNO, RECOVERYCLAIM,
       NOKTP, NPWPPERUSAHAAN, NOPOLIS, CLIENTID)
VALUES (:1, :2, :3, :4,
        :5, :6, :7, :8,
        :9, :10, :11, :12,
        :13, :14, :15, :16,
        :17, :18, :19, :20,
        :21, :22, :23, :24,
        :25, :26, :27, :28)

-- ============================================================================
-- DATA KLAIM SUMBER — tombol "Proses Data Klaim"
-- ============================================================================
--
-- Disalin dari `GetDataSlinkAllFOG-SQL.xml`, dengan empat perubahan yang seluruhnya
-- mempertahankan hasilnya:
--
--   1. Penyaring `{ASIS:…}` diganti parameter binding.
--   2. `to_date()` pada kolom tanggal dihapus; batasnya dibandingkan langsung.
--   3. `rownum AS "JumlahHariTunggakan"` DIBUANG — ia mengisi kolom "jumlah hari
--      tunggakan" dengan NOMOR URUT BARIS, angka yang berubah setiap kali urutannya
--      berubah dan tidak ada hubungannya dengan tunggakan. Nilainya dikirim KOSONG.
--   4. `CASE` bersarang lima pembacaan tabel polis diratakan menjadi satu `EXISTS`.
--
-- Perubahan 3 adalah satu-satunya yang mengubah NILAI, dan ia mengosongkan kolom yang
-- isinya memang tidak bermakna. Dicatat sebagai calon butir `P-5`.
--
-- ---------------------------------------------------------------------------
-- DUA KOLOM YANG TIDAK DAPAT DIISI DARI SINI
-- ---------------------------------------------------------------------------
--
-- `NOKTP` dan `NPWPPERUSAHAAN` TIDAK ada di `GetDataSlinkAllFOG` maupun di
-- `T_CLAIM_OBJECTLIST`. Pada jalur akseptasi keduanya diisi dari data klaim yang tidak
-- dilewati layar ini. Keduanya karena itu dikirim KOSONG, dan itu perbedaan nyata
-- terhadap baris yang disusun jalur akseptasi.

-- name: source_rows
SELECT k.PYID                AS CLAIM_ID,
       o.CONTRACTNO          AS CONTRACT_NO,
       o.NOMORREKENINGFASILITAS AS NOREK_FASILITAS,
       o.NOMORCIFDEBITUR     AS NOCIF_DEBITUR,
       o.KODEJENISFASILITAS  AS KODE_JENIS_FASILITAS,
       o.SUMBERDANA          AS SUMBER_DANA,
       (SELECT MAX(g.STARTDATE) FROM POOLDATA.T_GENERAL g WHERE g.NOPOLIS = k.POLICYNO)
                             AS TANGGAL_MULAI,
       (SELECT MAX(g.ENDDATE) FROM POOLDATA.T_GENERAL g WHERE g.NOPOLIS = k.POLICYNO)
                             AS TANGGAL_AKHIR,
       o.SUKUBUNGA           AS SUKU_BUNGA,
       (SELECT pnc.CURRENCY FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY AND pnc.NOPOLIS = k.POLICYNO)
                             AS KODE_VALUTA,
       (SELECT MAX(c.SUMTSI) FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
         WHERE c.CLAIMID = o.CLAIMID AND c.OBJECTID = o.OBJECTID
           AND c.CONTRACTNO = o.CONTRACTNO)
                             AS JUMLAH_KEWAJIBAN,
       o.KODEKOLEKTIBILITAS  AS KODE_KOLEKTIBILITAS,
       (SELECT pnc.REGISTERDATE FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY AND pnc.NOPOLIS = k.POLICYNO)
                             AS TANGGAL_MACET,
       o.KODESEBABMACET      AS KODE_SEBAB_MACET,
       (SELECT SUM(ad.NILAIAKSEPTASI) FROM POOLDATA.T_CLAIM_ADJUSTMENT ad
         WHERE ad.CLAIMID = o.CLAIMID AND ad.OBJECTID = o.OBJECTID
           AND ad.CONTRACTNO = o.CONTRACTNO)
                             AS TUNGGAKAN,
       o.KODEKONDISI         AS KODE_KONDISI,
       o.KETERANGAN          AS KETERANGAN,
       :1                    AS KODE_KANTOR_CABANG,
       k.POLICYNO            AS NO_POLIS
  FROM POOLDATA.T_CLAIM_OBJECTLIST o
  JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PZINSKEY = o.CLAIMID
 WHERE (:2 IS NULL OR k.REGISTERDATE_1 >= :3)
   AND (:4 IS NULL OR k.REGISTERDATE_1 < :5)
   AND (:6 IS NULL
        OR (:7 = 'INCLUDE' AND UPPER(TRIM(k.BUSINESSNAME)) LIKE :8)
        OR (:9 = 'EXCLUDE' AND (UPPER(TRIM(k.BUSINESSNAME)) NOT LIKE :10 OR k.BUSINESSNAME IS NULL)))
 ORDER BY k.REGISTERDATE_1 DESC, o.CLAIMID, o.OBJECTID, o.CONTRACTNO

-- ============================================================================
-- PENGIRIMAN KE SLIK — tombol "SLIK OJK"
-- ============================================================================

-- Nomor urut pengiriman berikutnya.
--
-- `nvl(max(id),0)+1` disalin apa adanya dari `QuerySLINKIndividu`. Ia BUKAN sequence,
-- dan karena itu tidak aman terhadap dua pengiriman bersamaan: keduanya dapat memperoleh
-- nomor yang sama. Cacat itu ada di sistem lama dan direplikasi; penggantinya sequence,
-- dan itu keputusan tersendiri yang belum diambil.

-- name: submission_next
SELECT NVL(MAX(ID), 0) + 1 AS NEXT_ID
  FROM POOLDATA.T_CLAIM_SLINK_INDIVIDU

-- Mencatat pengiriman SEBELUM dikirim.
--
-- Urutannya mengikuti sistem lama, dan itu yang membuat pengiriman gagal tetap
-- meninggalkan jejak: baris tanpa `ID_TRANSACTION` adalah pengiriman yang tidak sampai.

-- name: submission_insert
INSERT INTO POOLDATA.T_CLAIM_SLINK_INDIVIDU (NO_KLAIM, CONTRACT_NO, ID)
VALUES (:1, :2, :3)

-- Menyimpan jawaban sistem SLIK.

-- name: submission_done
UPDATE POOLDATA.T_CLAIM_SLINK_INDIVIDU
   SET CLIENTID = :1,
       ID_TRANSACTION = :2
 WHERE ID = :3
   AND NO_KLAIM = :4

-- ============================================================================
-- DATA DEBITUR UNTUK PENDAFTARAN KLIEN
-- ============================================================================
--
-- Muatan tombol "SLIK OJK" bukan isi laporan SLIK melainkan IDENTITAS debiturnya.
--
-- Sistem lama membacanya dari `pyWorkPagee.ClaimData.PolicyData.CIFData` — CIF pada
-- snapshot polis. Di sini sumbernya `T_CLAIM_OBJECTLIST`, yang memuat kolom padanannya dan
-- sudah dibaca kueri segmen F06 (`GetDataSlinkAllFOG-SQL.xml`).
--
-- Itu PILIHAN yang dicatat, bukan kesetaraan yang terbukti. Tiga field yang Pega kirim
-- tidak punya padanan di sini: NIK (`ASMIDCard`), NPWP, dan nama ibu kandung. Ketiganya
-- dibiarkan KOSONG — tidak diisi dari kolom lain yang kebetulan mirip.
--
-- Penerimanya hanya mewajibkan satu field per cabang, sehingga muatan yang lebih miskin
-- tetap diterima. Yang hilang adalah kelengkapan, bukan keberhasilan.

-- name: debtor_row
SELECT o.OBJECTNAME      AS NAMA,
       o.CUSTOMERTYPE    AS CUSTOMER_TYPE,
       o.OBJECTGENDER    AS JENIS_KELAMIN,
       o.DATEOFBIRTH     AS TANGGAL_LAHIR,
       o.ASMADDRESS      AS ALAMAT,
       o.ASMZIPCODE      AS KODE_POS,
       o.TELFAXNUMBER    AS TELEPON
  FROM POOLDATA.T_CLAIM_OBJECTLIST o
  JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PZINSKEY = o.CLAIMID
 WHERE k.PYID = :1
   AND o.CONTRACTNO = :2
 ORDER BY o.OBJECTID
 FETCH FIRST 1 ROWS ONLY
