# Lampiran Gap Export — Rincian R-01, R-04, R-07

Daftar lengkap artefak yang **dirujuk oleh aplikasi tetapi tidak ada di export XML**. Dokumen
ini melengkapi `16-RISK-ANALYSIS.md`, yang hanya menyebut jumlah dan beberapa contoh.

Gunakan daftar ini apa adanya sebagai lampiran permintaan ke DBA dan tim Pega.

Seluruh isi diekstraksi langsung dari 652 rule Connect-SQL dan 902 Activity di export.

> **PERINGATAN SNAPSHOT (2026-10-03) — angka di bawah berasal dari export yang LEBIH LAMA.**
>
> Export bertambah lagi, dan pertambahannya menyeluruh — bukan satu dua berkas. Dihitung
> langsung dari direktori:
>
> | Direktori | Snapshot v2.0 (2026-09-14) | **Sekarang** |
> |---|---|---|
> | Seluruh berkas XML | 2.634 | **2.960** |
> | `Activity/` | 902 | **1.075** |
> | `RDB List/` | 652 | **820** |
> | `Section/` | 269 | **331** |
> | `Harness/` | 74 | **84** |
> | `When/` | 70 | **87** |
> | `Data Transform/` | 80 | **85** |
> | `Report Definition/` | 56 | **67** |
> | `Flow Action/` | 29 | **54** |
>
> **307 berkas XML disentuh dalam tujuh hari terakhir.**
>
> Akibatnya: setiap pernyataan "tidak ada di export" pada dokumen ini — dan pada dokumen
> mana pun yang bersandar pada snapshot 2.634 — **wajib diperiksa ulang sebelum dipakai**.
> Satu contoh nyata sudah ditemukan: `UploadDocument` tercatat hilang pada 2026-09-20 dan
> **sekarang ada** (`Flow Action/UploadDocument-FA.xml`).
>
> Ini pengulangan keadaan `D-45`, yang menjalankan ulang Fase 1 terhadap snapshot baru.
> Audit ulang yang setara **belum dijalankan** untuk snapshot ini.

> **Diperbarui v2.0 (2026-09-14) — dokumen ini ternyata jauh dari lengkap.**
>
> Audit ulang terhadap snapshot baru menemukan **±242 rule dirujuk tetapi tidak ada di export**,
> bukan puluhan. Yang lebih penting: **tujuh tipe rule tidak diaudit dokumen ini sama sekali** —
> **When, Flow Action, Section, Data Transform, Ticket, Report Definition, dan Function library**.
>
> | Kategori | Diaudit dokumen ini | Temuan v2.0 |
> |---|---|---|
> | Objek database (`R-01`) | ✅ 64 objek | ✅ **62 diterima**; sisa `SET_ATTACHFILETEMPSALVAGE`, `UPDATEPREMIUMTEMPLATE` — **plus 12 dependensi baru** yang dipanggil 62 berkas itu |
> | Router (`R-04`) | ✅ 8 router | algoritma beban **terbaca dari kueri lain**; risiko turun menjadi verifikasi |
> | Activity (`R-07`) | ✅ 50 activity | tetap berlaku |
> | **When rule** | ❌ tidak diaudit | **137 When rule buatan sendiri hilang** — kategori terbesar |
> | **Ticket rule** | ❌ tidak diaudit | **17 nama dirujuk, 8 ada, 9 tanpa rule** (1 di antaranya bawaan Pega) |
> | Flow Action · Section · Data Transform · Report Definition · Function | ❌ tidak diaudit | tercakup dalam ±242 |
>
> Konsekuensinya dicatat sebagai **`R-16`**, dan permintaan penggantinya bukan lagi daftar manual
> per tipe melainkan **export ulang berbasis Product rule** (`D-39`) — karena daftar manual
> terbukti melewatkan tujuh tipe sekaligus.
>
> **Isi §-§ di bawah tetap sahih** untuk tiga kategori yang memang diauditnya; ia kurang lengkap,
> bukan salah.

---

## Koreksi jumlah stored procedure

Angka **86 stored procedure** yang tercantum di dokumen Steering lain adalah **hasil hitungan
yang terlalu tinggi**. Pemindaian awal menangkap pola `SKEMA.NAMA(` tanpa memeriksa kata di
depannya, sehingga nama tabel pada `INSERT INTO POOLDATA.T_CLAIM_SLIK_OJK (kolom, kolom, …)`
ikut terhitung sebagai pemanggilan procedure.

Setelah pemindaian ulang dengan pemeriksaan posisi sintaksis, angka terverifikasi:

| Kategori | Jumlah | Keterangan |
|---|---|---|
| **Stored procedure** | 56 | Dipanggil sebagai statement di dalam blok PL/SQL |
| **Function lokal** | 8 | Dipanggil sebagai ekspresi di dalam SELECT/WHERE |
| **Function remote (DB Link)** | 6 | Milik database lain — masuk lingkup **R-03**, bukan R-01 |
| **Total** | 70 |  |

> **Yang harus diminta ke DBA untuk R-01 adalah 64 objek** —
> 56 procedure dan 8 function lokal. Yang 6 lagi berada di database lain
> dan ditangani lewat penggantian DB Link (D-25).

---

## R-01 · Stored Procedure yang Source-nya Dibutuhkan

**56 procedure.** Kolom *Param* adalah jumlah argumen pada pemanggilan — berguna untuk
memverifikasi bahwa source yang diterima memang versi yang dipakai aplikasi.

