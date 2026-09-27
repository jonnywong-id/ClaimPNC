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

-- name: admin_scorecard_nonmbu
-- Kartu skor **NON-MBU** — satu baris, seluruh angkanya dihitung basis data.
-- — RDB List/GetDataKPIAdmin-SQL.xml
--
-- Empat pencacah di dalamnya membedakan LEADER dari MEMBER lewat `a.reinsurer`:
--
--   reinsurer  = '1'  leader
--   reinsurer <> '1'  member
--
-- dan "melewati SLA" berarti `tat_regis > 1` — TAT registrasi lebih dari satu hari kerja.
-- Satu hari kerja di sana adalah 28.800 detik (8 jam), dan pembaginya ditulis apa adanya.
--
-- `datamining.get_working_hours@asmd.sinarmas.co.id` DIPERTAHANKAN sebagai DB link.
-- Work Owner, 2026-09-24: yang berupa SUB-QUERY tetap memakai DB link. Fungsi ini dipanggil
-- di dalam subquery terhadap tabel POOLDATA; memindahkannya ke koneksi langsung berarti
-- satu perjalanan jaringan PER BARIS, dan itu menghancurkan kinerja laporan (`D-50`).
--
-- Bind: :1 periode dari · :2 periode sampai (dipakai empat kali, lihat urutannya di Go)
SELECT leader_over_sla                                          AS LEADER_OVER_SLA,
       leader_total                                             AS LEADER_TOTAL,
       (leader_over_sla / NULLIF(leader_total, 0)) * 100         AS LEADER_PERCENT,
       member_over_sla                                          AS MEMBER_OVER_SLA,
       member_total                                             AS MEMBER_TOTAL,
       (member_over_sla / NULLIF(member_total, 0)) * 100         AS MEMBER_PERCENT,
       leader_score                                             AS LEADER_SCORE,
       member_score                                             AS MEMBER_SCORE,
       (leader_score / 5) * 0.45 * 100                          AS LEADER_SUBTOTAL,
       (member_score / 5) * 0.40 * 100                          AS MEMBER_SUBTOTAL,
       (leader_score / 5) * 0.45 * 100
         + (member_score / 5) * 0.40 * 100                      AS QUANTITATIVE_TOTAL,
       ROUND(((leader_score / 5) * 0.45 * 100
         + (member_score / 5) * 0.40 * 100) / ((3 / 5) * 90), 2) AS ACHIEVEMENT_RATIO
  FROM (SELECT leader_over_sla,
               leader_total,
               member_over_sla,
               member_total,
               -- Tangga nilai 1–5. Ditiru kata demi kata, termasuk tumpang tindih
               -- BETWEEN-nya: `= 1` tidak pernah tercapai karena cabang `BETWEEN 0.5 AND 1`
               -- di atasnya sudah menangkapnya lebih dulu. Itu perilaku Pega hari ini.
               CASE
                 WHEN (leader_over_sla / NULLIF(leader_total, 0)) * 100 < 0.5            THEN 5
                 WHEN (leader_over_sla / NULLIF(leader_total, 0)) * 100 BETWEEN 0.5 AND 1 THEN 4
                 WHEN (leader_over_sla / NULLIF(leader_total, 0)) * 100 = 1              THEN 3
                 WHEN (leader_over_sla / NULLIF(leader_total, 0)) * 100 BETWEEN 1 AND 1.5 THEN 2
                 WHEN (leader_over_sla / NULLIF(leader_total, 0)) * 100 BETWEEN 1.5 AND 2 THEN 1
                 ELSE 0
               END AS leader_score,
               CASE
                 WHEN (member_over_sla / NULLIF(member_total, 0)) * 100 < 0.5            THEN 5
                 WHEN (member_over_sla / NULLIF(member_total, 0)) * 100 BETWEEN 0.5 AND 1 THEN 4
                 WHEN (member_over_sla / NULLIF(member_total, 0)) * 100 = 1              THEN 3
                 WHEN (member_over_sla / NULLIF(member_total, 0)) * 100 BETWEEN 1 AND 1.5 THEN 2
                 WHEN (member_over_sla / NULLIF(member_total, 0)) * 100 BETWEEN 1.5 AND 2 THEN 1
                 ELSE 0
               END AS member_score
          FROM (SELECT COUNT(CASE WHEN reinsurer =  '1' AND tat_regis > 1 THEN 1 END) AS leader_over_sla,
                       COUNT(CASE WHEN reinsurer =  '1'                   THEN 1 END) AS leader_total,
                       COUNT(CASE WHEN reinsurer <> '1' AND tat_regis > 1 THEN 1 END) AS member_over_sla,
                       COUNT(CASE WHEN reinsurer <> '1'                   THEN 1 END) AS member_total
                  FROM (SELECT a.reinsurer,
                               datamining.get_working_hours@asmd.sinarmas.co.id(
                                 b.REGISTERDATE, b.TRANSFERPIC_DATE) / 28800 AS tat_regis
                          FROM pooldata.pega_dashboardpnc a,
                               pooldata.t_claim_pnc b,
                               datapega.pc_asm_fw_gcnmfw_work d
                         WHERE a.noklaim = b.claimno
                           AND b.claimid = d.pzinskey
                           AND a.stsklaim <> '2'
                           AND d.pxcreateoperator IN
                                 ('SOPHIANOVITAEVELYN_1', 'SOPHIANOVITAEVELYN', 'RUTHCLARA')
                           AND b.group_panel IN ('003', '004', '006', '009')
                           AND b.groupbisnisid NOT IN ('09', '11', '16', '25')
                           AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
                           AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY)))

