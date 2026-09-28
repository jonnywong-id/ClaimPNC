-- ============================================================================
-- DAFTAR SYNTAX — CREATE TABLE dan ALTER TABLE, portal ASM (2026-09-26)
-- ============================================================================
--
-- Dibangkitkan dari tc_pnc_object_tree.sql. Isinya SAMA; bedanya hanya bentuk:
-- di sini ALTER ditulis polos per tabel, tanpa pemeriksaan ALL_TAB_COLS.
--
-- Akibatnya berkas ini TIDAK idempoten: dijalankan dua kali -> ORA-01430 (kolom sudah
-- ada). Untuk DIJALANKAN, pakai tc_pnc_object_tree.sql. Berkas ini untuk DIBACA dan
-- ditinjau.
--
-- Jumlah: 7 ALTER TABLE (147 kolom) · 6 CREATE TABLE · 2 CREATE INDEX
-- ============================================================================


-- ============================================================================
-- A. ALTER TABLE — kolom baru di tabel yang sudah ada
-- ============================================================================

-- A1. T_CLAIM_OBJECTLIST — 27 kolom
ALTER TABLE POOLDATA.T_CLAIM_OBJECTLIST ADD (
    OBJECTINDEX              NUMBER(5),       -- ObjectIndex
    OBJECTSURVEYLOCATION     VARCHAR2(1000),  -- ObjectSurveyLocation
    CITYID                   VARCHAR2(20),    -- CityID
    CITY                     VARCHAR2(100),   -- City
    DISTRICTID               VARCHAR2(20),    -- DistrictID
    DISTRICT                 VARCHAR2(100),   -- District
    RWID                     VARCHAR2(20),    -- RWID
    RWNOTE                   VARCHAR2(100),   -- ObjectLocationRWNote
    BRANCHCODE               VARCHAR2(10),    -- BranchCode
    OCCUPATIONID             VARCHAR2(20),    -- OccupationID
    OCCUPATIONNAME           VARCHAR2(500),   -- OccupationName
    SURVEYORTYPE             VARCHAR2(10),    -- SurveyorType
    CURRENCY                 VARCHAR2(10),    -- Currency ("IDR")
    CURRENCYAKSEP            VARCHAR2(10),    -- CurrencyAksep
    NILAIOSKLAIM             NUMBER,          -- NilaiOSKalim (salah ketik di Pega)
    NILAIOSADJUSTER          NUMBER,          -- NilaiOSAdjuster
    NILAIAKSEPALL            NUMBER,          -- NilaiAksepAll
    NILAIADJUSTERALL         NUMBER,          -- NilaiAdjusterAll
    ISKOMITEAPPROVE          NUMBER(1),       -- IsKomiteApprove
    OBJECTIDCARD             VARCHAR2(50),    -- ObjectIDCard (paspor) — DATA PRIBADI
    OBJECTHEIGHT             NUMBER,          -- ObjectHeight
    OBJECTWEIGHT             NUMBER,          -- ObjectWeight
    OBJECTJOB                VARCHAR2(200),   -- ObjectJob
    OBJECTLEFTHANDED         NUMBER(1),       -- ObjectLeftHanded
    OBJECTPARTICIPANTSTATUS  VARCHAR2(30),    -- ObjectParticipantStatus
    DIUBAH_OLEH              VARCHAR2(64),
    DIUBAH_PADA              TIMESTAMP
);

