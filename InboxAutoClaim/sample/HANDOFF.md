# Handoff — PEGA Rule-Tree Extractor (InboxAutoClaim)

Dokumen ini agar pekerjaan bisa **dilanjutkan di chat Claude yang baru**. Chat baru tidak punya ingatan percakapan lama, jadi ikuti langkah di bawah.

## 1. Tujuan

Membaca file XML rule PEGA (Harness → Section → Activity → RDB List / Data Transform) dan menyusun **struktur pohon berjenjang** ke dalam spreadsheet, meniru contoh `Sample_Activity.jpeg` dengan kolom: **No | Nama Rule | Shape ID | Jenis Rule | XML**. Ditambah sheet kedua **"XML needed"** berisi daftar rule yang masih perlu diunggah untuk penelusuran lebih dalam.

## 2. Metode (penting)

- Rule akar = Harness `InboxAutoClaim`.

- Anak tiap rule diambil dari **indeks referensi** di dalam XML-nya: blok `<rowdata ...>` yang memuat `Embed-Reference-Rule`, dibaca `pxRuleObjClass` + `pyRuleName`.

- Pemetaan jenis: `Rule-Obj-Activity`→Activity, `Rule-RDB-SQL`→RDB List, `Rule-Obj-Model`→Data Transform, `Rule-HTML-Section`→Section, `Rule-Obj-Flow`→Flow.

- Nama RDB List dibersihkan: ambil bagian setelah `' GCNM '` (buang prefix class).

- **Filter (Opsi 1):** buang rule OOTB (nama diawali `px`/`py`/`pz`, `CustomActivePage`) DAN activity engine PEGA (lihat set `ENGINE` di skrip).

- **Dedup (Opsi 2):** tiap rule diurai penuh **sekali** pada kemunculan pertama; kemunculan berikutnya ditandai baris abu-abu "↑ lihat No X" dan tidak diekspansi ulang.

- Level-2 (anak langsung Harness) memakai daftar tetap `level2` di skrip (hasil analisis layout harness), karena indeks referensi harness sendiri sudah usang/stale.

- Kolom **Shape ID** hanya terisi untuk Harness; kosong untuk sisanya (Shape ID berasal dari shape Flow).

- Nama file XML pada tabel disintesis: `{No} {NamaRule}{suffix}.xml` (suffix: -Act/-SQL/-DT/-Section/-Flow), kecuali akar memakai `InboxAutoClaim-Harness.xml`.

## 3. Status terkini

- Pohon: **337 baris** (148 di antaranya baris duplikat). Rincian jenis: {'Harness': 1, 'Section': 4, 'Activity': 168, 'RDB List': 159, 'Data Transform': 5}.

- File XML sudah diunggah: **101**.

- Masih dibutuhkan ("XML needed"): **110** — 13 non-RDB (bisa diurai lagi) + 97 RDB List.

## 4. Yang masih perlu diunggah dulu (paling berguna: Activity/DT)

- [Activity] `HTMLToPDF-Act.xml`
- [Activity] `PrintPDFAcceptanceCreditNote-Act.xml`
- [Activity] `SendAttachmenttoCashier_act-Act.xml`
- [Activity] `SendCashierAutoClaim-Act.xml`
- [Activity] `SendDLAAutoSaatGeneratedDLA-Act.xml`
- [Activity] `SendEmailNotification-Act.xml`
- [Activity] `SendPLADLA_SRB-Act.xml`
- [Activity] `SendSimpleEmail-Act.xml`
- [Activity] `SetSignatureNonMBU-Act.xml`
- [Activity] `SetSpreadingAsuransiKredit-Act.xml`
- [Activity] `SetSpreadingAsuransiKredit_PA-Act.xml`
- [Activity] `SetUploadDocument-Act.xml`
- [Data Transform] `SetJSONPage-DT.xml`

<details><summary>97 RDB List (klik untuk lihat) — umumnya leaf</summary>

