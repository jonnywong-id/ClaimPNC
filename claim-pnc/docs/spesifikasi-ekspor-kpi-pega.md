# Spesifikasi berkas ekspor Report KPI — menurut Pega

> **Dibangun otomatis** dari `Activity/*.xml` oleh `docs/tools/gen-spek-ekspor.js`.
> Jangan disunting dengan tangan.

Seluruh ekspor KPI di Pega berbentuk **CSV**, dibuat `pxConvertResultsToCSV`.
Baris judulnya TIDAK diturunkan dari alias SQL melainkan ditetapkan pemanggil lewat
parameter `CSVPropHeaders`, dan kolom yang ditulis dipilih lewat `CSVProperties`.

Akibatnya dua hal yang mudah keliru:

1. **Jumlah kolom berkas tidak sama dengan jumlah alias kuerinya.** Kueri Progress
   mengembalikan 43 alias; berkasnya 32 kolom.
2. **Judul kolomnya bahasa manusia**, bukan alias. `No Klaim`, bukan `CLAIMNO`.

## `PNCReportKPIAdjuster_act` — tab KPI Adjuster

### Berkas: "LAPORAN DETAIL KPI ADJUSTER"

**35 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | ADJUSTER | `UserTeknisGroup` |
| 2 | NO CASE | `CaseID` |
| 3 | STATUS SURVEY | `StatusWork` |
| 4 | NOTE | `SubjectEmail` |
| 5 | NO CLAIM | `ClaimNo` |
| 6 | PIC | `IsPLA` |
| 7 | DATE OF LOSS | `AreaClaimId` |
| 8 | QQ NAME | `Conveyance` |
| 9 | TGL PENUNJUKAN | `ContractNo` |
| 10 | PENJADWALAN SURVEY | `CauseOfLoss` |
| 11 | SKOR SURVEY | `ProdKe` |
| 12 | TGL SURVEY | `Status` |
| 13 | ALASAN TERLAMBAT | `TelpTertanggung` |
| 14 | TGL INITIAL ADVICE | `City` |
| 15 | SKOR INITIAL ADVICE | `CityID` |
| 16 | TGL PRELIMINARY ADVICE | `RCV_ID` |
| 17 | SKOR PRELIMINARY ADVICE | `ResponseNote` |
| 18 | REQUEST INTERIM REPORT | `UserAdmin` |
| 19 | TGL INTERIM REPORT | `UserTeknis` |
| 20 | SKOR INTERIM REPORT | `ReporterName` |
| 21 | TGL UPDATE PROGRESS | `FlagASO` |
| 22 | TGL INPUT NEXT PROGRESS | `InsuredName` |
| 23 | SKOR PROGRESS | `CountryID` |
| 24 | TGL KIRIM KOMUNIKASI | `ClaimID` |
| 25 | TGL BALAS KOMUNIKASI | `ConsultantID` |
| 26 | SKOR TANGGAPAN KOMUNIKASI | `RWID` |
| 27 | TGL DOK LENGKAP | `District` |
| 28 | TGL DFR | `DistrictID` |
| 29 | STATUS KLAIM | `Email` |
| 30 | SKOR DFR | `AlasanKlaim` |
| 31 | REQUEST FINAL REPORT | `CauseOfLossID` |
| 32 | TGL UPLOAD FINAL REPORT | `CASEDB` |
| 33 | SKOR FINAL REPORT | `ProvinceID` |
| 34 | TOTAL SKOR | `Keyword` |
| 35 | KATEGORI | `IsTransferPIC` |

### Berkas: "LAPORAN SUMMARY KPI ADJUSTER"

**11 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | ADJUSTER | `UserTeknisGroup` |
| 2 | PENJADWALAN SURVEY | `ProdKe` |
| 3 | IMMEDIATE ADVICE | `CityID` |
| 4 | PRELIMINARY ADVICE | `ResponseNote` |
| 5 | INTERIM REPORT | `ReporterName` |
| 6 | UPDATE PROGRESS | `CountryID` |
| 7 | PROPOSE ADJUSTMENT | `AlasanKlaim` |
| 8 | TANGGAPAN KOMUNIKASI | `RWID` |
| 9 | FINAL REPORT | `ProvinceID` |
| 10 | NILAI | `Keyword` |
| 11 | KATEGORY | `IsTransferPIC` |

## `PNCReportKPIAdmin_Act` — tab KPI Admin

### Berkas: "Detail Data KPI PA"+" , "+Param.awal +" - "+Param.akhir