-- A2. T_CLAIM_OBJECTCOVERAGE — 24 kolom
ALTER TABLE POOLDATA.T_CLAIM_OBJECTCOVERAGE ADD (
    OBJECTNAME              VARCHAR2(500),   -- ObjectName
    LEGALLIABILITY          VARCHAR2(4000),  -- LegalLiability
    REMARKINVESTIGATION     VARCHAR2(4000),  -- RemarkInvestigation
    DIAGNOSE                VARCHAR2(4000),  -- Diagnose — DATA MEDIS
    CODEDIAGNOSE            VARCHAR2(20),    -- CodeDiagnose — DATA MEDIS
    DESCDIAGNOSE            VARCHAR2(500),   -- DescDiagnose — DATA MEDIS
    TSISUBLIMIT             NUMBER,          -- TSISublimit
    CURRENCYESTIMASI        VARCHAR2(10),    -- CurrencyEstimasi
    TOTALESTIMASIJAMINAN    NUMBER,          -- TotalEstimasiJaminan (Travel)
    ISCFS                   NUMBER(1),       -- IsCFS
    ISPLA                   NUMBER(1),       -- IsPLA
    ISANALISTRANSFER        NUMBER(1),       -- IsAnalisTransfer
    ISKOMITETRANSFER        NUMBER(1),       -- IsKomiteTransfer
    ISKOMITETRAVELTRANSFER  NUMBER(1),       -- IsKomiteTravelTransfer
    CHECKCOVERAGE           NUMBER(1),       -- CheckCoverage
    CHECKINTEREST           NUMBER(1),       -- CheckInterest
    USERBUSINESSPA          NUMBER(1),       -- UserBusinessPA
    TEMPRECEIVER            VARCHAR2(10),    -- TempReceiver (-> IDReceiver)
    INITIALNAME             VARCHAR2(128),   -- Initial (INITIAL kata tercadang Oracle)
    TANGGALCOMITEE          TIMESTAMP,       -- TanggalComitee
    DIBUAT_OLEH             VARCHAR2(128),   -- pxCreateOperator (waktunya = CREATEDATETIME)
    DIBUAT_OLEH_NAMA        VARCHAR2(128),   -- pxCreateOpName
    DIUBAH_OLEH             VARCHAR2(64),
    DIUBAH_PADA             TIMESTAMP
);

-- A3. T_CLAIM_ESTIMASI — 7 kolom
ALTER TABLE POOLDATA.T_CLAIM_ESTIMASI ADD (
    KURSVALUEPOLIS    NUMBER,         -- KursValuePolis
    ESTIMATIONDATE    TIMESTAMP,      -- EstimationDate (CFSDATE tidak menyimpan jam)
    PRINTFACECLAIM    NUMBER(1),      -- PrintFaceClaim
    TRAVELID          VARCHAR2(10),   -- TravelID
    DIBUAT_PADA       TIMESTAMP,      -- pxCreateDateTime
    DIBUAT_OLEH       VARCHAR2(128),  -- pxCreateOperator
    DIBUAT_OLEH_NAMA  VARCHAR2(128)   -- pxCreateOpName
);

