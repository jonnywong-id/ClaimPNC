-- Tab Survey tahap Choose Surveyor — Section/TabSurvey_sect.xml (lihat registrasi/survey.go).
--
-- Isian survey per objek disimpan di kolom survey POOLDATA.T_CLAIM_OBJECTLIST
-- (docs/ddl/t_claim_objectlist_survey.sql). OBJECTSURVEYLOCATION, SURVEYORTYPE, dan BRANCHCODE
-- sudah ada di tabel; sisanya ditambahkan DDL itu.

-- name: survey_objek_daftar
-- Bind: :1 CLAIMID.
SELECT o.URUTAN, o.OBJECTID, o.OBJECTNAME, o.LOKASI,
       o.PILIHSURVEY, o.OBJECTSURVEYLOCATION, o.SURVEYORTYPE, o.OBJECTSURVEYOR,
       o.OBJECTSURVEYORLOGIN, o.SURVEYORADDRESS, o.OBJECTSURVEYOROTHERS, o.BRANCHCODE,
       o.SURVEYORBRANCHNAME, o.OBJECTSURVEYORMARINE, o.OBJECTSURVEYORMARINELOGIN,
       o.OBJECTSTATUS, o.OBJECTSURVEYID, o.OBJECTSURVEYIDMARINE
  FROM POOLDATA.T_CLAIM_OBJECTLIST o
 WHERE o.CLAIMID = :1
   AND o.DIHAPUS_PADA IS NULL
 ORDER BY o.URUTAN, o.OBJECTID

-- name: survey_objek_simpan
-- Hanya kolom survey; kolom objek lain milik penyimpanan klaim (claim.sql).
-- Bind: :1..:15 isian, :16 pengubah, :17 waktu, :18 CLAIMID, :19 OBJECTID.
UPDATE POOLDATA.T_CLAIM_OBJECTLIST
   SET PILIHSURVEY = :1, OBJECTSURVEYLOCATION = :2, SURVEYORTYPE = :3, OBJECTSURVEYOR = :4,
       OBJECTSURVEYORLOGIN = :5, SURVEYORADDRESS = :6, OBJECTSURVEYOROTHERS = :7, BRANCHCODE = :8,
       SURVEYORBRANCHNAME = :9, OBJECTSURVEYORMARINE = :10, OBJECTSURVEYORMARINELOGIN = :11,
       OBJECTSTATUS = :12, OBJECTSURVEYID = :13, OBJECTSURVEYIDMARINE = :14, ISKOMITEAPPROVE = :15,
       DIUBAH_OLEH = :16, DIUBAH_PADA = :17
 WHERE CLAIMID = :18
   AND OBJECTID = :19
   AND DIHAPUS_PADA IS NULL

-- name: survey_surveyor_internal
-- RDB List/BrowseSurveyorTypeInternalSurveyor-SQL.xml:85 — urut nama cabang.
SELECT a.D_SURVEY_ID, a.NAME, a.LOGIN_APLIKASI, a.ADDRESS, a.BRANCH, a.BRANCHNAME, a.EMAIL, a.OTHER_CONTACT
  FROM POOLDATA.V_D_SURVEYORS a
 WHERE a.EMAIL IS NOT NULL
   AND a.M_SURVEY_ID = '1001'
   AND a.LOGIN_APLIKASI IS NOT NULL
 ORDER BY a.BRANCHNAME

-- name: survey_surveyor_tipe
-- BrowseSurveyorTypeLossAdjuster / Expert / SurveyAgent — satu M_SURVEY_ID, urut nama.
-- Bind: :1 M_SURVEY_ID.
SELECT a.D_SURVEY_ID, a.NAME, a.LOGIN_APLIKASI, a.ADDRESS, a.BRANCH, a.BRANCHNAME, a.EMAIL, a.OTHER_CONTACT
  FROM POOLDATA.V_D_SURVEYORS a
 WHERE a.M_SURVEY_ID = :1
 ORDER BY a.NAME

-- name: survey_surveyor_nominasi
-- RDB List/BrowseNominatedLossAdjuster-SQL.xml:90 (browse).
SELECT a.D_SURVEY_ID, a.NAME, a.LOGIN_APLIKASI, a.ADDRESS, a.BRANCH, a.BRANCHNAME, a.EMAIL, a.OTHER_CONTACT
  FROM POOLDATA.V_D_SURVEYORS a
 WHERE a.M_SURVEY_ID IN ('1002', '1004')
 ORDER BY a.NAME

-- name: survey_nomor_berikut
-- Nomor survey SRVN.YY.n: deret per tahun dari nomor terbesar di T_SURVEYORLIST, tanpa nol di
-- depan — pola yang sama dengan PNCN/RCVN/KMTN. Bind: :1 'SRVN.YY.%'.
SELECT COALESCE(MAX(TO_NUMBER(SUBSTR(CASEID, 9))), 0) + 1
  FROM POOLDATA.T_SURVEYORLIST
 WHERE CASEID LIKE :1

