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
  AND (:2 IS NULL OR NOTE = :3)
ORDER BY ID

-- name: adjuster_categories
-- Pita KATEGORI adjuster — kolom terakhir grid Summary dan Detail KPI Adjuster.
--
-- Di Pega kolom ini diisi section `KategoriKPI`, yang mencari pita dengan cara yang sama
-- seperti `GetNilaiKPIPIC`: `BOTTOM <= nilai AND TOP >= nilai`. Pencariannya dikerjakan Go
-- (lihat reportkpi.CategoryFor); yang dilakukan di sini hanya membaca pitanya.
--
-- `TIPE` di sini ADJUSTER, bukan PIC — dan itulah satu-satunya sebab kueri ini tidak dapat
-- memakai ulang kueri `bands`.
SELECT
    BOTTOM,
    TOP,
    NOTE
FROM POOLDATA.M_KPI_PNC
WHERE TIPE = 'ADJUSTER'
  AND JOB = :1
  AND NOTE IS NOT NULL
ORDER BY ID

-- name: threshold_days
-- Ambang hari satu komponen. Meniru `GetDaySurveyAdjuster` apa adanya.
SELECT DAY
FROM POOLDATA.M_KPI_PNC
WHERE TIPE = 'PIC'
  AND JOB = :1
  AND DAY IS NOT NULL
  AND (:2 IS NULL OR NOTE = :3)
ORDER BY ID
FETCH FIRST 1 ROW ONLY

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

-- # Probe `-periksa`
--
-- Satu probe per objek yang dibaca tab ini. Masing-masing menyebut KOLOM yang benar-benar
-- dipakai, bukan sekadar `SELECT 1` — karena kolom yang hilang (`ORA-00904`) sama sering
-- terjadi dengan tabel yang hilang (`ORA-00942`), dan keduanya menghentikan tab ini.
--
-- `WHERE 1 = 0` membuat probe tidak membaca satu baris pun: yang diperiksa keberadaan
-- objek, hak SELECT, dan nama kolomnya — bukan isinya.

-- name: probe_pic
SELECT OPERATOR_ID, STS_LEADER, TYPE_BUSINESS
FROM POOLDATA.MST_USER_TEKNIK
WHERE 1 = 0

-- name: probe_tangga_nilai
SELECT ID, TIPE, JOB, NILAI, BOTTOM, TOP, NOTE, DAY
FROM POOLDATA.M_KPI_PNC
WHERE 1 = 0

-- name: probe_progres
SELECT PNCCASEID, ID_UPDATE, TGL_INPUT, KETERANGAN, USER_INPUT, NEXT_FOLLOWUP
FROM POOLDATA.GCNM_PROGRESS_CLAIM
WHERE 1 = 0

-- name: probe_dashboard
SELECT NOKLAIM, STSKLAIM, PIC
FROM POOLDATA.PEGA_DASHBOARDPNC
WHERE 1 = 0

-- name: probe_klaim
SELECT CLAIMID, PICTEKNIK, LEADER_MEMBER, REGISTERDATE, CLOSECLAIMDATE, STATUSWORK, TGLDOKLENGKAP
FROM POOLDATA.T_CLAIM_PNC
WHERE 1 = 0

-- name: probe_adjustment
SELECT CLAIMID, ADJUSTMENTID, NOAKSEPTASI, TGLAKSEPTASI, RECEIVEDATELOD, ACCEPTANCE_DATECOMITEE, ANALYST_TFKOMITEDATE
FROM POOLDATA.T_CLAIM_ADJUSTMENT
WHERE 1 = 0

-- name: probe_followup_aritmetika
-- Membuktikan NEXT_FOLLOWUP dapat DIHITUNG sebagai tanggal, bukan sekadar ada.
--
-- Ia dipakai sebagai `c.NEXT_FOLLOWUP + INTERVAL '1' DAY` di `progress_counts`. Bila
-- kolomnya bertipe teks, objek dan kolomnya tetap terbaca oleh probe di atas, tetapi
-- kuerinya tetap gagal dengan `ORA-00932`. Probe ini yang membedakan keduanya — tipenya
-- belum pernah dipastikan (`R-08`: DDL belum ada).
SELECT NEXT_FOLLOWUP + INTERVAL '1' DAY AS BESOK
FROM POOLDATA.GCNM_PROGRESS_CLAIM
WHERE 1 = 0

-- name: holidays_dblink
-- Kalender hari libur lewat DB LINK — jalur CADANGAN, dijalankan di koneksi POOLDATA.
--
-- # Kenapa ada dua jalur untuk satu kalender
--
-- Jalur utamanya `holidays` di `reportkpi_aneka.sql`, yang memakai koneksi kedua portal
-- sesuai `D-25`/`R-03`. Jalur ini dipakai HANYA bila koneksi kedua itu belum terpasang.
--
-- Ia bukan jalan pintas, melainkan PERSIS yang dilakukan sistem lama:
-- `RDB List/CheckHoliday_SQL-SQL.xml` membaca objek yang sama lewat DB Link yang sama.
-- Jadi memakainya berarti mengikuti Pega apa adanya (`P-5`), bukan menyimpang darinya.
--
-- # Kenapa ia tetap CADANGAN, bukan jalur utama
--
-- `D-25` menetapkan DB Link diganti, dan alasannya tidak hilang karena ia kebetulan masih
-- hidup hari ini: DB Link tidak ada di PostgreSQL, sehingga kueri ini TIDAK PORTABEL dan
-- akan mati pada perpindahan basis data (`D-01`, `D-24`). Ia jembatan selama masa paralel
-- — pemakaiannya dicatat sebagai peringatan di log supaya tidak diam-diam menjadi
-- permanen.
SELECT TANGGAL
FROM GENERAL.HRD_LBR@ASMD.SINARMAS.CO.ID
WHERE TANGGAL >= :1
  AND TANGGAL < :2 + INTERVAL '1' DAY
  AND TRIM(TO_CHAR(TANGGAL, 'DAY')) NOT IN ('SABTU', 'MINGGU', 'SATURDAY', 'SUNDAY')
ORDER BY TANGGAL

-- name: probe_hari_libur_dblink
-- Probe `-periksa` untuk jalur CADANGAN: kalender libur lewat DB Link.
SELECT TANGGAL
FROM GENERAL.HRD_LBR@ASMD.SINARMAS.CO.ID
WHERE 1 = 0