-- A4. T_CLAIM_ADJUSTMENT — 60 kolom
ALTER TABLE POOLDATA.T_CLAIM_ADJUSTMENT ADD (
    ESTIMATIONVALUE          NUMBER,          -- EstimationValue
    PROPOSEVALUETERTANGGUNG  NUMBER,          -- ProposeValueTertanggung
    PROPOSEADJUSTMENTFINAL   NUMBER,          -- ProposeAdjustmentFinal (bisa negatif)
    CONVERTADJUSTMENTVALUE   NUMBER,          -- ConvertAdjustmentValue (IDR; pembanding ambang komite)
    ACCEPTANCEVALUELOD       NUMBER,          -- AcceptanceValueLOD
    SALVAGEVALUE             NUMBER,          -- SalvageValue (sudah ada: NILAI_SALVAGE_A)
    ADJUSTERFEEVALUE         NUMBER,          -- AdjusterFeeValue
    TOTALADJUSTER            NUMBER,          -- TotalAdjuster
    TOTALIDRESTIMASI         NUMBER,          -- TotalIDREstimasi
    ASMFULL                  NUMBER(1),       -- ASMFull
    INTERIMPAYMENT           NUMBER(1),       -- InterimPayment
    VATTYPE                  VARCHAR2(10),    -- VATType
    EXTENTOFLOSS             VARCHAR2(4000),  -- ExtentOfLoss
    LEGALLIABILITY           VARCHAR2(4000),  -- LegalLiability
    SALVAGENOTE              VARCHAR2(1000),  -- Salvage (teks, mis. "Tidak ekonomis")
    REMARKS                  VARCHAR2(4000),  -- Remarks
    REMARKACCEPTED           VARCHAR2(1000),  -- RemarkAccepted
    INITIALNAME              VARCHAR2(128),   -- Initial
    KOMITETYPE               VARCHAR2(10),    -- KomiteType
    KOMITEACCEPTED           VARCHAR2(128),   -- KomiteAccepted (operator)
    ISKOMITETRANSFER         NUMBER(1),       -- IsKomiteTransfer
    ISKOMITESETUJU           NUMBER(1),       -- IsKomiteSetuju
    TANGGALCOMITEE           TIMESTAMP,       -- TanggalComitee (sudah ada ANALYST_TFKOMITEDATE, turunannya)
    ACCEPTANCEANALYSTSTATUS  VARCHAR2(10),    -- AcceptanceAnalystStatus
    ACCEPTEDDATETIME         TIMESTAMP,       -- AcceptedDateTime (TGLAKSEPTASI tanpa jam)
    TIPEAKSEPTASI            VARCHAR2(10),    -- TipeAkseptasi
    CONFIRMATIONANSWER       VARCHAR2(10),    -- ConfirmationAnswer
    ISANALISATORTRANSFER     NUMBER(1),       -- IsAnalisatorTransfer
    RECEIVEDATEANALIST       TIMESTAMP,       -- ReceiveDateAnalist
    ADJUSTERTYPE             VARCHAR2(10),    -- AdjusterType
    JOINPLACEMENT            NUMBER(1),       -- JoinPlacement
    STATUSCLAIM              VARCHAR2(30),    -- StatusClaim (teks, mis. "ACCEPTATION")
    POSISIPROGRESSID         VARCHAR2(10),    -- PosisiProgressID
    USERBUSINESSPA           NUMBER(1),       -- UserBusinessPA
    CONF_SCORE               VARCHAR2(20),    -- CONF_SCORE (hasil AI; arti belum dipastikan)
    FILEDGT                  NUMBER(1),       -- FileDGT
    ISDLA                    NUMBER(1),       -- IsDLA
    ISPRINTLOD               NUMBER(1),       -- IsPrintLOD
    TRANSFERLODDATE          TIMESTAMP,       -- TransferLODDate
    LOADDATE                 TIMESTAMP,       -- LOADDate
    STSPENERIMAKLAIM         VARCHAR2(10),    -- StsPenerimaKlaim
    PEMBAYARANTO             VARCHAR2(10),    -- PembayaranTo
    PENERIMAPEMBAYARAN       VARCHAR2(1000),  -- PenerimaPembayaran
    TANGGALBOLEHBAYAR        TIMESTAMP,       -- TanggalBolehBayar
    TGLKOMITEPENERIMA        TIMESTAMP,       -- TglKomitePenerima
    TGLKASIRPENERIMA         DATE,            -- TglKasirPenerima
    TRANSFERCASHIERSTATUS    VARCHAR2(10),    -- TransferCashierStatus
    CLAIMPAIDDATE            TIMESTAMP,       -- ClaimPaidDate
    STATUSBAYAR              VARCHAR2(10),    -- StatusBayar (dibaca prosedur lama, tidak ditulis)
    ACCEPTANCEBANK           VARCHAR2(100),   -- AcceptanceDocument.AcceptanceBank
    ACCOUNTNO                VARCHAR2(50),    -- AcceptanceDocument.AccountNo (teks: nol di depan bermakna)
    PAYABLETO                VARCHAR2(500),   -- AcceptanceDocument.PayableTo
    ACCEPTANCEDATE           TIMESTAMP,       -- AcceptanceDocument.AcceptanceDate
    DIBUAT_PADA              TIMESTAMP,       -- pxCreateDateTime
    DIBUAT_OLEH              VARCHAR2(128),   -- pxCreateOperator
    DIBUAT_OLEH_NAMA         VARCHAR2(128),   -- pxCreateOpName
    DIUBAH_OLEH              VARCHAR2(64),
    DIUBAH_PADA              TIMESTAMP,
    DIHAPUS_OLEH             VARCHAR2(64),
    DIHAPUS_PADA             TIMESTAMP
);

