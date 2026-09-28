-- ============================================================================
-- Pohon data klaim per OBJEK — kolom baru di tabel yang ada, dan tabel TC_PNC_*
-- ============================================================================
--
-- STATUS: RANCANGAN v3 (2026-09-26). Belum dijalankan dan belum menjadi migrasi.
-- Menempuh D-63 (permintaan tertulis -> persetujuan Work Owner -> DBA -> uji
-- Pega+Go bersamaan). Tahap pertama: PORTAL ASM SAJA (Work Owner, 2026-09-26);
-- portal lain menyusul, satu basis data per portal (ADR-0030).
--
-- ATURAN DARI WORK OWNER (2026-09-26)
--   * Tabel yang sudah dipakai (T_CLAIM_OBJECTLIST, T_CLAIM_OBJECTCOVERAGE, dst.)
--     TETAP dipakai. Kolom yang belum ada DITAMBAHKAN dengan ALTER TABLE.
--   * TC_PNC_* hanya untuk data yang BELUM PUNYA TABEL sama sekali.
--
-- SUMBER
--   * 11 contoh JSON_KLAIM di folder JSON/ (termasuk CONTOH DATA ANEKA COINS_1).
--   * Katalog Oracle portal ASM (ALL_TAB_COLUMNS, ALL_CONSTRAINTS), dibaca 2026-09-26.
--     Portal lain belum diperiksa — karena itu BAGIAN 1 idempoten: kolom yang ternyata
--     sudah ada di portal tertentu dilewati, bukan menggagalkan skrip.
--
-- ============================================================================
-- PETA: DARI JSON KE TABEL
-- ============================================================================
--
--   ObjectList[] ................................ T_CLAIM_OBJECTLIST        + ALTER
--   └ ObjectCoverageList[] ...................... T_CLAIM_OBJECTCOVERAGE    + ALTER
--     ├ ObjectItemList[] ........................ (belum ada)               -> TC_PNC_OBJECTITEM
--     │ ├ EstimationList[] ...................... T_CLAIM_ESTIMASI          + ALTER
--     │ └ BaggageLostList[] ..................... (belum ada)               -> TC_PNC_BAGGAGELOST
--     ├ ObjectCoverageDetailList[] .............. (belum ada)               -> TC_PNC_COVERAGEDETAIL
--     ├ CFSList[] ............................... (belum ada)               -> TC_PNC_CFS
--     │ └ EstimasiList[] ........................ (belum ada)               -> TC_PNC_CFS_ESTIMASI
--     ├ PLAList[] ............................... T_PLALIST                 + ALTER
--     ├ SpreadingList[] ......................... T_CLAIM_SPREADING         + ALTER
--     └ AdjustmentList[] ........................ T_CLAIM_ADJUSTMENT        + ALTER (termasuk AcceptanceDocument)
--       ├ ComiteeClaim[] ........................ T_CLAIM_KOMITE_LIST       (lengkap, tidak diubah)
--       ├ DocumentList[] ........................ (belum ada)               -> TC_PNC_ADJUSTMENT_DOC
--       ├ TempObjectItemList[] .................. T_CLAIM_OBJECTITEMLIST    (lengkap, tidak diubah)
--       └ SpreadingList[] ....................... T_CLAIM_DETAIL_SPREDING   + ALTER (termasuk kolom kunci)
--
-- TIDAK DISENTUH — domain POLIS (D-03, D-04): ObjectList[].CoverageList[] beserta
--   turunannya, dan PolicyData.CoinsList. Snapshot-nya di JSON_POLIS.POLICYDATA; tabel
--   GISFW-nya per NOPOLIS/PRODKE (T_COINSLIST, T_DEDUCTIBLELIST, T_OUTGOLIST).
--
-- ============================================================================
-- ATURAN RANCANGAN
-- ============================================================================
--
-- AMAN TERHADAP PEGA (P-4). Seluruh kolom tambahan NULLABLE tanpa DEFAULT: hanya
--   metadata, tidak ada baris yang ditulis ulang, dan INSERT Pega yang menyebut kolom
--   satu per satu tetap berjalan. Tidak ada kolom lama yang diubah atau dihapus.
--
-- KUNCI. Tabel lama tidak punya primary key yang dapat dirujuk (T_CLAIM_OBJECTLIST,
--   T_CLAIM_ADJUSTMENT, T_CLAIM_ESTIMASI tanpa PK; PK T_CLAIM_OBJECTCOVERAGE menyertakan
--   CREATEDATETIME). Tabel TC_PNC_* karena itu memakai kunci alami yang sama (CLAIMID,
--   OBJECTID, OBJECTCOVERAGEID, ...) tanpa FOREIGN KEY ke tabel lama.
--
-- NAMA KOLOM. Gaya tabel T_CLAIM_*: nama properti JSON huruf besar tanpa garis bawah
--   (INPATIENTDAYMAX, BLASTEMAILDATE). Kolom jejak mengikuti T_CLAIM_PNC (0007):
--   DIBUAT_*, DIUBAH_*, DIHAPUS_*.
--
-- TIPE. Uang dan persentase NUMBER tanpa skala — desimal eksak presisi penuh (D-51,
--   I-12). Waktu Pega "...GMT" -> TIMESTAMP UTC (DB-8). Tanggal 8 digit -> DATE.
--   Penanda "0"/"1"/"true"/"false" -> NUMBER(1). 19700101 di JSON = KOSONG -> NULL.
-- ============================================================================


