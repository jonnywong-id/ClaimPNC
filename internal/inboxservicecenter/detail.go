package inboxservicecenter

import (
	"strings"
	"time"
)

// ClaimDetail adalah satu klaim portal rekanan beserta SELURUH isian layar rinciannya.
//
// # Dari mana bentuknya
//
// Dari dua tempat yang saling menguatkan:
//
//	Section/InputClaimServiceCenter-Section.xml    ±85 isian, terbagi tujuh kelompok
//	RDB List/GetDataServiceCenter_Update-SQL.xml   peta kolom -> alias untuk seluruhnya
//	Database/PEGA_PORTAL_REKANAN.prc               daftar kolom yang benar-benar ditulis
//
// Procedure-nya menjadi penengah ketika kedua yang pertama berselisih: ia menyebutkan kolom
// apa adanya, tanpa alias.
//
// # Kenapa isiannya sebanyak ini
//
// Karena layar Pega memang sebanyak itu. `D-13` menetapkan tampilan meniru Pega, dan memangkas
// isian berarti petugas kehilangan data yang hari ini ia isi. Yang TIDAK dibawa hanyalah dua
// isian hasil pencarian email operator (`UserTeknisEmail`, `UserAdmin` pada kueri lama) —
// keduanya bukan data klaim melainkan pencarian ke `DATAPEGA.PR_OPERATORS` untuk keperluan
// kirim surel, dan surelnya sendiri belum dibangun.
//
// # Peringatan alias: `DistrictID` dipakai DUA KALI
//
// Kueri lama mengaliaskan `OTHER_FEE` dan `CANCELLED_REASON` ke nama yang sama,
// `"DistrictID"`. Pada klipboard Pega yang belakangan menimpa yang duluan. Di sini keduanya
// menjadi isian terpisah — `OtherFee` dan `CancelReason` — sehingga tidak ada yang hilang.
// Itu perbedaan yang DISENGAJA terhadap perilaku lama, dan dicatat di Limitations.
type ClaimDetail struct {
	// ── GENERAL INFORMATION ────────────────────────────────────────────────

	ID           string // ID
	RepairID     string // REPAIRID          — "REPAIR ID"
	ClaimNumber  string // CLAIMNO
	Type         string // TYPE              — 'SC' atau 'BROKER'
	Owner        string // LOGIN             — "Username"
	InputDate    *time.Time
	PolicyNumber string // NOPOLIS           — "NO. POLIS"
	Insurance    string // INSURANCE         — "ASURANSI"

	// WarrantyStart dan WarrantyEnd — "ACTIVE DATE WARRANTY" dan "EXPIRE DATE WARRANTY".
	//
	// Kolomnya bernama `STARTDATE`/`ENDDATE` dan dialiaskan `StartDateTreaty`/`EndDateTreaty`
	// — nama yang menyebut TREATY REASURANSI, padahal isinya masa berlaku garansi perangkat.
	WarrantyStart *time.Time
	WarrantyEnd   *time.Time

	CustomerName   string // CUST_NAME       — "Customer Name"
	InsuredName    string // QQNAME          — nama nasabah pada polis
	CustomerPhone  string // NOHP            — "NO. HP NASABAH"
	IDCardNumber   string // NOKTP           — "NO KTP"
	PrincipalBill  string // PRINCIPAL_BILL_NO
	InsuranceBill  string // INSURANCE_BILL_NO — "NO. INSURANCE BILL"
	QuotationNo    string // QUOTATIONNO     — "NO. QUOTATION"
	QuotationValue string // QUOTATION_AMOUNT — "QUOTATION AMOUNT"

	// ── INFORMASI UNIT ─────────────────────────────────────────────────────

	ProductGroup    string // PROD_GROUP     — "JENIS BARANG"
	ProductCategory string // PROD_CATEGORY  — "KATEGORI BARANG"
	Brand           string // BRAND
	Model           string // MODEL
	Colour          string // COLOUR         — "Color"
	Device          string // DEVICE
	IMEI            string // IMEI           — "NO. IMEI"
	SerialNumber    string // SERIALNO       — "NO. SERIAL"
	ItemWarranty    string // ITEMWARRANTY   — "ITEM WARRANTY"
	ObjectName      string // OBJECT

	// ── INFORMASI PERBAIKAN ────────────────────────────────────────────────

	CollectPoint  string // COLLECT_POINT
	RepairPoint   string // REPAIR_POINT
	IsDelivery    string // IS_DELIVERY
	SymptomCode   string // SYMPTOM_CODE
	SymptomDesc   string // SYMPTOM_DESC    — "SYMPTOM DESCRIPTION"
	Analysis      string // ANALISA         — "ANALISA KERUSAKAN" / "TECHNICAL ANALYSIS"
	TechnicalPIC  string // PIC             — "NAMA PIC"
	RepairStatus  string // STATUS          — "REPAIR STATUS", kode 1..9
	Reason        string // REASON          — "ALASAN"
	CancelReason  string // CANCELLED_REASON — "ALASAN BATAL"
	CustomerReply string // CUST_ARRIVAL    — "CUSTOMER ARRIVAL"

	// ── ESTIMASI DATE ──────────────────────────────────────────────────────

	AcknowledgeDate   *time.Time // ACKNOWLEDGE_DATE
	AssignedDate      *time.Time // ASSIGNED_DATE       — "TANGGAL PENGECEKAN"
	CompletedDate     *time.Time // COMPLETED_DATE      — "TANGGAL SELESAI PERBAIKAN"
	ReleaseDate       *time.Time // RELEASE_DATE
	InvoiceDate       *time.Time // INVOICE_DATE
	EstimatedPickupAt *time.Time // ESTIMATED_PICKUP_DATE — "TANGGAL PICK UP"
	CourierPickupAt   *time.Time // PICKUP_COURIER_DATE   — "TANGGAL KURIR PICK UP"
	DownPaymentValue  string     // DOWNPAYMENT
	DownPaymentNo     string     // DP_NO             — "NO. DOWNPAYMENT"
	DownPaymentMethod string     // DP_METHOD         — "DOWNPAYMENT METHOD"
	ApprovalStatus    string     // STS_APPROVAL      — "STATUS APPROVAL"
	ApprovalRemark    string     // REMARK            — "REMARK APPROVAL"
	CommitteeApprover string     // KOMITEAPPROVE     — "KOMITE APPROVAL"
	PartDetailJSON    string     // DETAILPART        — rincian sparepart dalam JSON
	AccessoriesOther  string     // ACCESSORIESLAINYA — "AKSESORIS LAINYA"

	// ── ESTIMASI BIAYA ─────────────────────────────────────────────────────
	//
	// Tujuh komponen biaya, masing-masing BERPASANGAN: nilai estimasi dan nilai yang
	// disetujui komite. Kolom "…APPROVE" itulah isi kelompok "HARGA APPROVE KOMITE".
	//
	// Seluruhnya teks, bukan angka — lihat catatan di ujung berkas ini.

	ServiceFee         string // SERVICE_FEE          — "BIAYA JASA"
	SparepartFee       string // SPAREPART_FEE        — "BIAYA SPAREPART"
	SparepartValue     string // SUKUCADANG           — "BIAYA SUKU CADANG"
	SparepartValueAppr string // SUKUCADANGAPPROVE
	TaxFee             string // TAX_FEE              — "BIAYA PAJAK"
	TaxFeeAppr         string // TAX_FEEAPPROVE
	VAT                string // PPN
	VATAppr            string // PPNAPPROVE
	DeliveryFee        string // DELIVERY_FEE         — "BIAYA PENGIRIMAN"
	DeliveryFeeAppr    string // DELIVERY_FEEAPPROVE
	OtherFee           string // OTHER_FEE            — "BIAYA LAINNYA"
	Excess             string // EXCESS               — "EXCESS DIBAYAR CUSTOMER"
	ExcessAppr         string // EXCESSAPPROVE
	Deductible         string // DEDUCTIBLE
	DeductibleAppr     string // DEDUCAPPROVE
	TotalFee           string // TOTAL_FEE            — "TOTAL BIAYA"
	TotalFeeAppr       string // TOTAL_FEEAPPROVE

	// ── ACCESSORIES UNIT ───────────────────────────────────────────────────
	//
	// Sembilan penanda kelengkapan yang ikut diserahkan pelanggan bersama perangkatnya.
	//
	// Nama alias lamanya tidak ada hubungannya sama sekali dengan isinya — `HEADSET`
	// dialiaskan `IsPLA`, `BATTERY` menjadi `IsReservedClaim`, `SIMCARD` menjadi
	// `IsSendPremi`. Ketiganya nama properti alur klaim PNC yang dipakai ulang.

	ChargerAdaptor   string // DESKCHARGER      — "CHARGER ADAPTOR"
	ChargerCable     string // CGARGERCABLE     — "CHARGER CABLE" (salah ketik di kolomnya)
	CarKit           string // CARKIT
	RemovableAntenna string // REMOVABLEANTENNA
	Headset          string // HEADSET
	Battery          string // BATTERY
	SimCard          string // SIMCARD          — "SIM CARD"
	ExternalCover    string // EXTRACOVER       — "EXTERNAL COVER"
	BatteryCover     string // BATTERYCOVER
	LCDText          string // LCD_TEXT
	UnitCase         string // CASE
	BoxUnit          string // BOXUNIT          — "BOX UNIT"
}

