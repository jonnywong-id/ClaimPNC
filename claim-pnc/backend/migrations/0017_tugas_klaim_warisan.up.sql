-- 0017 — Membuatkan tugas bagi klaim warisan `PNC-xxxx`, supaya dikerjakan di aplikasi ini
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Ia MENYISIPKAN ke `POOLDATA.CPNC_TUGAS` — tabel tugas milik aplikasi ini. Tidak ada satu
-- pun baris Pega yang disentuh: berkas ini hanya MEMBACA `POOLDATA.T_CLAIMLIST_ADMIN`.
--
-- ============================================================================
-- KENAPA MIGRASI INI ADA
-- ============================================================================
--
-- Sebuah klaim hanya dapat DIKERJAKAN di aplikasi ini bila ia punya tugas terbuka di
-- `CPNC_TUGAS`. Tugas itu lahir saat klaim didaftarkan lewat aplikasi ini.
--
-- Klaim warisan `PNC-xxxx` dibuat di Pega, sehingga tidak pernah punya tugas di sini. Diukur
-- 2026-10-12 atas 368 baris `CPNC_TUGAS`: **359 milik klaim `PNCN.`, hanya 2 milik klaim
-- warisan**. Akibatnya seluruh klaim warisan terbuka sebagai layar baca-saja, dan satu-satunya
-- tempat mengerjakannya adalah Pega.
--
-- Work Owner menegaskan 2026-10-12: **Pega sudah tidak dipakai lagi.** Migration Strategy
-- Tahap 7 memang menyediakan jalur ini — Pega dimatikan setelah klaim warisan selesai "atau
-- DIPINDAHKAN SECARA SADAR". Berkas ini adalah pemindahan itu.
--
-- ============================================================================
-- DARI MANA TAHAP DAN PEMILIKNYA DIKETAHUI
-- ============================================================================
--
-- `POOLDATA.T_CLAIMLIST_ADMIN` sudah menyimpan keduanya, dan TIDAK perlu menyentuh schema
-- `DATAPEGA` sama sekali:
--
--	PXTASKLABEL            nama tahap tempat klaim berhenti  ("Input Register", …)
--	PXASSIGNEDOPERATORID   pemiliknya
--
-- Diukur 2026-10-12 atas klaim warisan yang masih berjalan:
--
--	klaim warisan seluruhnya        2.646
--	...yang masih berjalan          1.992
--	...punya tahap DAN pemilik      1.901   (95%)
--	...tanpa keterangan tahap          91
--
-- Sebaran tahapnya, dan seluruhnya bernama SAMA dengan tahap alur di aplikasi ini:
--
--	Input Register       596      Send To Analis        179
--	Choose Surveyor      480      Send To PIC Teknik     21
--	Input Estimasi       211      RCLDokter               1
--	Estimation           198      (FixCorrespondence)    23   <- bukan tahap alur ini
--	View Polis           181      (InputPanel)            7   <- bukan tahap alur ini
--	                              (Review Klaim OCR)      4   <- bukan tahap alur ini
--
-- ============================================================================
-- YANG SENGAJA TIDAK DIPINDAHKAN
-- ============================================================================
--
--	klaim yang SUDAH SELESAI       tidak butuh tugas; tetap terbaca sebagai riwayat
--	tahap di luar alur Register    FixCorrespondence, InputPanel, Review Klaim OCR —
--	                               ketiganya bukan tahap `Register_Flow`, dan menebak
--	                               padanannya akan menaruh klaim di tahap yang salah
--	klaim tanpa PXTASKLABEL        91 baris; tahapnya memang tidak diketahui
--	klaim yang SUDAH punya tugas   2 baris; tidak digandakan
--
-- Keempatnya tetap terbuka sebagai layar baca-saja — persis seperti sebelum migrasi ini.
-- Tidak ada klaim yang menjadi lebih buruk keadaannya.
--
-- ============================================================================
-- PEMETAAN TAHAP, DAN SATU YANG AMBIGU
-- ============================================================================
--
-- Nama tahap Pega dipetakan ke pengenal tahap aplikasi ini (`internal/registrasi/flow.go`).
-- Satu nama menunjuk DUA tahap:
--
--	"Input Estimasi"  ->  estimasi-travel   bila Group Panel = 005 (Travel)
--	                  ->  estimasi-admin    selainnya
--
-- Pembedanya `GROUPPANEL_1`, bukan tebakan: `Assignment10` (Travel) dan `Assignment7`
-- (non-Travel) adalah dua assignment berbeda pada `Flow/Register_Flow.xml` yang kebetulan
-- berlabel sama.
--
-- ANTREAN seluruh tahap yang dipindahkan adalah WORKLIST. Ketiga tahap Workbasket
-- (Investigator, RCL/PUCL, Compliance) TIDAK muncul pada sebaran di atas, sehingga tidak ada
-- baris yang perlu memakai WORKBASKET — dan `CK_CPNC_TUGAS_ANTREAN` akan menolaknya bila
-- kelak muncul, alih-alih menerimanya diam-diam.
--
-- ============================================================================
-- DAPAT DIJALANKAN ULANG
-- ============================================================================
--
-- `WHERE NOT EXISTS` menjaga klaim yang sudah punya tugas terbuka tidak digandakan, sehingga
-- menjalankan berkas ini dua kali tidak menambah satu baris pun.

