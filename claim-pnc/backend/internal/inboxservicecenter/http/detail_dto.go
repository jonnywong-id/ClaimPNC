package inboxservicecenterhttp

import (
	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/inboxservicecenter/usecase"
)

// FieldGroupDTO adalah satu kelompok isian pada layar rincian.
//
// Layar menggambar kelompok dan urutannya dari sini, bukan dari daftar tetap di frontend:
// kelompoknya hasil pembacaan `Section/InputClaimServiceCenter-Section.xml`, dan tempat
// pembacaan itu tercatat adalah backend.
type FieldGroupDTO struct {
	Code  string `json:"kode"`
	Title string `json:"judul"`

	// Fields memakai bentuk yang sama dengan kolom grid — kunci dan judul — supaya layar
	// punya SATU cara menggambar sel, baik di tabel maupun di layar rincian.
	Fields []ColumnDTO `json:"isian"`
}

// ClaimDetailDTO adalah seluruh isian layar rincian.
//
// Nama field JSON berbahasa Indonesia mengikuti JUDUL ISIAN di layar Pega (`D-13`, `D-80`),
// bukan nama kolom dan bukan alias lamanya. `biaya_lainnya` karena itu bernama begitu,
// meskipun kolomnya `OTHER_FEE` dan aliasnya dulu `DistrictID`.
type ClaimDetailDTO struct {
	// ── General Information ──
	ID                 string  `json:"id"`
	RepairID           string  `json:"repair_id"`
	ClaimNumber        string  `json:"no_klaim"`
	Type               string  `json:"tipe"`
	Owner              string  `json:"username"`
	InputDate          *string `json:"tanggal_input"`
	PolicyNumber       string  `json:"no_polis"`
	Insurance          string  `json:"asuransi"`
	ActiveDateWarranty *string `json:"active_date_warranty"`
	ExpireDateWarranty *string `json:"expire_date_warranty"`
	CustomerName       string  `json:"customer_name"`
	InsuredName        string  `json:"nasabah"`
	CustomerPhone      string  `json:"no_hp_nasabah"`
	IDCardNumber       string  `json:"no_ktp"`
	PrincipalBill      string  `json:"principal_bill_no"`
	InsuranceBill      string  `json:"no_insurance_bill"`
	QuotationNo        string  `json:"no_quotation"`
	QuotationValue     string  `json:"quotation_amount"`

	// ── Informasi Unit ──
	ProductGroup    string `json:"jenis_barang"`
	ProductCategory string `json:"kategori_barang"`
	Brand           string `json:"brand"`
	Model           string `json:"model"`
	Colour          string `json:"color"`
	Device          string `json:"device"`
	IMEI            string `json:"no_imei"`
	SerialNumber    string `json:"no_serial"`
	ItemWarranty    string `json:"item_warranty"`
	ObjectName      string `json:"object_name"`

	// ── Informasi Perbaikan ──
	CollectPoint      string `json:"collect_point"`
	RepairPoint       string `json:"repair_point"`
	IsDelivery        string `json:"is_delivery"`
	SymptomCode       string `json:"symptom_code"`
	SymptomDesc       string `json:"symptom_description"`
	Analysis          string `json:"analisa_kerusakan"`
	TechnicalPIC      string `json:"nama_pic"`
	RepairStatus      string `json:"repair_status"`
	RepairStatusLabel string `json:"repair_status_label"`
	Reason            string `json:"alasan"`
	CancelReason      string `json:"alasan_batal"`
	CustomerReply     string `json:"customer_arrival"`

	// ── Estimasi Date ──
	AcknowledgeDate     *string `json:"acknowledge_date"`
	AssignedDate        *string `json:"tanggal_pengecekan"`
	CompletedDate       *string `json:"tanggal_selesai_perbaikan"`
	ReleaseDate         *string `json:"release_date"`
	InvoiceDate         *string `json:"invoice_date"`
	EstimatedPickupAt   *string `json:"tanggal_pick_up"`
	CourierPickupAt     *string `json:"tanggal_kurir_pick_up"`
	DownPaymentValue    string  `json:"downpayment"`
	DownPaymentNo       string  `json:"no_downpayment"`
	DownPaymentMethod   string  `json:"downpayment_method"`
	ApprovalStatus      string  `json:"status_approval"`
	ApprovalStatusLabel string  `json:"status_approval_label"`
	ApprovalRemark      string  `json:"remark_approval"`
	CommitteeApprover   string  `json:"komite_approval"`
	AccessoriesOther    string  `json:"aksesoris_lainya"`

	// ── Estimasi Biaya ──
	ServiceFee         string `json:"biaya_jasa"`
	SparepartFee       string `json:"biaya_sparepart"`
	SparepartValue     string `json:"biaya_suku_cadang"`
	SparepartValueAppr string `json:"biaya_suku_cadang_approve"`
	TaxFee             string `json:"biaya_pajak"`
	TaxFeeAppr         string `json:"biaya_pajak_approve"`
	VAT                string `json:"ppn"`
	VATAppr            string `json:"ppn_approve"`
	DeliveryFee        string `json:"biaya_pengiriman"`
	DeliveryFeeAppr    string `json:"biaya_pengiriman_approve"`
	OtherFee           string `json:"biaya_lainnya"`
	Excess             string `json:"excess"`
	ExcessAppr         string `json:"excess_approve"`
	Deductible         string `json:"deductible"`
	DeductibleAppr     string `json:"deductible_approve"`
	TotalFee           string `json:"total_biaya"`
	TotalFeeAppr       string `json:"total_biaya_approve"`

	// ── Accessories Unit ──
	ChargerAdaptor   string `json:"charger_adaptor"`
	ChargerCable     string `json:"charger_cable"`
	CarKit           string `json:"car_kit"`
	RemovableAntenna string `json:"removable_antenna"`
	Headset          string `json:"headset"`
	Battery          string `json:"battery"`
	SimCard          string `json:"sim_card"`
	ExternalCover    string `json:"external_cover"`
	BatteryCover     string `json:"battery_cover"`
	LCDText          string `json:"lcd_text"`
	UnitCase         string `json:"case"`
	BoxUnit          string `json:"box_unit"`

	// PartDetailJSON adalah isi kolom DETAILPART apa adanya.
	//
	// Ia dikirim sebagai TEKS, belum diurai. Bentuk JSON-nya belum pernah dibaca dari data
	// sungguhan, dan menguraikannya berdasarkan tebakan akan menghasilkan grid yang kosong
	// atau salah kolom tanpa satu pun galat. Layar menampilkannya apa adanya sampai
	// bentuknya diketahui.
	PartDetailJSON string `json:"detail_part_json"`
}