- `ASM-FW-GISFW-Int-CURRENCY ASM GetIDCurrencyByID-SQL.xml`
- `ASM-FW-GISFW-Int-CURRENCYSTANDARD ASM CurrencyStandard-SQL.xml`
- `BrowseAutoKlaim-SQL.xml`
- `BrowseClaimSPK1_AsuransiKredit-SQL.xml`
- `BrowseClaimSPK1_AutoClaim-SQL.xml`
- `BrowseClaimSPK1_Travel-SQL.xml`
- `BrowseClaimSPK_COUNT_AsuransiKredit-SQL.xml`
- `BrowseClaimSPK_COUNT_AutoClaim-SQL.xml`
- `BrowseClaimSPK_COUNT_Travel-SQL.xml`
- `BrowseClaimSPK_detail_AsuransiKredit-SQL.xml`
- `BrowseClaimSPK_detail_AutoClaim-SQL.xml`
- `BrowseClaimSPK_detail_Travel-SQL.xml`
- `BrowseClaimSPKAutoClaim-SQL.xml`
- `BrowseClaimSPKClaimKredit-SQL.xml`
- `BrowseClaimSPKTravel-SQL.xml`
- `BrowseCompanyClaimCredit-SQL.xml`
- `BrowseDataReas-SQL.xml`
- `BrowseDataReasCoinsDLA-SQL.xml`
- `BrowseEmailReas-SQL.xml`
- `BrowsePolisForKredit-SQL.xml`
- `BrowseReportClaimSPK_AsuransiKredit-SQL.xml`
- `BrowseReportClaimSPK_AutoClaim-SQL.xml`
- `BrowseReportClaimSPK_Travel-SQL.xml`
- `BrowseServiceName_sql-SQL.xml`
- `CallPackageGLAsuransiKredit-SQL.xml`
- `CallProccedureInsertMitra-SQL.xml`
- `CallProccedureInsertPLADLA2-SQL.xml`
- `CheckExistsTreatyLoss_SQL-SQL.xml`
- `GetAdd1MonthSLIK-SQL.xml`
- `GetBisnisFromInskey-SQL.xml`
- `GetBusinessCode2-SQL.xml`
- `GetCauseofLossDesc-SQL.xml`
- `GetCreditClaimTable2-SQL.xml`
- `GetDataDLA-SQL.xml`
- `GetDataPreDLA-SQL.xml`
- `GetDataProsesClaimTravel-SQL.xml`
- `GetDetailTanggalBayarKlaims-SQL.xml`
- `GetGroupPanelKlaimKredit-SQL.xml`
- `GetKlaimKreditAdjustment-SQL.xml`
- `GetListDataLoginReas-SQL.xml`
- `GetListKlaimKredit-SQL.xml`
- `GetListObjectAneka_SpkUpload-SQL.xml`
- `GetListObjectTravel-SQL.xml`
- `GetListSumbis-SQL.xml`
- `GetMandalaFinancePremi_SQL-SQL.xml`
- `GetMaxBatchAsuransiKredit-SQL.xml`
- `GetObjectAsuransiKredit-SQL.xml`
- `GetObjectAsuransiKredit_PA-SQL.xml`
- `GetObjekKreditOutstanding-SQL.xml`
- `GetOldPolisDataAsuransiKredit-SQL.xml`
- `GetReceiverClaimAsuransiKredit-SQL.xml`
- `GetShareFacout_askredit-SQL.xml`
- `GetShareFacout_askredit_smmf-SQL.xml`
- `GetShareTKA-SQL.xml`
- `GetSumbis-SQL.xml`
- `GetTipeKlaimKredit-SQL.xml`
- `GetTotalKlaimAksep_ASIKredit-SQL.xml`
- `GetTotalKlaimCreditValue-SQL.xml`
- `GetTotalKlaimCreditValue_API-SQL.xml`
- `GetTotalKlaimCreditValue_APIAkseptasiW-SQL.xml`
- `GetTreatyGroupID-SQL.xml`
- `GetTypeReinsuranceSpreading-SQL.xml`
- `GetwomPremi-SQL.xml`
- `GroupingAsuransiKredit-SQL.xml`
- `GroupingAsuransiKreditAdjustment-SQL.xml`
- `GroupingAsuransiKreditOutstanding-SQL.xml`
- `GroupingAutoClaim-SQL.xml`
- `GroupingAutoClaim2-SQL.xml`
- `GroupingAutoClaim3-SQL.xml`
- `GroupingKlaimKredit-SQL.xml`
- `InsertClaimPNC-SQL.xml`
- `InsertDataSlikOJKF06-SQL.xml`
- `InsertDokumenPLADLA-SQL.xml`
- `InsertOSAkseptasiClaimPNC-SQL.xml`
- `InsertPLADLA_SQL-SQL.xml`
- `InsertTempAsuransiKredit-SQL.xml`
- `InsertTempAutoClaim-SQL.xml`
- `InsertTempAutoTravel-SQL.xml`
- `InsertUpdateAutoClaim-SQL.xml`
- `InsertUpdateAutoClaimTravel-SQL.xml`
- `KonversiDLAKlaimNonMBU-SQL.xml`
- `KonversiOSAkseptasiKlaimNonMBU-SQL.xml`
- `Loging_InsertClaimPNC-SQL.xml`
- `QueryClaimBatchAneka-SQL.xml`
- `RunConvertJSONKLAIM-SQL.xml`
- `SearchCodeBank_sql-SQL.xml`
- `SearchLinkAttachment-SQL.xml`
- `searchQSReins2_SQL-SQL.xml`
- `searchQSReins_SQL-SQL.xml`
- `SelectProportionalArrg-SQL.xml`
- `SelectTreatyReinsurer-SQL.xml`
- `UpdateAkseptasiKlaimKredit-SQL.xml`
- `UpdateAsuransiKredit-SQL.xml`
- `UpdateAsuransiKredit_tanpanoasuransi-SQL.xml`
- `UpdateEmailReas-SQL.xml`
- `UpdateNoAksepPreDLA-SQL.xml`
- `UpdatesetstatusdantanggalKirimDLA-SQL.xml`
</details>

