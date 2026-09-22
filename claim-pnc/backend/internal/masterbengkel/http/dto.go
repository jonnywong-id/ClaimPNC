// Package masterbengkelhttp adalah lapisan transport modul Master Bengkel.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// masterautoclaim/http: foldernya `http` supaya letaknya seragam antarmodul, nama
// paketnya `masterbengkelhttp` supaya tidak menutupi `net/http`.
package masterbengkelhttp

import "claim-pnc/internal/masterbengkel"

// WorkshopDTO adalah bentuk satu baris master bengkel yang dikirim ke peramban.
//
// Terpisah dari masterbengkel.Workshop supaya perubahan internal tidak bocor ke klien
// dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya berbahasa Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
// Yang dipakai adalah nama yang MENCERMINKAN ISI, bukan nama kolomnya apa adanya —
// `NAMA_KABUPATEN` menjadi `nama_kota` karena label layarnya memang "NAMA KOTA", dan
// `NO_ACCOUNT` menjadi `no_rekening` karena itulah yang dibacanya.
type WorkshopDTO struct {
	// ID adalah kolom ID_BENGKEL — kunci baris ini, diterbitkan server.
	ID string `json:"id_bengkel"`

	Name    string `json:"nama_bengkel"`
	Address string `json:"alamat_bengkel"`
	Phone   string `json:"telp_bengkel"`
	Mobile  string `json:"nohp_bengkel"`
	Email   string `json:"email"`

	WorkOrderEmail string `json:"email_wo"`

	BranchID   string `json:"id_cabang"`
	BranchName string `json:"nama_cabang"`
	CityID     string `json:"id_kota"`
	CityName   string `json:"nama_kota"`

	PartnerStatus  string `json:"status_rekanan"`
	WorkshopStatus string `json:"status_bengkel"`
	StatusReason   string `json:"alasan_status_bengkel"`
	StatusDate     string `json:"tanggal_status"`

	Login string `json:"login_aplikasi"`

	BankID        string `json:"id_bank"`
	BankName      string `json:"nama_bank"`
	AccountNumber string `json:"no_rekening"`
	AccountName   string `json:"nama_rekening"`
	AccountID     string `json:"id_rekening"`

	TaxName       string `json:"nama_npwp"`
	TaxNumber     string `json:"no_npwp"`
	TaxAddress    string `json:"alamat_npwp"`
	IncomeTaxType string `json:"jenis_pph"`

	ValueAddedTax   string `json:"ppn"`
	ServiceDiscount string `json:"diskon_jasa"`
	PartDiscount    string `json:"diskon_sparepart"`
	MaterialPercent string `json:"persen_material"`

	PriceListGapPercent string `json:"pct_selisih_pl"`
	SLA                 string `json:"sla"`

	SuppliedByASM     string `json:"status_disupply_asm"`
	Supplier          string `json:"supplier"`
	EClaimStatus      string `json:"status_eklaim"`
	AutoAcceptStatus  string `json:"status_auto_aksep"`
	PaymentStatus     string `json:"status_payment"`
	AutoPaymentStatus string `json:"status_autopayment"`
	TeknoStatus       string `json:"status_tekno"`
	OrderStatus       string `json:"status_order"`

	// DocumentID adalah kolom DOKUMENID.
	//
	// Ia dikirim meski layar ini tidak menyediakan unggahan lampiran: baris lama dapat
	// memilikinya, dan menyembunyikannya berarti petugas tidak punya cara mengetahui
	// bahwa lampirannya masih ada.
	DocumentID string `json:"id_dokumen"`

	// Status adalah kolom APPROVAL: "0" menunggu, "1" disetujui, "2" ditolak.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status dalam bahasa yang dibaca pengguna, dihitung
	// server supaya layar tidak menyimpan salinan ketiga sandinya.
	StatusLabel string `json:"status_label"`

	// Partner menyatakan bengkel ini berstatus rekanan.
	//
	// Dihitung server dari STATUS_REKANAN supaya layar tidak perlu menyimpan salinan
	// sandi non-rekanan — sandi yang artinya terbaca dari percabangan Pega, bukan dari
	// label mana pun (lihat masterbengkel.PartnerStatusNonPartner).
	Partner bool `json:"rekanan"`
}