**9 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | No Klaim | `UserTeknis` |
| 2 | No Polis | `Remark` |
| 3 | Business Name | `Country` |
| 4 | Tanggal Terima Document | `ClaimAmount` |
| 5 | Register Date | `ClaimNo` |
| 6 | Tanggal Terima LOD | `BranchID` |
| 7 | Tanggal Akseptasi | `BranchName` |
| 8 | Regist Klaim PA > SLA | `City` |
| 9 | Pembayaran Klaim PA > SLA | `CityID` |

### Berkas: "Detail Data KPI "+" , "+Param.awal +" - "+Param.akhir

**7 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | No Klaim | `UserTeknis` |
| 2 | No Polis | `Remark` |
| 3 | Bisnis | `Country` |
| 4 | Status | `CountryID` |
| 5 | Tgl Registrasi | `ClaimNo` |
| 6 | Tgl Trf PIC | `ClaimAmount` |
| 7 | TAT Regist | `City` |

## `PNCReportKPI_act` — tab KPI PIC Teknik

### Berkas: "Laporan KPI "+" , "+Param.awal +" - "+Param.akhir

**6 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | PIC | `UserTeknis` |
| 2 | KETERANGAN | `Remark` |
| 3 | TOTAL DATA | `Country` |
| 4 | JUMLAH TERCAPAI | `CountryID` |
| 5 | TERCAPAI (%) | `ClaimNo` |
| 6 | NILAI | `ClaimAmount` |

## `EksportDataAllKPIPICKlaim` — tab KPI PIC Teknik - Pilih Data KPI

### Berkas: "Data KPI Progress Klaim"+Param.awal +" - "+Param.akhir + " "+ "NonMBU"

**32 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | No Klaim | `CaseID` |
| 2 | No Polis | `ClaimNo` |
| 3 | Nama tertanggung | `LossCoverage` |
| 4 | Nama bisnis | `AlasanDokterRejectRCL` |
| 5 | Sumber bisnis | `DollarCurrencyVal` |
| 6 | Klaim Leader/Member | `NamaDokterRCL` |
| 7 | Tahun Of Loss | `CauseOfLoss` |
| 8 | Tgl kejadian | `DaftarObjek` |
| 9 | Tgl terima dokumen | `CommentKomiteClosecase` |
| 10 | Tgl Registrasi | `City` |
| 11 | Tgl selesai registrasi | `Location` |
| 12 | Tgl tf PLA | `InsuredRelationshipOthers` |
| 13 | Lama proses regis-tf ke teknik | `CloseClaimNote` |
| 14 | Tgl permintaan kelengkapan dokumen | `isComplianceTransfer` |
| 15 | Tgl Survey | `UserAdmin` |
| 16 | Status klaim | `AnalystDoctorRemaks` |
| 17 | PIC Teknik | `NoKTP` |
| 18 | Tgl tutup kaim | `ExGratiaNote` |
| 19 | ctt tutup kaim | `NIK` |
| 20 | Alasan keterlambatan | `Remark` |
| 21 | Tgl Trf PIC | `NewNoKTP` |
| 22 | PIC Admin | `pyLabel` |
| 23 | Tgl Dok Lengkap | `pyGroup` |
| 24 | Tgl Trf Komite | `pyCountryName` |
| 25 | Tgl Trf Kasir | `NewEmail` |
| 26 | Report Date | `CASEDB` |
| 27 | Tgl Input Progress | `DISC` |
| 28 | Next Follow UP | `DISC2` |
| 29 | Status Tercapai | `ASMCityId` |
| 30 | Status Prog 1 | `AidaAlamat` |
| 31 | Status Prog 2 | `AidaNama` |
| 32 | Keterangan | `CARI6` |

### Berkas: "Data KPI SLA Klaim"+Param.awal +" - "+Param.akhir + " "+ "NonMBU"

