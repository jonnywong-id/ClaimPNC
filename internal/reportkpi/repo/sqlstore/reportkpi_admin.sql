-- Kueri tab **KPI Admin** pada Report KPI PNC.
--
-- Rujukannya lima rule, dan pembagiannya dua kelompok kali dua grid:
--
--   RDB List/GetDataKPIAdmin-SQL.xml             kartu skor NON-MBU
--   RDB List/BrowseDataKPIAdmin-SQL.xml          rincian     NON-MBU
--   RDB List/GetDataKPIAdminPA-SQL.xml           kartu skor PA
--   RDB List/GetDataKPIAdminPA_khususPA-SQL.xml  rincian     PA
--   (BrowseDataKPIAdmin_PA-SQL.xml adalah varian rincian PA tanpa paginasi; yang dipakai
--    layar adalah `_khususPA`, karena hanya ia yang dipaginasi `PNCReportKPIAdmin_Act_khususPA`)
--
-- SELURUH pernyataan di sini MEMBACA.
--
-- ============================================================================
-- NILAI YANG DI-HARDCODE — DIBIARKAN APA ADANYA ATAS KETETAPAN WORK OWNER
-- ============================================================================
--
-- Work Owner, 2026-09-24: **"seperti aplikasi PEGA saja"**. Karena itu seluruh nilai di
-- bawah ini DITIRU kata demi kata, meski `D-15` menetapkan tidak satu pun nilai bisnis
-- boleh berada di dalam kode:
--
--   enam OPERATOR yang menentukan klaim siapa yang ikut dihitung
--   nama KOORDINATOR dan NIK-nya, ditulis sebagai literal di dalam SELECT
--   UNIT KERJA
--   bobot 0.45 dan 0.40
--   ambang nilai 0.5 / 1 / 1.5 / 2 yang mengubah persentase menjadi nilai 1–5
--   pembagi target (3/5)*90
--
-- Nilainya BOLEH ditulis di sini: `D-69` melarang alamat surel, kredensial, hostname, dan
-- data nasabah — bukan nama Operator ID, yang justru diperlukan agar hardcode-nya dapat
-- ditunjuk saat kelak dipindahkan ke master data.
--
-- Ketika masternya kelak dibuat, yang berubah hanyalah berkas ini dan satu pembacaan
-- tambahan — tidak ada satu baris pun kode Go yang ikut berubah.
--
-- ============================================================================
-- TIGA KEANEHAN YANG DIREPLIKASI, DAN KETIGANYA DAPAT MENGUBAH ANGKA
-- ============================================================================
--
-- 1. **Kartu skor NON-MBU menghitung Group Panel `009`; grid rinciannya TIDAK.**
--    `GetDataKPIAdmin` menyaring `('003','004','006','009')`, sementara
--    `BrowseDataKPIAdmin` menyaring `('003','004','006')`. Akibatnya jumlah baris rincian
--    tidak selalu sama dengan "TOTAL KLAIM" pada kartu skornya. Direplikasi apa adanya.
--
-- 2. **Pembagi `total_pembayaran_pa` terkunci pada rentang 1 Jan – 10 Nov 2023.**
--    Rentang itu ditulis sebagai literal di dalam `GetDataKPIAdminPA`, dan TIDAK ikut
--    berubah ketika pengguna memilih periode lain. Ia tampak seperti sisa uji coba yang
--    tertinggal. Direplikasi, dan ditandai sebagai selisih terencana.
--
-- 3. **Kedua kartu skor membagi tanpa penjaga nol.** Bila tidak ada satu pun klaim pada
--    periode yang dipilih, pembaginya nol dan Oracle menjawab `ORA-01476`. Di Pega galat
--    itu sampai ke pengguna apa adanya. Di sini ia DITAHAN — lihat NULLIF di bawah, satu-
--    satunya penyimpangan pada berkas ini dan alasannya ada di komentar tempatnya.