-- A5. T_CLAIM_SPREADING — 12 kolom
ALTER TABLE POOLDATA.T_CLAIM_SPREADING ADD (
    TREATYGROUP          VARCHAR2(20),  -- TreatyGroup
    TREATYYEAR           NUMBER(4),     -- TreatyYear
    TREATYLIMIT          NUMBER,        -- TreatyLimit
    TREATYLIMITPERCEN    NUMBER,        -- TreatyLimitPercen
    TSISPREADED          NUMBER,        -- TSISpreaded
    PREMIUMSPREADED      NUMBER,        -- PremiumSpreaded
    PREMINET             NUMBER,        -- PremiNet
    PERCENTTABARUFUND    NUMBER,        -- PercentTabaruFund (syariah)
    ISFILLTSISPREADED    NUMBER(1),     -- IsFillTSISpreaded
    ISFILLPREMISPREADED  NUMBER(1),     -- IsFillPremiSpreaded
    FLAGOLDDATA          NUMBER(1),     -- FlagOldData
    FLAGEDITDATA         NUMBER(1)      -- FlagEditData
);

-- A6. T_CLAIM_DETAIL_SPREDING — 14 kolom
ALTER TABLE POOLDATA.T_CLAIM_DETAIL_SPREDING ADD (
    OBJECTID             VARCHAR2(30),  -- kunci: objek
    OBJECTCOVERAGEID     VARCHAR2(30),  -- kunci: coverage klaim
    ADJUSTMENTID         VARCHAR2(10),  -- kunci: adjustment
    URUTAN               NUMBER(10),    -- kunci: urutan di SpreadingList
    TREATYLIMIT          NUMBER,        -- TreatyLimit
    TREATYLIMITPERCEN    NUMBER,        -- TreatyLimitPercen
    PREMIUMSPREADED      NUMBER,        -- PremiumSpreaded
    PREMINET             NUMBER,        -- PremiNet
    PERCENTTABARUFUND    NUMBER,        -- PercentTabaruFund
    ISFILLTSISPREADED    NUMBER(1),     -- IsFillTSISpreaded
    ISFILLPREMISPREADED  NUMBER(1),     -- IsFillPremiSpreaded
    FLAGOLDDATA          NUMBER(1),     -- FlagOldData
    FLAGEDITDATA         NUMBER(1),     -- FlagEditData
    DIHAPUS_PADA         TIMESTAMP      -- FlagDelete / soft delete (D-66)
);

-- A7. T_PLALIST — 3 kolom
ALTER TABLE POOLDATA.T_PLALIST ADD (
    INTEREST           VARCHAR2(1000),  -- Interest
    ESTIMATIONRESERVE  NUMBER,          -- EstimasiList[].EstimastionReserve (salah ketik di Pega)
    PLACOMMITDATE      TIMESTAMP        -- pxCommitDateTime
);

CREATE INDEX POOLDATA.IX_T_CLAIM_DETSPR_ADJ
    ON POOLDATA.T_CLAIM_DETAIL_SPREDING (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID);


-- ============================================================================
-- B. CREATE TABLE — tabel baru TC_PNC_*
-- ============================================================================

