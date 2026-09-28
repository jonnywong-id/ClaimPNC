-- Kueri tab KPI PIC Teknik.
--
-- Seluruhnya membaca KOLOM RELASIONAL LANGSUNG — `T_CLAIM_PNC`, `T_CLAIM_ADJUSTMENT`,
-- `PEGA_DASHBOARDPNC`, `GCNM_PROGRESS_CLAIM`, `MST_USER_TEKNIK`, `M_KPI_PNC`. Tidak ada satu
-- pun `JSON_VALUE` maupun `JSON_TABLE` di sini, sehingga `DB-3` tidak tertekan di jalur ini.
--
-- Empat aturan `D-20` yang dijaga di seluruh berkas ini:
--
--   * tanpa `SELECT *` — kolomnya disebut satu per satu
--   * tanpa `ROWNUM`, `NVL`, `SYSDATE`, `DECODE`
--   * tanpa `TO_CHAR` untuk memformat — tanggal dikembalikan sebagai tanggal, diformat di Go
--   * tanpa `TRUNC` pada KOLOM — rentang tanggalnya setengah terbuka supaya index terpakai
--
-- Yang terakhir perlu disebut khusus. Kueri lama menulis
-- `trunc(a.registerdate) >= to_date(:1) and trunc(a.registerdate) <= to_date(:2)`, yang
-- mematikan index pada kolom itu dan memaksa pemindaian tabel penuh. Penggantinya
-- `>= :1 AND < :2 + INTERVAL '1' DAY` mencakup hari yang sama persis, tetapi membiarkan
-- index terpakai.
--
-- # Penanda /*LINE_FILTER*/
--
-- Penyaring lini bisnis TIDAK dapat dijadikan parameter: Non-MBU menyaring tiga nilai
-- `GROUP_PANEL` sekaligus MENGECUALIKAN empat kode kelompok bisnis, sedangkan Bonding justru
-- menyaring keempat kode itu dan tidak menyentuh `GROUP_PANEL` sama sekali. Keduanya bukan
-- dua nilai pada kolom yang sama, melainkan dua bentuk predikat yang berbeda.
--
-- Karena itu penyaringnya disisipkan dari sebuah daftar TERTUTUP di dalam kode, dipilih oleh
-- kode lini yang sudah divalidasi menjadi salah satu dari empat. Tidak ada satu pun nilai
-- dari pengguna yang menyentuh teks SQL — yang dipilih adalah potongan tetap, bukan nilai.

-- name: pic_list
-- Petugas satu lini bisnis. Meniru `GetDataPIC`.
SELECT
    OPERATOR_ID,
    STS_LEADER
FROM POOLDATA.MST_USER_TEKNIK
WHERE TYPE_BUSINESS = :1
ORDER BY OPERATOR_ID

-- name: bands
-- Pita nilai satu komponen. Meniru `GetNilaiKPIPIC`, dengan tiga penajaman.
--
-- PERTAMA, `TIPE = 'PIC'` ditambahkan. Kueri lama tidak menyaringnya, dan hari ini itu belum
-- menggigit karena nama JOB milik PIC dan milik ADJUSTER kebetulan tidak bertabrakan. Satu
-- baris ADJUSTER baru bernama sama akan mengubah nilai seseorang tanpa galat apa pun.
--
-- KEDUA, `NILAI IS NOT NULL`. Seluruh baris ADJUSTER pada tabel itu berkolom `NILAI` kosong;
-- tanpa penyaring ini, baris semacam itu akan terbaca sebagai nilai nol.
--
-- KETIGA, `ORDER BY ID`. Kueri lama tidak punya urutan sama sekali, sementara pita
-- bertetangga BERTINDIH di titik batasnya — sehingga jawabannya di sana bergantung pada
-- urutan baris yang kebetulan dikembalikan Oracle. Urutan tetap membuatnya dapat ditentukan.
-- Dinyatakan sebagai selisih terencana.
SELECT
    JOB,
    NILAI,
    BOTTOM,
    TOP,
    NOTE
FROM POOLDATA.M_KPI_PNC
WHERE TIPE = 'PIC'
  AND JOB = :1
  AND NILAI IS NOT NULL
  AND (:2 IS NULL OR NOTE = :2)
ORDER BY ID

-- name: threshold_days
-- Ambang hari satu komponen. Meniru `GetDaySurveyAdjuster` apa adanya.
SELECT DAY
FROM POOLDATA.M_KPI_PNC
WHERE TIPE = 'PIC'
  AND JOB = :1
  AND DAY IS NOT NULL
  AND (:2 IS NULL OR NOTE = :2)
ORDER BY ID
FETCH FIRST 1 ROW ONLY

-- name: holidays
-- Hari libur pada satu rentang, DI LUAR akhir pekan. Meniru `CheckHoliday_SQL`.
--
-- Akhir pekan dikecualikan di sini, bukan di Go, dan itu disengaja: begitulah kueri lama
-- melakukannya, sehingga libur yang jatuh pada Sabtu atau Minggu tidak terpotong dua kali.
--
-- Nama harinya dibandingkan dalam dua bahasa karena `TO_CHAR(...,'DAY')` mengikuti setelan
-- bahasa sesi basis data — dan setelan itu tidak dijamin sama antar lingkungan.
SELECT TANGGAL
FROM GENERAL.HRD_LBR
WHERE TANGGAL >= :1
  AND TANGGAL < :2 + INTERVAL '1' DAY
  AND TRIM(TO_CHAR(TANGGAL, 'DAY')) NOT IN ('SABTU', 'MINGGU', 'SATURDAY', 'SUNDAY')
