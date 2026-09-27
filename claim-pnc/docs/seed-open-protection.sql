-- ============================================================================
-- Data uji POOLDATA.T_CLAIM_OPENPROTECTION
-- ============================================================================
--
-- BUKAN migrasi. Berkas ini sengaja diletakkan di docs/, bukan di migrations/,
-- supaya tidak pernah ikut terjalankan otomatis. Dijalankan tangan, di
-- lingkungan pengembangan saja.
--
-- ============================================================================
-- PERINGATAN: BERKAS INI BELUM DAPAT DIJALANKAN
-- ============================================================================
--
-- Nomor klaim contoh di bawah berbentuk PNCN.26.90xx -- DUA BELAS huruf, mengikuti D-71.
-- Kolom CLAIM_NO sejak revisi 2026-09-24 hanya VARCHAR2(10), sehingga setiap INSERT yang
-- mengisinya akan ditolak:
--
--     ORA-12899: value too large for column "POOLDATA"."T_CLAIM_OPENPROTECTION"."CLAIM_NO"
--
-- Jalankan lebih dulu:
--
--     ALTER TABLE POOLDATA.T_CLAIM_OPENPROTECTION
--         MODIFY (CLAIM_NO VARCHAR2(32 BYTE));
--
-- Alasannya di docs/kolom-open-protection.md §1. Nomor contoh di sini SENGAJA tidak
-- dipendekkan agar muat: memendekkannya akan membuat berkas ini lulus sambil
-- menyembunyikan bahwa kolomnya memang terlalu sempit untuk nomor klaim yang sesungguhnya.
--
-- ============================================================================
-- SELURUH NILAI DI SINI KARANGAN
-- ============================================================================
--
-- Tidak ada satu pun nomor polis, nomor klaim, atau nama nasabah sungguhan
-- (`D-69`). Awalan `UJI-` dipilih supaya baris uji dapat dibedakan dan dibuang
-- dengan satu penyaring.
--
-- ============================================================================
-- KENAPA SYS_EXTRACT_UTC, BUKAN SYSTIMESTAMP
-- ============================================================================
--
-- Aplikasi menulis CREATE_DATE dalam **UTC** (`at.UTC()` di protection.go), dan
-- kolomnya TIMESTAMP tanpa zona. Menyeed dengan SYSTIMESTAMP menaruh waktu
-- server, sehingga baris uji melenceng 7 jam dari baris yang dibuat aplikasi.
--
-- Selisih itu BUKAN sekadar tampilan: pemeriksaan proteksi ganda membandingkan
-- `CAST(CREATE_DATE AS DATE)` dengan tanggal dari aplikasi, sehingga baris yang
-- diseed sore hari WIB akan terhitung sebagai HARI BERIKUTNYA dan pengujian
-- duplikatnya diam-diam tidak membuktikan apa pun.
--
-- ============================================================================
-- APPROVAL_STATUS DIBIARKAN TIDAK DISEBUT
-- ============================================================================
--
-- Pada baris yang belum diakseptasi, kolomnya sengaja tidak masuk daftar
-- sehingga jatuh ke NULL. Mengisinya dengan teks kosong ('') di Oracle memang
-- juga menjadi NULL — tetapi menuliskannya mengaburkan maksud, dan di
-- PostgreSQL kelak teks kosong BUKAN NULL.


-- ── 1. PREMI, lengkap, menunggu keputusan ───────────────────────────────────
-- Muncul di daftar DAN di antrean PREMI. Tautan suntingnya MATI karena sudah
-- tertaut klaim.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0001', SYS_EXTRACT_UTC(SYSTIMESTAMP), 'UJICOBA01',
   'UJI-POLIS-0001', 'PNCN.26.9001', 'PNCN.26.9001',
   '2', 'Uji antrean PREMI', 'OBJEK UJI SATU', 'CABANG UJI', '1');