-- B1. TC_PNC_OBJECTITEM
CREATE TABLE POOLDATA.TC_PNC_OBJECTITEM (
    CLAIMID                     VARCHAR2(100)   NOT NULL,
    OBJECTID                    VARCHAR2(30)    NOT NULL,
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,
    OBJECTITEMID                VARCHAR2(10)    NOT NULL,   -- urutan + 1 — sama dengan T_CLAIM_ESTIMASI.OBJECTITEMID
    OBJECTITEMCODE              VARCHAR2(20),               -- ObjectItemID (Travel: kode manfaat, mis. 15002)
    OBJECTITEMTYPEID            VARCHAR2(20),               -- ObjectItemTypeID (Fire)
    OBJECTITEMNAME              VARCHAR2(500),              -- ObjectItemName
    PROPERTYITEMGROUP           VARCHAR2(50),               -- PropertyItemGroup
    DESKRIPSIOBJECT             VARCHAR2(4000),             -- DeskripsiObject
    LOCATIONID                  VARCHAR2(100),              -- LocationID
    TRAVELID                    VARCHAR2(10),               -- TravelID
    TSIPEROBJECT                NUMBER,                     -- TSIPerObject
    TSIPERCOVERAGE              NUMBER,                     -- TSIperCoverage
    SUMESTIMATION               NUMBER,                     -- SumEstimation
    GROSSVALUE                  NUMBER,                     -- GrossValue
    DEDUCTIBLE                  NUMBER,                     -- Deductible
    ISCFS                       NUMBER(1),                  -- IsCFS
    DB_PLACECARRIER             VARCHAR2(1000),             -- DelayedBaggage.PlaceCarrier (gaya nama T_CLAIM_OBJECTITEMLIST)
    DIBUAT_PADA                 TIMESTAMP,                  -- pxCreateDateTime
    DIBUAT_OLEH                 VARCHAR2(128),              -- pxCreateOperator
    DIBUAT_OLEH_NAMA            VARCHAR2(128),              -- pxCreateOpName
    DIUBAH_OLEH                 VARCHAR2(64),
    DIUBAH_PADA                 TIMESTAMP,
    DIHAPUS_OLEH                VARCHAR2(64),
    DIHAPUS_PADA                TIMESTAMP,                  -- soft delete (D-66)
    CONSTRAINT PK_TC_PNC_OBJECTITEM PRIMARY KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID)
);

-- B2. TC_PNC_BAGGAGELOST
CREATE TABLE POOLDATA.TC_PNC_BAGGAGELOST (
    CLAIMID                     VARCHAR2(100)   NOT NULL,
    OBJECTID                    VARCHAR2(30)    NOT NULL,
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,
    OBJECTITEMID                VARCHAR2(10)    NOT NULL,
    URUTAN                      NUMBER(5)       NOT NULL,
    ITEMDESCRIPTION             VARCHAR2(2000),             -- ItemDescription
    PURCHASEDATE                DATE,                       -- PurchaseDate
    PURCHASEPLACE               VARCHAR2(1000),             -- PurchasePlace
    NETTPRICE                   NUMBER,                     -- NettPrice
    CURRENCY                    VARCHAR2(10),               -- Currency (kode)
    ISCFS                       NUMBER(1),                  -- IsCFS
    REMARKS                     VARCHAR2(1000),             -- Remarks
    DIBUAT_PADA                 TIMESTAMP,
    DIBUAT_OLEH                 VARCHAR2(128),
    DIBUAT_OLEH_NAMA            VARCHAR2(128),
    DIHAPUS_OLEH                VARCHAR2(64),
    DIHAPUS_PADA                TIMESTAMP,
    CONSTRAINT PK_TC_PNC_BAGGAGELOST PRIMARY KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, URUTAN),
    CONSTRAINT FK_TC_PNC_BAGGAGELOST FOREIGN KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID)
        REFERENCES POOLDATA.TC_PNC_OBJECTITEM (CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID)
);

