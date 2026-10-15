-- Isi popup rincian klaim, dibaca dari TABEL RELASIONAL.
--
-- ============================================================================
-- KENAPA BUKAN DARI JSON
-- ============================================================================
--
-- Sampai 2026-10-12 seluruh isi popup datang dari satu dokumen
-- `POOLDATA.JSON_KLAIM.DATA_JSON`, meniru Pega apa adanya:
--
--     select data_json as "LSC_ID" from pooldata.json_klaim where idpega={…}
--     (RDB List/GetJsonKlaimPNC-SQL.xml)
--
-- Kolom itu terisi **721 dari 16.284 baris**, dan hanya **348 dari 2.758** klaim dashboard
-- yang punya isinya. Jadi popup tampil kosong bagi mayoritas klaim — di Pega pun demikian.
--
-- Work Owner memutuskan 2026-10-12: **migrasi tidak memakai JSON lagi, langsung ke kolom.**
-- Itu juga arah Steering — `R-10` menetapkan di sistem baru data klaim disimpan sebagai
-- tabel relasional, dan JSON hanya untuk snapshot polis (`D-04`).
--
-- ============================================================================
-- KUNCI PENAUT — DIUKUR, BUKAN DIDUGA
-- ============================================================================
--
--     tabel                      kunci                     cocok dari 2.758 klaim
--     T_CLAIM_ADJUSTMENT         CLAIMID   = PZINSKEY       1.752
--     T_CLAIM_OBJECTLIST         CLAIMID   = PZINSKEY       1.434
--     T_CLAIM_OBJECTCOVERAGE     CLAIMID   = PZINSKEY       1.409
--     T_CLAIM_RECEIVER           CLAIMID   = PZINSKEY       1.352
--     DATA_ATTACHFILE            IDPEGA    = PZINSKEY         779
--     T_SURVEYORLIST             PNCCASEID = PZINSKEY         113
--     GCNM_PROGRESS_CLAIM        PNCCASEID = **PYID**        1.725
--
-- Baris terakhir yang paling mudah salah: `GCNM_PROGRESS_CLAIM.PNCCASEID` menyimpan **nomor
-- klaim**, bukan kunci Pega. Memakai `PZINSKEY` di sana mengembalikan 98 — dan 98 baris tidak
-- terlihat seperti kesalahan, hanya seperti klaim yang belum punya progres.
--
-- ============================================================================
-- SOFT DELETE
-- ============================================================================
--
-- `T_CLAIM_OBJECTLIST` dan `T_CLAIM_OBJECTCOVERAGE` punya kolom `DIHAPUS_PADA`. `D-66`
-- menetapkan penghapusan dinyatakan lewat penanda, dan setiap kueri pembaca **wajib
-- menyaringnya**. Terisi pada 10 baris objek hari ini — kecil, tetapi satu kueri yang lupa
-- akan menampilkan objek yang sudah dihapus petugas, tanpa galat.

-- name: rincian_penerima
-- Tab **Penerima Pembayaran Klaim** — `Section/ViewShowReceiver-Section.xml`.
--
-- Kolomnya mengikuti grid layar lama: Nama dan Alamat. `NAMEOFBANK` dan `NOACCOUNT` ADA di
-- tabel tetapi TIDAK ditampilkan grid itu, dan tidak dibawa ke sini — nomor rekening adalah
-- data nasabah (`D-69`), dan mengirim yang tidak digambar hanya memperluas yang bocor.
-- Bank dan nomor rekening ikut dibaca untuk SUB-POPUP "Detail Penerima", yang membuka baris
-- grid ini. Keduanya ada di tabel yang sama, sehingga tidak ada kueri tambahan.
--
-- Tabel ini hanya punya ENAM kolom seluruhnya — dihitung dari katalog, bukan diperkirakan.
-- Karena itu tiga isian sub-popup (Telepon, Nomor Kontrak, Nama Cabang Bank) tidak punya
-- padanan, dan ketiadaannya pasti, bukan belum ditelusuri.
SELECT R.NAME, R.ADDRESS, R.NAMEOFBANK, R.NOACCOUNT
  FROM POOLDATA.T_CLAIM_RECEIVER R
 WHERE R.CLAIMID = :1
 ORDER BY R.IDRECEIVER