| No | Objek database | Param | Rule pemanggil | Modul terdampak |
|---|---|---|---|---|
| 1 | `COLLECTION.P_GET_DATA_REFUND` | 5 | `BrowseDataRefund` | perlu ditentukan |
| 2 | `GENERAL.GET_TOKEN_STORAGE` | 4 | `GenerateTokenPNCDokumen` | S-1 Dokumen |
| 3 | `KONVERSIT_VEHICLELIST` | 2 | `KonversiT_Vehiclelist_SQL` | B-1 Polis/Snapshot |
| 4 | `POOLDATA.ADD_NEWMASTERVIRTUALACCOUNT` | 8 | `InsertDataMSTVirtualAccount` | B-10 Akseptasi |
| 5 | `POOLDATA.CONVERTJSONPRODUCTION` | 3 | `ConvertJSONProduction_SQL` | B-1 Polis/Snapshot |
| 6 | `POOLDATA.DOCTRAVEL_CVG` | 4 | `UpdateMstDocTravel` | S-1 Dokumen |
| 7 | `POOLDATA.INSERT_KPIADJUSTER` | 14 | `CallProcedureInsertKPISurvey` | B-8 Survey |
| 8 | `POOLDATA.INSERT_PLADLA` | 28 | `CallProccedureInsertPLADLA2` | B-9 PLA/DLA |
| 9 | `POOLDATA.INSERT_PNCCHRONOLOGYTAT` | 13 | `CallProccedureInsertMitra` | S-7 Progres/TAT |
| 10 | `POOLDATA.INSERT_SALVAGE` | 29 | `InsertNewSalvage` | B-12 Salvage |
| 11 | `POOLDATA.INSERT_SALVAGE_DETAILS` | 18 | `InsertDetailsSalvageperObject` | B-12 Salvage |
| 12 | `POOLDATA.INSERT_SURVEYORLIST` | 17 | `CallProcedureInsertSurvey` | B-8 Survey |
| 13 | `POOLDATA.INSERT_UPDATE_MST_XOL` | 18 | `SetMasterXOL` | B-9 PLA/DLA |
| 14 | `POOLDATA.INSERTDATAAIKLAIMPNC` | 30 | `InsertDataAIToTableFlatKlaimPA` | perlu ditentukan |
| 15 | `POOLDATA.INSERTDATAKLAIMCABANG` | 15 | `InsertDataCabang` | perlu ditentukan |
| 16 | `POOLDATA.INSERTDATAKOMITELIST` | 21 | `InsertDataToFlatTableKomiteList` | B-7 Komite |
| 17 | `POOLDATA.INSERTDATASFILLINGARCHIVE` | 18 | `InsertToClaimArchive` | S-1 Dokumen |
| 18 | `POOLDATA.INSERTMASTERRECOVERYKLAIM` | 21 | `InsertMasterRecoveryKlaimASM` | B-12 Salvage |
| 19 | `POOLDATA.INSERTMASTERREJECTEDKOMITE` | 3 | `MasterRejectedKlaimPNC` | B-7 Komite |
| 20 | `POOLDATA.INSERTOBJECTBACKUP` | 6 | `INSERTJSON_OBJECTPERSONPEGA` | B-1 Polis/Snapshot |
| 21 | `POOLDATA.INSERTPOLISTOJSON` | 2 | `InsertPolicyToJSON` | B-1 Polis/Snapshot |
| 22 | `POOLDATA.INSERTT_PERSONLIST` | 7 | `INSERTJSON_OBJECTPERSONPEGA` | B-1 Polis/Snapshot |
| 23 | `POOLDATA.MASTERPENOLAKANKLAIM1` | 3 | `SaveDataPenolakanKlaimMaster` | F-4 Master Data |
| 24 | `POOLDATA.MASTERPENOLAKANKLAIM2` | 6 | `SaveDataPenolakanKlaimMaster2` | F-4 Master Data |
| 25 | `POOLDATA.PEGA_CONVERT_JSONKLAIM_PNC` | 2 | `RunConvertJSONKLAIM` | B-1 Polis/Snapshot |
| 26 | `POOLDATA.PEGA_D_CAUSE_OF_LOSS` | 3 | `UpdateDCauseOfLoss` | F-4 Master Data |
| 27 | `POOLDATA.PEGA_D_PASAL_MASTER` | 4 | `UpdateDPasalDataMaster` | F-4 Master Data |
| 28 | `POOLDATA.PEGA_D_SURVEYORS` | 3 | `UpdateDetailSurveyors` | B-8 Survey |
| 29 | `POOLDATA.PEGA_JSON_INSERT_HISTORY_CLAIM_PNC` | 4 | `InsertHistoryClaimPNC` | B-1 Polis/Snapshot |
| 30 | `POOLDATA.PEGA_JSON_KLAIM_PNC` | 7 | `InsertClaimPNC` | B-1 Polis/Snapshot |
| 31 | `POOLDATA.PEGA_JSON_OS_AKSEP_KLAIM` | 7 | `InsertOSAkseptasiClaimPNC` | B-10 Akseptasi |
| 32 | `POOLDATA.PEGA_LOGJSON_LOG` | 3 | `Loging_InsertClaimPNC` | B-1 Polis/Snapshot |
| 33 | `POOLDATA.PEGA_LST_DET_TYPE_DOC` | 3 | `UpdateDetTypeDoc` | S-1 Dokumen |
| 34 | `POOLDATA.PEGA_LST_DET_TYPE_DOC_BUSINESS` | 14 | `UpdateDetTypeDocBusiness_SQL` | S-1 Dokumen |
| 35 | `POOLDATA.PEGA_LST_DOC_TYPE` | 3 | `UpdateLstDocType` | S-1 Dokumen |
| 36 | `POOLDATA.PEGA_M_BENGKEL_HE` | 3 | `UpdateBengkelHE` | F-4 Master Data |
| 37 | `POOLDATA.PEGA_M_CAUSE_OF_LOSS` | 3 | `UpdateMCauseOfLoss` | F-4 Master Data |
| 38 | `POOLDATA.PEGA_M_DOMINAN_FACTOR` | 4 | `InsertDominanfactor` | F-4 Master Data |
| 39 | `POOLDATA.PEGA_M_GROUPING_SPAREPART_HE` | 3 | `UpdateGroupingSparepartHE` | F-4 Master Data |
| 40 | `POOLDATA.PEGA_M_PANEL_HE` | 3 | `UpdatePanel_HE` | F-4 Master Data |
| 41 | `POOLDATA.PEGA_M_SPAREPART_HE` | 3 | `UpdateSparepartHE` | F-4 Master Data |
| 42 | `POOLDATA.PEGA_M_STS_CLAIM` | 3 | `UpdateStsClaim` | F-4 Master Data |
| 43 | `POOLDATA.PEGA_M_SUPPLIER` | 5 | `KonversiMasterSupplier_SQL` | F-4 Master Data |
| 44 | `POOLDATA.PEGA_M_SURVEYORS` | 3 | `UpdateMSurveyors` | B-8 Survey |
| 45 | `POOLDATA.PEGA_MST_USER_TEKNIS` | 10 | `UpdateMasterUserTeknis` | F-4 Master Data |
| 46 | `POOLDATA.PNC_INSERT_EMAIL_ADJUSTER` | 9 | `InsertEmailAdjuster_PNC` | B-8 Survey |
| 47 | `POOLDATA.PROC_JOBPERSONALACCIDENTKLAIM` | 13 | `ProcLODPersonalAccident_job` | perlu ditentukan |
| 48 | `POOLDATA.PROCINSERTDATARECIVEDKLAIM` | 27 | `Rcv_ProcInsertRecivedDocument` | B-14 Receive Doc |
| 49 | `POOLDATA.PROGRESS_CLAIM_PNC` | 16 | `InsertStatusProgress` | S-7 Progres/TAT |
| 50 | `POOLDATA.SET_ATTACHFILETEMPSALVAGE` | 10 | `SaveAttachmentToDBTemp_Sql` | B-12 Salvage |
| 51 | `POOLDATA.SET_ATTACHMENT_64BIT` | 11 | `SaveAttachmentToDB_Sql` | S-1 Dokumen |
| 52 | `POOLDATA.UPDATE_LOG_PROTEKSI` | 15 | `SaveMstProteksi_SQL` | B-13 Open Protection |
| 53 | `POOLDATA.UPDATEINSERT_PENGKINIANDATA` | 12 | `InsertPengkinianData_SQL` | B-2 Registrasi |
| 54 | `POOLDATA.UPDATEPREMIUMTEMPLATE` | 2 | `UpdateTemplatePKS` | perlu ditentukan |
| 55 | `POOLDATA.UPDATEREAS` | 7 | `UpdateEmailReas` | B-9 PLA/DLA |
| 56 | `PROCESSNEWEDMOBJECTDATA` | 3 | `ProcessNewEDMObjectData` | B-1 Polis/Snapshot |