-- ── 2. Perubahan DOL, lengkap, menunggu keputusan ───────────────────────────
-- OLD_DATA / NEW_DATA berisi TANGGAL berformat YYYY-MM-DD; itulah bentuk yang
-- dibaca decodeChangeDetail. Bentuk lain tidak menggagalkan daftar — ia hanya
-- tampil kosong.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, OLD_DATA, NEW_DATA, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0002', SYS_EXTRACT_UTC(SYSTIMESTAMP), 'UJICOBA01',
   'UJI-POLIS-0002', 'PNCN.26.9002', 'PNCN.26.9002',
   '7', 'Uji perubahan DOL', '2026-09-01', '2026-09-05',
   'OBJEK UJI DUA', 'CABANG UJI', '1');


-- ── 3. Perubahan Cause of Loss, lengkap, menunggu keputusan ─────────────────
-- Untuk tipe '8', kedua kolom berisi KODE penyebab kerugian, bukan tanggal.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, OLD_DATA, NEW_DATA, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0003', SYS_EXTRACT_UTC(SYSTIMESTAMP), 'UJICOBA02',
   'UJI-POLIS-0003', 'PNCN.26.9003', 'PNCN.26.9003',
   '8', 'Uji perubahan penyebab kerugian', '12002', '12005',
   'OBJEK UJI TIGA', 'CABANG UJI', '1');


-- ── 4. Tipe 9 — kode yang labelnya TIDAK diketahui ──────────────────────────
-- Tipe ini nyata dipakai produksi (25 baris) tetapi nol kemunculan di export
-- Pega. Layar harus menampilkan "9" apa adanya, bukan kosong dan bukan tebakan.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0004', SYS_EXTRACT_UTC(SYSTIMESTAMP), 'UJICOBA02',
   'UJI-POLIS-0004', 'PNCN.26.9004', 'PNCN.26.9004',
   '9', 'Uji tipe tanpa label', 'OBJEK UJI EMPAT', 'CABANG UJI', '1');


-- ── 5. BELUM tertaut klaim — satu-satunya baris yang MASIH DAPAT DISUNTING ──
-- Muncul di daftar dengan tautan sunting HIDUP, dan TIDAK pernah muncul di
-- antrean akseptasi. Inilah pembeda kedua layar.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO,
   PROTECTION_TYPE_ID, NOTES, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0005', SYS_EXTRACT_UTC(SYSTIMESTAMP), 'UJICOBA01',
   'UJI-POLIS-0005',
   '2', 'Uji permintaan yang belum tertaut klaim', 'OBJEK UJI LIMA',
   'CABANG UJI', '1');


-- ── 6. Tanpa nomor polis — lolos daftar, TERTAHAN di antrean ────────────────
-- Membuktikan syarat POLICY_NO IS NOT NULL benar-benar dijalankan, bukan hanya
-- tertulis.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0006', SYS_EXTRACT_UTC(SYSTIMESTAMP), 'UJICOBA02',
   'PNCN.26.9006', 'PNCN.26.9006',
   '7', 'Uji tanpa nomor polis', 'OBJEK UJI ENAM', 'CABANG UJI', '1');


-- ── 7. SUDAH DISETUJUI — tidak muncul di mana pun ───────────────────────────
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, APPROVAL_STATUS, RESOLVED_BY, RESOLVED_DATETIME,
   OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0007', SYS_EXTRACT_UTC(SYSTIMESTAMP) - INTERVAL '2' DAY, 'UJICOBA01',
   'UJI-POLIS-0007', 'PNCN.26.9007', 'PNCN.26.9007',
   '2', 'Uji sudah disetujui', '1', 'UJIAKSEP01',
   SYS_EXTRACT_UTC(SYSTIMESTAMP) - INTERVAL '1' DAY,
   'OBJEK UJI TUJUH', 'CABANG UJI', '1');