-- name: rincian_objek
-- Tab **Registrasi** — grid objek pertanggungan, `Section/ViewInputRegisterDetail-Section.xml`.
--
-- Kedelapan kolom mengikuti `KOLOM_OBJEK` yang dibaca dari `pyValue` section-nya. Kolom
-- kendaraan ikut dibawa karena grid yang sama melayani lini kendaraan dan lini orang —
-- yang tidak terpakai tampil kosong, persis seperti di layar lama.
SELECT O.OBJECTNAME,
       O.LOKASI,
       O.OBJECTPARTICIPANTSTATUS,
       O.OBJECTIDCARD,
       O.DATEOFBIRTH,
       O.OBJECTJOB,
       O.TYPENAME,
       O.CHASISSNUMBER
  FROM POOLDATA.T_CLAIM_OBJECTLIST O
 WHERE O.CLAIMID = :1
   AND O.DIHAPUS_PADA IS NULL
 ORDER BY O.URUTAN, O.OBJECTID

-- name: rincian_coverage
-- Tab **Estimasi** — grid coverage, `Section/ViewInputEstimasiDetail-Section.xml`.
--
-- `COVERAGEID` dibawa sebagai "Nama Plan" mengikuti grid lama, yang memang menampilkan
-- kodenya — bukan namanya. Itu diwarisi apa adanya (`P-5`).
SELECT C.COVERAGENAME,
       C.COVERAGEID,
       C.SUMTSI,
       C.CURRENCY,
       C.CAUSEOFLOSS,
       C.OBJECTNAME
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE C
 WHERE C.CLAIMID = :1
   AND C.DIHAPUS_PADA IS NULL
 ORDER BY C.URUTAN_OBJEK, C.URUTAN, C.OBJECTCOVERAGEID

-- name: rincian_adjustment
-- Tab **Adjustment & Akseptasi** — `Section/ViewShowObjectAdj-Section.xml`.
--
-- Nilai uang dibawa APA ADANYA tanpa pembulatan: `D-51` menetapkan nilai uang disimpan
-- presisi penuh dan dibulatkan hanya saat ditampilkan. Pembulatan di SQL akan memindahkan
-- keputusan itu ke tempat yang tidak terlihat pembacanya.
SELECT A.NOAKSEPTASI,
       A.TGLAKSEPTASI,
       A.STATUSAKSEPTASI,
       A.PAYMENTTYPE,
       A.RECEIVERNAME,
       A.CURRENCY,
       A.NILAIAKSEPTASI,
       A.PROPOSE_VALUE,
       A.TOTAL_CLAIM,
       A.TRANSFER_CASHIER_DATE,
       A.CLAIM_STATUS_PAID
  FROM POOLDATA.T_CLAIM_ADJUSTMENT A
 WHERE A.CLAIMID = :1
 ORDER BY A.OBJECTID, A.OBJECTCOVERAGEID, A.ADJUSTMENTID

-- name: rincian_survey
-- Tab **Hasil Survey** — `Section/ViewHasilSurvey-Section.xml`.
--
-- Satu baris per LANGKAH survei, bukan per berkas — berbeda dari tile Loss Adjuster yang
-- mengambil langkah terakhir saja. Di sini riwayatnya memang yang ditampilkan.
-- Empat kolom terakhir dibaca untuk SUB-POPUP yang membuka baris ini — "Hasil Survey" dan
-- "Detail Hasil Surveyor". Keempatnya ada di tabel yang sama.
--
-- `LOCATION_SURVEY` dan `RESCHEDULE_LOCATION` keduanya dibawa karena keduanya NILAI BERBEDA:
-- yang pertama lokasi survei, yang kedua lokasi penjadwalan ulang. Grid menggambar yang kedua
-- (meniru layar lama), sub-popup menggambar yang pertama.
SELECT S.SURVEYDATE,
       S.SURVEYOR_NAME,
       S.OBJECT_NAME,
       S.RESCHEDULE_LOCATION,
       S.STS_SURVEY,
       S.INDEX_SURVEY,
       S.REFNO,
       S.KETERANGAN,
       S.LOCATION_SURVEY,
       S.LOSSTYPE
  FROM POOLDATA.T_SURVEYORLIST S
 WHERE S.PNCCASEID = :1
 ORDER BY S.CASEID, LPAD(S.INDEX_SURVEY, 5, '0')