### Function lokal

**8 function.** Dipanggil di dalam SELECT sehingga logikanya ikut menentukan hasil query —
harus ikut ditulis ulang di Go (D-02).

| No | Objek database | Param | Dipakai di rule | Modul terdampak |
|---|---|---|---|---|
| 1 | `MBU.F_VALIDASI_KLAIM_PENGKINIAN` | 3 | 1 | B-2 Registrasi |
| 2 | `POOLDATA.BASE64ENCODE` | 1 | 5 | S-1 Dokumen |
| 3 | `POOLDATA.GET_GROUPBUSINESS_XOL` | 3 | 1 | B-9 PLA/DLA |
| 4 | `POOLDATA.GET_INTERPOLASIPNC` | 1 | 4 | B-5 Settlement |
| 5 | `POOLDATA.GET_POSISI_PROGRESS_PNC` | 2 | 2 | S-7 Progres/TAT |
| 6 | `POOLDATA.GET_POSISI_PROGRESS2` | 3 | 6 | S-7 Progres/TAT |
| 7 | `POOLDATA.GETCURRENCYSTANDARD` | 2 | 2 | B-5 Settlement |
| 8 | `POOLDATA.GETSELISIHJAM` | 2 | 1 | S-7 Progres/TAT |

### Function remote via DB Link — lingkup R-03

**6 function** milik database lain. Tidak diminta ke DBA POOLDATA; digantikan API sesuai D-25.

| No | Function | DB Link | Param | Dipakai di rule |
|---|---|---|---|---|
| 1 | `DATAMINING.GET_WORKING_HOURS` | `asmd.sinarmas.co.id` | 2 | 6 |
| 2 | `GET_NAMA_AGEN` | `asmd.sinarmas.co.id` | 1 | 2 |
| 3 | `GET_NAMA_BISNIS` | `asmd.sinarmas.co.id` | 1 | 2 |
| 4 | `GET_NAMA_CABANG` | `asmd.sinarmas.co.id` | 1 | 2 |
| 5 | `GET_NAMA_CLIENT` | `asmd.sinarmas.co.id` | 1 | 2 |
| 6 | `GET_NAMA_MO` | `asmd.sinarmas.co.id` | 1 | 2 |

---

## R-04 · Router Penugasan

Seluruh router yang dirujuk oleh 4 flow, beserta status ketersediaannya di export.

| Router | Status | Dipakai oleh shape | Dampak bila hilang |
|---|---|---|---|
| `PNCAdminRouter` | **HILANG** | Register / Estimation · Register / Input Estimasi · Register / Input Register · Register / Input Estimasi | Menentukan penerima tugas |
| `PNCTeknikRouter` | **HILANG** | Register / Send To Analis · Register / Choose Surveyor · Register / Send To PIC Teknik | Menentukan penerima tugas |
| `RouterRCLDokter` | **HILANG** | Register / RCLDokter | Menentukan penerima tugas |
| `ToWorkList` | **HILANG** | Register / Analyst Doctor | Menentukan penerima tugas |
| `KomiteRouter` | Ada | Komite / KomiteRouter | — |
| `PNCAdminRouterRCV` | Ada | ReceiveDocument / InputReceiveDocument | — |
| `ToCurrentOperator` | Ada | CreateProtection / InputProtection · Register / View Polis | — |
| `ToWorkbasket` | Ada | CreateProtection / AksepProtection · Register / RCL/PUCL · Register / Investigator · Register / Compliance | — |

**4 router hilang**, tetapi hanya **3 yang benar-benar kritis**:

- `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` — **buatan sendiri (ruleset GCNMFW)**,
  berisi aturan bisnis penentuan penerima tugas. Tanpa ini, modul B-6 tidak dapat diselesaikan.
- `ToWorkList` — **bawaan Pega** (`Pega-ProcessEngine`), sama seperti `ToWorkbasket` dan
  `ToCurrentOperator` yang ada di export. Perilakunya baku: menugaskan ke worklist operator.
  **Bukan gap nyata** — tidak perlu diminta.

Sebagai pembanding, dua router buatan sendiri yang **ada** di export dapat dipakai untuk menebak
pola ketiganya:

- `KomiteRouter` (GCNMFW 01-01-50) — memilih **worklist** `komitepnc` (tanpa angka), `komitepnc2`, `komitepnc3`, `komitepnc4` berdasarkan `KomiteCount`. **Worklist, bukan workbasket** — dua model penugasan yang berbeda (`D-26`).
- `PNCAdminRouterRCV` (GCNMFW 01-01-86) — menugaskan ke `ReceiveDocument.UserAdmin`, jatuh ke
  pembuat kasus bila kosong.

> Pola keduanya sederhana: menetapkan `Param.AssignTo` berdasarkan kondisi. Kemungkinan besar
> ketiga router yang hilang juga demikian — memilih PIC berdasarkan lini bisnis, cabang, dan
> beban kerja. Tapi ini **dugaan**, bukan fakta, dan tidak boleh dipakai sebagai dasar implementasi.