// RepairStatusName adalah label status perbaikan baris ini.
func (d ClaimDetail) RepairStatusName() string {
	return RepairStatusLabel(d.RepairStatus)
}

// ApprovalStatusName adalah label status persetujuan baris ini.
func (d ClaimDetail) ApprovalStatusName() string {
	return ApprovalStatusLabel(d.ApprovalStatus)
}

// ProgressNote adalah satu catatan progres pada riwayat klaim.
//
// Sumbernya `RDB List/GetListDataServiceCenter-SQL.xml`, yang membaca
// `POOLDATA.PROGRESS_SERVICECENTER_CLAIM` dikunci `REPAIRID` dan diurutkan `INSERTDATE ASC`
// — terlama di atas, seperti riwayat pada umumnya.
type ProgressNote struct {
	// ClaimID adalah `REPAIRID`, dialiaskan `"CaseID"` pada kueri lama.
	ClaimID string

	// RecordedAt adalah `INSERTDATE`, dialiaskan `"DateReceived"`.
	RecordedAt *time.Time

	// Note adalah `NOTEPROGRESS`, dialiaskan `"Remark"`.
	Note string

	// RecordedBy adalah `USERUPDATE`, dialiaskan `"UserName"`.
	RecordedBy string
}

// FieldGroup adalah satu kelompok isian pada layar rincian.
//
// Kelompoknya dibaca dari judul kontainer di `Section/InputClaimServiceCenter-Section.xml`,
// termasuk yang judulnya tertulis sebagai potongan HTML (`<h3><u> INFORMASI UNIT</u></h3>`) —
// di sini diambil teksnya saja.
type FieldGroup struct {
	// Code adalah nama kelompok pada kontrak API.
	Code string

	// Title adalah judul yang dibaca pengguna, mengikuti Pega (`D-13`).
	Title string

	// Fields adalah isian milik kelompok ini, berurutan seperti di Pega.
	//
	// Judulnya ikut dikirim — bukan hanya kuncinya — dengan alasan yang sama seperti kolom
	// grid: judul itu hasil pembacaan `pyCaption` di export, dan tempat pembacaan itu
	// tercatat adalah backend. Menyalinnya ke layar berarti 83 judul hidup di dua tempat.
	Fields []Column
}