## 5. File XML yang SUDAH diunggah (perlu diunggah ulang di chat baru)

<details><summary>Daftar 101 file (klik)</summary>

- `ASMCollectAttachments-Act.xml`
- `ASMForceCaseClose-Act.xml`
- `ASMSendsEmailAttachments-Act.xml`
- `AddWork-Act.xml`
- `AllCoveredResolved-Act.xml`
- `AttachAsPDFC-Act.xml`
- `BPPDANCekNilaiDla-Act.xml`
- `BrowseAutoKlaim-Section.xml`
- `BrowseAutoKlaim_act-Act.xml`
- `ButtonPagingInbox-Section.xml`
- `CekNilaiKlaimBerdasarkanNGPWSimasnet-Act.xml`
- `CekPremi-Act.xml`
- `CekTotalPremiDanAkseptasiKlaimPerdagangan-Act.xml`
- `CheckDuplicates-Act.xml`
- `CheckKeys-Act.xml`
- `ConnectRestPNC_act-Act.xml`
- `CreateCasePNCAgent_AsuransiKredit-Act.xml`
- `CreateCasePNCAgent_AutoClaim-Act.xml`
- `CreateCasePNCAgent_Travel-Act.xml`
- `CreateCasePNCAuto_Travel-Act.xml`
- `CreateCasePNC_AsKredit-Act.xml`
- `CreateCasePNC_AutoClaim-Act.xml`
- `CreateCasePNC_Kredit_PA-Act.xml`
- `CreateInstance-Act.xml`
- `CreateWorkPage-Act.xml`
- `DETAIL_ASURANSI_KREDIT-Act.xml`
- `DETAIL_AUTO_CLAIM-Act.xml`
- `DETAIL_TRAVEL-Act.xml`
- `DLABPPDAN_act-Act.xml`
- `DLACoins_act-Act.xml`
- `DLAFacoutKredit_act-Act.xml`
- `DLAFacout_act-Act.xml`
- `DLATreaty_Act-Act.xml`
- `DeleteAttachment-Act.xml`
- `DictionaryValidation-Act.xml`
- `DownloadAllDocumentDlaByAkseptasi-Act.xml`
- `DownloadDLA-Act.xml`
- `FindDataReinsurerDLA-Act.xml`
- `FindOpenProtecionAsKredit-Act.xml`
- `GCNMCreateOperator-Act.xml`
- `GenerateDLAList-Act.xml`
- `GenerateDLA_Askredit-Act.xml`
- `GetLinkAppClaim-Act.xml`
- `GetListKorwil-Act.xml`
- `GetObjectFromTable-Act.xml`
- `GetObjectFromTable_AutoClaim-Act.xml`
- `GetObjectFromTable_Kredit_PA-Act.xml`
- `GetObjectFromTable_Travel-Act.xml`
- `GetReportClaimAsuransiKredit-Act.xml`
- `GetReportClaimAutoClaim-Act.xml`
- `GetReportClaimTravel-Act.xml`
- `GetTotalPremi-Act.xml`
- `HitServiceOSAkseptasiClaimNonMBU-Act.xml`
- `INBOX_AS_KREDIT_ACT_AUTOCLAIM-Act.xml`
- `INBOX_AS_KREDIT_ACT_CLAIMKREDIT-Act.xml`
- `INBOX_AS_KREDIT_ACT_TRAVEL-Act.xml`
- `InboxAutoClaim-Harness.xml`
- `Inbox_AS_KREDIT_Sect-Section.xml`
- `InsertAdjustmentList-Act.xml`
- `InsertAdjustmentListKredit-Act.xml`
- `InsertAdjustmentList_AutoClaim-Act.xml`
- `InsertAdjustmentList_Kredit_PA-Act.xml`
- `InsertCasePNCAuto_Travel-Act.xml`
- `InsertCasePNC_AsuransiKredit-Act.xml`
- `InsertCasePNC_AutoClaim-Act.xml`
- `InsertCasePNC_Kredit_PA-Act.xml`
- `InsertJsonClaimNonMBU_act-Act.xml`
- `InsertObjectList-Act.xml`
- `InsertObjectListAuto_Travel-Act.xml`
- `InsertObjectList_AutoClaim-Act.xml`
- `InsertObjectList_Kredit_PA-Act.xml`
- `InsertOutstandingAdjustment-Act.xml`
- `NewDefaults-Act.xml`
- `NewInternalDefaults-Act.xml`
- `OsAkseptasiKlaimKredit-Act.xml`
- `PNCInsertMitraLog_Act-Act.xml`
- `PNCInsertPLADLA-Act.xml`
- `PreSave-Act.xml`
- `REPORT_ASURANSI_KREDIT_ACT-Act.xml`
- `REPORT_AUTO_CLAIM_ACT-Act.xml`
- `REPORT_TRAVEL_ACT-Act.xml`
- `RecalculateAndSave-Act.xml`
- `RemoveNotNeedPolicyData-DT.xml`
- `RemovePaginationFU-Act.xml`
- `RemovePaginationOS-Act.xml`
- `Resolve-Act.xml`
- `Save-Act.xml`
- `SaveSetup-Act.xml`
- `SetDataClaimKredit-Act.xml`
- `SpreadingTKA-Act.xml`
- `StandardValidate-Act.xml`
- `SuspendFlows-Act.xml`
- `UpdateDetailDLA2-Act.xml`
- `UpdateStatus-Act.xml`
- `Validate-Act.xml`
- `ValidationInitial_act-Act.xml`
- `WorkUnlock-Act.xml`
- `commitWithErrorHandling-Act.xml`
- `performAssignmentCheck-Act.xml`
- `setOutput-Act.xml`
- `svcAddWorkObject-Act.xml`
</details>

