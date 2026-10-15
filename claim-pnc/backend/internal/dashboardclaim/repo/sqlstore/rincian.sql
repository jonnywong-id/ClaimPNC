-- Rincian satu klaim — isi popup yang terbuka saat nomor klaim diklik.
--
-- # Apa yang digantikan
--
-- `Section/DashboardClaimShow_Sec-Section.xml` memasang kolom nomor klaim sebagai **Link**:
--
--	pyControlDisplayTitle  Link
--	pyUIElement            link
--	pyEvent                click
--	pyAction               showHarness
--	pyHarnessName          ViewTempDetailClaim
--	pyActivity             setDataViewKlaim_Act
--	pyTarget               popup
--
-- Jadi klik nomor klaim menjalankan `setDataViewKlaim_Act` lalu membuka harness
-- `ViewTempDetailClaim` sebagai popup. Kueri ini menggantikan langkah pertamanya.
--
-- # SATU dokumen JSON, bukan belasan kueri
--
-- `setDataViewKlaim_Act` mengambil seluruh isi klaim dari satu baris:
--
--	RDB List/GetJsonKlaimPNC-SQL.xml
--	  select data_json as "LSC_ID" from pooldata.json_klaim where idpega = {…}
--
-- lalu menguraikannya menjadi ClaimData, ObjectList, ObjectCoverageList, AdjustmentList,
-- ComiteeClaim, ClaimReceiver, Policy, SpreadingRisk, dan Coins.
--
-- ============================================================================
-- JSON TIDAK DIPAKAI LAGI — KEPUTUSAN WORK OWNER 2026-10-12
-- ============================================================================
--
-- Kueri di bawah dulu menggabung `POOLDATA.JSON_KLAIM` dan mengembalikan dokumennya utuh.
-- Gabungan itu DILEPAS. Isi popup kini dibaca dari tabel relasional — lihat
-- `rincian_bagian.sql`.
--
-- Sebabnya terukur: `data_json` terisi **721 dari 16.284 baris**, dan hanya **348 dari 2.758**
-- klaim dashboard punya isinya. Popup karena itu tampil kosong bagi mayoritas klaim — dan di
-- Pega pun demikian, karena rule-nya membaca kolom yang sama.
--
-- Arahnya juga sudah tertulis di Steering: `R-10` menetapkan di sistem baru data klaim
-- disimpan sebagai tabel relasional, dan JSON hanya untuk snapshot polis (`D-04`).
--
-- # Kuncinya PZINSKEY, dan grid sudah memegangnya
--
-- Baris grid membawa `klaim_id` = PZINSKEY, sehingga popup tidak perlu mencarinya ulang lewat
-- nomor klaim — dan dengan begitu tidak ada klaim bernomor kembar yang tertukar.

-- name: klaim_rincian
-- Bind: :1 PZINSKEY klaim
SELECT W.PYID            AS NOMOR_KLAIM,
       W.PZINSKEY        AS KLAIM_ID,
       W.PYSTATUSWORK    AS STATUS_PROSES,
       -- Status klaim dari tabel datar, DENGAN cadangan dari tabel klaim warisan.
       --
       -- Diukur 2026-10-12 atas 2.758 klaim di `T_CLAIMLIST_ADMIN`:
       --
       --     STATUSCLAIM_1 terisi            2.084
       --     setelah cadangan ini            2.193   (+109)
       --
       -- Tanpa cadangan, popup menjawab "—" pada klaim yang halaman `/registrasi/klaim/:nomor`
       -- justru menampilkan statusnya — dua layar menampilkan hal berbeda untuk klaim yang
       -- sama. Itu yang dilaporkan Work Owner pada `PNC-11`.
       --
       -- Halaman itu membaca `POOLDATA.T_CLAIM_PNC.STATUSCLAIM`, dan ini membaca kolom yang
       -- sama — bukan menebak nilainya.
       COALESCE(W.STATUSCLAIM_1, P.STATUSCLAIM) AS STATUS_KLAIM,
       W.USERTEKNIS_1    AS PIC_TEKNIK,
       W.PXCREATEOPNAME  AS ADMIN_PNC,
       W.PXCREATEDATETIME AS DIDAFTARKAN_PADA,
       -- Lini bisnis klaim, dipakai layar memilih varian grid.
       --
       -- Dulu kedua nilai ini diambil dari dokumen JSON. Setelah dokumen itu dilepas,
       -- keduanya HARUS ikut di kepala: tanpa itu penjaga IsPA/IsHE/IsFire tidak punya
       -- bahan, dan layar akan menyatakan setiap klaim "tidak termasuk yang digambar".
       W.GROUPPANEL_1    AS GROUP_PANEL,
       P.POLIS_JENIS_BISNIS AS JENIS_BISNIS,
       -- "Catatan dari Inputor" pada kepala popup — dulu dari ClaimData.Remark di dokumen.
       --
       -- "Aging Amount" yang berdampingan dengannya TIDAK punya padanan relasional: ia milik
       -- data polis (Policy.AgingAmount), dan polis bukan milik bounded context ini (D-04).
       -- Isiannya karena itu tetap kosong, dan itu dinyatakan — bukan diisi nilai karangan.
       P.REMARK          AS CATATAN_INPUTOR,
       -- ---------------------------------------------------------------------
       -- Isian tab "Registrasi" — `ISIAN_REGISTER` pada layar.
       -- ---------------------------------------------------------------------
       --
       -- Keenam belas isian ini dulu digali dari dokumen JSON, dan setelah dokumen dilepas
       -- tab Registrasi tampil sebagai penanda "belum tersambung". Work Owner meminta
       -- isiannya PERSIS, bukan ditautkan ke layar lain yang kebetulan memuatnya.
       --
       -- Ketiga belas yang punya padanan kolom dibaca di sini. Kolomnya SAMA dengan yang
       -- dibaca `registrasi/repo/sqlstore/claim.sql` `klaim_ambil` — bukan kolom lain yang
       -- kebetulan mirip namanya, sehingga kedua layar tidak dapat berbeda isi.
       P.RECEIVEDATE            AS TANGGAL_TERIMA_DOKUMEN,
       P.DATEOFLOSS             AS TANGGAL_KEJADIAN,
       P.TANGGALSELESAIRAWATINAP AS TANGGAL_KELUAR_RAWAT_INAP,
       P.REPORTDATE             AS TANGGAL_LAPOR,
       P.NO_HP                  AS TELEPON_PELAPOR,
       P.PELAPOR_HUBUNGAN       AS STATUS_PELAPOR,
       P.PELAPOR_HUBUNGAN_LAIN  AS STATUS_PELAPOR_LAIN,
       P.LOCATION               AS LOKASI_KEJADIAN,
       P.DISTRICT               AS KELURAHAN,
       P.STS_TKI                AS TERTANGGUNG_TKI,
       P.REMARK                 AS CATATAN_KE_ANALYST,
       P.REMARKRECOMENDATION    AS CATATAN_KE_PIC_TEKNIS,
       P.EMAIL_LOD              AS EMAIL_PELAPOR
  FROM POOLDATA.T_CLAIMLIST_ADMIN W
       -- LEFT JOIN, bukan INNER: klaim yang belum ada di tabel warisan tetap harus tampil.
       --
       -- Kuncinya `CLAIMID`, bukan `CLAIMNO`. Keduanya diukur berdampingan — `CLAIMID` cocok
       -- pada 1.513 klaim, `CLAIMNO` pada 1.464 — dan `CLAIMID` TUNGGAL (nol ganda),
       -- sedangkan `CLAIMNO` terbukti punya nilai ganda di tabel itu.
       LEFT JOIN POOLDATA.T_CLAIM_PNC P
              ON P.CLAIMID = W.PZINSKEY