-- ============================================================================
-- BAGIAN 1 — ALTER TABLE: kolom yang belum ada di tabel yang sudah dipakai
-- ============================================================================
--
-- Idempoten: setiap kolom diperiksa di ALL_TAB_COLS lebih dulu, dan ditambahkan hanya
-- bila belum ada. Aman dijalankan ulang, dan aman pada portal yang skemanya sedikit
-- berbeda. Pola sama dengan migrasi 0008.

DECLARE
    TYPE t_kolom IS RECORD (tabel VARCHAR2(30), nama VARCHAR2(30), tipe VARCHAR2(40));
    TYPE t_daftar IS TABLE OF t_kolom;

    daftar t_daftar := t_daftar(
        -- --------------------------------------------------------------------
        -- T_CLAIM_OBJECTLIST  <- ObjectList[]
        -- Sudah ada: OBJECTNAME, kendaraan, kredit/SLIK, NOKTP, OBJECTGENDER,
        -- DATEOFBIRTH, LOKASI, LOCATIONID, CONTRACTNO, URUTAN, DIHAPUS_PADA
        -- --------------------------------------------------------------------
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTINDEX',             'NUMBER(5)'),        -- ObjectIndex
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTSURVEYLOCATION',    'VARCHAR2(1000)'),   -- ObjectSurveyLocation
        t_kolom('T_CLAIM_OBJECTLIST', 'CITYID',                  'VARCHAR2(20)'),     -- CityID
        t_kolom('T_CLAIM_OBJECTLIST', 'CITY',                    'VARCHAR2(100)'),    -- City
        t_kolom('T_CLAIM_OBJECTLIST', 'DISTRICTID',              'VARCHAR2(20)'),     -- DistrictID
        t_kolom('T_CLAIM_OBJECTLIST', 'DISTRICT',                'VARCHAR2(100)'),    -- District
        t_kolom('T_CLAIM_OBJECTLIST', 'RWID',                    'VARCHAR2(20)'),     -- RWID
        t_kolom('T_CLAIM_OBJECTLIST', 'RWNOTE',                  'VARCHAR2(100)'),    -- ObjectLocationRWNote
        t_kolom('T_CLAIM_OBJECTLIST', 'BRANCHCODE',              'VARCHAR2(10)'),     -- BranchCode
        t_kolom('T_CLAIM_OBJECTLIST', 'OCCUPATIONID',            'VARCHAR2(20)'),     -- OccupationID
        t_kolom('T_CLAIM_OBJECTLIST', 'OCCUPATIONNAME',          'VARCHAR2(500)'),    -- OccupationName
        t_kolom('T_CLAIM_OBJECTLIST', 'SURVEYORTYPE',            'VARCHAR2(10)'),     -- SurveyorType
        t_kolom('T_CLAIM_OBJECTLIST', 'CURRENCY',                'VARCHAR2(10)'),     -- Currency ("IDR")
        t_kolom('T_CLAIM_OBJECTLIST', 'CURRENCYAKSEP',           'VARCHAR2(10)'),     -- CurrencyAksep
        t_kolom('T_CLAIM_OBJECTLIST', 'NILAIOSKLAIM',            'NUMBER'),           -- NilaiOSKalim (salah ketik di Pega)
        t_kolom('T_CLAIM_OBJECTLIST', 'NILAIOSADJUSTER',         'NUMBER'),           -- NilaiOSAdjuster
        t_kolom('T_CLAIM_OBJECTLIST', 'NILAIAKSEPALL',           'NUMBER'),           -- NilaiAksepAll
        t_kolom('T_CLAIM_OBJECTLIST', 'NILAIADJUSTERALL',        'NUMBER'),           -- NilaiAdjusterAll
        t_kolom('T_CLAIM_OBJECTLIST', 'ISKOMITEAPPROVE',         'NUMBER(1)'),        -- IsKomiteApprove
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTIDCARD',            'VARCHAR2(50)'),     -- ObjectIDCard (paspor) — DATA PRIBADI
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTHEIGHT',            'NUMBER'),           -- ObjectHeight
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTWEIGHT',            'NUMBER'),           -- ObjectWeight
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTJOB',               'VARCHAR2(200)'),    -- ObjectJob
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTLEFTHANDED',        'NUMBER(1)'),        -- ObjectLeftHanded
        t_kolom('T_CLAIM_OBJECTLIST', 'OBJECTPARTICIPANTSTATUS', 'VARCHAR2(30)'),     -- ObjectParticipantStatus
        t_kolom('T_CLAIM_OBJECTLIST', 'DIUBAH_OLEH',             'VARCHAR2(64)'),
        t_kolom('T_CLAIM_OBJECTLIST', 'DIUBAH_PADA',             'TIMESTAMP'),

        -- --------------------------------------------------------------------
        -- T_CLAIM_OBJECTCOVERAGE  <- ObjectCoverageList[]
        -- Sudah ada: CAUSEOFLOSS*, COVERAGEID, COVERAGENAME, CREATEDATETIME, SUMTSI,
        -- CURICUMOFLOSS, EXTENTOFLOSS, INPATIENTDAY*, REMARKS, BLASTEMAILDATE, CURRENCY,
        -- LOCATIONID, CONTRACTNO, URUTAN_OBJEK, URUTAN, DIHAPUS_PADA
        -- --------------------------------------------------------------------
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'OBJECTNAME',             'VARCHAR2(500)'),   -- ObjectName
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'LEGALLIABILITY',         'VARCHAR2(4000)'),  -- LegalLiability
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'REMARKINVESTIGATION',    'VARCHAR2(4000)'),  -- RemarkInvestigation
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DIAGNOSE',               'VARCHAR2(4000)'),  -- Diagnose — DATA MEDIS
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'CODEDIAGNOSE',           'VARCHAR2(20)'),    -- CodeDiagnose — DATA MEDIS
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DESCDIAGNOSE',           'VARCHAR2(500)'),   -- DescDiagnose — DATA MEDIS
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'TSISUBLIMIT',            'NUMBER'),          -- TSISublimit
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'CURRENCYESTIMASI',       'VARCHAR2(10)'),    -- CurrencyEstimasi
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'TOTALESTIMASIJAMINAN',   'NUMBER'),          -- TotalEstimasiJaminan (Travel)
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'ISCFS',                  'NUMBER(1)'),       -- IsCFS
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'ISPLA',                  'NUMBER(1)'),       -- IsPLA
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'ISANALISTRANSFER',       'NUMBER(1)'),       -- IsAnalisTransfer
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'ISKOMITETRANSFER',       'NUMBER(1)'),       -- IsKomiteTransfer
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'ISKOMITETRAVELTRANSFER', 'NUMBER(1)'),       -- IsKomiteTravelTransfer
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'CHECKCOVERAGE',          'NUMBER(1)'),       -- CheckCoverage
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'CHECKINTEREST',          'NUMBER(1)'),       -- CheckInterest
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'USERBUSINESSPA',         'NUMBER(1)'),       -- UserBusinessPA
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'TEMPRECEIVER',           'VARCHAR2(10)'),    -- TempReceiver (-> IDReceiver)
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'INITIALNAME',            'VARCHAR2(128)'),   -- Initial (INITIAL kata tercadang Oracle)
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'TANGGALCOMITEE',         'TIMESTAMP'),       -- TanggalComitee
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DIBUAT_OLEH',            'VARCHAR2(128)'),   -- pxCreateOperator (waktunya = CREATEDATETIME)
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DIBUAT_OLEH_NAMA',       'VARCHAR2(128)'),   -- pxCreateOpName
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DIUBAH_OLEH',            'VARCHAR2(64)'),
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DIUBAH_PADA',            'TIMESTAMP'),

        -- --------------------------------------------------------------------
        -- T_CLAIM_ESTIMASI  <- ObjectItemList[].EstimationList[]
        -- Sudah ada: ESTIMATIONTYPE, ESTIMATIONVALUE, KURSID, KURSVALUE, CONVERTVALUE,
        -- CFSDATE (DATE), INSERTDATE, LOCATIONID, CONTRACTNO
        -- --------------------------------------------------------------------
        t_kolom('T_CLAIM_ESTIMASI', 'KURSVALUEPOLIS',   'NUMBER'),         -- KursValuePolis
        t_kolom('T_CLAIM_ESTIMASI', 'ESTIMATIONDATE',   'TIMESTAMP'),      -- EstimationDate (CFSDATE tidak menyimpan jam)
        t_kolom('T_CLAIM_ESTIMASI', 'PRINTFACECLAIM',   'NUMBER(1)'),      -- PrintFaceClaim
        t_kolom('T_CLAIM_ESTIMASI', 'TRAVELID',         'VARCHAR2(10)'),   -- TravelID
        t_kolom('T_CLAIM_ESTIMASI', 'DIBUAT_PADA',      'TIMESTAMP'),      -- pxCreateDateTime
        t_kolom('T_CLAIM_ESTIMASI', 'DIBUAT_OLEH',      'VARCHAR2(128)'),  -- pxCreateOperator
        t_kolom('T_CLAIM_ESTIMASI', 'DIBUAT_OLEH_NAMA', 'VARCHAR2(128)'),  -- pxCreateOpName

        -- --------------------------------------------------------------------
        -- T_CLAIM_ADJUSTMENT  <- ObjectCoverageList[].AdjustmentList[] + AcceptanceDocument
        -- Sudah ada: NOAKSEPTASI, TGLAKSEPTASI, STATUSAKSEPTASI, PAYMENTTYPE, NILAIAKSEPTASI,
        -- GROSSVALUE, TOTAL_CLAIM (=ProposeAdjustmentValue), ASM_SHARE_VALUE (=AdjustmentValue),
        -- PROPOSE_VALUE, INDIVIDUAL_RISK_*, LOC, ASM_SHARE, EXGRATIA, CASEIDKOMITE,
        -- PRINTLOD_DATE, RECEIVEDATELOD, TRANSFER_CASHIER_DATE, CLAIM_STATUS_PAID,
        -- TGLBAYAR, IDCHASIER, NOTES, ALASANTOLAK, CIRCUMCAUSEOFLOSS, PDFTYPE, ...
        -- --------------------------------------------------------------------
        -- nilai
        t_kolom('T_CLAIM_ADJUSTMENT', 'ESTIMATIONVALUE',         'NUMBER'),          -- EstimationValue
        t_kolom('T_CLAIM_ADJUSTMENT', 'PROPOSEVALUETERTANGGUNG', 'NUMBER'),          -- ProposeValueTertanggung
        t_kolom('T_CLAIM_ADJUSTMENT', 'PROPOSEADJUSTMENTFINAL',  'NUMBER'),          -- ProposeAdjustmentFinal (bisa negatif)
        t_kolom('T_CLAIM_ADJUSTMENT', 'CONVERTADJUSTMENTVALUE',  'NUMBER'),          -- ConvertAdjustmentValue (IDR; pembanding ambang komite)
        t_kolom('T_CLAIM_ADJUSTMENT', 'ACCEPTANCEVALUELOD',      'NUMBER'),          -- AcceptanceValueLOD
        t_kolom('T_CLAIM_ADJUSTMENT', 'SALVAGEVALUE',            'NUMBER'),          -- SalvageValue (sudah ada: NILAI_SALVAGE_A)
        t_kolom('T_CLAIM_ADJUSTMENT', 'ADJUSTERFEEVALUE',        'NUMBER'),          -- AdjusterFeeValue
        t_kolom('T_CLAIM_ADJUSTMENT', 'TOTALADJUSTER',           'NUMBER'),          -- TotalAdjuster
        t_kolom('T_CLAIM_ADJUSTMENT', 'TOTALIDRESTIMASI',        'NUMBER'),          -- TotalIDREstimasi
        t_kolom('T_CLAIM_ADJUSTMENT', 'ASMFULL',                 'NUMBER(1)'),       -- ASMFull
        t_kolom('T_CLAIM_ADJUSTMENT', 'INTERIMPAYMENT',          'NUMBER(1)'),       -- InterimPayment
        t_kolom('T_CLAIM_ADJUSTMENT', 'VATTYPE',                 'VARCHAR2(10)'),    -- VATType
        -- narasi
        t_kolom('T_CLAIM_ADJUSTMENT', 'EXTENTOFLOSS',            'VARCHAR2(4000)'),  -- ExtentOfLoss
        t_kolom('T_CLAIM_ADJUSTMENT', 'LEGALLIABILITY',          'VARCHAR2(4000)'),  -- LegalLiability
        t_kolom('T_CLAIM_ADJUSTMENT', 'SALVAGENOTE',             'VARCHAR2(1000)'),  -- Salvage (teks, mis. "Tidak ekonomis")
        t_kolom('T_CLAIM_ADJUSTMENT', 'REMARKS',                 'VARCHAR2(4000)'),  -- Remarks
        t_kolom('T_CLAIM_ADJUSTMENT', 'REMARKACCEPTED',          'VARCHAR2(1000)'),  -- RemarkAccepted
        t_kolom('T_CLAIM_ADJUSTMENT', 'INITIALNAME',             'VARCHAR2(128)'),   -- Initial
        -- komite
        t_kolom('T_CLAIM_ADJUSTMENT', 'KOMITETYPE',              'VARCHAR2(10)'),    -- KomiteType
        t_kolom('T_CLAIM_ADJUSTMENT', 'KOMITEACCEPTED',          'VARCHAR2(128)'),   -- KomiteAccepted (operator)
        t_kolom('T_CLAIM_ADJUSTMENT', 'ISKOMITETRANSFER',        'NUMBER(1)'),       -- IsKomiteTransfer
        t_kolom('T_CLAIM_ADJUSTMENT', 'ISKOMITESETUJU',          'NUMBER(1)'),       -- IsKomiteSetuju
        t_kolom('T_CLAIM_ADJUSTMENT', 'TANGGALCOMITEE',          'TIMESTAMP'),       -- TanggalComitee (sudah ada ANALYST_TFKOMITEDATE, turunannya)
        -- akseptasi dan analis
        t_kolom('T_CLAIM_ADJUSTMENT', 'ACCEPTANCEANALYSTSTATUS', 'VARCHAR2(10)'),    -- AcceptanceAnalystStatus
        t_kolom('T_CLAIM_ADJUSTMENT', 'ACCEPTEDDATETIME',        'TIMESTAMP'),       -- AcceptedDateTime (TGLAKSEPTASI tanpa jam)
        t_kolom('T_CLAIM_ADJUSTMENT', 'TIPEAKSEPTASI',           'VARCHAR2(10)'),    -- TipeAkseptasi
        t_kolom('T_CLAIM_ADJUSTMENT', 'CONFIRMATIONANSWER',      'VARCHAR2(10)'),    -- ConfirmationAnswer
        t_kolom('T_CLAIM_ADJUSTMENT', 'ISANALISATORTRANSFER',    'NUMBER(1)'),       -- IsAnalisatorTransfer
        t_kolom('T_CLAIM_ADJUSTMENT', 'RECEIVEDATEANALIST',      'TIMESTAMP'),       -- ReceiveDateAnalist
        t_kolom('T_CLAIM_ADJUSTMENT', 'ADJUSTERTYPE',            'VARCHAR2(10)'),    -- AdjusterType
        t_kolom('T_CLAIM_ADJUSTMENT', 'JOINPLACEMENT',           'NUMBER(1)'),       -- JoinPlacement
        t_kolom('T_CLAIM_ADJUSTMENT', 'STATUSCLAIM',             'VARCHAR2(30)'),    -- StatusClaim (teks, mis. "ACCEPTATION")
        t_kolom('T_CLAIM_ADJUSTMENT', 'POSISIPROGRESSID',        'VARCHAR2(10)'),    -- PosisiProgressID
        t_kolom('T_CLAIM_ADJUSTMENT', 'USERBUSINESSPA',          'NUMBER(1)'),       -- UserBusinessPA
        t_kolom('T_CLAIM_ADJUSTMENT', 'CONF_SCORE',              'VARCHAR2(20)'),    -- CONF_SCORE (hasil AI; arti belum dipastikan)
        t_kolom('T_CLAIM_ADJUSTMENT', 'FILEDGT',                 'NUMBER(1)'),       -- FileDGT
        t_kolom('T_CLAIM_ADJUSTMENT', 'ISDLA',                   'NUMBER(1)'),       -- IsDLA
        -- LOD
        t_kolom('T_CLAIM_ADJUSTMENT', 'ISPRINTLOD',              'NUMBER(1)'),       -- IsPrintLOD
        t_kolom('T_CLAIM_ADJUSTMENT', 'TRANSFERLODDATE',         'TIMESTAMP'),       -- TransferLODDate
        t_kolom('T_CLAIM_ADJUSTMENT', 'LOADDATE',                'TIMESTAMP'),       -- LOADDate
        -- penerima dan pembayaran
        t_kolom('T_CLAIM_ADJUSTMENT', 'STSPENERIMAKLAIM',        'VARCHAR2(10)'),    -- StsPenerimaKlaim
        t_kolom('T_CLAIM_ADJUSTMENT', 'PEMBAYARANTO',            'VARCHAR2(10)'),    -- PembayaranTo
        t_kolom('T_CLAIM_ADJUSTMENT', 'PENERIMAPEMBAYARAN',      'VARCHAR2(1000)'),  -- PenerimaPembayaran
        t_kolom('T_CLAIM_ADJUSTMENT', 'TANGGALBOLEHBAYAR',       'TIMESTAMP'),       -- TanggalBolehBayar
        t_kolom('T_CLAIM_ADJUSTMENT', 'TGLKOMITEPENERIMA',       'TIMESTAMP'),       -- TglKomitePenerima
        t_kolom('T_CLAIM_ADJUSTMENT', 'TGLKASIRPENERIMA',        'DATE'),            -- TglKasirPenerima
        t_kolom('T_CLAIM_ADJUSTMENT', 'TRANSFERCASHIERSTATUS',   'VARCHAR2(10)'),    -- TransferCashierStatus
        t_kolom('T_CLAIM_ADJUSTMENT', 'CLAIMPAIDDATE',           'TIMESTAMP'),       -- ClaimPaidDate
        t_kolom('T_CLAIM_ADJUSTMENT', 'STATUSBAYAR',             'VARCHAR2(10)'),    -- StatusBayar (dibaca prosedur lama, tidak ditulis)
        -- AcceptanceDocument — REKENING NASABAH, jangan dicatat di log (§12 2.4)
        t_kolom('T_CLAIM_ADJUSTMENT', 'ACCEPTANCEBANK',          'VARCHAR2(100)'),   -- AcceptanceDocument.AcceptanceBank
        t_kolom('T_CLAIM_ADJUSTMENT', 'ACCOUNTNO',               'VARCHAR2(50)'),    -- AcceptanceDocument.AccountNo (teks: nol di depan bermakna)
        t_kolom('T_CLAIM_ADJUSTMENT', 'PAYABLETO',               'VARCHAR2(500)'),   -- AcceptanceDocument.PayableTo
        t_kolom('T_CLAIM_ADJUSTMENT', 'ACCEPTANCEDATE',          'TIMESTAMP'),       -- AcceptanceDocument.AcceptanceDate
        -- jejak dan soft delete (D-66)
        t_kolom('T_CLAIM_ADJUSTMENT', 'DIBUAT_PADA',             'TIMESTAMP'),       -- pxCreateDateTime
        t_kolom('T_CLAIM_ADJUSTMENT', 'DIBUAT_OLEH',             'VARCHAR2(128)'),   -- pxCreateOperator
        t_kolom('T_CLAIM_ADJUSTMENT', 'DIBUAT_OLEH_NAMA',        'VARCHAR2(128)'),   -- pxCreateOpName
        t_kolom('T_CLAIM_ADJUSTMENT', 'DIUBAH_OLEH',             'VARCHAR2(64)'),
        t_kolom('T_CLAIM_ADJUSTMENT', 'DIUBAH_PADA',             'TIMESTAMP'),
        t_kolom('T_CLAIM_ADJUSTMENT', 'DIHAPUS_OLEH',            'VARCHAR2(64)'),
        t_kolom('T_CLAIM_ADJUSTMENT', 'DIHAPUS_PADA',            'TIMESTAMP'),

        -- --------------------------------------------------------------------
        -- T_CLAIM_SPREADING (0008)  <- ObjectCoverageList[].SpreadingList[]
        -- Sudah ada: JENIS_TREATY, NAMA, SHARE_E4, DIHAPUS, OBJEK_FAC_OFFER, DIHAPUS_PADA
        -- --------------------------------------------------------------------
        t_kolom('T_CLAIM_SPREADING', 'TREATYGROUP',         'VARCHAR2(20)'),  -- TreatyGroup
        t_kolom('T_CLAIM_SPREADING', 'TREATYYEAR',          'NUMBER(4)'),     -- TreatyYear
        t_kolom('T_CLAIM_SPREADING', 'TREATYLIMIT',         'NUMBER'),        -- TreatyLimit
        t_kolom('T_CLAIM_SPREADING', 'TREATYLIMITPERCEN',   'NUMBER'),        -- TreatyLimitPercen
        t_kolom('T_CLAIM_SPREADING', 'TSISPREADED',         'NUMBER'),        -- TSISpreaded
        t_kolom('T_CLAIM_SPREADING', 'PREMIUMSPREADED',     'NUMBER'),        -- PremiumSpreaded
        t_kolom('T_CLAIM_SPREADING', 'PREMINET',            'NUMBER'),        -- PremiNet
        t_kolom('T_CLAIM_SPREADING', 'PERCENTTABARUFUND',   'NUMBER'),        -- PercentTabaruFund (syariah)
        t_kolom('T_CLAIM_SPREADING', 'ISFILLTSISPREADED',   'NUMBER(1)'),     -- IsFillTSISpreaded
        t_kolom('T_CLAIM_SPREADING', 'ISFILLPREMISPREADED', 'NUMBER(1)'),     -- IsFillPremiSpreaded
        t_kolom('T_CLAIM_SPREADING', 'FLAGOLDDATA',         'NUMBER(1)'),     -- FlagOldData
        t_kolom('T_CLAIM_SPREADING', 'FLAGEDITDATA',        'NUMBER(1)'),     -- FlagEditData

        -- --------------------------------------------------------------------
        -- T_CLAIM_DETAIL_SPREDING  <- AdjustmentList[].SpreadingList[]
        -- Sudah ada: CLAIMID, NOAKSEPTASI, TREATYNAME/TYPE/YEAR/GROUP, TSISPREADED,
        -- SHAREPERCENTAGE, CONTRACTNO — TANPA kolom yang mengenali barisnya.
        -- Empat kolom kunci pertama membuat baris baru dapat dikaitkan ke adjustment-nya
        -- dan diurutkan. Baris lama tetap NULL di keempatnya (lihat catatan akhir).
        -- --------------------------------------------------------------------
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'OBJECTID',            'VARCHAR2(30)'),  -- kunci: objek
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'OBJECTCOVERAGEID',    'VARCHAR2(30)'),  -- kunci: coverage klaim
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'ADJUSTMENTID',        'VARCHAR2(10)'),  -- kunci: adjustment
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'URUTAN',              'NUMBER(10)'),    -- kunci: urutan di SpreadingList
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'TREATYLIMIT',         'NUMBER'),        -- TreatyLimit
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'TREATYLIMITPERCEN',   'NUMBER'),        -- TreatyLimitPercen
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'PREMIUMSPREADED',     'NUMBER'),        -- PremiumSpreaded
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'PREMINET',            'NUMBER'),        -- PremiNet
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'PERCENTTABARUFUND',   'NUMBER'),        -- PercentTabaruFund
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'ISFILLTSISPREADED',   'NUMBER(1)'),     -- IsFillTSISpreaded
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'ISFILLPREMISPREADED', 'NUMBER(1)'),     -- IsFillPremiSpreaded
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'FLAGOLDDATA',         'NUMBER(1)'),     -- FlagOldData
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'FLAGEDITDATA',        'NUMBER(1)'),     -- FlagEditData
        t_kolom('T_CLAIM_DETAIL_SPREDING', 'DIHAPUS_PADA',        'TIMESTAMP'),     -- FlagDelete / soft delete (D-66)

        -- --------------------------------------------------------------------
        -- T_PLALIST  <- ObjectCoverageList[].PLAList[]
        -- Sudah ada: NOPLA, PLAREINSURER, REVISI, TIPEPLA, TGLPLA, ISKIRIM, EMAILPLA,
        -- REINSCODE, CURRENCYPOLIS, COUNTRY, NOTES, ESTIMASI, PERCENTPLA, NILAIPLA,
        -- ESTIMASISHARE. EstimasiList berisi tepat satu baris per PLA di sampel.
        -- --------------------------------------------------------------------
        t_kolom('T_PLALIST', 'INTEREST',          'VARCHAR2(1000)'),  -- Interest
        t_kolom('T_PLALIST', 'ESTIMATIONRESERVE', 'NUMBER'),          -- EstimasiList[].EstimastionReserve (salah ketik di Pega)
        t_kolom('T_PLALIST', 'PLACOMMITDATE',     'TIMESTAMP')        -- pxCommitDateTime
    );

    ada NUMBER;