-- ── 8. SUDAH DITOLAK — tidak muncul di mana pun ─────────────────────────────
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, APPROVAL_STATUS, RESOLVED_BY, RESOLVED_DATETIME,
   OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0008', SYS_EXTRACT_UTC(SYSTIMESTAMP) - INTERVAL '3' DAY, 'UJICOBA02',
   'UJI-POLIS-0008', 'PNCN.26.9008', 'PNCN.26.9008',
   '7', 'Uji sudah ditolak', '2', 'UJIAKSEP01',
   SYS_EXTRACT_UTC(SYSTIMESTAMP) - INTERVAL '1' DAY,
   'OBJEK UJI DELAPAN', 'CABANG UJI', '1');


-- ── 9. TERHAPUS LUNAK — tidak muncul di mana pun ────────────────────────────
-- Barisnya tetap ada di tabel (`D-66`), tetapi tidak pernah tampil. Ini yang
-- membuat perbandingan COUNT(*) tabel TIDAK dapat dipakai sebagai uji
-- kesetaraan (`14-TESTING-STRATEGY.md` §6.4).
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0009', SYS_EXTRACT_UTC(SYSTIMESTAMP), 'UJICOBA01',
   'UJI-POLIS-0009', 'PNCN.26.9009', 'PNCN.26.9009',
   '2', 'Uji baris terhapus lunak', 'OBJEK UJI SEMBILAN', 'CABANG UJI', '0');


-- ── 10. Polis SAMA dengan baris 1, tetapi 30 hari lalu ──────────────────────
-- Pasangan penguji aturan proteksi ganda. Aturannya per HARI KALENDER, bukan
-- per polis: baris ini TIDAK boleh menghalangi pembuatan proteksi baru dengan
-- polis dan tipe yang sama hari ini.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION
  (OPEN_PROTECTION_ID, CREATE_DATE, CREATED_BY, POLICY_NO, CLAIM_NO, ID_CLAIM,
   PROTECTION_TYPE_ID, NOTES, OLD_DATA, NEW_DATA, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE)
VALUES
  ('OPCN.26.0010', SYS_EXTRACT_UTC(SYSTIMESTAMP) - INTERVAL '30' DAY, 'UJICOBA01',
   'UJI-POLIS-0001', 'PNCN.26.9010', 'PNCN.26.9010',
   '7', 'Uji aturan ganda per hari', '2026-08-01', '2026-08-03',
   'OBJEK UJI SEPULUH', 'CABANG UJI', '1');

COMMIT;


-- ============================================================================
-- VERIFIKASI — apa yang SEHARUSNYA tampil
-- ============================================================================
--
-- Angka di bawah sudah memperhitungkan OPC-216 yang disalin sebelumnya.

-- Daftar Input Req Protection  → 8 baris
--   OPCN.26.0001 0002 0003 0004 0005 0006 0010, ditambah OPC-216
SELECT COUNT(*) AS daftar
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1');

-- Antrean akseptasi PREMI      → 1 baris  (OPCN.26.0001)
SELECT COUNT(*) AS premi
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND CLAIM_NO IS NOT NULL AND POLICY_NO IS NOT NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND TRIM(PROTECTION_TYPE_ID) = '2';

-- Antrean akseptasi NON PREMI  → 5 baris  (0002 0003 0004 0010 + OPC-216)
SELECT COUNT(*) AS non_premi
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND CLAIM_NO IS NOT NULL AND POLICY_NO IS NOT NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND (PROTECTION_TYPE_ID IS NULL OR TRIM(PROTECTION_TYPE_ID) <> '2');


-- ============================================================================
-- MEMBUANG DATA UJI
-- ============================================================================
--
-- DELETE fisik di sini SAH karena ini pembersihan tangan atas baris karangan,
-- bukan penghapusan data bisnis oleh aplikasi. `D-66` mengikat APLIKASI —
-- akun aplikasi bahkan tidak diberi hak DELETE.
--
-- DELETE FROM POOLDATA.T_CLAIM_OPENPROTECTION WHERE OPEN_PROTECTION_ID LIKE 'OPCN.26.00%';
-- COMMIT;