---

## R-07 · Activity yang Dipanggil tapi Tidak Ada di Export

**50 activity** dipanggil oleh activity lain tetapi tidak ikut diekspor:
**45 buatan sendiri** dan **5 bawaan Pega**.

### Buatan sendiri — perlu diminta

| No | Activity | Dipanggil | Oleh | Modul terdampak |
|---|---|---|---|---|
| 1 | `SendEmailNotification` | 15× | `ASMSendsEmailAttachments`, `ASMSendsEmailAttachments_PDF`, `CNMUpdatePanelHE_act`, `InputSurveyorBCAF_PostAct`, `InputSurveyorBRIF_PostAct`, `JobSendEmailNotificationPAYDI`, `SendWorkMailReport_Act`, `SubmitTanggalLengkapTKA`, `UpdateBengkelHE_act`, `UpdateGroupingSparepartHE_act`, `UpdateKategoriSparepart2_act2`, `UpdateKategoriSparepart_act2`, `UpdateSparepartHE_act`, `UpdateTypeSparepartReject_act2`, `UpdateTypeSparepart_act2` | S-3 Notifikasi |
| 2 | `CreateWorkPage` | 5× | `CreateNewMasterSupplier_post`, `EditMasterSupplier_post`, `KomitePost_Survey`, `SetChildKomitePerAdjustment_act` | perlu ditentukan |
| 3 | `SetTicket` | 4× | `AllCoveredResolved`, `KomitePost_Adjustment`, `KomitePost_Reject`, `SetListComiteeClaimPerObjAdj` | perlu ditentukan |
| 4 | `SetKasir_Act` | 3× | `CreateCasePNC_AsKredit`, `InsertCasePNC_AsuransiKredit`, `InsertCasePNC_Kredit_PA` | B-10 Akseptasi |
| 5 | `generatePD4ML` | 2× | `CreatePDFPolisAdjuster_act`, `GenerateViewPolisClaim` | perlu ditentukan |
| 6 | `generatePDF` | 2× | `CreatePDFPolisAdjuster_act`, `GenerateViewPolisClaim` | perlu ditentukan |
| 7 | `getRandomTeam_act` | 2× | `CallActivityInputRegister`, `InsertObjekAneka` | perlu ditentukan |
| 8 | `PNCSalvageHistorySemuaKlaim` | 2× | `SetDataSalavage_act`, `SetDataSalavage_act_ASI` | B-12 Salvage |
| 9 | `SetPageErrors` | 2× | `createWorkPage`, `pzCheckFieldSecurity` | perlu ditentukan |
| 10 | `AddCoveredDefaults` | 1× | `AddCoveredWork` | perlu ditentukan |
| 11 | `addWork` | 1× | `svcAddWorkObject` | perlu ditentukan |
| 12 | `CopyCoverage_act` | 1× | `ValidateInputEstimate_act` | perlu ditentukan |
| 13 | `ExportToExcel` | 1× | `ExportProduktivitasClaim_Act` | perlu ditentukan |
| 14 | `GCNMGetInternalSurveyor_Act` | 1× | `SetDashboardClaim` | B-8 Survey |
| 15 | `GCNMGetLossAdjuster_Act` | 1× | `SetDashboardClaim` | B-8 Survey |
| 16 | `GetAttchmentBase64CsvOrFile` | 1× | `ExportHasilDataAIKlaim` | S-1 Dokumen |
| 17 | `GetEmailSenderInfo` | 1× | `SendSimpleEmail` | S-3 Notifikasi |
| 18 | `GetKomunikasiRequestDokumen_Act` | 1× | `SetDataViewKlaimRequestDoc_Act` | B-10 Akseptasi |
| 19 | `InsertDataSlinkOJKIndividu` | 1× | `InsertAdjustmentListKredit` | S-4 Integrasi |
| 20 | `InsertJsonClaimNonMBUforKomite_act` | 1× | `KomitePost_Reject` | B-7 Komite |
| 21 | `InsertToSalvageDocument` | 1× | `SetStsSalvagePNC_act` | B-12 Salvage |
| 22 | `NotificationPAYDI` | 1× | `JobSendEmailNotificationPAYDI` | S-3 Notifikasi |
| 23 | `OsAkseptasiKlaim` | 1× | `KomitePost_Reject` | B-10 Akseptasi |
| 24 | `PartyNewSetup` | 1× | `createWorkPage` | perlu ditentukan |
| 25 | `PerformFlowAction` | 1× | `pzProcessIndividualDepAssignment` | perlu ditentukan |
| 26 | `RejectedCaseSurvey_Act` | 1× | `InputSurveyorBCAF_PostAct` | B-8 Survey |
| 27 | `RemoveFromCover` | 1× | `AddCoveredWork` | perlu ditentukan |
| 28 | `ReportKPILoginAdjuster_act` | 1× | `GetReportKPIAdjuster` | B-8 Survey |
| 29 | `runReport` | 1× | `pxShowReport` | perlu ditentukan |
| 30 | `SendEmailAlertPIC` | 1× | `ValidationUploadRegister` | S-3 Notifikasi |
| 31 | `SendFilePendukungLelangKeSimasBit` | 1× | `Insert_salvageToSimasBid` | B-12 Salvage |
| 32 | `SendUpdateCIF_act` | 1× | `InputRegister_act` | B-2 Registrasi |
| 33 | `SendWorkMail_Act` | 1× | `ServiceGetProductionData` | perlu ditentukan |
| 34 | `setCauseOfLossID_act` | 1× | `CallActivityInputRegister` | perlu ditentukan |
| 35 | `SetFlagactivityleaderKomite` | 1× | `SetAssignmentKomiteAI` | B-7 Komite |
| 36 | `SetListRegistKlaimPA` | 1× | `GetReportClaimRegistList` | perlu ditentukan |
| 37 | `SetspreadingtoCoverage` | 1× | `CallActivityInputRegister` | perlu ditentukan |
| 38 | `SetSumOfOutgo_Act` | 1× | `SetSumOfTSIPremium_Act` | perlu ditentukan |
| 39 | `SetSumOfTSIPerLocationList_Act` | 1× | `SetSumOfTSIPremium_Act` | perlu ditentukan |
| 40 | `SetViewAttachmentReas` | 1× | `SetDataViewKlaimReas_Act` | B-9 PLA/DLA |
| 41 | `ShowCoverage` | 1× | `CallActivityInputRegister` | perlu ditentukan |
| 42 | `TransferToKasir_act_Leader` | 1× | `SaveApprovalAkseptasiPaymentLeader_Act` | B-10 Akseptasi |
| 43 | `UpdateStatusMasterKomitexol` | 1× | `InsertUpdateMasterXOL` | B-9 PLA/DLA |
| 44 | `UploadDocumentToGoogleStorage` | 1× | `SetStsSalvagePNC_act` | S-1 Dokumen |
| 45 | `ValidasiSisaTSI` | 1× | `SetNilaiResikoSendiri` | perlu ditentukan |