BEGIN
    FOR i IN 1 .. daftar.COUNT LOOP
        SELECT COUNT(*) INTO ada
          FROM all_tab_cols
         WHERE owner = 'POOLDATA'
           AND table_name = daftar(i).tabel
           AND column_name = daftar(i).nama;

        IF ada = 0 THEN
            EXECUTE IMMEDIATE
                'ALTER TABLE POOLDATA.' || daftar(i).tabel || ' ADD (' ||
                daftar(i).nama || ' ' || daftar(i).tipe || ')';
        END IF;
    END LOOP;
END;
/

COMMENT ON COLUMN POOLDATA.T_CLAIM_OBJECTCOVERAGE.DIAGNOSE IS 'Data medis PA; akses terbatas Analyst Doctor / RCL Dokter, juga di staging (FR-R2, D-64)';
COMMENT ON COLUMN POOLDATA.T_CLAIM_ADJUSTMENT.ACCOUNTNO IS 'Rekening penerima (AcceptanceDocument.AccountNo); teks, jangan dicatat di log';
COMMENT ON COLUMN POOLDATA.T_CLAIM_ADJUSTMENT.CONVERTADJUSTMENTVALUE IS 'Nilai adjustment dalam IDR; pembanding ambang komite (D-47, D-48)';
COMMENT ON COLUMN POOLDATA.T_CLAIM_DETAIL_SPREDING.URUTAN IS 'Urutan di SpreadingList; NULL pada baris yang ditulis sebelum kolom ini ada';