INSERT INTO CPNC_TUGAS (
    ID, KLAIM_ID, NOMOR_KLAIM, TAHAP, ANTREAN, WORKBASKET, PEMILIK,
    DIBUAT_PADA, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI
)
SELECT
    -- ID 32 karakter, seperti tugas yang lahir dari aplikasi ini.
    LOWER(RAWTOHEX(SYS_GUID())),
    W.PZINSKEY,
    W.PYID,
    CASE TRIM(W.PXTASKLABEL)
        WHEN 'View Polis'          THEN 'view-polis'
        WHEN 'Input Register'      THEN 'input-register'
        WHEN 'Estimation'          THEN 'estimasi-pa'
        WHEN 'Choose Surveyor'     THEN 'pilih-surveyor'
        WHEN 'Send To Analis'      THEN 'kirim-analis'
        WHEN 'Send To PIC Teknik'  THEN 'kirim-pic-teknik'
        WHEN 'RCLDokter'           THEN 'rcl-dokter'
        WHEN 'Input Estimasi'      THEN
            CASE WHEN TRIM(W.GROUPPANEL_1) = '005' THEN 'estimasi-travel' ELSE 'estimasi-admin' END
    END,
    'WORKLIST',
    NULL,
    TRIM(W.PXASSIGNEDOPERATORID),
    -- Waktu lahir tugas = waktu klaimnya dibuat. Memakai waktu migrasi akan membuat seluruh
    -- 1.901 tugas tampak lahir serentak, dan perhitungan umur tugas menjadi salah seluruhnya.
    COALESCE(W.PXCREATEDATETIME, SYSTIMESTAMP),
    NULL,
    NULL,
    NULL
FROM POOLDATA.T_CLAIMLIST_ADMIN W
WHERE W.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
  AND W.PYID LIKE 'PNC-%'
  AND W.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
  AND W.PXASSIGNEDOPERATORID IS NOT NULL
  AND TRIM(W.PXTASKLABEL) IN (
        'View Polis', 'Input Register', 'Estimation', 'Choose Surveyor',
        'Send To Analis', 'Send To PIC Teknik', 'RCLDokter', 'Input Estimasi')
  AND NOT EXISTS (
        SELECT 1 FROM CPNC_TUGAS T
         WHERE T.KLAIM_ID = W.PZINSKEY
           AND T.SELESAI_PADA IS NULL);

COMMIT;
