// Package mastersupplierhttp adalah lapisan transport modul Master Supplier.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// masterbengkel/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `mastersupplierhttp` supaya tidak menutupi `net/http`.
package mastersupplierhttp

import "claim-pnc/internal/mastersupplier"

// SupplierDTO adalah bentuk satu baris master supplier yang dikirim ke peramban.
//
// Terpisah dari mastersupplier.Supplier supaya perubahan internal tidak bocor ke klien dan
// sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya berbahasa Indonesia karena ia KONTRAK, bukan nama internal (`D-80`). Yang
// dipakai adalah nama yang MENCERMINKAN ISI, bukan nama kunci JSON-nya apa adanya:
// `JENIS_STATUS` menjadi `status_supply` karena label layarnya memang "Status Supply", dan
// `STS_AKTIF_PROMLIST` menjadi `status_aktif` karena itulah yang tertulis di atas isiannya.
type SupplierDTO struct {
	// ID adalah kolom ID — kunci baris ini, diterbitkan server.
	ID string `json:"id_supplier"`

	// OldID adalah kolom OLDID.
	//
	// Ia dikirim meski tidak dapat disunting: baris warisan dapat memilikinya, dan
	// menyembunyikannya berarti petugas tidak punya cara menautkan baris ini dengan
	// catatan lama yang menyebut nomor itu.
	OldID string `json:"id_lama"`

	Name       string `json:"nama"`
	Address    string `json:"alamat"`
	City       string `json:"kota"`
	BranchName string `json:"nama_cabang"`
	PostalCode string `json:"kode_pos"`
	Country    string `json:"negara"`

	Phone string `json:"telepon"`
	Fax   string `json:"fax"`
	Email string `json:"email"`

	TaxNumber     string `json:"npwp"`
	ContactPerson string `json:"contact_person"`

	PartnerStatus string `json:"status_rekanan"`
	SupplyType    string `json:"status_supply"`

	// HeavyEquipment adalah kunci SUPPLIER_HE — turunan status_supply, bukan isian.
	//
	// Ia dikirim supaya layar dapat memperlihatkan nilai yang benar-benar tersimpan, dan
	// TIDAK diterima kembali pada penyimpanan: yang menurunkannya adalah server.
	HeavyEquipment string `json:"supplier_he"`

	TermOfPayment  string `json:"term_of_payment"`
	TermOfDelivery string `json:"term_of_delivery"`
	Note           string `json:"keterangan"`

	Bank          string `json:"bank"`
	AccountNumber string `json:"no_account"`
	AccountName   string `json:"account_name"`
	BankBranch    string `json:"bank_branch"`

	SupplierType string `json:"jenis_supplier"`

	// ActiveRequested adalah kunci STS_AKTIF_PROMLIST — isian berlabel "Status Aktif".
	ActiveRequested string `json:"status_aktif"`

	// Active adalah kunci STS_AKTIF — status aktif yang SEBENARNYA berlaku.
	//
	// Ia bukan isian dan tidak pernah tampil di form Pega, tetapi tetap dikirim: pada
	// supplier yang baru ditambahkan keduanya BERBEDA — yang diminta dapat "1" sementara
	// yang berlaku selalu "0" sampai persetujuannya turun
	// (`CreateNewMasterSupplier_post` step 6). Tanpa nilai ini, layar tidak punya cara
	// menjelaskan kenapa supplier yang baru disimpan sebagai aktif belum juga aktif.
	Active string `json:"status_aktif_berlaku"`

	AutoPayment string `json:"status_autopayment"`

	// UpdatedBy dan UpdatedAt adalah kunci USERKLAIMID dan TGL_INSERT.
	//
	// Keduanya ditulis sistem lama tetapi TIDAK PERNAH dibacanya kembali. Menampilkannya
	// adalah tambahan terhadap Pega, dan tidak mengubah apa pun yang tersimpan.
	//
	// UpdatedAt berbentuk teks `dd/MM/yyyy` zona WIB, bukan waktu — lihat
	// mastersupplier.FormatJakartaDate untuk alasan bentuknya dipertahankan.
	UpdatedBy string `json:"diubah_oleh"`
	UpdatedAt string `json:"diubah_pada"`

	// HeavyEquipmentLabel menyatakan supplier ini termasuk Heavy Equipment.
	//
	// Dihitung server dari SUPPLIER_HE supaya layar tidak perlu menyimpan salinan sandinya
	// — sandi yang artinya terbaca dari percabangan activity, bukan dari label mana pun.
	IsHeavyEquipment bool `json:"heavy_equipment"`

	// IsActive menyatakan supplier ini berlaku aktif.
	//
	// Dihitung dari status_aktif_berlaku, bukan dari status_aktif: yang menentukan apakah
	// sebuah supplier dapat dipakai adalah yang berlaku, bukan yang diminta.
	IsActive bool `json:"aktif"`
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

// CountryDTO adalah satu pilihan pada isian Negara.
type CountryDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// BankDTO adalah satu pilihan pada dropdown Bank.
//
// Kodenya dikirim meski TIDAK disimpan ke dokumen supplier — dokumennya tidak punya kunci
// BANK_ID. Ia hanya pembeda bila ada dua bank bernama mirip.
type BankDTO struct {
	Code string `json:"kode"`
	Name string `json:"nama"`
}

// CodeDTO adalah satu pilihan pada dropdown bersandi.
type CodeDTO struct {
	Value string `json:"nilai"`
	Label string `json:"label"`
}

// ListResponse adalah jawaban GET /api/master/supplier.
type ListResponse struct {
	Supplier []SupplierDTO `json:"supplier"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna. Pada aplikasi yang melayani empat badan hukum,
	// "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban penambahan, pengambilan, dan penyimpanan.
type SingleResponse struct {
	Supplier SupplierDTO `json:"supplier"`
	Portal   string      `json:"portal"`
}

// BranchListResponse adalah jawaban GET /api/master/supplier/cabang.
type BranchListResponse struct {
	Branch []BranchDTO `json:"cabang"`
	Portal string      `json:"portal"`
}

// CityListResponse adalah jawaban GET /api/master/supplier/kota.
type CityListResponse struct {
	City   []CityDTO `json:"kota"`
	Portal string    `json:"portal"`
}

// CountryListResponse adalah jawaban GET /api/master/supplier/negara.
type CountryListResponse struct {
	Country []CountryDTO `json:"negara"`
	Portal  string       `json:"portal"`
}

// BankListResponse adalah jawaban GET /api/master/supplier/bank.
type BankListResponse struct {
	Bank   []BankDTO `json:"bank"`
	Portal string    `json:"portal"`
}

// CodeListResponse adalah jawaban GET /api/master/supplier/sandi.
//
// Kelima daftar dikirim sekaligus, bukan lima endpoint: kelimanya dibaca dari tabel yang
// sama dalam satu kali pemindaian, dan memecahnya berarti lima kali memindai tabel yang
// sama untuk mengisi satu form.
type CodeListResponse struct {
	PartnerStatus []CodeDTO `json:"status_rekanan"`
	SupplyType    []CodeDTO `json:"status_supply"`
	SupplierType  []CodeDTO `json:"jenis_supplier"`
	Active        []CodeDTO `json:"status_aktif"`
	AutoPayment   []CodeDTO `json:"status_autopayment"`
	Portal        string    `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan DAN penyimpanan.
//
// # Kenapa satu bentuk untuk dua jalur
//
// Karena isiannya memang sama: `Section/CreateMasterSupplier_Sec-Section.xml` dipakai
// KEDUA Flow Action — `CreateMasterSupplier` dan `EditMasterSupplier` — dan yang berbeda
// hanyalah activity pre dan post-nya.
//
// # Yang TIDAK ada di sini, dan kenapa
//
//	id_supplier   kunci baris. Pada penambahan ia diterbitkan server dari kode situs dan
//	              sequence; pada penyimpanan ia diambil dari jalur URL. Menerimanya dari
//	              badan permintaan berarti dua sumber untuk satu nilai.
//	id_lama       kolom warisan yang tidak pernah ditulis rule mana pun.
//	supplier_he   turunan status_supply, bukan isian.
//	status_aktif_berlaku  bukan isian melainkan akibat — "0" pada penambahan, dan
//	              menyusul status_aktif pada penyimpanan.
//	diubah_oleh   identitas pemanggil, diambil dari sesi.
//	diubah_pada   waktu penyimpanan, diambil dari seam Clock.
//
// # nama tetap DITERIMA meski tidak dapat diubah
//
// Layar mengirimkannya kembali apa adanya karena isiannya memang ada di form, hanya
// terkunci. Server membandingkannya dengan yang tersimpan dan MENOLAK bila berbeda, alih
// alih mengabaikannya diam-diam: permintaan yang mencoba menggantinya harus tahu bahwa
// usahanya tidak berlaku.
type SaveRequest struct {
	Name       string `json:"nama"`
	Address    string `json:"alamat"`
	City       string `json:"kota"`
	BranchName string `json:"nama_cabang"`
	PostalCode string `json:"kode_pos"`
	Country    string `json:"negara"`

	Phone string `json:"telepon"`
	Fax   string `json:"fax"`
	Email string `json:"email"`

	TaxNumber     string `json:"npwp"`
	ContactPerson string `json:"contact_person"`

	PartnerStatus string `json:"status_rekanan"`
	SupplyType    string `json:"status_supply"`

	TermOfPayment  string `json:"term_of_payment"`
	TermOfDelivery string `json:"term_of_delivery"`
	Note           string `json:"keterangan"`

	Bank          string `json:"bank"`
	AccountNumber string `json:"no_account"`
	AccountName   string `json:"account_name"`
	BankBranch    string `json:"bank_branch"`

	SupplierType    string `json:"jenis_supplier"`
	ActiveRequested string `json:"status_aktif"`
	AutoPayment     string `json:"status_autopayment"`
}

// toInput memetakan badan permintaan menjadi isian domain.
func (r SaveRequest) toInput() mastersupplier.Input {
	return mastersupplier.Input{
		Name:       r.Name,
		Address:    r.Address,
		City:       r.City,
		BranchName: r.BranchName,
		PostalCode: r.PostalCode,
		Country:    r.Country,

		Phone: r.Phone,
		Fax:   r.Fax,
		Email: r.Email,

		TaxNumber:     r.TaxNumber,
		ContactPerson: r.ContactPerson,

		PartnerStatus: r.PartnerStatus,
		SupplyType:    r.SupplyType,

		TermOfPayment:  r.TermOfPayment,
		TermOfDelivery: r.TermOfDelivery,
		Note:           r.Note,

		Bank:          r.Bank,
		AccountNumber: r.AccountNumber,
		AccountName:   r.AccountName,
		BankBranch:    r.BankBranch,

		SupplierType:    r.SupplierType,
		ActiveRequested: r.ActiveRequested,
		AutoPayment:     r.AutoPayment,
	}
}

// ViolationDTO adalah satu isian yang ditolak, beserta pesannya.
//
// # Nama fieldnya `kolom`, dan itu BUKAN pilihan bebas
//
// Klien bersama `api/client.ts` membaca nama isian dari `field` ATAU `kolom`, dan pesannya
// dari `pesan`. Nama lain — sekalipun lebih tepat artinya — membuat `APIError.violations()`
// mengembalikan peta kosong, sehingga layar menampilkan satu kotak galat umum alih-alih
// menyorot isian yang salah satu per satu.
//
// Pada form berisi lima belas isian wajib, akibatnya nyata: pengguna diberi tahu "ada
// isian yang belum benar" tanpa satu pun petunjuk yang mana.
//
// `kolom` dipilih karena itu yang dipakai Master Bengkel dan Master Auto Claim; Master
// Pasal Kerugian memakai `field`. Keduanya sah hari ini, dan penyeragamannya menunggu
// TKT-F1-004.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya `{kode, pesan, detail}`, sama dengan galat modul lain, supaya klien tidak
// menghadapi dua bentuk galat yang berbeda.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// toDTO memetakan satu baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(s mastersupplier.Supplier) SupplierDTO {
	return SupplierDTO{
		ID:    s.ID,
		OldID: s.OldID,
		Name:  s.Name,

		Address:    s.Address,
		City:       s.City,
		BranchName: s.BranchName,
		PostalCode: s.PostalCode,
		Country:    s.Country,

		Phone: s.Phone,
		Fax:   s.Fax,
		Email: s.Email,

		TaxNumber:     s.TaxNumber,
		ContactPerson: s.ContactPerson,

		PartnerStatus:  s.PartnerStatus,
		SupplyType:     s.SupplyType,
		HeavyEquipment: s.HeavyEquipment,

		TermOfPayment:  s.TermOfPayment,
		TermOfDelivery: s.TermOfDelivery,
		Note:           s.Note,

		Bank:          s.Bank,
		AccountNumber: s.AccountNumber,
		AccountName:   s.AccountName,
		BankBranch:    s.BankBranch,

		SupplierType:    s.SupplierType,
		ActiveRequested: s.ActiveRequested,
		Active:          s.Active,
		AutoPayment:     s.AutoPayment,

		UpdatedBy: s.UpdatedBy,
		UpdatedAt: s.UpdatedAt,

		IsHeavyEquipment: s.HeavyEquipment == mastersupplier.SupplyTypeHeavyEquipment,
		IsActive:         s.Active == mastersupplier.ActiveYes,
	}
}

// toListDTO memetakan sekumpulan baris.
//
// Senarai kosong dikembalikan sebagai senarai kosong, BUKAN nil: `[]` dan `null` diurai
// berbeda di peramban, dan yang kedua memaksa setiap layar memeriksanya sendiri.
func toListDTO(list []mastersupplier.Supplier) []SupplierDTO {
	result := make([]SupplierDTO, 0, len(list))
	for _, one := range list {
		result = append(result, toDTO(one))
	}
	return result
}

func toBranchListDTO(list []mastersupplier.Branch) []BranchDTO {
	result := make([]BranchDTO, 0, len(list))
	for _, one := range list {
		result = append(result, BranchDTO{ID: one.ID, Name: one.Name})
	}
	return result
}

func toCityListDTO(list []mastersupplier.City) []CityDTO {
	result := make([]CityDTO, 0, len(list))
	for _, one := range list {
		result = append(result, CityDTO{ID: one.ID, Name: one.Name})
	}
	return result
}

func toCountryListDTO(list []mastersupplier.Country) []CountryDTO {
	result := make([]CountryDTO, 0, len(list))
	for _, one := range list {
		result = append(result, CountryDTO{ID: one.ID, Name: one.Name})
	}
	return result
}

func toBankListDTO(list []mastersupplier.Bank) []BankDTO {
	result := make([]BankDTO, 0, len(list))
	for _, one := range list {
		result = append(result, BankDTO{Code: one.Code, Name: one.Name})
	}
	return result
}

func toCodeListDTO(list []mastersupplier.CodeOption) []CodeDTO {
	result := make([]CodeDTO, 0, len(list))
	for _, one := range list {
		result = append(result, CodeDTO{Value: one.Value, Label: one.Label})
	}
	return result
}