-- Index untuk kunci baru T_CLAIM_DETAIL_SPREDING — hari ini tabel itu dicari lewat
-- NOAKSEPTASI saja, dan tidak ada index sama sekali.
DECLARE
    ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada
      FROM all_indexes WHERE owner = 'POOLDATA' AND index_name = 'IX_T_CLAIM_DETSPR_ADJ';

    IF ada = 0 THEN
        EXECUTE IMMEDIATE
            'CREATE INDEX POOLDATA.IX_T_CLAIM_DETSPR_ADJ
                 ON POOLDATA.T_CLAIM_DETAIL_SPREDING (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID)';
    END IF;
END;
/


-- ============================================================================
-- BAGIAN 2 — TABEL BARU TC_PNC_*: data yang belum punya tabel sama sekali
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. TC_PNC_OBJECTITEM — ObjectCoverageList[].ObjectItemList[]
--    Rincian barang/manfaat yang diklaim. Hari ini hanya estimasinya yang tersimpan
--    (T_CLAIM_ESTIMASI.OBJECTITEMID); item itu sendiri tidak.
--    T_CLAIM_OBJECTITEMLIST BUKAN tabel ini: ia per ADJUSTMENTID (TempObjectItemList).
-- ----------------------------------------------------------------------------
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

-- ----------------------------------------------------------------------------
-- 2. TC_PNC_BAGGAGELOST — ObjectItemList[].BaggageLostList[] (Travel)
--    Pada tingkat adjustment, bagasi sudah ada di T_CLAIM_OBJECTITEMLIST.BL_* —
--    tetapi hanya SATU per baris; di tingkat item klaim ia DERET.
-- ----------------------------------------------------------------------------
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