-- name: rincian_lampiran
-- Riwayat lampiran — `Section/GCNMViewAttachmentHistory-Section.xml`.
--
-- Digambar DI BAWAH tab, bukan sebagai tab, mengikuti harness `ViewTempDetailClaim`.
--
-- `ATTACHFILE` dan `IMAGEID` TIDAK dibawa: yang pertama isi berkasnya, yang kedua rujukan ke
-- storage. Popup hanya menampilkan daftar, dan mengirim isi berkas untuk daftar berarti
-- memindahkan megabita untuk menggambar beberapa baris.
SELECT D.CATEGORY,
       D.ATTACHNAME,
       D.INPUTDATE,
       D.INPUTOPERATOR
  FROM POOLDATA.DATA_ATTACHFILE D
 WHERE D.IDPEGA = :1
 ORDER BY D.INPUTDATE DESC, D.DATAID

-- name: rincian_progress
-- Tab **Progress Klaim** — `Section/ProgressCloseClaim-Section.xml`.
--
-- KUNCINYA NOMOR KLAIM, bukan kunci Pega. Diukur: `PNCCASEID = PYID` cocok pada 1.725 klaim,
-- sedangkan `PNCCASEID = PZINSKEY` hanya 98. Keduanya berjalan tanpa galat; yang kedua hanya
-- mengembalikan sedikit baris, dan itu tidak terlihat seperti kesalahan.
--
-- Nama tahapannya dari master `GCNM_MST_PROGRESS_KLAIM` lewat LEFT JOIN: baris progres yang
-- kodenya tidak terdaftar di master tetap tampil, dengan namanya kosong. Membuangnya akan
-- menyembunyikan progres yang benar-benar terjadi hanya karena masternya belum lengkap.
-- `NEXT_FOLLOWUP` dibaca untuk sub-popup "Detail Progress" yang membuka baris ini.
SELECT P.TGL_INPUT,
       P.STATUS_PROGRESS1,
       P.STATUS,
       P.KETERANGAN,
       P.USER_INPUT,
       M.STATUS AS NAMA_TAHAP,
       P.NEXT_FOLLOWUP
  FROM POOLDATA.GCNM_PROGRESS_CLAIM P
       LEFT JOIN POOLDATA.GCNM_MST_PROGRESS_KLAIM M
              ON M.ID_PROGRESS = P.STATUS_PROGRESS1
 WHERE P.PNCCASEID = :1
 ORDER BY P.TGL_INPUT DESC

-- name: rincian_permintaan_survey
-- Isian tab **Hasil Survey** bagian atas — `ISIAN_SURVEY` pada layar.
--
-- Kedelapan isiannya dulu digali dari `ClaimData.SurveyData.*` di dokumen JSON. Seluruhnya
-- punya padanan SATU LAWAN SATU di `POOLDATA.T_REQ_SURVEY`, dan kecocokan 8-dari-8 itulah
-- yang membuktikan tabel ini memang rumahnya — bukan kemiripan nama:
--
--     RequestorName      -> REQ_NAME          Cabang        -> BRANCH
--     LocationSurvey     -> LOCATION          SurveyorName  -> SURVEYOR
--     NoTelpDiHubungi    -> NOTELP            EmailSurvey   -> EMAIL_SURVEYOR
--     RequestSurveyDate  -> DATE_SURVEY       ObjectName    -> OBJECTNAME
--
-- Dipakai permintaan survei TERBARU bila ada lebih dari satu. Layar lama menggambar satu
-- himpunan isian, bukan daftar, sehingga memilih yang terbaru meniru perilakunya; memilih
-- yang terlama akan menampilkan permintaan yang sudah digantikan.
--
-- Bind: :1 PZINSKEY klaim
SELECT REQ_NAME, LOCATION, NOTELP, DATE_SURVEY,
       BRANCH, SURVEYOR, EMAIL_SURVEYOR, OBJECTNAME
  FROM POOLDATA.T_REQ_SURVEY
 WHERE CLAIMID = :1
 ORDER BY INPUTDATE DESC, SURVEYID DESC
 FETCH FIRST 1 ROWS ONLY

-- name: rincian_pucl
-- Dua isian tab **Catatan Untuk Analis** — `ISIAN_CATATAN` pada layar.
--
-- `TanggalTerimaDokumenPUCL` dan `AlasanDokterRejectRCL` dicari ke SELURUH skema POOLDATA,
-- bukan ditebak dari `T_CLAIM_PNC`; keduanya ada di `TC_PNC_PUCL` dan tidak di tempat lain.
--
-- Dua isian lain pada tab yang sama — `PilihanCompliance` dan `NotePilihanCompliance` —
-- **tidak punya kolom di mana pun**. Pencarian menyeluruh atas `ALL_TAB_COLUMNS` POOLDATA
-- tidak menemukan satu pun kolom yang sepadan. Keduanya dinyatakan kosong di layar, bukan
-- diisi dari kolom lain yang kebetulan mirip.
--
-- Bind: :1 PZINSKEY klaim
SELECT TGL_TERIMA_DOKUMEN_PUCL, ALASAN_DOKTER_REJECT_RCL
  FROM POOLDATA.TC_PNC_PUCL
 WHERE CLAIMID = :1
 ORDER BY TGL_CREATE_PUCL DESC
 FETCH FIRST 1 ROWS ONLY