-- B3. TC_PNC_COVERAGEDETAIL
CREATE TABLE POOLDATA.TC_PNC_COVERAGEDETAIL (
    CLAIMID                     VARCHAR2(100)   NOT NULL,
    OBJECTID                    VARCHAR2(30)    NOT NULL,
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,
    URUTAN                      NUMBER(5)       NOT NULL,
    COVERAGEDETAILID            VARCHAR2(20),               -- CoverageDetailID
    COVERAGEDETAILFORM          VARCHAR2(10),               -- CoverageDetailForm
    COVERAGEDETAILNAME          VARCHAR2(500),              -- CoverageDetailName
    COVERAGEDETAILLIMIT         NUMBER,                     -- CoverageDetailLimit
    DEDUCTIBLE                  NUMBER,                     -- Deductible
    CONSTRAINT PK_TC_PNC_COVERAGEDETAIL PRIMARY KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, URUTAN)
);

-- B4. TC_PNC_CFS
CREATE TABLE POOLDATA.TC_PNC_CFS (
    CLAIMID                     VARCHAR2(100)   NOT NULL,
    OBJECTID                    VARCHAR2(30)    NOT NULL,
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,
    REVISI                      NUMBER(5)       NOT NULL,   -- Revisi
    CFSDATE                     TIMESTAMP,                  -- Date
    FILENAME                    VARCHAR2(500),              -- FileName
    CONSTRAINT PK_TC_PNC_CFS PRIMARY KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI)
);

-- B5. TC_PNC_CFS_ESTIMASI
CREATE TABLE POOLDATA.TC_PNC_CFS_ESTIMASI (
    CLAIMID                     VARCHAR2(100)   NOT NULL,
    OBJECTID                    VARCHAR2(30)    NOT NULL,
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,
    REVISI                      NUMBER(5)       NOT NULL,
    URUTAN                      NUMBER(5)       NOT NULL,
    CURRENCY                    VARCHAR2(10),               -- Currency — TEKS ("IDR", "Rp."), beda dengan kode 10026 di estimasi item
    ESTIMATIONDATE              TIMESTAMP,                  -- EstimationDate
    ESTIMATIONVALUE             NUMBER,                     -- EstimationValue
    CONSTRAINT PK_TC_PNC_CFS_ESTIMASI PRIMARY KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI, URUTAN),
    CONSTRAINT FK_TC_PNC_CFS_ESTIMASI FOREIGN KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI)
        REFERENCES POOLDATA.TC_PNC_CFS (CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI)
);

-- B6. TC_PNC_ADJUSTMENT_DOC
CREATE TABLE POOLDATA.TC_PNC_ADJUSTMENT_DOC (
    CLAIMID                     VARCHAR2(100)   NOT NULL,
    OBJECTID                    VARCHAR2(30)    NOT NULL,
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,
    ADJUSTMENTID                VARCHAR2(10)    NOT NULL,
    URUTAN                      NUMBER(5)       NOT NULL,
    PZINSKEY                    VARCHAR2(50),               -- pzInsKey (kunci lampiran)
    PYCATEGORY                  VARCHAR2(50),               -- pyCategory (Adjustment · AdjustmentLOD)
    GCNMCATEGORY                VARCHAR2(50),               -- GCNMCategory
    GCNMTYPE                    VARCHAR2(50),               -- GCNMType
    KBRUTYPE                    VARCHAR2(20),               -- KBRUType
    DOCUMENTCATEGORYID          VARCHAR2(20),               -- pxSessionID (berisi kode kategori, bukan sesi)
    DOCUMENTDESCRIPTION         VARCHAR2(500),              -- pxCustomerID (berisi uraian dokumen)
    UPLOADEDFILE                VARCHAR2(500),              -- pyUploadedFile
    UPLOADEDFILENAME            VARCHAR2(500),              -- UploadedFileName
    DIBUAT_PADA                 TIMESTAMP,
    DIBUAT_OLEH                 VARCHAR2(128),
    DIHAPUS_OLEH                VARCHAR2(64),
    DIHAPUS_PADA                TIMESTAMP,
    CONSTRAINT PK_TC_PNC_ADJUSTMENT_DOC PRIMARY KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID, URUTAN)
);

CREATE INDEX POOLDATA.IX_TC_PNC_ADJDOC_INSKEY ON POOLDATA.TC_PNC_ADJUSTMENT_DOC (PZINSKEY);