## 6. Cara melanjutkan di chat baru — langkah

1. Buka chat Claude baru **dengan Code Execution / File Creation aktif**.

2. Lampirkan: (a) semua file `*.xml` yang sudah dikumpulkan (daftar di bagian 5) + file baru, (b) `Sample_Activity.jpeg` sebagai acuan format, (c) file ini (`HANDOFF.md`), dan (d) skrip `build.py` & `render.py`.

3. Tempel **prompt** di bagian 7.

4. Claude menjalankan `build.py` lalu `render.py` untuk menghasilkan ulang `InboxAutoClaim-Rule-Structure.xlsx` (2 sheet), dan memberi daftar "XML needed" terbaru.

5. Ulangi: unggah rule dari daftar "XML needed" → jalankan lagi → sampai daftar tinggal RDB List.

## 7. Prompt siap-tempel untuk chat baru

> Saya melanjutkan pembuatan struktur pohon rule PEGA untuk Harness `InboxAutoClaim`, meniru format `Sample_Activity.jpeg` (kolom: No | Nama Rule | Shape ID | Jenis Rule | XML) plus sheet "XML needed". Metode dan aturan lengkap ada di `HANDOFF.md` yang saya lampirkan. Gunakan skrip `build.py` dan `render.py` yang saya lampirkan **apa adanya** (jangan ubah logika filter Opsi 1 = buang rule engine PEGA, dan Opsi 2 = dedup global dengan penanda "↑ lihat No X"). Semua file `*.xml` sudah saya lampirkan di folder upload. Jalankan `build.py` lalu `render.py`, hasilkan `InboxAutoClaim-Rule-Structure.xlsx` (2 sheet: "Rule Structure" dan "XML needed"), tampilkan filenya, lalu berikan daftar "XML needed" terbaru. Jika ada file XML baru yang saya tambahkan, otomatis ikut terurai karena skrip membaca seluruh folder upload.

## 8. Catatan / keputusan yang sudah disepakati

- Rule engine PEGA (Save, WorkLock, NewDefaults, createWorkPage, Resolve, Validate, dll) **dikecualikan**.

- Duplikat **tidak** diekspansi ulang (dedup global) untuk mencegah ukuran meledak.

- Jika ingin `HTMLToPDF`/`SendSimpleEmail` dianggap OOTB juga, tambahkan namanya ke set `ENGINE` di `build.py`.

- Skrip membaca folder `/mnt/user-data/uploads` — cukup lampirkan file, tak perlu ubah path.