// Kode kelompok isian.
const (
	GroupGeneral      = "general"
	GroupUnit         = "unit"
	GroupRepair       = "perbaikan"
	GroupDates        = "tanggal"
	GroupCost         = "biaya"
	GroupCostApproved = "biaya_approve"
	GroupAccessories  = "aksesoris"
)

// detailGroups adalah ketujuh kelompok beserta isinya.
//
// Kelompok "Detail Part" TIDAK ada di sini: ia bukan kumpulan isian melainkan GRID tersendiri
// yang isinya tersimpan sebagai JSON di kolom `DETAILPART`. Ia belum dibangun — lihat
// Limitations.
var detailGroups = []FieldGroup{
	{
		Code:  GroupGeneral,
		Title: "General Information",
		Fields: []Column{
			{Key: "id", Title: "ID"},
			{Key: "repair_id", Title: "REPAIR ID"},
			{Key: "no_klaim", Title: "No Klaim"},
			{Key: "tipe", Title: "Tipe"},
			{Key: "username", Title: "Username"},
			{Key: "tanggal_input", Title: "Input Date"},
			{Key: "no_polis", Title: "NO. POLIS"},
			{Key: "asuransi", Title: "ASURANSI"},
			{Key: "active_date_warranty", Title: "ACTIVE DATE WARRANTY"},
			{Key: "expire_date_warranty", Title: "EXPIRE DATE WARRANTY"},
			{Key: "customer_name", Title: "Customer Name"},
			{Key: "nasabah", Title: "Nasabah"},
			{Key: "no_hp_nasabah", Title: "NO. HP NASABAH"},
			{Key: "no_ktp", Title: "NO KTP"},
			{Key: "principal_bill_no", Title: "Principal Bill No"},
			{Key: "no_insurance_bill", Title: "NO. INSURANCE BILL"},
			{Key: "no_quotation", Title: "NO. QUOTATION"},
			{Key: "quotation_amount", Title: "QUOTATION AMOUNT"},
		},
	},
	{
		Code:  GroupUnit,
		Title: "Informasi Unit",
		Fields: []Column{
			{Key: "jenis_barang", Title: "JENIS BARANG"},
			{Key: "kategori_barang", Title: "KATEGORI BARANG"},
			{Key: "brand", Title: "BRAND"},
			{Key: "model", Title: "MODEL"},
			{Key: "color", Title: "Color"},
			{Key: "device", Title: "DEVICE"},
			{Key: "no_imei", Title: "NO. IMEI"},
			{Key: "no_serial", Title: "NO. SERIAL"},
			{Key: "item_warranty", Title: "ITEM WARRANTY"},
			{Key: "object_name", Title: "Object Name"},
		},
	},
	{
		Code:  GroupRepair,
		Title: "Informasi Perbaikan",
		Fields: []Column{
			{Key: "collect_point", Title: "COLLECT POINT"},
			{Key: "repair_point", Title: "REPAIR POINT"},
			{Key: "is_delivery", Title: "Is Delivery"},
			{Key: "symptom_code", Title: "SYMPTOM CODE"},
			{Key: "symptom_description", Title: "SYMPTOM DESCRIPTION"},
			{Key: "analisa_kerusakan", Title: "ANALISA KERUSAKAN"},
			{Key: "nama_pic", Title: "NAMA PIC"},
			{Key: "repair_status_label", Title: "REPAIR STATUS"},
			{Key: "alasan", Title: "ALASAN"},
			{Key: "alasan_batal", Title: "ALASAN BATAL"},
			{Key: "customer_arrival", Title: "CUSTOMER ARRIVAL"},
		},
	},
	{
		Code:  GroupDates,
		Title: "Estimasi Date",
		Fields: []Column{
			{Key: "acknowledge_date", Title: "ACKNOWLEDGE DATE"},
			{Key: "tanggal_pengecekan", Title: "TANGGAL PENGECEKAN"},
			{Key: "tanggal_selesai_perbaikan", Title: "TANGGAL SELESAI PERBAIKAN"},
			{Key: "tanggal_pick_up", Title: "TANGGAL PICK UP"},
			{Key: "tanggal_kurir_pick_up", Title: "TANGGAL KURIR PICK UP"},
			{Key: "release_date", Title: "Release Date"},
			{Key: "invoice_date", Title: "INVOICE DATE"},
			{Key: "downpayment", Title: "DOWNPAYMENT"},
			{Key: "no_downpayment", Title: "NO. DOWNPAYMENT"},
			{Key: "downpayment_method", Title: "DOWNPAYMENT METHOD"},
			{Key: "status_approval_label", Title: "STATUS APPROVAL"},
			{Key: "remark_approval", Title: "REMARK APPROVAL"},
			{Key: "komite_approval", Title: "KOMITE APPROVAL"},
			{Key: "aksesoris_lainya", Title: "AKSESORIS LAINYA"},
		},
	},
	{
		Code:  GroupCost,
		Title: "Estimasi Biaya",
		Fields: []Column{
			{Key: "biaya_jasa", Title: "BIAYA JASA"},
			{Key: "biaya_sparepart", Title: "BIAYA SPAREPART"},
			{Key: "biaya_suku_cadang", Title: "BIAYA SUKU CADANG"},
			{Key: "biaya_pajak", Title: "BIAYA PAJAK"},
			{Key: "ppn", Title: "PPN"},
			{Key: "biaya_pengiriman", Title: "BIAYA PENGIRIMAN"},
			{Key: "biaya_lainnya", Title: "BIAYA LAINNYA"},
			{Key: "excess", Title: "EXCESS DIBAYAR CUSTOMER"},
			{Key: "deductible", Title: "DEDUCTIBLE"},
			{Key: "total_biaya", Title: "TOTAL BIAYA"},
		},
	},
	{
		Code:  GroupCostApproved,
		Title: "Harga Approve Komite",
		Fields: []Column{
			{Key: "biaya_suku_cadang_approve", Title: "BIAYA SUKU CADANG"},
			{Key: "biaya_pajak_approve", Title: "BIAYA PAJAK"},
			{Key: "ppn_approve", Title: "PPN"},
			{Key: "biaya_pengiriman_approve", Title: "BIAYA PENGIRIMAN"},
			{Key: "excess_approve", Title: "EXCESS DIBAYAR CUSTOMER"},
			{Key: "deductible_approve", Title: "DEDUCTIBLE"},
			{Key: "total_biaya_approve", Title: "TOTAL BIAYA"},
		},
	},
	{
		Code:  GroupAccessories,
		Title: "Accessories Unit",
		Fields: []Column{
			{Key: "charger_adaptor", Title: "CHARGER ADAPTOR"},
			{Key: "charger_cable", Title: "CHARGER CABLE"},
			{Key: "car_kit", Title: "CAR KIT"},
			{Key: "removable_antenna", Title: "Removable Antenna"},
			{Key: "headset", Title: "HEADSET"},
			{Key: "battery", Title: "BATTERY"},
			{Key: "sim_card", Title: "SIM CARD"},
			{Key: "external_cover", Title: "EXTERNAL COVER"},
			{Key: "battery_cover", Title: "Battery Cover"},
			{Key: "lcd_text", Title: "LCD Text"},
			{Key: "case", Title: "CASE"},
			{Key: "box_unit", Title: "BOX UNIT"},
		},
	},
}