ORDER BY TANGGAL

-- name: progress_counts
-- Cacah pembaruan progres per PIC. Meniru `GetProgressForKPIPIC`.
--
-- `ON_TIME` mencacah pembaruan yang TIDAK terlambat: entah karena belum ada pembaruan
-- berikutnya, atau karena pembaruan berikutnya datang sebelum tanggal janji terlampaui.
-- Namanya di sistem lama `STSKLAIM` dan tidak menyiratkan itu sama sekali; yang
-- menjelaskannya adalah nama variabel penerimanya, `local.tdkterlambat`.
--
-- Tiga penyaring dibawa apa adanya: status klaim bukan 1/2/3, keterangan bukan AUTO maupun
-- SYSTEM, dan PIC bukan 'ASNET'. Ketiganya membuang pembaruan yang dibuat sistem, bukan
-- orang — dan menilai orang atas pembaruan yang dibuat sistem jelas keliru.
SELECT
    a.PIC,
    COUNT(c.ID_UPDATE) AS TOTAL,
    SUM(
        CASE
            WHEN (
                SELECT t.TGL_INPUT
                FROM POOLDATA.GCNM_PROGRESS_CLAIM t
                WHERE t.PNCCASEID = c.PNCCASEID
                  AND t.ID_UPDATE > c.ID_UPDATE
                ORDER BY t.ID_UPDATE ASC
                FETCH NEXT 1 ROW ONLY
            ) IS NULL THEN 1
            ELSE
                CASE
                    WHEN c.NEXT_FOLLOWUP + INTERVAL '1' DAY >= (
                        SELECT t.TGL_INPUT
                        FROM POOLDATA.GCNM_PROGRESS_CLAIM t
                        WHERE t.PNCCASEID = c.PNCCASEID
                          AND t.ID_UPDATE > c.ID_UPDATE
                        ORDER BY t.ID_UPDATE ASC
                        FETCH NEXT 1 ROW ONLY
                    ) THEN 1
                    ELSE 0
                END
        END
    ) AS ON_TIME
FROM POOLDATA.GCNM_PROGRESS_CLAIM c
JOIN POOLDATA.PEGA_DASHBOARDPNC a ON c.PNCCASEID = a.NOKLAIM
WHERE a.STSKLAIM NOT IN ('1', '2', '3')
  AND UPPER(c.KETERANGAN) NOT LIKE '%AUTO%'
  AND UPPER(c.KETERANGAN) NOT LIKE '%SYSTEM%'
  AND a.PIC = c.USER_INPUT
  AND a.PIC <> 'ASNET'
  AND c.TGL_INPUT >= :1
  AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
  /*LINE_FILTER*/
GROUP BY a.PIC

-- name: analysis_spans
-- Pasangan tanggal penilaian Analisa Klaim. Meniru `DataAnalisaPIC`.
--
-- Hanya adjustment PERTAMA tiap klaim yang dinilai — `MIN(ADJUSTMENTID)`. Klaim dengan
-- beberapa adjustment tetap terhitung satu, dan itu perilaku lama.
SELECT
    a.PICTEKNIK,
    a.TGLDOKLENGKAP,
    b.ANALYST_TFKOMITEDATE
FROM POOLDATA.T_CLAIM_PNC a
JOIN POOLDATA.T_CLAIM_ADJUSTMENT b ON a.CLAIMID = b.CLAIMID
WHERE b.ADJUSTMENTID = (
        SELECT MIN(x.ADJUSTMENTID)
        FROM POOLDATA.T_CLAIM_ADJUSTMENT x
        WHERE x.CLAIMID = b.CLAIMID
    )
  AND b.ANALYST_TFKOMITEDATE IS NOT NULL
  AND a.PICTEKNIK IS NOT NULL
  AND a.REGISTERDATE >= :1
  AND a.REGISTERDATE < :2 + INTERVAL '1' DAY
  /*LINE_FILTER*/

-- name: acceptance_spans
-- Pasangan tanggal penilaian Akseptasi Klaim. Meniru `DataAkseptasiPIC`.
--
-- Ketiga tanggal dikembalikan sekaligus; mana yang dipakai ditentukan `LEADER_MEMBER` di
-- lapisan domain, bukan di sini — percabangannya aturan bisnis, bukan pengambilan data.
SELECT
    a.PICTEKNIK,
    a.LEADER_MEMBER,
    b.RECEIVEDATELOD,
    b.ACCEPTANCE_DATECOMITEE,
    b.TGLAKSEPTASI
FROM POOLDATA.T_CLAIM_PNC a
JOIN POOLDATA.T_CLAIM_ADJUSTMENT b ON a.CLAIMID = b.CLAIMID
WHERE b.NOAKSEPTASI IS NOT NULL
  AND a.PICTEKNIK IS NOT NULL
  AND b.TGLAKSEPTASI >= :1
  AND b.TGLAKSEPTASI < :2 + INTERVAL '1' DAY
  /*LINE_FILTER*/

-- name: closure_spans
-- Pasangan tanggal penilaian SLA Klaim. Meniru `GetDataClosePIC`.
SELECT
    a.PICTEKNIK,
    a.LEADER_MEMBER,
    a.REGISTERDATE,
    a.CLOSECLAIMDATE
FROM POOLDATA.T_CLAIM_PNC a
WHERE a.STATUSWORK = 'Resolved-Completed'
  AND a.CLOSECLAIMDATE IS NOT NULL
  AND a.PICTEKNIK IS NOT NULL
  AND a.REGISTERDATE >= :1
  AND a.REGISTERDATE < :2 + INTERVAL '1' DAY
  /*LINE_FILTER*/
