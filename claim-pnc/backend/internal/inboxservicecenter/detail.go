package inboxservicecenter

import (
	"strings"
	"time"

	"claim-pnc/internal/platform/tabletext"
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
		Fields: tabletext.Rows[Column](`
			Key                  | Title
			id                   | ID
			repair_id            | REPAIR ID
			no_klaim             | No Klaim
			tipe                 | Tipe
			username             | Username
			tanggal_input        | Input Date
			no_polis             | NO. POLIS
			asuransi             | ASURANSI
			active_date_warranty | ACTIVE DATE WARRANTY
			expire_date_warranty | EXPIRE DATE WARRANTY
			customer_name        | Customer Name
			nasabah              | Nasabah
			no_hp_nasabah        | NO. HP NASABAH
			no_ktp               | NO KTP
			principal_bill_no    | Principal Bill No
			no_insurance_bill    | NO. INSURANCE BILL
			no_quotation         | NO. QUOTATION
			quotation_amount     | QUOTATION AMOUNT
		`),
	},
	{
		Code:  GroupUnit,
		Title: "Informasi Unit",
		Fields: tabletext.Rows[Column](`
			Key             | Title
			jenis_barang    | JENIS BARANG
			kategori_barang | KATEGORI BARANG
			brand           | BRAND
			model           | MODEL
			color           | Color
			device          | DEVICE
			no_imei         | NO. IMEI
			no_serial       | NO. SERIAL
			item_warranty   | ITEM WARRANTY
			object_name     | Object Name
		`),
	},
	{
		Code:  GroupRepair,
		Title: "Informasi Perbaikan",
		Fields: tabletext.Rows[Column](`
			Key                 | Title
			collect_point       | COLLECT POINT
			repair_point        | REPAIR POINT
			is_delivery         | Is Delivery
			symptom_code        | SYMPTOM CODE
			symptom_description | SYMPTOM DESCRIPTION
			analisa_kerusakan   | ANALISA KERUSAKAN
			nama_pic            | NAMA PIC
			repair_status_label | REPAIR STATUS
			alasan              | ALASAN
			alasan_batal        | ALASAN BATAL
			customer_arrival    | CUSTOMER ARRIVAL
		`),
	},
	{
		Code:  GroupDates,
		Title: "Estimasi Date",
		Fields: tabletext.Rows[Column](`
			Key                       | Title
			acknowledge_date          | ACKNOWLEDGE DATE
			tanggal_pengecekan        | TANGGAL PENGECEKAN
			tanggal_selesai_perbaikan | TANGGAL SELESAI PERBAIKAN
			tanggal_pick_up           | TANGGAL PICK UP
			tanggal_kurir_pick_up     | TANGGAL KURIR PICK UP
			release_date              | Release Date
			invoice_date              | INVOICE DATE
			downpayment               | DOWNPAYMENT
			no_downpayment            | NO. DOWNPAYMENT
			downpayment_method        | DOWNPAYMENT METHOD
			status_approval_label     | STATUS APPROVAL
			remark_approval           | REMARK APPROVAL
			komite_approval           | KOMITE APPROVAL
			aksesoris_lainya          | AKSESORIS LAINYA
		`),
	},
	{
		Code:  GroupCost,
		Title: "Estimasi Biaya",
		Fields: tabletext.Rows[Column](`
			Key               | Title
			biaya_jasa        | BIAYA JASA
			biaya_sparepart   | BIAYA SPAREPART
			biaya_suku_cadang | BIAYA SUKU CADANG
			biaya_pajak       | BIAYA PAJAK
			ppn               | PPN
			biaya_pengiriman  | BIAYA PENGIRIMAN
			biaya_lainnya     | BIAYA LAINNYA
			excess            | EXCESS DIBAYAR CUSTOMER
			deductible        | DEDUCTIBLE
			total_biaya       | TOTAL BIAYA
		`),
	},
	{
		Code:  GroupCostApproved,
		Title: "Harga Approve Komite",
		Fields: tabletext.Rows[Column](`
			Key                       | Title
			biaya_suku_cadang_approve | BIAYA SUKU CADANG
			biaya_pajak_approve       | BIAYA PAJAK
			ppn_approve               | PPN
			biaya_pengiriman_approve  | BIAYA PENGIRIMAN
			excess_approve            | EXCESS DIBAYAR CUSTOMER
			deductible_approve        | DEDUCTIBLE
			total_biaya_approve       | TOTAL BIAYA
		`),
	},
	{
		Code:  GroupAccessories,
		Title: "Accessories Unit",
		Fields: tabletext.Rows[Column](`
			Key               | Title
			charger_adaptor   | CHARGER ADAPTOR
			charger_cable     | CHARGER CABLE
			car_kit           | CAR KIT
			removable_antenna | Removable Antenna
			headset           | HEADSET
			battery           | BATTERY
			sim_card          | SIM CARD
			external_cover    | EXTERNAL COVER
			battery_cover     | Battery Cover
			lcd_text          | LCD Text
			case              | CASE
			box_unit          | BOX UNIT
		`),
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