-- ----------------------------------------------------------------------------
-- 3. TC_PNC_COVERAGEDETAIL — ObjectCoverageList[].ObjectCoverageDetailList[]
--    Jadwal manfaat Travel (A.1, A.2, B.1 ...), ~50 baris per coverage.
-- ----------------------------------------------------------------------------
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

-- ----------------------------------------------------------------------------
-- 4. TC_PNC_CFS — ObjectCoverageList[].CFSList[] (Claim Face Sheet per revisi)
-- ----------------------------------------------------------------------------
CREATE TABLE POOLDATA.TC_PNC_CFS (
    CLAIMID                     VARCHAR2(100)   NOT NULL,
    OBJECTID                    VARCHAR2(30)    NOT NULL,
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,
    REVISI                      NUMBER(5)       NOT NULL,   -- Revisi
    CFSDATE                     TIMESTAMP,                  -- Date
    FILENAME                    VARCHAR2(500),              -- FileName
    CONSTRAINT PK_TC_PNC_CFS PRIMARY KEY (CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI)
);

-- ----------------------------------------------------------------------------
-- 5. TC_PNC_CFS_ESTIMASI — ObjectCoverageList[].CFSList[].EstimasiList[]
-- ----------------------------------------------------------------------------
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

-- ----------------------------------------------------------------------------
-- 6. TC_PNC_ADJUSTMENT_DOC — AdjustmentList[].DocumentList[]
--    Kaitan settlement -> lampiran. Berkasnya tetap di storage (D-16) dan metadatanya
--    di DATA_ATTACHFILE; yang belum ada adalah LAMPIRAN MANA milik SETTLEMENT MANA.
-- ----------------------------------------------------------------------------
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