### Bawaan Pega — tidak perlu diminta

Berawalan `px`/`py`/`pz`, milik ruleset Pega. Perilakunya baku dan akan digantikan mekanisme
setara di sistem baru, bukan diporting.

| Activity | Dipanggil | Oleh |
|---|---|---|
| `pxAddChildWork` | 1× | `SetChildKomitePerAdjustment_act` |
| `pxChooseBestRuleSet` | 1× | `pxShowReport` |
| `pxTransferAssignment` | 1× | `GCNMTransferDataKlaim_act` |
| `pxUploadCSVResults` | 1× | `PNCUploadAutoClaimSlikOJK` |
| `pyDeleteAttachmentContent` | 1× | `DeleteAttachment` |


---

## Cara Meminta ke DBA

Daftar di atas dapat dikirim apa adanya. Untuk mempercepat, berikut query yang dapat dijalankan
DBA guna menarik seluruh source sekaligus.

**Query penarikan source:**

```sql
SELECT owner, name, type, line, text
  FROM all_source
 WHERE (owner, name) IN (
       ('COLLECTION', 'P_GET_DATA_REFUND'),
       ('GENERAL', 'GET_TOKEN_STORAGE'),
       ('POOLDATA', 'KONVERSIT_VEHICLELIST'),
       ('POOLDATA', 'ADD_NEWMASTERVIRTUALACCOUNT'),
       ('POOLDATA', 'CONVERTJSONPRODUCTION'),
       ('POOLDATA', 'DOCTRAVEL_CVG'),
       ('POOLDATA', 'INSERT_KPIADJUSTER'),
       ('POOLDATA', 'INSERT_PLADLA'),
       ('POOLDATA', 'INSERT_PNCCHRONOLOGYTAT'),
       ('POOLDATA', 'INSERT_SALVAGE'),
       ('POOLDATA', 'INSERT_SALVAGE_DETAILS'),
       ('POOLDATA', 'INSERT_SURVEYORLIST'),
       ('POOLDATA', 'INSERT_UPDATE_MST_XOL'),
       ('POOLDATA', 'INSERTDATAAIKLAIMPNC'),
       ('POOLDATA', 'INSERTDATAKLAIMCABANG'),
       ('POOLDATA', 'INSERTDATAKOMITELIST'),
       ('POOLDATA', 'INSERTDATASFILLINGARCHIVE'),
       ('POOLDATA', 'INSERTMASTERRECOVERYKLAIM'),
       ('POOLDATA', 'INSERTMASTERREJECTEDKOMITE'),
       ('POOLDATA', 'INSERTPOLISTOJSON'),
       ('POOLDATA', 'MASTERPENOLAKANKLAIM1'),
       ('POOLDATA', 'MASTERPENOLAKANKLAIM2'),
       ('POOLDATA', 'PEGA_CONVERT_JSONKLAIM_PNC'),
       ('POOLDATA', 'PEGA_D_CAUSE_OF_LOSS'),
       ('POOLDATA', 'PEGA_D_PASAL_MASTER'),
       ('POOLDATA', 'PEGA_D_SURVEYORS'),
       ('POOLDATA', 'PEGA_JSON_INSERT_HISTORY_CLAIM_PNC'),
       ('POOLDATA', 'PEGA_JSON_KLAIM_PNC'),
       ('POOLDATA', 'PEGA_JSON_OS_AKSEP_KLAIM'),
       ('POOLDATA', 'PEGA_LOGJSON_LOG'),
       ('POOLDATA', 'PEGA_LST_DET_TYPE_DOC'),
       ('POOLDATA', 'PEGA_LST_DET_TYPE_DOC_BUSINESS'),
       ('POOLDATA', 'PEGA_LST_DOC_TYPE'),
       ('POOLDATA', 'PEGA_M_BENGKEL_HE'),
       ('POOLDATA', 'PEGA_M_CAUSE_OF_LOSS'),
       ('POOLDATA', 'PEGA_M_DOMINAN_FACTOR'),
       ('POOLDATA', 'PEGA_M_GROUPING_SPAREPART_HE'),
       ('POOLDATA', 'PEGA_M_PANEL_HE'),
       ('POOLDATA', 'PEGA_M_SPAREPART_HE'),
       ('POOLDATA', 'PEGA_M_STS_CLAIM'),
       ('POOLDATA', 'PEGA_M_SUPPLIER'),
       ('POOLDATA', 'PEGA_M_SURVEYORS'),
       ('POOLDATA', 'PEGA_MST_USER_TEKNIS'),
       ('POOLDATA', 'PNC_INSERT_EMAIL_ADJUSTER'),
       ('POOLDATA', 'PROC_JOBPERSONALACCIDENTKLAIM'),
       ('POOLDATA', 'PROCINSERTDATARECIVEDKLAIM'),
       ('POOLDATA', 'PROGRESS_CLAIM_PNC'),
       ('POOLDATA', 'SET_ATTACHFILETEMPSALVAGE'),
       ('POOLDATA', 'SET_ATTACHMENT_64BIT'),
       ('POOLDATA', 'UPDATE_LOG_PROTEKSI'),
       ('POOLDATA', 'UPDATEINSERT_PENGKINIANDATA'),
       ('POOLDATA', 'UPDATEPREMIUMTEMPLATE'),
       ('POOLDATA', 'UPDATEREAS'),
       ('POOLDATA', 'PROCESSNEWEDMOBJECTDATA'),
       ('POOLDATA', 'INSERTOBJECTBACKUP'),
       ('POOLDATA', 'INSERTT_PERSONLIST'),
       ('MBU', 'F_VALIDASI_KLAIM_PENGKINIAN'),
       ('POOLDATA', 'BASE64ENCODE'),
       ('POOLDATA', 'GET_GROUPBUSINESS_XOL'),
       ('POOLDATA', 'GET_INTERPOLASIPNC'),
       ('POOLDATA', 'GET_POSISI_PROGRESS_PNC'),
       ('POOLDATA', 'GET_POSISI_PROGRESS2'),
       ('POOLDATA', 'GETCURRENCYSTANDARD'),
       ('POOLDATA', 'GETSELISIHJAM')
 )
 ORDER BY owner, name, type, line;
```