**27 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | No Klaim | `CaseID` |
| 2 | No Polis | `ClaimNo` |
| 3 | Nama tertanggung | `LossCoverage` |
| 4 | Nama bisnis | `AlasanDokterRejectRCL` |
| 5 | Sumber bisnis | `DollarCurrencyVal` |
| 6 | Klaim Leader/Member | `NamaDokterRCL` |
| 7 | Tahun Of Loss | `CauseOfLoss` |
| 8 | Tgl kejadian | `DaftarObjek` |
| 9 | Tgl terima dokumen | `CommentKomiteClosecase` |
| 10 | Tgl Registrasi | `City` |
| 11 | Tgl selesai registrasi | `Location` |
| 12 | Tgl tf PLA | `InsuredRelationshipOthers` |
| 13 | Lama proses regis-tf ke teknik | `CloseClaimNote` |
| 14 | Tgl permintaan kelengkapan dokumen | `isComplianceTransfer` |
| 15 | Tgl Survey | `UserAdmin` |
| 16 | Status klaim | `AnalystDoctorRemaks` |
| 17 | PIC Teknik | `NoKTP` |
| 18 | Tgl tutup kaim | `ExGratiaNote` |
| 19 | ctt tutup kaim | `NIK` |
| 20 | Alasan keterlambatan | `Remark` |
| 21 | Tgl Trf PIC | `NewNoKTP` |
| 22 | PIC Admin | `pyLabel` |
| 23 | Tgl Dok Lengkap | `pyGroup` |
| 24 | Tgl Trf Komite | `pyCountryName` |
| 25 | Tgl Trf Kasir | `NewEmail` |
| 26 | Report Date | `CASEDB` |
| 27 | Status SLA | `DISC` |

### Berkas: "Data KPI Akseptasi Klaim "+Param.awal +" - "+Param.akhir + " "+ "NonMBU"

**40 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | No Klaim | `CaseID` |
| 2 | No Polis | `ClaimNo` |
| 3 | Nama tertanggung | `LossCoverage` |
| 4 | Nama bisnis | `AlasanDokterRejectRCL` |
| 5 | Sumber bisnis | `DollarCurrencyVal` |
| 6 | Klaim Leader/Member | `NamaDokterRCL` |
| 7 | Nomor Akseptasi | `IsAnalisTransfer` |
| 8 | Tahun Of Loss | `CauseOfLoss` |
| 9 | Tgl kejadian | `DaftarObjek` |
| 10 | Tgl terima dokumen | `CommentKomiteClosecase` |
| 11 | Tgl Registrasi | `City` |
| 12 | Tgl selesai registrasi | `Location` |
| 13 | Tgl tf PLA | `InsuredRelationshipOthers` |
| 14 | Lama proses regis-tf ke teknik | `CloseClaimNote` |
| 15 | Tgl permintaan kelengkapan dokumen | `isComplianceTransfer` |
| 16 | Tgl Survey | `UserAdmin` |
| 17 | Tgl aksep komite | `AnaylstRemarks` |
| 18 | Tgl Selesai Cek Komite | `LokasiSurveyor` |
| 19 | Tgl krm propose adjustment | `CountryID` |
| 20 | Tgl Terima LOD | `ResponseNote` |
| 21 | Tgl Akseptasi | `ComplianceRemark` |
| 22 | Tgl akseptasi manual | `IsTransferAnalisator` |
| 23 | Tgl DLA | `RCV_ID` |
| 24 | Tgl bayar | `UserBusinessPA` |
| 25 | Status Bayar | `DistrictID` |
| 26 | Lama regis - tanggal trf kasir | `CityID` |
| 27 | Lama aksep - trf kasir | `OccupationCode` |
| 28 | Lama akseptasi - tanggal bayar | `Conveyance` |
| 29 | Status klaim | `AnalystDoctorRemaks` |
| 30 | PIC Teknik | `NoKTP` |
| 31 | Tgl tutup kaim | `ExGratiaNote` |
| 32 | ctt tutup kaim | `NIK` |
| 33 | Alasan keterlambatan | `Remark` |
| 34 | Tgl Trf PIC | `NewNoKTP` |
| 35 | PIC Admin | `pyLabel` |
| 36 | Tgl Dok Lengkap | `pyGroup` |
| 37 | Tgl Trf Komite | `pyCountryName` |
| 38 | Tgl Trf Kasir | `NewEmail` |
| 39 | Report Date | `CASEDB` |
| 40 | Status Akseptasi | `DISC` |

### Berkas: "Data KPI Analisa Klaim "+Param.awal +" - "+Param.akhir + " "+ "NonMBU"