// DetailGroups mengembalikan ketujuh kelompok isian layar rincian.
//
// Salinan sampai ke daftar isiannya, dengan alasan yang sama seperti copyTab: // adalah senarai, dan salinan dangkal masih berbagi lariknya.
func DetailGroups() []FieldGroup {
	result := make([]FieldGroup, 0, len(detailGroups))
	for _, group := range detailGroups {
		clone := group
		clone.Fields = make([]Column, len(group.Fields))
		copy(clone.Fields, group.Fields)
		result = append(result, clone)
	}
	return result
}

// DetailQuery adalah permintaan satu rincian klaim yang sudah tervalidasi.
type DetailQuery struct {
	// ID adalah `T_KLAIM_PORTAL_REKANAN.ID`.
	ID string

	// Caller adalah identitas pemanggil.
	//
	// Rincian DISARING menurut `PIC`-nya, sama seperti daftar. Lihat NewDetailQuery.
	Caller Caller
}

// NewDetailQuery membentuk permintaan rincian yang sah, atau menyatakan apa yang salah.
//
// # Kenapa rincian ikut disaring menurut PIC
//
// Karena tanpa itu, `ID` dapat ditebak. Daftar sudah dipersempit ke klaim milik pemanggil,
// tetapi rincian dipanggil dengan ID pada jalurnya — dan ID di tabel ini berurutan. Membuka
// rincian tanpa penyaring berarti siapa pun yang sudah masuk dapat membaca nama nasabah,
// nomor IMEI, dan nilai klaim milik petugas lain hanya dengan menaikkan angkanya.
//
// Kueri lama tidak menyaringnya karena di Pega rincian hanya dapat dicapai lewat klik pada
// baris yang SUDAH tersaring; pada API yang dapat dipanggil langsung, jaminan itu hilang.
// Ini perbedaan yang DISENGAJA dan dicatat di Limitations.
func NewDetailQuery(id string, caller Caller) (DetailQuery, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return DetailQuery{}, ErrCallerUnknown
	}

	clean := strings.TrimSpace(id)
	if clean == "" {
		return DetailQuery{}, NewValidationError([]Violation{{
			Field:   FieldID,
			Message: "ID klaim wajib diisi.",
		}})
	}

	return DetailQuery{ID: clean, Caller: cleanCaller}, nil
}