**Sebaran per skema:**

| Skema | Jumlah objek |
|---|---|
| `POOLDATA` | 59 |
| `(tanpa skema)` | 2 |
| `COLLECTION` | 1 |
| `GENERAL` | 1 |
| `MBU` | 1 |

> Catatan: objek `KONVERSIT_VEHICLELIST` dan `PROCESSNEWEDMOBJECTDATA`
> dipanggil **tanpa nama skema**, sehingga skemanya mengikuti user koneksi aplikasi. DBA perlu
> memastikan skema sebenarnya saat menarik source.

**Yang juga perlu diminta bersamaan** — tanpa ini source procedure saja belum cukup:

- **Definisi tabel** yang disentuh tiap procedure (R-08), agar perilaku insert/update dapat dibaca utuh.
- **Grant dan sinonim** — beberapa objek dipanggil tanpa skema, kemungkinan lewat sinonim publik.
- **Isi master `V_STS_CLAIM`** (R-06), karena beberapa procedure menulis kolom status.


---

## Cara Meminta ke Tim Pega

Bagian ini padanan **Cara Meminta ke DBA** di atas, dan `D-39` memang memintanya ada.
Isinya dapat dikirim apa adanya.

Berbeda dari permintaan ke DBA yang berupa satu kueri, permintaan ke Tim Pega selalu
berbentuk **export rule**. Dua bentuk yang dipakai:

| Bentuk | Kapan |
|---|---|
| **Export ulang berbasis Product rule** dengan opsi *include dependent rules* | untuk ±242 rule yang hilang menyeluruh (`R-16`) — ini yang utama (`D-39`) |
| **Permintaan rule bernama**, seperti di bawah | bila yang dibutuhkan sedikit dan namanya sudah pasti |

---

### Permintaan 1 — tiga jalur unggah CSV master

**Pemohon:** tim pengembang Claim PNC · **Modul terdampak:** Master Panel (MENU_ID 30) dan
**Master Sparepart**
**Diperiksa ulang:** 2026-10-03, terhadap snapshot **2.960 berkas XML**

#### Yang diminta

Tiga **Flow Action** berikut, masing-masing **beserta section form dan activity
pemrosesnya**:

| Flow Action | Dipanggil dari | Tombol di layar | Status |
|---|---|---|---|
| `PNCUploadMasterPanelCSV` | `Section/BrowsePanelHE` | "Upload Data Master Panel" | **belum ada** |

> **KOREKSI 2026-10-06 — activity pemrosesnya ADA, hanya namanya berbeda.**
> `Activity/PNCUploadMasterPanel_Act-Act.xml` dan
> `Activity/PNCUploadLokasiSisiPanel_Act-Act.xml` keduanya ada di snapshot
> (perhatikan **LokasiSisi**, bukan Lokasi). Keduanya memuat seluruh yang dibutuhkan:
> nama kolom CSV, penerjemahan kata ke sandi, dan penanganan baris anak.
>
> Yang masih belum ada hanyalah **Flow Action** beserta section formnya — dan itu hanya
> pemilih berkas, yang nilainya rendah karena bentuknya sudah ditiru dari
> `PNCUploadMasterSparepartCSV`.
>
> **Pelajaran untuk permintaan berikutnya:** cari nama Flow Action **dan** nama activity
> berpola `<nama tanpa CSV>_Act` sebelum menyatakan sebuah jalur hilang. Dua kali di
> berkas ini sebuah rule dinyatakan hilang padahal yang dicari hanya satu nama dari dua.
| `PNCUploadLokasiPanelCSV` | `Section/BrowsePanelHE` | "Upload Data Lokasi Panel" | **belum ada** |
| `PNCUploadMasterSparepartCSV` | `Section/BrowseMasterSparepartHE` | "Upload Data Master Sparepart" | ✅ **flow action + activity SUDAH masuk 2026-10-04** — sisa satu SQL rule |

Ketiganya dirujuk sebagai `<pyLocalAction>` pada section yang memanggilnya, masing-masing
dua kali — yang sparepart pada pos 46933 dan 51890.

**Ditambahkan 2026-10-03.** Yang sparepart baru ketahuan setelah pemeriksaan kelengkapan
Master Sparepart; sebelumnya tidak terdeteksi karena dicari di **harness**, yang hanya
merangkai section dan tidak memuat satu pun tombolnya.

#### Sparepart — nyaris lengkap, sisa satu SQL rule

Dua pengiriman pada **2026-10-04** menutup hampir seluruh jalur ini.

| Rule | Tipe | Kelas · Ruleset | Status |
|---|---|---|---|
| `PNCUploadMasterSparepartCSV` | Rule-Obj-FlowAction | `@baseclass` · GCNMFW 01-01-89 | ✅ **diterima** |
| `PNCUploadMasterSparepart_Act` | Rule-Obj-Activity | `@baseclass` · GCNMFW | ✅ **diterima** |
| **`ASM-FW-GCNMFW-Int-SPAREPART_HE GCNM GetSparepartFromNoSparepart`** | **Rule-Connect-SQL** | `ASM-FW-GCNMFW-Int-SPAREPART_HE` · GCNMFW | ❌ **masih diminta** |
| `GCNM UpdateSparepartHE` | Rule-RDB-SQL | `ASM-FW-GCNMFW-Int-SPAREPART_HE` | ✅ sudah ada |
| `pyCaption PNCUploadMasterSparepartCSV` | Rule-Obj-FieldValue | `@baseclass` | ❌ belum (kosmetik) |
| `pxUploadCSVResults` · `PNCSaveAttachmentToDB` | — | Pega OOTB · sudah tercatat | — jangan diminta |

Nama activity-nya dibaca dari blok `rowdata` di dalam FA yang masuk lebih dulu, bukan ditebak
dari pola penamaan — yang justru terbukti tidak konsisten (lihat di bawah).