-- name: admin_detail_nonmbu
-- Grid rincian **NON-MBU** — satu baris per klaim.
-- — RDB List/BrowseDataKPIAdmin-SQL.xml
--
-- PERHATIKAN penyaring Group Panel-nya: `('003','004','006')` — TANPA `009`, berbeda dari
-- kartu skor di atas. Itu keanehan nomor 1 pada kepala berkas ini, direplikasi apa adanya.
--
-- Tanggal diambil sebagai TANGGAL, bukan `to_char` seperti kueri lama: pemformatan
-- dikerjakan Go (`08-TECHNICAL-STRATEGY.md` §4.3).
--
-- Bind: :1 periode dari · :2 periode sampai · :3 offset · :4 jumlah baris
SELECT a.noklaim                AS CLAIM_NUMBER,
       b.nopolis                AS POLICY_NUMBER,
       b.businessname           AS BUSINESS_NAME,
       b.REGISTERDATE           AS REGISTER_DATE,
       b.TRANSFERPIC_DATE       AS TRANSFER_DATE,
       b.leader_member          AS TEAM_FLAG,
       datamining.get_working_hours@asmd.sinarmas.co.id(
         b.REGISTERDATE, b.TRANSFERPIC_DATE) / 28800 AS REGISTER_AGING,
       COUNT(*) OVER ()         AS TOTAL_ROWS
  FROM pooldata.pega_dashboardpnc a,
       pooldata.t_claim_pnc b,
       datapega.pc_asm_fw_gcnmfw_work d
 WHERE a.noklaim = b.claimno
   AND a.stsklaim <> '2'
   AND b.claimid = d.pzinskey
   AND d.pxcreateoperator IN
         ('SOPHIANOVITAEVELYN_1', 'SOPHIANOVITAEVELYN', 'RUTHCLARA')
   AND b.group_panel IN ('003', '004', '006')
   AND b.groupbisnisid NOT IN ('09', '11', '16', '25')
   AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
   AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY
 ORDER BY b.REGISTERDATE DESC, a.noklaim
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: admin_scorecard_pa
-- Kartu skor **PA** — satu baris.
-- — RDB List/GetDataKPIAdminPA-SQL.xml
--
-- Bentuknya BERBEDA dari NON-MBU, dan itu bukan penyederhanaan: yang diukur memang bukan
-- leader versus member melainkan DUA TAHAP — registrasi dan pembayaran.
--
--   regist_klaim_pa      klaim yang TAT registrasinya > 0 hari kerja
--   pembayaran_klaim_pa  klaim yang TAT pembayarannya > 0 hari kerja
--
-- Perhatikan ambangnya `> 0`, bukan `> 1` seperti NON-MBU. Ditiru apa adanya.
--
-- ============================================================================
-- RENTANG 2023 YANG TERTANAM — KEANEHAN NOMOR 2
-- ============================================================================
--
-- Pembagi `total_pembayaran_pa` menyaring `b.receivedate BETWEEN 01/01/2023 AND
-- 10/11/2023` — rentang TETAP yang tidak ikut berubah ketika pengguna memilih periode
-- lain. Ia tampak seperti sisa uji coba yang tertinggal di produksi.
--
-- Direplikasi apa adanya (`P-5`), dan ditandai sebagai selisih terencana supaya penguji
-- tidak melaporkannya sebagai cacat sistem baru. Bila Work Owner memutuskan memperbaikinya,
-- yang berubah hanya dua baris di bawah.
--
-- Bind: :1 periode dari · :2 periode sampai
SELECT register_over_sla                                       AS REGISTER_OVER_SLA,
       payment_over_sla                                        AS PAYMENT_OVER_SLA,
       claim_total                                             AS CLAIM_TOTAL,
       payment_total                                           AS PAYMENT_TOTAL,
       CASE
         WHEN (register_over_sla / NULLIF(claim_total, 0)) * 100 < 0.5             THEN 5
         WHEN (register_over_sla / NULLIF(claim_total, 0)) * 100 BETWEEN 0.5 AND 1 THEN 4
         WHEN (register_over_sla / NULLIF(claim_total, 0)) * 100 = 1               THEN 3
         WHEN (register_over_sla / NULLIF(claim_total, 0)) * 100 BETWEEN 1 AND 1.5 THEN 2
         WHEN (register_over_sla / NULLIF(claim_total, 0)) * 100 BETWEEN 1.5 AND 2 THEN 1
         ELSE 0
       END                                                     AS REGISTER_SCORE,
       CASE
         WHEN (payment_over_sla / NULLIF(claim_total, 0)) * 100 < 0.5              THEN 5
         WHEN (payment_over_sla / NULLIF(claim_total, 0)) * 100 BETWEEN 0.5 AND 1  THEN 4
         WHEN (payment_over_sla / NULLIF(claim_total, 0)) * 100 = 1                THEN 3
         WHEN (payment_over_sla / NULLIF(claim_total, 0)) * 100 BETWEEN 1 AND 1.5  THEN 2
         WHEN (payment_over_sla / NULLIF(claim_total, 0)) * 100 BETWEEN 1.5 AND 2  THEN 1
         ELSE 0
       END                                                     AS PAYMENT_SCORE
  FROM (SELECT
          (SELECT COUNT(CLAIMNO)
             FROM (SELECT b.CLAIMNO,
                          datamining.get_working_hours@asmd.sinarmas.co.id(
                            b.receivedate, b.REGISTERDATE) / 28800 AS tat_regis
                     FROM pooldata.t_claim_pnc b,
                          datapega.pc_asm_fw_gcnmfw_work d
                    WHERE b.claimid = d.pzinskey
                      AND b.STATUSWORK <> 'Resolved-Rejected'
                      AND d.pxcreateoperator IN
                            ('IRMANOPITAPURBA_1', 'IRMANOPITAPURBA', 'YUNIARPAMORSUARI')
                      AND b.GROUPPANEL IN ('002')
                      AND b.REGISTERDATE >= TO_DATE(:1, 'YYYY-MM-DD')
                      AND b.REGISTERDATE <  TO_DATE(:2, 'YYYY-MM-DD') + INTERVAL '1' DAY)
            WHERE tat_regis > 0) AS register_over_sla,

          (SELECT COUNT(DISTINCT CLAIMNO)
             FROM (SELECT b.CLAIMNO,
                          datamining.get_working_hours@asmd.sinarmas.co.id(
                            c.RECEIVEDATELOD, c.TGLAKSEPTASI) / 28800 AS tat_bayar
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
                      AND b.REGISTERDATE >= TO_DATE(:3, 'YYYY-MM-DD')
                      AND b.REGISTERDATE <  TO_DATE(:4, 'YYYY-MM-DD') + INTERVAL '1' DAY)
            WHERE tat_bayar > 0) AS payment_over_sla,

          (SELECT COUNT(CLAIMNO)
             FROM (SELECT b.CLAIMNO
                     FROM pooldata.t_claim_pnc b,
                          datapega.pc_asm_fw_gcnmfw_work d
                    WHERE b.claimid = d.pzinskey
                      AND b.STATUSWORK <> 'Resolved-Rejected'
                      AND d.pxcreateoperator IN
                            ('IRMANOPITAPURBA_1', 'IRMANOPITAPURBA', 'YUNIARPAMORSUARI')
                      AND b.GROUPPANEL IN ('002')
                      AND b.REGISTERDATE >= TO_DATE(:5, 'YYYY-MM-DD')
                      AND b.REGISTERDATE <  TO_DATE(:6, 'YYYY-MM-DD') + INTERVAL '1' DAY))
            AS claim_total,

          -- RENTANG 2023 YANG TERTANAM. Dua baris `TO_DATE` di bawah TIDAK memakai periode
          -- yang dipilih pengguna — persis seperti di Pega. Lihat kepala kueri ini.
          (SELECT COUNT(DISTINCT CLAIMNO)
             FROM (SELECT b.CLAIMNO,
                          datamining.get_working_hours@asmd.sinarmas.co.id(
                            c.RECEIVEDATELOD, c.TGLAKSEPTASI) / 28800 AS tat_bayar
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
                      AND b.receivedate <  TO_DATE('2023-11-10', 'YYYY-MM-DD') + INTERVAL '1' DAY)
            WHERE tat_bayar > 0) AS payment_total
       FROM DUAL)