-- Index lain tidak diperlukan: setiap tabel TC_PNC_* diakses per klaim, dan CLAIMID
-- adalah kolom terdepan primary key-nya.


-- ============================================================================
-- ROLLBACK (P-4) — dijalankan DBA HANYA bila perubahan ini harus ditarik
-- ============================================================================
--
-- Bagian 2 — tabel baru, urutan anak dulu:
--   DROP TABLE POOLDATA.TC_PNC_BAGGAGELOST;
--   DROP TABLE POOLDATA.TC_PNC_OBJECTITEM;
--   DROP TABLE POOLDATA.TC_PNC_CFS_ESTIMASI;
--   DROP TABLE POOLDATA.TC_PNC_CFS;
--   DROP TABLE POOLDATA.TC_PNC_COVERAGEDETAIL;
--   DROP TABLE POOLDATA.TC_PNC_ADJUSTMENT_DOC;
--
-- Bagian 1 — kolom tambahan. JANGAN dijalankan setelah aplikasi Go mulai menulis ke
-- kolom-kolom ini: isinya ikut hilang. Kolom yang SUDAH ADA sebelum skrip ini (dilewati
-- oleh pemeriksaan ALL_TAB_COLS) tidak boleh ikut di-DROP — periksa per portal dulu.
--   DROP INDEX POOLDATA.IX_T_CLAIM_DETSPR_ADJ;
--   ALTER TABLE POOLDATA.T_CLAIM_OBJECTLIST DROP (OBJECTINDEX, OBJECTSURVEYLOCATION,
--       CITYID, CITY, DISTRICTID, DISTRICT, RWID, RWNOTE, BRANCHCODE, OCCUPATIONID,
--       OCCUPATIONNAME, SURVEYORTYPE, CURRENCY, CURRENCYAKSEP, NILAIOSKLAIM,
--       NILAIOSADJUSTER, NILAIAKSEPALL, NILAIADJUSTERALL, ISKOMITEAPPROVE, OBJECTIDCARD,
--       OBJECTHEIGHT, OBJECTWEIGHT, OBJECTJOB, OBJECTLEFTHANDED, OBJECTPARTICIPANTSTATUS,
--       DIUBAH_OLEH, DIUBAH_PADA);
--   ALTER TABLE POOLDATA.T_CLAIM_OBJECTCOVERAGE DROP (OBJECTNAME, LEGALLIABILITY,
--       REMARKINVESTIGATION, DIAGNOSE, CODEDIAGNOSE, DESCDIAGNOSE, TSISUBLIMIT,
--       CURRENCYESTIMASI, TOTALESTIMASIJAMINAN, ISCFS, ISPLA, ISANALISTRANSFER,
--       ISKOMITETRANSFER, ISKOMITETRAVELTRANSFER, CHECKCOVERAGE, CHECKINTEREST,
--       USERBUSINESSPA, TEMPRECEIVER, INITIALNAME, TANGGALCOMITEE, DIBUAT_OLEH,
--       DIBUAT_OLEH_NAMA, DIUBAH_OLEH, DIUBAH_PADA);
--   ALTER TABLE POOLDATA.T_CLAIM_ESTIMASI DROP (KURSVALUEPOLIS, ESTIMATIONDATE,
--       PRINTFACECLAIM, TRAVELID, DIBUAT_PADA, DIBUAT_OLEH, DIBUAT_OLEH_NAMA);
--   ALTER TABLE POOLDATA.T_CLAIM_ADJUSTMENT DROP (ESTIMATIONVALUE, PROPOSEVALUETERTANGGUNG,
--       PROPOSEADJUSTMENTFINAL, CONVERTADJUSTMENTVALUE, ACCEPTANCEVALUELOD, SALVAGEVALUE,
--       ADJUSTERFEEVALUE, TOTALADJUSTER, TOTALIDRESTIMASI, ASMFULL, INTERIMPAYMENT, VATTYPE,
--       EXTENTOFLOSS, LEGALLIABILITY, SALVAGENOTE, REMARKS, REMARKACCEPTED, INITIALNAME,
--       KOMITETYPE, KOMITEACCEPTED, ISKOMITETRANSFER, ISKOMITESETUJU, TANGGALCOMITEE,
--       ACCEPTANCEANALYSTSTATUS, ACCEPTEDDATETIME, TIPEAKSEPTASI, CONFIRMATIONANSWER,
--       ISANALISATORTRANSFER, RECEIVEDATEANALIST, ADJUSTERTYPE, JOINPLACEMENT, STATUSCLAIM,
--       POSISIPROGRESSID, USERBUSINESSPA, CONF_SCORE, FILEDGT, ISDLA, ISPRINTLOD,
--       TRANSFERLODDATE, LOADDATE, STSPENERIMAKLAIM, PEMBAYARANTO, PENERIMAPEMBAYARAN,
--       TANGGALBOLEHBAYAR, TGLKOMITEPENERIMA, TGLKASIRPENERIMA, TRANSFERCASHIERSTATUS,
--       CLAIMPAIDDATE, STATUSBAYAR, ACCEPTANCEBANK, ACCOUNTNO, PAYABLETO, ACCEPTANCEDATE,
--       DIBUAT_PADA, DIBUAT_OLEH, DIBUAT_OLEH_NAMA, DIUBAH_OLEH, DIUBAH_PADA,
--       DIHAPUS_OLEH, DIHAPUS_PADA);
--   ALTER TABLE POOLDATA.T_CLAIM_SPREADING DROP (TREATYGROUP, TREATYYEAR, TREATYLIMIT,
--       TREATYLIMITPERCEN, TSISPREADED, PREMIUMSPREADED, PREMINET, PERCENTTABARUFUND,
--       ISFILLTSISPREADED, ISFILLPREMISPREADED, FLAGOLDDATA, FLAGEDITDATA);
--   ALTER TABLE POOLDATA.T_CLAIM_DETAIL_SPREDING DROP (OBJECTID, OBJECTCOVERAGEID,
--       ADJUSTMENTID, URUTAN, TREATYLIMIT, TREATYLIMITPERCEN, PREMIUMSPREADED, PREMINET,
--       PERCENTTABARUFUND, ISFILLTSISPREADED, ISFILLPREMISPREADED, FLAGOLDDATA,
--       FLAGEDITDATA, DIHAPUS_PADA);
--   ALTER TABLE POOLDATA.T_PLALIST DROP (INTEREST, ESTIMATIONRESERVE, PLACOMMITDATE);