-- name: rincian_bagian_check
-- Membuktikan kesembilan tabel bagian terbaca akun aplikasi, beserta kolom yang dipakai.
--
-- Satu pernyataan, bukan tujuh: yang diperiksa parse-nya, dan tujuh probe terpisah hanya
-- menambah tujuh baris pada keluaran `-periksa` tanpa menambah satu pun yang dibuktikan.
--
-- Kolomnya disebut satu per satu. Probe yang hanya membuktikan tabelnya ada akan LULUS
-- terhadap kolom yang tidak ada — kegagalan yang sudah terjadi tiga kali di modul ini
-- (`TOTAL_JOB`, `CASEID_1`, lalu `NOKLAIM`).
SELECT R.NAME, R.ADDRESS, R.CLAIMID, R.NAMEOFBANK, R.NOACCOUNT, R.IDRECEIVER,
       S.REFNO, S.KETERANGAN AS S_KETERANGAN, S.LOCATION_SURVEY, S.LOSSTYPE,
       P.NEXT_FOLLOWUP,
       O.OBJECTNAME, O.LOKASI, O.OBJECTPARTICIPANTSTATUS, O.OBJECTIDCARD,
       O.DATEOFBIRTH, O.OBJECTJOB, O.TYPENAME, O.CHASISSNUMBER, O.DIHAPUS_PADA, O.URUTAN,
       C.COVERAGENAME, C.COVERAGEID, C.SUMTSI, C.CURRENCY, C.CAUSEOFLOSS,
       C.OBJECTNAME AS C_OBJECTNAME, C.DIHAPUS_PADA AS C_DIHAPUS, C.URUTAN_OBJEK,
       A.NOAKSEPTASI, A.TGLAKSEPTASI, A.STATUSAKSEPTASI, A.PAYMENTTYPE, A.RECEIVERNAME,
       A.NILAIAKSEPTASI, A.PROPOSE_VALUE, A.TOTAL_CLAIM, A.TRANSFER_CASHIER_DATE,
       A.CLAIM_STATUS_PAID, A.ADJUSTMENTID,
       S.SURVEYDATE, S.SURVEYOR_NAME, S.OBJECT_NAME, S.RESCHEDULE_LOCATION,
       S.STS_SURVEY, S.INDEX_SURVEY, S.PNCCASEID,
       D.CATEGORY, D.ATTACHNAME, D.INPUTDATE, D.INPUTOPERATOR, D.IDPEGA,
       P.TGL_INPUT, P.STATUS_PROGRESS1, P.STATUS, P.KETERANGAN, P.USER_INPUT, P.PNCCASEID AS P_KUNCI,
       M.ID_PROGRESS, M.STATUS AS M_STATUS,
       Q.REQ_NAME, Q.LOCATION AS Q_LOCATION, Q.NOTELP, Q.DATE_SURVEY, Q.BRANCH,
       Q.SURVEYOR, Q.EMAIL_SURVEYOR, Q.OBJECTNAME AS Q_OBJECTNAME, Q.CLAIMID AS Q_KUNCI,
       Q.INPUTDATE, Q.SURVEYID,
       U.TGL_TERIMA_DOKUMEN_PUCL, U.ALASAN_DOKTER_REJECT_RCL, U.CLAIMID AS U_KUNCI,
       U.TGL_CREATE_PUCL
  FROM POOLDATA.T_CLAIM_RECEIVER R,
       POOLDATA.T_CLAIM_OBJECTLIST O,
       POOLDATA.T_CLAIM_OBJECTCOVERAGE C,
       POOLDATA.T_CLAIM_ADJUSTMENT A,
       POOLDATA.T_SURVEYORLIST S,
       POOLDATA.DATA_ATTACHFILE D,
       POOLDATA.GCNM_PROGRESS_CLAIM P,
       POOLDATA.GCNM_MST_PROGRESS_KLAIM M,
       POOLDATA.T_REQ_SURVEY Q,
       POOLDATA.TC_PNC_PUCL U
 WHERE 1 = 0