-- name: admin_detail_pa
-- Grid rincian **PA** — satu baris per klaim, dipaginasi.
-- — RDB List/GetDataKPIAdminPA_khususPA-SQL.xml
--
-- Ia membawa DUA umur dan DUA penanda SLA, karena yang diukur dua tahap. Penanda SLA
-- ditulis basis data sebagai teks `SLA` / `TIDAK SLA`, dan itu dibawa apa adanya (`D-13`).
--
-- `RECEIVEDATELOD` yang kosong jatuh ke `TGLAKSEPTASI` — akibatnya umur pembayaran menjadi
-- nol pada baris seperti itu, bukan kosong. Ditiru apa adanya.
--
-- Paginasinya `OFFSET … FETCH NEXT`, bukan `rownum BETWEEN` seperti kueri lama: `ROWNUM`
-- khas Oracle dan `D-20` menggantinya. Halaman 25 baris milik Pega tidak dibawa; ukuran
-- halaman di sini ditentukan pemanggil.
--
-- Bind: :1 periode dari · :2 periode sampai · :3 offset · :4 jumlah baris
SELECT b.CLAIMNO             AS CLAIM_NUMBER,
       b.NOPOLIS             AS POLICY_NUMBER,
       g.STARTDATE           AS POLICY_START,
       g.ENDDATE             AS POLICY_END,
       d.pxcreateoperator    AS ADMIN_NAME,
       b.RECEIVEDATE         AS RECEIVE_DATE,
       b.REGISTERDATE        AS REGISTER_DATE,
       COALESCE(c.RECEIVEDATELOD, c.TGLAKSEPTASI) AS LOD_RECEIVE_DATE,
       c.TGLAKSEPTASI        AS ACCEPTANCE_DATE,
       datamining.get_working_hours@asmd.sinarmas.co.id(
         b.RECEIVEDATE, b.REGISTERDATE) / 28800 AS REGISTER_AGING,
       datamining.get_working_hours@asmd.sinarmas.co.id(
         COALESCE(c.RECEIVEDATELOD, c.TGLAKSEPTASI), c.TGLAKSEPTASI) / 28800 AS PAYMENT_AGING,
       CASE WHEN datamining.get_working_hours@asmd.sinarmas.co.id(
                   b.RECEIVEDATE, b.REGISTERDATE) / 28800 > 1
            THEN 'TIDAK SLA' ELSE 'SLA' END AS REGISTER_SLA,
       CASE WHEN datamining.get_working_hours@asmd.sinarmas.co.id(
                   COALESCE(c.RECEIVEDATELOD, c.TGLAKSEPTASI), c.TGLAKSEPTASI) / 28800 > 1
            THEN 'TIDAK SLA' ELSE 'SLA' END AS PAYMENT_SLA,
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
