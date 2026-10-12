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
-- ComiteeClaim, ClaimReceiver, Policy, SpreadingRisk, dan Coins — kesembilan kelas itu
-- terbaca dari daftar kelas yang dirujuk activity-nya.
--
-- # KOLOMNYA `data_json`, BUKAN `DATA_JSONBLOB`
--
-- `POOLDATA.JSON_KLAIM` memuat KEDUANYA, dan rule yang berbeda membaca yang berbeda:
--
--	data_json        GetJsonKlaimPNC            ← layar INI
--	DATA_JSONBLOB    GetBrowseDataAIPA, GetClaimTreaty_SQL, BroswseKlaimByRegisterDate
--
-- Yang dipakai di sini `data_json`, karena itu yang dibaca rule layar ini. Memilih yang lain
-- karena "modul sebelah memakainya" adalah tebakan atas kolom yang isinya belum pernah
-- dibandingkan — dan keduanya mungkin memang berbeda isi.
--
-- # Kuncinya PZINSKEY, dan grid sudah memegangnya
--
-- `GetJsonKlaimPNC` menyaring `idpega`, yang berpasangan dengan `PZINSKEY` pada tabel kerja.
-- Baris grid sudah membawa `klaim_id` = PZINSKEY, sehingga popup tidak perlu mencarinya ulang
-- lewat nomor klaim — dan dengan begitu tidak ada klaim bernomor kembar yang tertukar.
--
-- # LEFT JOIN, bukan INNER
--
-- Klaim yang belum punya baris di JSON_KLAIM tetap DAPAT DIBUKA, dengan isian kosong. INNER
-- JOIN akan membuatnya menjawab "klaim tidak ditemukan" — padahal klaimnya ada, hanya
-- dokumennya yang belum terbentuk.

-- name: klaim_rincian
-- Bind: :1 PZINSKEY klaim
SELECT W.PYID            AS NOMOR_KLAIM,
       W.PZINSKEY        AS KLAIM_ID,
       W.PYSTATUSWORK    AS STATUS_PROSES,
       W.STATUSCLAIM_1   AS STATUS_KLAIM,
       W.USERTEKNIS_1    AS PIC_TEKNIK,
       W.PXCREATEOPNAME  AS ADMIN_PNC,
       W.PXCREATEDATETIME AS DIDAFTARKAN_PADA,
       J.DATA_JSON       AS DOKUMEN
  FROM POOLDATA.T_CLAIMLIST_ADMIN W
       LEFT JOIN POOLDATA.JSON_KLAIM J
              ON W.PZINSKEY = J.IDPEGA
 WHERE W.PZINSKEY = :1
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
       W.USERTEKNIS_1, W.PXCREATEOPNAME, W.PXCREATEDATETIME, J.DATA_JSON
  FROM POOLDATA.T_CLAIMLIST_ADMIN W
       LEFT JOIN POOLDATA.JSON_KLAIM J
              ON W.PZINSKEY = J.IDPEGA
 WHERE 1 = 0
   AND W.PXOBJCLASS IS NOT NULL