-- name: survey_sisip
-- INSERT_SURVEYORLIST.prc:27-35 (cabang sisip). TGLINPUT = waktu simpan.
INSERT INTO POOLDATA.T_SURVEYORLIST
       (CASEID, PNCCASEID, SURVEYTYPE, LOSSTYPE, SURVEYOR_NAME, SURVEYDATE, LOCATION_SURVEY,
        OBJECT_NAME, LOCATION_OBJECT, IDOBJECT, INDEX_SURVEY, TGLPERAWATAN, NAMA_PASIEN,
        STS_SURVEY, KETERANGAN, TGLINPUT, RESCHEDULE_LOCATION, RESCHEDULE_DATE, SURVEYOR_NAME_MARINE)
VALUES (:1, :2, :3, NULL, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, NULL, :14, :15, :16, :17)

-- name: survey_baris_klaim
-- Survey tersimpan klaim (pengganti .ClaimData.SurveyResults). Bind: :1 :2 :3 kunci klaim.
SELECT s.CASEID, s.IDOBJECT, s.SURVEYOR_NAME, s.STS_SURVEY, s.PYSTATUSWORK
  FROM POOLDATA.T_SURVEYORLIST s
 WHERE s.PNCCASEID IN (:1, :2, :3)

-- name: survey_batal
-- Batal Survei: kasus survey ditutup Resolved-Rejected (CancelSurvey langkah 7) dan
-- STS_SURVEY "Batal Survey" (Work Owner 2026-10-11). Bind: :1 status, :2 status kerja, :3 CASEID.
UPDATE POOLDATA.T_SURVEYORLIST
   SET STS_SURVEY = :1, PYSTATUSWORK = :2
 WHERE CASEID = :3

-- name: survey_direksi
-- SearchCodeDireksi_SQL: surveyor ber-TRFKOMITE 1 harus sampai Direksi. Aslinya
-- `NAME LIKE '%nama%'` dirangkai {ASIS}; di sini diikat. Bind: :1 pola nama.
SELECT COUNT(*)
  FROM POOLDATA.V_D_SURVEYORS
 WHERE NAME LIKE :1
   AND TRIM(TRFKOMITE) = '1'

-- name: survey_limit
-- RDB List/GetLimitSurvey-SQL.xml:87. Bind: :1 kode bisnis polis, :2 status syariah.
SELECT A.LIMIT
  FROM POOLDATA.LIMIT_LOSSADJUSTER A, POOLDATA.BUSINESS B
 WHERE B.ID = :1
   AND A.BUSINESSID = B.BUSINESSGROUPID
   AND A.STSSYARIAH = :2

-- name: survey_komite_anggota
-- EmailKomiteSurvey_sql (STS_SURVEY = '1') — ORDER BY DEGREE. Bind: :1 TYPE_BUSINESS.
SELECT OPERATOR_ID, EMAIL
  FROM POOLDATA.EMAILKOMITE
 WHERE STS_ADJ = '1' AND STS_AKTIF = '1'
   AND TYPE_BUSINESS = :1
   AND STS_SURVEY = '1'
 ORDER BY DEGREE

-- name: survey_komite_anggota_bawah
-- Sama, dengan `and limit_bottom<=100000001` (ValidationAnalysis 16: tanpa Direksi).
-- Bind: :1 TYPE_BUSINESS.
SELECT OPERATOR_ID, EMAIL
  FROM POOLDATA.EMAILKOMITE
 WHERE STS_ADJ = '1' AND STS_AKTIF = '1'
   AND TYPE_BUSINESS = :1
   AND STS_SURVEY = '1'
   AND LIMIT_BOTTOM <= 100000001
 ORDER BY DEGREE

-- name: survey_nominasi_sisip
-- BrowseNominatedLossAdjuster-SQL.xml:36 (save). Bind: :1 CLAIMID, :2 SURVEYORID, :3 NAME, :4 LOGIN.
INSERT INTO POOLDATA.NOMINATE_LOSSADJUSTER_PNC (CLAIMID, SURVEYORID, NAME, LOGIN)
VALUES (:1, :2, :3, :4)

-- name: survey_komite_catatan
-- Isian modal Transfer Komite pada kepala kasus komite survey (docs/ddl/t_claim_objectlist_survey.sql).
UPDATE POOLDATA.TC_PNC_KOMITE
   SET SURVEY_TANGGAL = :1, SURVEY_INISIAL = :2, SURVEY_TIPE_ANALISIS = :3, SURVEY_KRONOLOGI = :4,
       SURVEY_NOMINATED = :5, SURVEY_REMARKS = :6, SURVEY_PERUSAHAAN = :7, SURVEY_KONTAK = :8,
       SURVEY_TELEPON = :9, SURVEY_EMAIL = :10
 WHERE KOMITE_ID = :11

-- name: survey_permintaan
-- Permintaan Survey (RequestSurvey) dari POOLDATA.T_REQ_SURVEY, yang terbaru.
-- Bind: :1 :2 :3 kunci klaim.
SELECT REQ_NAME, LOCATION, NOTELP, DATE_SURVEY, BRANCH, SURVEYOR, EMAIL_SURVEYOR, OBJECTNAME
  FROM POOLDATA.T_REQ_SURVEY
 WHERE CLAIMID IN (:1, :2, :3)
 ORDER BY INPUTDATE DESC NULLS LAST
 FETCH FIRST 1 ROWS ONLY