**Flow action-nya sendiri tidak memuat satu pun informasi yang dibutuhkan.** Kata `Column`
muncul 20 kali di dalamnya dan `Template` 14 kali, tetapi seluruhnya properti **tata letak**
(`pyColumnCount`, `pyHideRightColumn`, `pyLayoutGroupTemplateType`); `<pyFileName/>` kosong.
Susunan kolom CSV, validasinya, dan perlakuan baris bentrok **seluruhnya ada di activity** —
dan activity itu kini di tangan.

**Yang terjawab begitu activity-nya dibaca** (rinciannya di
`claim-pnc/docs/keputusan-implementasi.md` §184):

| Pertanyaan | Jawaban |
|---|---|
| Apakah unggah massal **memintas persetujuan**? | **Tidak** — `APPROVAL := "0"` tanpa syarat; baris mendarat di Waiting Approval |
| Kunci upsert-nya apa? | **`NO_SPART`**, bukan Kode maupun Nama |
| ID baris baru dari mana? | sentinel `"UnknownID"` → `PEGA_M_SPAREPART_HE.prc:19` membuat `id_site \|\| lpad(seq,10,'0')` |
| Ditulis ke mana? | **`POOLDATA.M_SPAREPART_HE_BU`**, kolom **`JSONDATA`** — dokumen JSON, bukan kolom |

Baris terakhir menyambung ke penghalang DBA: `M_SPAREPART_HE_BU` adalah tabel yang memberi
**`ORA-00942`**. Ia bukan salah tulis — ia **tabel penyimpanan sebenarnya** master ini, dan
aplikasi belum diberi hak atasnya.

#### Cara menyebut rule SQL yang diminta — nama pendeknya TIDAK cukup

Connect-SQL modul ini memakai nama lengkap yang memuat **kelas di depan** dan sisipan
**`GCNM`**. Saudaranya yang sudah diterima membuktikan polanya:

| Hal | Nilai pada `UpdateSparepartHE` yang SUDAH ada |
|---|---|
| `pyRuleName` | `ASM-FW-GCNMFW-Int-SPAREPART_HE GCNM UpdateSparepartHE` |
| `pyClassName` | `ASM-FW-GCNMFW-Int-SPAREPART_HE` |
| `pyRuleSet` · versi | `GCNMFW` · `01-01-88` |
| `pzInsKey` | `RULE-CONNECT-SQL ASM-FW-GCNMFW-INT-SPAREPART_HE GCNM!UPDATESPAREPARTHE` |
| Nama berkasnya di export | `RDB List/UpdateSparepartHE-SQL.xml` |

Perhatikan **nama berkas ≠ nama rule**: berkasnya bernama pendek, rule-nya bernama panjang.
Meminta dengan nama pendek saja berisiko dilaporkan "tidak ditemukan".

Sisa yang diminta hanya **`GetSparepartFromNoSparepart`**. Kunci pencariannya sudah diketahui
(`NO_SPART`); yang belum terbaca adalah penyaring lain yang mungkin menyertainya.

#### Yang SUDAH diterima — jangan diminta lagi

| Flow Action | Berkas | Kelas · Ruleset | Activity pemrosesnya |
|---|---|---|---|
| `UploadDocument` | `Flow Action/UploadDocument-FA.xml` | `@baseclass` · GCNMFW 01-01-89 | `Activity/SaveFilePenunjang-Act.xml` ✅ |

Ia sempat tercatat hilang pada pemeriksaan **2026-09-20**, dan memang belum ada saat itu —
direktori `Flow Action/` masih berisi 29 berkas. Ia masuk pada pengiriman berikutnya.
Tombol "Upload Document" **sudah dibangun** pada 2026-10-04; yang tertahan hanya ketiga
unggahan CSV.

> **Flow action-nya saja TIDAK cukup, dan ini layak dicatat untuk permintaan berikutnya.**
> `SaveFilePenunjang` ternyata hanya **menahan** berkas di halaman sementara; yang
> menyimpannya adalah rantai `CNMUpdatePanelHE_act` → `PNCSaveAttachmentToDB` →
> `SET_ATTACHMENT_64BIT.prc` → `InsertDokumenPNC` → Connect REST `UploadDokumenPNC`
> (`POST /api/v1/upload`) → `InsertDataPNCStorage`. Isi berkasnya **tidak pernah masuk**
> `DATA_ATTACHFILE`; yang tersimpan di sana hanya metadata beserta `IMAGEID`.
>
> Artinya satu flow action unggah **selalu** menyeret rantai penyimpanan di belakangnya.
> Saat meminta ketiga CSV di atas, mintakan pula **activity pemrosesnya beserta rule yang
> dipanggil activity itu** — bukan flow action dan section-nya saja. Permintaan yang
> berhenti di flow action akan terlihat lengkap dan tetap tidak dapat dibangun.

Perhatikan kelasnya: `@baseclass`, bukan kelas modul. Itu pula sebabnya satu rule yang sama
dipanggil **13 section** lintas modul — Surveyor, Master Rekening, Bengkel, Sparepart,
Panel, dan Recovery.

#### Cara memastikan keduanya memang belum terkirim

| Pemeriksaan | Hasil |
|---|---|
| Rule ber-`pyRuleName` persis kedua nama Panel itu, di seluruh snapshot | **nol** |
| Kedua nama itu di berkas lain | **tidak ada** — hanya muncul di section yang memanggilnya |
| Isi direktori `Flow Action/` | **55 berkas** — jadi Flow Action memang ikut diekspor |
| `find . -iname "*PNCUploadMasterSparepart*"` (2026-10-04) | **1 berkas** — FA-nya saja; **activity-nya tidak ada** |

> Baris terakhir diperbarui 2026-10-04. Sebelumnya berbunyi *"tak satu pun sparepart"*, dan
> itu benar sampai flow action-nya masuk. Ia contoh langsung dari pelajaran §181: klaim
> "tidak ada di export" punya tanggal kedaluwarsa.

Butir terakhir yang menjadikan ini **gap**, bukan kategori yang tidak pernah dikirim.

#### Bentuk yang diharapkan

Padanannya sudah ada di snapshot dan dapat dipakai sebagai contoh:
**`Flow Action/PNCUploadDataKlaimSlikOJK-FA.xml`** (ruleset `GCNMFW`), yang menunjuk
activity `PNCUploadAutoClaimSlikOJK` dan `pxUploadCSVResults`.

Padanan yang **lebih dekat lagi** untuk ketiganya, karena sama-sama unggah CSV *master*:
**`Flow Action/PNCUploadMasterBengkelCSV-FA.xml`** beserta
`Activity/PNCUploadMasterBengkel_Act-Act.xml`.