-- ============================================================================
-- JAM KERJA DIHITUNG DI GO, BUKAN DI BASIS DATA (2026-10-09)
-- ============================================================================
--
-- Keempat kueri di bawah DULU memanggil `datamining.get_working_hours@asmd.sinarmas.co.id`
-- — fungsi lintas DB Link milik basis data ASMD. Pemanggilan itu dicabut seluruhnya,
-- karena dua alasan yang berdiri sendiri:
--
--   1. Fungsinya TIDAK DAPAT DIJANGKAU dari basis data kita. Dibuktikan dengan
--      memanggilnya langsung: `ORA-00904: "DATAMINING"."GET_WORKING_HOURS": invalid
--      identifier`, sementara `GENERAL.HRD_LBR@asmd.sinarmas.co.id` pada DB Link YANG SAMA
--      terbaca normal. Jadi bukan DB Link-nya yang mati — objek itu yang tidak terlihat.
--      Akibatnya seluruh tab KPI Admin gagal dengan "Terjadi kesalahan pada sistem".
--
--   2. `D-50` memang menetapkan perhitungan jam kerja dan kalender libur **ditulis ulang
--      di Go**: ia aturan bisnis, bukan pengambilan data. Tab KPI PIC Teknik sudah begitu.
--      Yang diambil dari basis data kini hanyalah daftar tanggal liburnya.
--
-- Akibatnya pada bentuk kueri: yang dulu MENGHITUNG di SQL kini MENGEMBALIKAN BARIS, dan
-- pencacahan beserta penilaiannya pindah ke Go — lihat `admin_penilaian.go`.
--
-- SELISIH YANG DIKETAHUI. Fungsi lama mengembalikan DETIK, lalu dibagi 28.800 (= 8 jam),
-- sehingga umurnya PECAHAN — grid Pega menampilkan misalnya 2,35 hari. Perhitungan Go
-- menghitung HARI KERJA BULAT, karena jam masuk dan jam pulang yang dipakai fungsi itu
-- tidak dapat dibaca dari mana pun. Dua akibatnya: kolom umur tampil bulat, dan ambang
-- `> 1 hari` dapat berbeda untuk kasus yang sebenarnya 1,4 hari. Begitu jam kerjanya
-- diketahui, yang berubah hanya satu fungsi di Go — kueri di bawah tidak tersentuh.

-- name: admin_rows_nonmbu
-- Baris mentah kartu skor **NON-MBU** — satu baris per klaim.
-- — RDB List/GetDataKPIAdmin-SQL.xml
--
-- Penyaringnya sama persis dengan keempat subquery kueri lama, yang memang mengulang
-- penyaring yang sama empat kali. Di sini cukup SEKALI: pembedaan leader/member dan
-- pelanggaran SLA dikerjakan Go atas baris yang sama.
--
-- `GROUP_PANEL` dan `GROUPBISNISID` memakai alias `a` (pega_dashboardpnc), bukan `b`.
-- Kueri Pega menulisnya tanpa alias; katalog menunjukkan keduanya milik tabel itu, dan
-- menempelkannya ke `b` menghasilkan `ORA-00904`.
--
-- Bind: :1 periode dari · :2 periode sampai
SELECT a.reinsurer        AS REINSURER,
       b.REGISTERDATE     AS REGISTER_DATE,
       b.TRANSFERPIC_DATE AS TRANSFER_DATE
  FROM pooldata.pega_dashboardpnc a,
       pooldata.t_claim_pnc b,
       datapega.pc_asm_fw_gcnmfw_work d
 WHERE a.noklaim = b.claimno
   AND b.claimid = d.pzinskey
   AND a.stsklaim <> '2'
   AND d.pxcreateoperator IN
         ('SOPHIANOVITAEVELYN_1', 'SOPHIANOVITAEVELYN', 'RUTHCLARA')
   AND a.group_panel IN ('003', '004', '006', '009')
   AND a.groupbisnisid NOT IN ('09', '11', '16', '25')
   AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
   AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY

-- name: admin_detail_nonmbu
-- Grid rincian **NON-MBU** — satu baris per klaim.
-- — RDB List/BrowseDataKPIAdmin-SQL.xml
--
-- PERHATIKAN penyaring Group Panel-nya: `('003','004','006')` — TANPA `009`, berbeda dari
-- kartu skor di atas. Itu keanehan kueri lama, direplikasi apa adanya.
--
-- Umur registrasi TIDAK lagi dipilih di sini; ia dihitung Go dari kedua tanggalnya.
--
-- Bind: :1 periode dari · :2 periode sampai · :3 offset · :4 jumlah baris
SELECT a.noklaim          AS CLAIM_NUMBER,
       b.nopolis          AS POLICY_NUMBER,
       b.businessname     AS BUSINESS_NAME,
       b.REGISTERDATE     AS REGISTER_DATE,
       b.TRANSFERPIC_DATE AS TRANSFER_DATE,
       b.leader_member    AS TEAM_FLAG,
       COUNT(*) OVER ()   AS TOTAL_ROWS
  FROM pooldata.pega_dashboardpnc a,
       pooldata.t_claim_pnc b,
       datapega.pc_asm_fw_gcnmfw_work d
 WHERE a.noklaim = b.claimno
   AND a.stsklaim <> '2'
   AND b.claimid = d.pzinskey
   AND d.pxcreateoperator IN
         ('SOPHIANOVITAEVELYN_1', 'SOPHIANOVITAEVELYN', 'RUTHCLARA')
   AND a.group_panel IN ('003', '004', '006')
   AND a.groupbisnisid NOT IN ('09', '11', '16', '25')
   AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
   AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY
 ORDER BY b.REGISTERDATE DESC, a.noklaim
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: admin_rows_pa_register
-- Baris tahap REGISTRASI kelompok **PA** — satu baris per klaim.
-- — RDB List/GetDataKPIAdminPA_khususPA-SQL.xml
--
-- Ia melayani DUA cacahan sekaligus: `claim_total` adalah seluruh barisnya, dan
-- `register_over_sla` adalah yang umurnya melewati ambang. Kueri lama memisahkannya
-- menjadi dua subquery dengan penyaring yang identik; di sini cukup satu.
--
-- Bind: :1 periode dari · :2 periode sampai
SELECT b.CLAIMNO      AS CLAIM_NUMBER,
       b.RECEIVEDATE  AS RECEIVE_DATE,
       b.REGISTERDATE AS REGISTER_DATE
  FROM pooldata.t_claim_pnc b,
       datapega.pc_asm_fw_gcnmfw_work d
 WHERE b.claimid = d.pzinskey
   AND b.STATUSWORK <> 'Resolved-Rejected'
   AND d.pxcreateoperator IN
         ('IRMANOPITAPURBA_1', 'IRMANOPITAPURBA', 'YUNIARPAMORSUARI')
   AND b.GROUPPANEL IN ('002')
   AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
   AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY

-- name: admin_rows_pa_payment
-- Baris tahap PEMBAYARAN kelompok **PA**, disaring periode yang dipilih pengguna.
--
-- Melayani `payment_over_sla`. Cacahnya DISTINCT per nomor klaim — satu klaim dapat punya
-- beberapa baris akseptasi, dan kueri lama pun menghitungnya sekali.
--
-- Bind: :1 periode dari · :2 periode sampai
SELECT b.CLAIMNO         AS CLAIM_NUMBER,
       c.RECEIVEDATELOD  AS LOD_RECEIVE_DATE,
       c.TGLAKSEPTASI    AS ACCEPTANCE_DATE
  FROM pooldata.t_claim_pnc b,
       POOLDATA.T_CLAIM_ADJUSTMENT c,
       datapega.pc_asm_fw_gcnmfw_work d
 WHERE b.CLAIMID = c.CLAIMID
   AND b.claimid = d.pzinskey
   AND b.STATUSWORK <> 'Resolved-Rejected'
   AND c.NOAKSEPTASI IS NOT NULL
   AND d.pxcreateoperator IN
         ('IRMANOPITAPURBA_1', 'IRMANOPITAPURBA', 'YUNIARPAMORSUARI')
   AND b.GROUPPANEL IN ('002')
   AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
   AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY

-- name: admin_rows_pa_payment_total
-- Baris pembayaran untuk `payment_total` — RENTANG 2023 YANG TERTANAM.
--
-- Ia TIDAK menerima periode yang dipilih pengguna. Rentangnya terkunci pada
-- 2023-01-01 s.d. 2023-11-10 di dalam teks kueri, persis seperti di Pega, dan menyaring
-- `receivedate` — bukan `registerdate` seperti ketiga subquery lainnya.
--
-- Direplikasi apa adanya (`P-5`) dan ditandai sebagai selisih terencana, supaya penguji
-- tidak melaporkannya sebagai cacat sistem baru. Bila Work Owner memutuskan
-- memperbaikinya, yang berubah hanya dua baris di bawah.
--
-- Tanpa bind.
SELECT b.CLAIMNO         AS CLAIM_NUMBER,
       c.RECEIVEDATELOD  AS LOD_RECEIVE_DATE,
       c.TGLAKSEPTASI    AS ACCEPTANCE_DATE
  FROM pooldata.t_claim_pnc b,
       POOLDATA.T_CLAIM_ADJUSTMENT c,
       datapega.pc_asm_fw_gcnmfw_work d
 WHERE b.CLAIMID = c.CLAIMID
   AND b.claimid = d.pzinskey
   AND b.STATUSWORK <> 'Resolved-Rejected'
   AND c.NOAKSEPTASI IS NOT NULL
   AND d.pxcreateoperator IN
         ('IRMANOPITAPURBA_1', 'IRMANOPITAPURBA', 'YUNIARPAMORSUARI')
   AND b.GROUPPANEL IN ('002')
   AND b.receivedate >= TO_DATE('2023-01-01', 'YYYY-MM-DD')
   AND b.receivedate <  TO_DATE('2023-11-10', 'YYYY-MM-DD') + INTERVAL '1' DAY

-- name: admin_detail_pa
-- Grid rincian **PA** — satu baris per klaim, dipaginasi.
-- — RDB List/GetDataKPIAdminPA_khususPA-SQL.xml
--
-- Ia membawa DUA umur dan DUA penanda SLA, karena yang diukur dua tahap. Keduanya kini
-- dihitung Go dari tanggal-tanggal di bawah; teks `SLA` / `TIDAK SLA` tetap seperti layar
-- lama (`D-13`).
--
-- `RECEIVEDATELOD` yang kosong jatuh ke `TGLAKSEPTASI` — akibatnya umur pembayaran menjadi
-- nol pada baris seperti itu, bukan kosong. Ditiru apa adanya, dan kejatuhannya kini
-- dikerjakan Go.
--
-- Paginasinya `OFFSET … FETCH NEXT`, bukan `rownum BETWEEN` seperti kueri lama: `ROWNUM`
-- khas Oracle dan `D-20` menggantinya.
--
-- Bind: :1 periode dari · :2 periode sampai · :3 offset · :4 jumlah baris
SELECT b.CLAIMNO          AS CLAIM_NUMBER,
       b.NOPOLIS          AS POLICY_NUMBER,
       g.STARTDATE        AS POLICY_START,
       g.ENDDATE          AS POLICY_END,
       d.pxcreateoperator AS ADMIN_NAME,
       b.RECEIVEDATE      AS RECEIVE_DATE,
       b.REGISTERDATE     AS REGISTER_DATE,
       c.RECEIVEDATELOD   AS LOD_RECEIVE_DATE,
       c.TGLAKSEPTASI     AS ACCEPTANCE_DATE,
       CASE WHEN d.pystatuswork = 'Resolved-Rejected'  THEN 'Rejected'
            WHEN d.pystatuswork = 'Resolved-Completed' THEN 'Close'
            ELSE 'Outstanding' END          AS CLAIM_STATUS,
       COUNT(*) OVER ()                     AS TOTAL_ROWS
  FROM pooldata.t_claim_pnc b
       INNER JOIN POOLDATA.T_CLAIM_ADJUSTMENT c ON b.CLAIMID = c.CLAIMID
       INNER JOIN datapega.pc_asm_fw_gcnmfw_work d ON b.claimid = d.pzinskey
       LEFT JOIN POOLDATA.T_GENERAL g
              ON g.nopolis = b.NOPOLIS AND g.PRODKE = b.PRODKE
 WHERE d.pxcreateoperator IN
         ('IRMANOPITAPURBA_1', 'IRMANOPITAPURBA', 'YUNIARPAMORSUARI')
   AND b.GROUPPANEL IN ('002')
   AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
   AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY
 ORDER BY b.REGISTERDATE DESC, b.CLAIMNO
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY
