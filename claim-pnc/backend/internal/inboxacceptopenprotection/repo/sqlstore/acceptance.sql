-- Kueri modul Inbox Accept Open Protection: antrean akseptasi permintaan proteksi.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data.
--
-- ============================================================================
-- TIGA SYARAT YANG BUKAN PILIHAN PENGGUNA
-- ============================================================================
--
-- `Report Definition/InboxOpenProtection2_RD-RD.xml` menyaring
--
--     CaseID IS NOT NULL  AND  PolicyNo IS NOT NULL  AND  AcceptStatus IS NULL
--
-- Ketiganya DEFINISI layar ini, bukan penyaring yang dapat dimatikan pemanggil: yang
-- ditampilkan hanyalah permintaan yang sudah lengkap dan belum diputuskan.
--
-- Perhatikan bedanya dengan modul `inputreqprotection`, yang penyaringnya HANYA
-- `AcceptStatus IS NULL`. Perbedaan itulah yang membuat permintaan rancangan — yang belum
-- tertaut klaim — tidak pernah sampai ke meja petugas akseptasi.
--
-- ============================================================================
-- PEMISAHAN ANTREAN PREMI / NON PREMI
-- ============================================================================
--
--     InboxOpenProtection2_RD_collection   PROTECTION_TYPE  = '2'   -> PREMI
--     InboxOpenProtection2_RD              PROTECTION_TYPE <> '2'   -> NON PREMI
--
-- Kedua antrean memakai KUERI YANG SAMA dengan parameter pembeda, bukan dua kueri terpisah.
-- Dua kueri yang nyaris sama akan berbeda isinya cepat atau lambat, dan yang berbeda akan
-- menampilkan antrean yang salah tanpa satu pun gejala.
--
-- Penanda pertama dan ketiga membawa penentu antrean (1 = PREMI, 0 = NON PREMI); penanda
-- kedua dan keempat membawa kode pembandingnya. Keduanya dikirim DUA KALI karena muncul
-- dua kali — lihat bagian berikut.
--
-- ============================================================================
-- SETIAP KEMUNCULAN BIND BERNOMOR SENDIRI
-- ============================================================================
--
-- Driver mengikat argumen menurut urutan KEMUNCULAN penanda, bukan menurut nomornya.
-- Memakai penanda yang sama dua kali lalu mengirim satu argumen menghasilkan
-- **ORA-01008: not all variables bound**.
--
-- Pola pencarian pun dibentuk DI GO, bukan dirangkai di SQL: teks yang memuat tanda persen
-- atau garis bawah akan menjadi wildcard tanpa disengaja.
--
-- ============================================================================
-- KOLOM YANG DITULIS MODUL INI — HANYA TIGA
-- ============================================================================
--
--     APPROVAL_STATUS   keputusan: '1' disetujui, '2' ditolak
--     RESOLVED_BY       pelakunya
--     RESOLVED_DATE_TIME  waktunya
--
-- Kolom pembuatan — ID, POLICY_NO, CLAIM_NO, ID_CLAIM, PROTECTION_TYPE, CREATE_DATE, CREATED_BY,
-- NOTES, OLD_DATA, NEW_DATA, OBJECT_NAME, BRANCH_NAME — dimiliki modul `inputreqprotection` dan
-- TIDAK PERNAH disentuh di sini (`P-1`). Dijaga uji di query_test.go.


-- name: acceptance_count
-- Jumlah permintaan yang menunggu keputusan pada satu antrean.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND CLAIM_NO IS NOT NULL
   AND POLICY_NO IS NOT NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( ( :1 = 1 AND TRIM(PROTECTION_TYPE) = :2 )
         OR ( :3 = 0 AND (PROTECTION_TYPE IS NULL OR TRIM(PROTECTION_TYPE) <> :4) ) )
   AND ( :5 IS NULL
         OR UPPER(ID)      LIKE :6
         OR UPPER(POLICY_NO) LIKE :7
         OR UPPER(CLAIM_NO) LIKE :8 )


-- name: acceptance_list
-- Satu halaman antrean akseptasi.
--
-- Diurutkan MENURUN menurut tanggal permintaan dibuat, dengan nomor sebagai pemutus seri
-- supaya paginasi tidak menampilkan satu baris dua kali.
SELECT ID,
       POLICY_NO,
       CLAIM_NO,
       PROTECTION_TYPE,
       CREATE_DATE,
       NOTES,
       CREATED_BY,
       APPROVAL_STATUS,
       RESOLVED_DATE_TIME,
       RESOLVED_BY
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND CLAIM_NO IS NOT NULL
   AND POLICY_NO IS NOT NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( ( :1 = 1 AND TRIM(PROTECTION_TYPE) = :2 )
         OR ( :3 = 0 AND (PROTECTION_TYPE IS NULL OR TRIM(PROTECTION_TYPE) <> :4) ) )
   AND ( :5 IS NULL
         OR UPPER(ID)      LIKE :6
         OR UPPER(POLICY_NO) LIKE :7
         OR UPPER(CLAIM_NO) LIKE :8 )
 ORDER BY CREATE_DATE DESC, ID DESC
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY


-- name: acceptance_get
-- Satu permintaan untuk form akseptasi.
--
-- TIDAK menyaring APPROVAL_STATUS, dan itu disengaja: layar ini antrean BERSAMA
-- (`Flow/CreateProtection_Flow.xml` menempatkannya di workbasket ProtectionPNC), sehingga
-- sebuah baris dapat diputuskan petugas lain kapan saja. Form harus tetap terbuka supaya
-- pesannya dapat menyatakan keputusan siapa dan kapan — bukan sekadar "tidak ditemukan".
SELECT ID,
       POLICY_NO,
       CLAIM_NO,
       PROTECTION_TYPE,
       CREATE_DATE,
       NOTES,
       CREATED_BY,
       APPROVAL_STATUS,
       RESOLVED_DATE_TIME,
       RESOLVED_BY
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE UPPER(TRIM(ID)) = :1
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')


-- name: acceptance_decide
-- Menuliskan keputusan akseptasi.
--
-- `APPROVAL_STATUS IS NULL` ada DI DALAM WHERE, bukan hanya diperiksa lebih dulu di Go.
-- Itulah yang benar-benar menahan petugas KEDUA pada antrean bersama: pemeriksaan di Go
-- dilewati keduanya bila mereka menekan tombol bersamaan.
--
-- Kedua syarat kelengkapan ikut diulang. Sebuah permintaan yang belum tertaut klaim tidak
-- muncul di antrean, tetapi tautan yang disimpan masih dapat membukanya — dan keputusan
-- atasnya harus ditolak, bukan diterima diam-diam.
UPDATE POOLDATA.T_CLAIM_OPENPROTECTION
   SET APPROVAL_STATUS  = :1,
       RESOLVED_BY      = :2,
       RESOLVED_DATE_TIME = :3
 WHERE UPPER(TRIM(ID)) = :4
   AND APPROVAL_STATUS IS NULL
   AND CLAIM_NO IS NOT NULL
   AND POLICY_NO IS NOT NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