> **Padanan bukan pengganti.** Keempat contoh itu memperlihatkan **bentuknya**, tetapi tidak
> satu pun menyatakan **kolom apa** yang diterima berkas CSV panel maupun sparepart,
> urutannya, dan apa yang terjadi pada baris yang bentrok dengan baris yang sudah ada.
> Karena jalur ini **menulis massal ke master yang menentukan harga**, ketiganya tetap
> diminta apa adanya — tidak disusun dari analogi.

#### Identifikasi presisi rule yang diminta

Dibaca per blok `rowdata` dari `Flow Action/PNCUploadMasterBengkelCSV-FA.xml` — bukan dari
jendela teks, yang terbukti menyerempet blok tetangga.

| Rule | Tipe | Kelas | Ruleset | Diminta? |
|---|---|---|---|---|
| `PNCUploadMasterSparepartCSV` | **Rule-Obj-FlowAction** | `@baseclass` | `GCNMFW` | **YA** |
| *activity pemrosesnya* — **namanya tidak diketahui** | **Rule-Obj-Activity** | `@baseclass` | `GCNMFW` | **YA** |
| `pyCaption PNCUploadMasterSparepartCSV` | Rule-Obj-FieldValue | `@baseclass` | `GCNMFW` | ikut |
| `pxUploadCSVResults` | Rule-HTML-Section | `@baseclass` | **Pega OOTB** | **tidak** — bawaan |

Ruleset `GCNMFW` dan versi `01-01-89` diambil dari saudara kembarnya; keduanya **dugaan
yang wajar**, bukan hasil pembacaan rule sparepart itu sendiri — rule-nya belum ada.

#### Kenapa nama activity pemrosesnya TIDAK boleh ditebak

Ketiga unggah CSV yang ada di snapshot memakai **tiga pola penamaan yang berbeda**:

| Flow Action | Activity pemrosesnya | Pola |
|---|---|---|
| `PNCUploadMasterBengkelCSV` | `PNCUploadMasterBengkel_Act` | buang `CSV`, tambah `_Act` |
| `PNCUploadDataKlaimSlikOJK` | `PNCUploadAutoClaimSlikOJK` | nama lain sama sekali |
| `PNCUploadClaimCSV` | `PNCUploadAutoClaim_Act` | sisip `Auto`, tambah `_Act` |

Tiga contoh, tiga pola. Karena itu nama activity ketiganya **tidak dicantumkan sebagai nama
yang diminta** — menuliskannya berisiko membuat Tim Pega mencari rule yang tidak ada, lalu
melaporkan "tidak ditemukan" untuk sesuatu yang sebenarnya ada dengan nama lain.

> **Terjawab untuk sparepart, 2026-10-04.** Setelah flow action-nya masuk, nama activity-nya
> terbaca langsung dari dalamnya: **`PNCUploadMasterSparepart_Act`** — kebetulan mengikuti
> pola Bengkel. Kehati-hatian di atas tetap benar saat ditulis: dua dari tiga contoh TIDAK
> mengikuti pola itu, jadi menebaknya saat itu berpeluang salah sekitar dua pertiga.
>
> Untuk **`PNCUploadMasterPanelCSV`** dan **`PNCUploadLokasiPanelCSV`** namanya masih belum
> diketahui, dan peringatan ini berlaku penuh.

**Cara yang benar:** minta Flow Action-nya **beserta dependent rules**, bukan menyebut nama
activity. Pega dapat melakukannya dalam satu langkah, dan itu persis yang `D-39` tetapkan —
export berbasis Product rule dengan opsi *include dependent rules*.

Pengiriman 2026-10-04 memperlihatkan kenapa opsi itu penting: yang masuk **hanya satu
berkas**, flow action-nya saja. Opsi *include dependent rules* tampaknya tidak dicentang,
sehingga permintaannya terlihat terpenuhi padahal bagian yang memuat seluruh logikanya
tertinggal.

Yang harus ikut terbawa oleh opsi itu:

1. Rule Flow Action-nya sendiri.
2. **Section form** yang dirujuknya (`pyHTMLStreamName`), bila ada — pada Bengkel **tidak
   ada section terpisah**, jadi ketiadaannya bukan tanda export kurang.
3. **Activity pemrosesnya**, beserta seluruh rule yang dipanggil activity itu.

#### Kenapa menebak tidak dapat diterima di sini

| Tidak diketahui | Akibat bila ditebak |
|---|---|
| **Susunan kolom CSV** | berkas diterima sistem tetapi kolomnya dipetakan ke tempat yang salah — tidak muncul sebagai galat |
| Validasi yang berjalan | nama ganda atau sandi `STS_*` yang tidak sah lolos masuk |
| **Apakah baris hasil unggah masuk antrean persetujuan** | unggah massal yang melewati `APPROVAL="0"` **memintas seluruh kontrol persetujuan** |

Butir ketiga yang paling menentukan: salah menebak di sana berarti membuat pintu masuk
data yang tidak diperiksa siapa pun.

#### Jalan pintas bila rule-nya sulit dicari

Satu **contoh berkas CSV** untuk Master Panel dan satu untuk Lokasi Panel sudah cukup
untuk membaca susunan kolomnya. Yang tetap perlu dijawab terpisah — dan tidak terbaca dari
CSV mana pun — adalah apakah baris hasil unggah masuk **Waiting Approval** atau langsung
**Approve**.

---

### Yang juga perlu diminta bersamaan

Keempatnya sudah tercatat di tempat lain pada dokumen ini, dan sebaiknya dikirim dalam
satu permintaan yang sama:

- **Export ulang berbasis Product rule** (`D-39`) — ±242 rule hilang, terberat **137 When
  rule** yang memblokir percabangan bisnis hampir semua modul (`R-16`).
- **Tiga router penugasan** `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`
  (`R-04`).
- **40 activity buatan sendiri** yang dirujuk tetapi tidak ada (`R-07`), diprioritaskan
  `SendEmailNotification`, `ValidasiSisaTSI`, `SetKasir_Act`.
- **Konfirmasi cacat proses export**: dua berkas berisi rule yang berbeda dari namanya
  (`Activity/SendEmailNotification-Act.xml` memuat `CompressImage_Act`), yang bila benar
  berarti seluruh berkas perlu divalidasi ulang lewat `pyRuleName`, bukan lewat nama
  berkas.