// BranchDTO adalah satu pilihan pada dropdown Cabang.
type BranchDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// CityDTO adalah satu pilihan pada lookup Kota.
type CityDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// BankDTO adalah satu pilihan pada dropdown Bank.
//
// Berbeda dari Master Auto Claim yang hanya menyimpan nama banknya, tabel bengkel punya
// kolom kode banknya sendiri — sehingga keduanya ikut disimpan, bukan hanya ditampilkan.
type BankDTO struct {
	Code string `json:"kode"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/bengkel.
type ListResponse struct {
	Bengkel []WorkshopDTO `json:"bengkel"`

	// Status menyebut penyaring yang benar-benar dipakai, bukan yang diminta. Keduanya
	// sama pada jalur normal; menyebutkannya membuat layar dapat memastikan tab yang
	// ditampilkan memang tab yang dijawab.
	Status string `json:"status"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna. Pada aplikasi yang melayani empat badan
	// hukum, "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban penambahan, pengambilan, dan penyimpanan.
type SingleResponse struct {
	Bengkel WorkshopDTO `json:"bengkel"`
	Portal  string      `json:"portal"`
}

// DecisionResponse adalah jawaban keputusan borongan.
type DecisionResponse struct {
	// Changed adalah jumlah baris yang BENAR-BENAR berubah, bukan jumlah yang dikirim.
	//
	// Keduanya dapat berbeda: baris yang sudah berstatus itu, atau yang sudah tidak ada,
	// tidak ikut terhitung. Menyebutkannya membuat layar dapat mengatakan "3 dari 5"
	// alih-alih melaporkan keberhasilan atas baris yang tidak tersentuh.
	Changed int `json:"jumlah_berubah"`

	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Portal      string `json:"portal"`
}

// BranchListResponse adalah jawaban GET /api/master/bengkel/cabang.
type BranchListResponse struct {
	Branch []BranchDTO `json:"cabang"`
	Portal string      `json:"portal"`
}

// CityListResponse adalah jawaban GET /api/master/bengkel/kota.
type CityListResponse struct {
	City   []CityDTO `json:"kota"`
	Portal string    `json:"portal"`
}

// BankListResponse adalah jawaban GET /api/master/bengkel/bank.
type BankListResponse struct {
	Bank   []BankDTO `json:"bank"`
	Portal string    `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan DAN penyimpanan.
//
// # Kenapa satu bentuk untuk dua jalur
//
// Karena isiannya memang sama: `Section/BrowseMasterHEApprove-Section.xml` memakai form
// yang sama untuk menambah dan mengubah, dan `Activity/UpdateBengkelHE_act` melayani
// keduanya — yang membedakannya hanya `TempBengkel.ID_BENGKEL=="UnknownID"`.
//
// # Yang TIDAK ada di sini, dan kenapa
//
//	id_bengkel  kunci baris. Pada penambahan ia diterbitkan server dari kode situs dan
//	            sequence; pada penyimpanan ia diambil dari jalur URL. Menerimanya dari
//	            badan permintaan berarti dua sumber untuk satu nilai, dan yang mana yang
//	            menang menjadi pertanyaan yang tidak perlu ada.
//	status      bukan isian melainkan akibat. Penambahan dan penyimpanan SELALU
//	            menghasilkan status menunggu (`UpdateBengkelHE_act` step 7); keputusan
//	            komite menempuh endpoint tersendiri.
//	id_dokumen  hasil unggah lampiran, jalur yang tidak dibawa modul ini. Nilainya
//	            dipertahankan dari baris yang tersimpan.
type SaveRequest struct {
	Name    string `json:"nama_bengkel"`
	Address string `json:"alamat_bengkel"`
	Phone   string `json:"telp_bengkel"`
	Mobile  string `json:"nohp_bengkel"`
	Email   string `json:"email"`

	WorkOrderEmail string `json:"email_wo"`

	BranchID   string `json:"id_cabang"`
	BranchName string `json:"nama_cabang"`
	CityID     string `json:"id_kota"`
	CityName   string `json:"nama_kota"`

	PartnerStatus  string `json:"status_rekanan"`
	WorkshopStatus string `json:"status_bengkel"`
	StatusReason   string `json:"alasan_status_bengkel"`
	StatusDate     string `json:"tanggal_status"`

	Login string `json:"login_aplikasi"`

	BankID        string `json:"id_bank"`
	BankName      string `json:"nama_bank"`
	AccountNumber string `json:"no_rekening"`
	AccountName   string `json:"nama_rekening"`
	AccountID     string `json:"id_rekening"`

	TaxName       string `json:"nama_npwp"`
	TaxNumber     string `json:"no_npwp"`
	TaxAddress    string `json:"alamat_npwp"`
	IncomeTaxType string `json:"jenis_pph"`

	ValueAddedTax   string `json:"ppn"`
	ServiceDiscount string `json:"diskon_jasa"`
	PartDiscount    string `json:"diskon_sparepart"`
	MaterialPercent string `json:"persen_material"`

	PriceListGapPercent string `json:"pct_selisih_pl"`
	SLA                 string `json:"sla"`

	SuppliedByASM     string `json:"status_disupply_asm"`
	Supplier          string `json:"supplier"`
	EClaimStatus      string `json:"status_eklaim"`
	AutoAcceptStatus  string `json:"status_auto_aksep"`
	PaymentStatus     string `json:"status_payment"`
	AutoPaymentStatus string `json:"status_autopayment"`
	TeknoStatus       string `json:"status_tekno"`
	OrderStatus       string `json:"status_order"`
}

// DecisionRequest adalah badan permintaan keputusan borongan.
//
// Bentuknya — DAFTAR kunci ditambah satu status — mengikuti sistem lama:
// `Activity/SetApprovalAllMaster` menelusuri baris yang dicentang lalu menetapkan status
// yang sama pada seluruhnya. Satu permintaan per baris akan mengubah operasi yang di
// Pega berupa satu tindakan menjadi sederet tindakan yang dapat gagal separuh jalan.
type DecisionRequest struct {
	// ID adalah kunci baris yang dicentang pengguna.
	ID []string `json:"id_bengkel"`

	// Status adalah keputusan yang dikehendaki: "1" approve, "2" reject.
	//
	// "0" juga diterima — ia yang mengembalikan baris ke antrean, dan
	// `SetApprovalAllMaster` pun menerima nilai apa pun lewat `Param.approval`.
	Status string `json:"status"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — ditambah `detail` untuk
// pelanggaran per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan
// mencocokkan teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
//
// Nama kuncinya `kolom`, mengikuti masterstatusprogres dan masterautoclaim. Ketiga modul
// master belum sepakat menamainya — masterstatus memakai `field` — dan penyeragamannya
// adalah TKT-F1-004 yang masih terhalang. Frontend sudah menampung keduanya lewat
// `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(w masterbengkel.Workshop) WorkshopDTO {
	return WorkshopDTO{
		ID:                  w.ID,
		Name:                w.Name,
		Address:             w.Address,
		Phone:               w.Phone,
		Mobile:              w.Mobile,
		Email:               w.Email,
		WorkOrderEmail:      w.WorkOrderEmail,
		BranchID:            w.BranchID,
		BranchName:          w.BranchName,
		CityID:              w.CityID,
		CityName:            w.CityName,
		PartnerStatus:       w.PartnerStatus,
		WorkshopStatus:      w.WorkshopStatus,
		StatusReason:        w.StatusReason,
		StatusDate:          w.StatusDate,
		Login:               w.Login,
		BankID:              w.BankID,
		BankName:            w.BankName,
		AccountNumber:       w.AccountNumber,
		AccountName:         w.AccountName,
		AccountID:           w.AccountID,
		TaxName:             w.TaxName,
		TaxNumber:           w.TaxNumber,
		TaxAddress:          w.TaxAddress,
		IncomeTaxType:       w.IncomeTaxType,
		ValueAddedTax:       w.ValueAddedTax,
		ServiceDiscount:     w.ServiceDiscount,
		PartDiscount:        w.PartDiscount,
		MaterialPercent:     w.MaterialPercent,
		PriceListGapPercent: w.PriceListGapPercent,
		SLA:                 w.SLA,
		SuppliedByASM:       w.SuppliedByASM,
		Supplier:            w.Supplier,
		EClaimStatus:        w.EClaimStatus,
		AutoAcceptStatus:    w.AutoAcceptStatus,
		PaymentStatus:       w.PaymentStatus,
		AutoPaymentStatus:   w.AutoPaymentStatus,
		TeknoStatus:         w.TeknoStatus,
		OrderStatus:         w.OrderStatus,
		DocumentID:          w.DocumentID,
		Status:              string(w.Status),
		StatusLabel:         w.Status.Label(),
		Partner:             w.PartnerStatus != masterbengkel.PartnerStatusNonPartner,
	}
}

// toInput mengubah badan permintaan menjadi isian domain.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya keduanya tidak pernah berbeda soal
// isian mana yang diterima.
func (r SaveRequest) toInput() masterbengkel.Input {
	return masterbengkel.Input{
		Name:                r.Name,
		Address:             r.Address,
		Phone:               r.Phone,
		Mobile:              r.Mobile,
		Email:               r.Email,
		WorkOrderEmail:      r.WorkOrderEmail,
		BranchID:            r.BranchID,
		BranchName:          r.BranchName,
		CityID:              r.CityID,
		CityName:            r.CityName,
		PartnerStatus:       r.PartnerStatus,
		WorkshopStatus:      r.WorkshopStatus,
		StatusReason:        r.StatusReason,
		StatusDate:          r.StatusDate,
		Login:               r.Login,
		BankID:              r.BankID,
		BankName:            r.BankName,
		AccountNumber:       r.AccountNumber,
		AccountName:         r.AccountName,
		AccountID:           r.AccountID,
		TaxName:             r.TaxName,
		TaxNumber:           r.TaxNumber,
		TaxAddress:          r.TaxAddress,
		IncomeTaxType:       r.IncomeTaxType,
		ValueAddedTax:       r.ValueAddedTax,
		ServiceDiscount:     r.ServiceDiscount,
		PartDiscount:        r.PartDiscount,
		MaterialPercent:     r.MaterialPercent,
		PriceListGapPercent: r.PriceListGapPercent,
		SLA:                 r.SLA,
		SuppliedByASM:       r.SuppliedByASM,
		Supplier:            r.Supplier,
		EClaimStatus:        r.EClaimStatus,
		AutoAcceptStatus:    r.AutoAcceptStatus,
		PaymentStatus:       r.PaymentStatus,
		AutoPaymentStatus:   r.AutoPaymentStatus,
		TeknoStatus:         r.TeknoStatus,
		OrderStatus:         r.OrderStatus,
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []masterbengkel.Workshop) []WorkshopDTO {
	result := make([]WorkshopDTO, 0, len(list))
	for _, w := range list {
		result = append(result, toDTO(w))
	}
	return result
}

func toBranchListDTO(list []masterbengkel.Branch) []BranchDTO {
	result := make([]BranchDTO, 0, len(list))
	for _, b := range list {
		result = append(result, BranchDTO{ID: b.ID, Name: b.Name})
	}
	return result
}

func toCityListDTO(list []masterbengkel.City) []CityDTO {
	result := make([]CityDTO, 0, len(list))
	for _, c := range list {
		result = append(result, CityDTO{ID: c.ID, Name: c.Name})
	}
	return result
}

func toBankListDTO(list []masterbengkel.Bank) []BankDTO {
	result := make([]BankDTO, 0, len(list))
	for _, b := range list {
		result = append(result, BankDTO{Code: b.Code, Name: b.Name})
	}
	return result
}
