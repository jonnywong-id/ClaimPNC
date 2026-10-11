-- ============================================================================
-- Kolom survey per objek — tab Survey tahap Choose Surveyor (Oracle 19c)
-- ============================================================================
--
-- STATUS: DIAJUKAN 2026-10-11 atas keputusan Work Owner ("Tambah kolom di T_CLAIM_OBJECTLIST").
-- Pelaksanaan menempuh `D-63`: akun aplikasi tidak memiliki hak DDL. Portal ASM lebih dulu.
--
-- DUA pernyataan ALTER, berurutan. Tidak ada baris kosong di dalam satu pernyataan
-- (SQL*Plus `SET SQLBLANKLINES OFF` memotong pernyataan pada baris kosong).
--
-- ============================================================================
-- APA YANG DIGANTIKANNYA
-- ============================================================================
--
-- Grid Tambah Survey (`Section/InputSurvey-sect.xml`) berada di atas `.ClaimData.ObjectList`.
-- Di Pega isiannya tersimpan di objek kerja (BLOB). Tiga kolomnya sudah ada di
-- T_CLAIM_OBJECTLIST dan dipakai apa adanya:
--
--   .ObjectSurveyLocation  -> OBJECTSURVEYLOCATION   (sudah ada)
--   .SurveyorType          -> SURVEYORTYPE           (sudah ada)
--   .BranchCode            -> BRANCHCODE             (sudah ada)
--   .IsKomiteApprove       -> ISKOMITEAPPROVE        (sudah ada)
--
-- Sisanya ditambahkan pernyataan (1):
--
--   .pySelected                 -> PILIHSURVEY               '1' / '0'
--   .ObjectStatus               -> OBJECTSTATUS              1 Menunggu Persetujuan · 2 Ditolak Komite ·
--                                                            3 Sedang Proses · 4 Batal Survey · 5 Selesai
--   .ObjectSurveyor             -> OBJECTSURVEYOR            nama surveyor (V_D_SURVEYORS.NAME)
--   .ObjectSurveyorLogin        -> OBJECTSURVEYORLOGIN       V_D_SURVEYORS.LOGIN_APLIKASI
--   .SurveyorAddrress           -> SURVEYORADDRESS           V_D_SURVEYORS.ADDRESS
--   .ObjectSurveyorOthers       -> OBJECTSURVEYOROTHERS      email surveyor internal
--   .BranchName                 -> SURVEYORBRANCHNAME        cabang surveyor internal
--   .ObjectSurveyorMarine       -> OBJECTSURVEYORMARINE      Surveyor Marine (Marine Hull)
--   .ObjectSurveyorMarineLogin  -> OBJECTSURVEYORMARINELOGIN
--   .ObjectSurveyID             -> OBJECTSURVEYID            nomor survey (SRVN.YY.n)
--   .ObjectSurveyIDMarine       -> OBJECTSURVEYIDMARINE
--
-- Pernyataan (2) menambah isian modal Transfer Komite (`Section/ClaimComitee-sect.xml`) pada
-- kepala kasus komite survey POOLDATA.TC_PNC_KOMITE (TRANSFERTYPE '1').
--
-- Kolom baru semuanya NULLABLE, sehingga Pega yang masih menulis tabel yang sama tidak
-- terganggu (`P-4`).

ALTER TABLE POOLDATA.T_CLAIM_OBJECTLIST ADD (
  PILIHSURVEY               VARCHAR2(1),
  OBJECTSTATUS              VARCHAR2(5),
  OBJECTSURVEYOR            VARCHAR2(500),
  OBJECTSURVEYORLOGIN       VARCHAR2(200),
  SURVEYORADDRESS           VARCHAR2(2000),
  OBJECTSURVEYOROTHERS      VARCHAR2(500),
  SURVEYORBRANCHNAME        VARCHAR2(500),
  OBJECTSURVEYORMARINE      VARCHAR2(500),
  OBJECTSURVEYORMARINELOGIN VARCHAR2(200),
  OBJECTSURVEYID            VARCHAR2(50),
  OBJECTSURVEYIDMARINE      VARCHAR2(50)
);

ALTER TABLE POOLDATA.TC_PNC_KOMITE ADD (
  SURVEY_TANGGAL       TIMESTAMP,
  SURVEY_INISIAL       VARCHAR2(100),
  SURVEY_TIPE_ANALISIS VARCHAR2(5),
  SURVEY_KRONOLOGI     VARCHAR2(4000),
  SURVEY_NOMINATED     VARCHAR2(4000),
  SURVEY_REMARKS       VARCHAR2(4000),
  SURVEY_PERUSAHAAN    VARCHAR2(500),
  SURVEY_KONTAK        VARCHAR2(500),
  SURVEY_TELEPON       VARCHAR2(100),
  SURVEY_EMAIL         VARCHAR2(500)
);