-- Dicari lewat KUNCI PEGA **atau** NOMOR KLAIM, dan keduanya memakai nilai yang sama.
--
-- # Kenapa dua-duanya
--
-- Grid memegang `PZINSKEY` dan memakainya langsung. Halaman detail memakai NOMOR klaim di
-- alamatnya, karena kunci teknis Pega tidak pernah dipakai sebagai alamat (`D-22`) — dan
-- `PZINSKEY` memuat spasi (`ASM-FW-GCNMFW-WORK PNC-13`), yang akan tampak di bilah alamat.
--
-- # Kenapa aman
--
-- `PYID` diukur TUNGGAL pada 2026-10-12: 2.777 baris dashboard, **nol** `PYID` ganda. Tanpa
-- pengukuran itu, pencarian lewat nomor dapat mengembalikan klaim yang salah tanpa satu pun
-- galat — dan yang terlihat hanyalah rincian milik orang lain.
--
-- Bind: :1 dipakai untuk KEDUANYA, dikirim dua kali sebagai :1 dan :2.
 WHERE (W.PZINSKEY = :1 OR W.PYID = :2)
   AND W.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'

-- name: klaim_rincian_check
-- Membuktikan kedua tabel terbaca beserta kolom yang dipakai `klaim_rincian`.
--
-- Kolomnya disebut satu per satu, bukan `SELECT 1`: kegagalan nyata di modul ini dua kali
-- berupa kolom yang tidak ada, dan probe `SELECT 1` lulus terhadapnya.
--
-- `PXOBJCLASS` ikut disebut sejak 2026-10-08. Ia hanya muncul di klausa WHERE kueri nyatanya,
-- sehingga versi sebelumnya tidak membuktikannya ada — lihat `TestProbesNameEveryColumnTheyGuard`.
SELECT W.PYID, W.PZINSKEY, W.PYSTATUSWORK, W.STATUSCLAIM_1,
       W.USERTEKNIS_1, W.PXCREATEOPNAME, W.PXCREATEDATETIME,
       P.STATUSCLAIM, P.CLAIMID, W.GROUPPANEL_1, P.POLIS_JENIS_BISNIS, P.REMARK,
       P.RECEIVEDATE, P.DATEOFLOSS, P.TANGGALSELESAIRAWATINAP, P.REPORTDATE,
       P.NO_HP, P.PELAPOR_HUBUNGAN, P.PELAPOR_HUBUNGAN_LAIN, P.LOCATION,
       P.DISTRICT, P.STS_TKI, P.REMARKRECOMENDATION, P.EMAIL_LOD
  FROM POOLDATA.T_CLAIMLIST_ADMIN W
       LEFT JOIN POOLDATA.T_CLAIM_PNC P
              ON P.CLAIMID = W.PZINSKEY
 WHERE 1 = 0
   AND W.PXOBJCLASS IS NOT NULL