-- ============================================================================
-- CATATAN YANG MASIH TERBUKA (pemilik: Work Owner)
-- ============================================================================
--
-- 1. DIPUTUSKAN (Work Owner, 2026-09-26): baris lama T_CLAIM_DETAIL_SPREDING DIBIARKAN
--    NULL pada OBJECTID/OBJECTCOVERAGEID/ADJUSTMENTID/URUTAN — tidak diisi ulang.
--    Akibatnya: pembaca yang menyaring lewat keempat kolom itu hanya menemukan baris
--    yang ditulis setelah skrip ini; baris lama tetap dicari lewat NOAKSEPTASI.
--
-- 2. PENULIS KOLOM BARU. Kolom-kolom di Bagian 1 berada di tabel yang JUGA ditulis
--    PEGA_CONVERT_JSONKLAIM_PNC. Prosedur itu tidak mengisinya, sehingga untuk klaim
--    Pega (PNC-xxxx) kolom baru tetap kosong. Aturan P-1 tetap terjaga lewat kunci
--    (klaim PNCN hanya ditulis Go), sama seperti T_CLAIM_RECIVEDCLAIM.
--
-- 3. Kosong di SELURUH 11 sampel, kolomnya tidak dikarang: AccidentTravel, CancelTrip,
--    DelayedTravel (tingkat item klaim), NotesAI, PageResultAI, RESULT_CRAWLING,
--    PUCLStatus per adjustment. Deret Old* dianggap data polis.
--
-- 4. DIPUTUSKAN (Work Owner, 2026-09-26): skrip ini dijalankan untuk PORTAL ASM SAJA
--    lebih dulu. Portal lain menyusul; sebelum itu katalog portal tersebut diperiksa
--    ulang, karena panjang/tipe kolom yang sudah ada di sana tidak diubah skrip ini.