// ProgressNoteDTO adalah satu catatan progres.
type ProgressNoteDTO struct {
	RecordedAt *string `json:"tanggal"`
	Note       string  `json:"catatan"`
	RecordedBy string  `json:"oleh"`
}

// DetailResponse adalah jawaban GET /api/inbox-service-center/{id}.
type DetailResponse struct {
	Claim    ClaimDetailDTO    `json:"klaim"`
	Progress []ProgressNoteDTO `json:"riwayat_progres"`
	Groups   []FieldGroupDTO   `json:"kelompok"`
	Portal   string            `json:"portal"`
}

// toDetailResponse merakit jawaban rincian.
func toDetailResponse(detailed usecase.Detailed, portalAlias string) DetailResponse {
	groups := make([]FieldGroupDTO, 0, len(detailed.Groups))
	for _, group := range detailed.Groups {
		fields := make([]ColumnDTO, 0, len(group.Fields))
		for _, field := range group.Fields {
			fields = append(fields, ColumnDTO{Key: field.Key, Title: field.Title})
		}
		groups = append(groups, FieldGroupDTO{
			Code:   group.Code,
			Title:  group.Title,
			Fields: fields,
		})
	}

	notes := make([]ProgressNoteDTO, 0, len(detailed.Progress))
	for _, note := range detailed.Progress {
		notes = append(notes, ProgressNoteDTO{
			RecordedAt: toDateString(note.RecordedAt),
			Note:       note.Note,
			RecordedBy: note.RecordedBy,
		})
	}

	return DetailResponse{
		Claim:    toClaimDetailDTO(detailed.Claim),
		Progress: notes,
		Groups:   groups,
		Portal:   portalAlias,
	}
}

// toClaimDetailDTO mengubah satu rincian klaim.
func toClaimDetailDTO(d inboxservicecenter.ClaimDetail) ClaimDetailDTO {
	return ClaimDetailDTO{
		ID:                 d.ID,
		RepairID:           d.RepairID,
		ClaimNumber:        d.ClaimNumber,
		Type:               d.Type,
		Owner:              d.Owner,
		InputDate:          toDateString(d.InputDate),
		PolicyNumber:       d.PolicyNumber,
		Insurance:          d.Insurance,
		ActiveDateWarranty: toDateString(d.WarrantyStart),
		ExpireDateWarranty: toDateString(d.WarrantyEnd),
		CustomerName:       d.CustomerName,
		InsuredName:        d.InsuredName,
		CustomerPhone:      d.CustomerPhone,
		IDCardNumber:       d.IDCardNumber,
		PrincipalBill:      d.PrincipalBill,
		InsuranceBill:      d.InsuranceBill,
		QuotationNo:        d.QuotationNo,
		QuotationValue:     d.QuotationValue,

		ProductGroup:    d.ProductGroup,
		ProductCategory: d.ProductCategory,
		Brand:           d.Brand,
		Model:           d.Model,
		Colour:          d.Colour,
		Device:          d.Device,
		IMEI:            d.IMEI,
		SerialNumber:    d.SerialNumber,
		ItemWarranty:    d.ItemWarranty,
		ObjectName:      d.ObjectName,

		CollectPoint:      d.CollectPoint,
		RepairPoint:       d.RepairPoint,
		IsDelivery:        d.IsDelivery,
		SymptomCode:       d.SymptomCode,
		SymptomDesc:       d.SymptomDesc,
		Analysis:          d.Analysis,
		TechnicalPIC:      d.TechnicalPIC,
		RepairStatus:      d.RepairStatus,
		RepairStatusLabel: d.RepairStatusName(),
		Reason:            d.Reason,
		CancelReason:      d.CancelReason,
		CustomerReply:     d.CustomerReply,

		AcknowledgeDate:     toDateString(d.AcknowledgeDate),
		AssignedDate:        toDateString(d.AssignedDate),
		CompletedDate:       toDateString(d.CompletedDate),
		ReleaseDate:         toDateString(d.ReleaseDate),
		InvoiceDate:         toDateString(d.InvoiceDate),
		EstimatedPickupAt:   toDateString(d.EstimatedPickupAt),
		CourierPickupAt:     toDateString(d.CourierPickupAt),
		DownPaymentValue:    d.DownPaymentValue,
		DownPaymentNo:       d.DownPaymentNo,
		DownPaymentMethod:   d.DownPaymentMethod,
		ApprovalStatus:      d.ApprovalStatus,
		ApprovalStatusLabel: d.ApprovalStatusName(),
		ApprovalRemark:      d.ApprovalRemark,
		CommitteeApprover:   d.CommitteeApprover,
		AccessoriesOther:    d.AccessoriesOther,

		ServiceFee:         d.ServiceFee,
		SparepartFee:       d.SparepartFee,
		SparepartValue:     d.SparepartValue,
		SparepartValueAppr: d.SparepartValueAppr,
		TaxFee:             d.TaxFee,
		TaxFeeAppr:         d.TaxFeeAppr,
		VAT:                d.VAT,
		VATAppr:            d.VATAppr,
		DeliveryFee:        d.DeliveryFee,
		DeliveryFeeAppr:    d.DeliveryFeeAppr,
		OtherFee:           d.OtherFee,
		Excess:             d.Excess,
		ExcessAppr:         d.ExcessAppr,
		Deductible:         d.Deductible,
		DeductibleAppr:     d.DeductibleAppr,
		TotalFee:           d.TotalFee,
		TotalFeeAppr:       d.TotalFeeAppr,

		ChargerAdaptor:   d.ChargerAdaptor,
		ChargerCable:     d.ChargerCable,
		CarKit:           d.CarKit,
		RemovableAntenna: d.RemovableAntenna,
		Headset:          d.Headset,
		Battery:          d.Battery,
		SimCard:          d.SimCard,
		ExternalCover:    d.ExternalCover,
		BatteryCover:     d.BatteryCover,
		LCDText:          d.LCDText,
		UnitCase:         d.UnitCase,
		BoxUnit:          d.BoxUnit,

		PartDetailJSON: d.PartDetailJSON,
	}
}