**40 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | No Klaim | `CaseID` |
| 2 | No Polis | `ClaimNo` |
| 3 | Nama tertanggung | `LossCoverage` |
| 4 | Nama bisnis | `AlasanDokterRejectRCL` |
| 5 | Sumber bisnis | `DollarCurrencyVal` |
| 6 | Klaim Leader/Member | `NamaDokterRCL` |
| 7 | Nomor Akseptasi | `IsAnalisTransfer` |
| 8 | Tahun Of Loss | `CauseOfLoss` |
| 9 | Tgl kejadian | `DaftarObjek` |
| 10 | Tgl terima dokumen | `CommentKomiteClosecase` |
| 11 | Tgl Registrasi | `City` |
| 12 | Tgl selesai registrasi | `Location` |
| 13 | Tgl tf PLA | `InsuredRelationshipOthers` |
| 14 | Lama proses regis-tf ke teknik | `CloseClaimNote` |
| 15 | Tgl permintaan kelengkapan dokumen | `isComplianceTransfer` |
| 16 | Tgl Survey | `UserAdmin` |
| 17 | Tgl aksep komite | `AnaylstRemarks` |
| 18 | Tgl Selesai Cek Komite | `LokasiSurveyor` |
| 19 | Tgl krm propose adjustment | `CountryID` |
| 20 | Tgl Terima LOD | `ResponseNote` |
| 21 | Tgl Akseptasi | `ComplianceRemark` |
| 22 | Tgl akseptasi manual | `IsTransferAnalisator` |
| 23 | Tgl DLA | `RCV_ID` |
| 24 | Tgl bayar | `UserBusinessPA` |
| 25 | Status Bayar | `DistrictID` |
| 26 | Lama regis - tanggal trf kasir | `CityID` |
| 27 | Lama aksep - trf kasir | `OccupationCode` |
| 28 | Lama akseptasi - tanggal bayar | `Conveyance` |
| 29 | Status klaim | `AnalystDoctorRemaks` |
| 30 | PIC Teknik | `NoKTP` |
| 31 | Tgl tutup kaim | `ExGratiaNote` |
| 32 | ctt tutup kaim | `NIK` |
| 33 | Alasan keterlambatan | `Remark` |
| 34 | Tgl Trf PIC | `NewNoKTP` |
| 35 | PIC Admin | `pyLabel` |
| 36 | Tgl Dok Lengkap | `pyGroup` |
| 37 | Tgl Trf Komite | `pyCountryName` |
| 38 | Tgl Trf Kasir | `NewEmail` |
| 39 | Report Date | `CASEDB` |
| 40 | Status Analisa | `DISC` |

## `ExportKPILoginAdjuster` — KPI Login Adjuster (di luar lingkup layar ini)

### Berkas: "LAPORAN DETAIL KPI ADJUSTER"

**34 kolom.**

| # | Judul kolom | Properti sumber |
|---:|---|---|
| 1 | TIPE | `TKI` |
| 2 | NO CASE | `CaseID` |
| 3 | STATUS SURVEY | `StatusWork` |
| 4 | NOTE | `SubjectEmail` |
| 5 | NO CLAIM | `ClaimNo` |
| 6 | PIC | `IsPLA` |
| 7 | DATE OF LOSS | `AreaClaimId` |
| 8 | QQ NAME | `Conveyance` |
| 9 | TGL PENUNJUKAN | `ContractNo` |
| 10 | TGL SURVEY | `CauseOfLoss` |
| 11 | ALASAN TERLAMBAT | `TelpTertanggung` |
| 12 | SKOR SURVEY | `ProdKe` |
| 13 | TGL INITIAL ADVICE | `City` |
| 14 | SKOR INITIAL ADVICE | `CityID` |
| 15 | TGL PRELIMINARY ADVICE | `RCV_ID` |
| 16 | SKOR PRELIMINARY ADVICE | `ResponseNote` |
| 17 | REQUEST INTERIM REPORT | `UserAdmin` |
| 18 | TGL INTERIM REPORT | `UserTeknis` |
| 19 | SKOR INTERIM REPORT | `ReporterName` |
| 20 | TGL UPDATE PROGRESS | `FlagASO` |
| 21 | TGL INPUT NEXT PROGRESS | `InsuredName` |
| 22 | SKOR PROGRESS | `CountryID` |
| 23 | TGL KIRIM KOMUNIKASI | `ClaimID` |
| 24 | TGL BALAS KOMUNIKASI | `ConsultantID` |
| 25 | SKOR TANGGAPAN KOMUNIKASI | `RWID` |
| 26 | TGL DOK LENGKAP | `District` |
| 27 | TGL DFR | `DistrictID` |
| 28 | STATUS KLAIM | `Email` |
| 29 | SKOR DFR | `AlasanKlaim` |
| 30 | REQUEST FINAL REPORT | `CauseOfLossID` |
| 31 | TGL UPLOAD FINAL REPORT | `CASEDB` |
| 32 | SKOR FINAL REPORT | `ProvinceID` |
| 33 | TOTAL SKOR | `Keyword` |
| 34 | KATEGORI | `IsTransferPIC` |

