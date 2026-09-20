// Package masterautoclaimhttp adalah lapisan transport modul Master Auto Claim.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// masterstatusprogres/http: foldernya `http` supaya letaknya seragam antarmodul, nama
// paketnya `masterautoclaimhttp` supaya tidak menutupi `net/http`.
package masterautoclaimhttp

import "claim-pnc/internal/masterautoclaim"

// AutoClaimDTO adalah bentuk satu baris master yang dikirim ke peramban.
//
// Terpisah dari masterautoclaim.AutoClaim supaya perubahan internal tidak bocor ke klien
// dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya berbahasa Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
type AutoClaimDTO struct {
	// Inisial adalah kolom INISIALID — kode Sumber Bisnis, kunci baris ini.
	Initial string `json:"inisial"`

	// ReceiverName adalah kolom NAMA_PENERIMA. Ia tidak dapat diubah lewat penyuntingan;
	// lihat SaveRequest.
	ReceiverName string `json:"nama_penerima"`

	BankName        string `json:"nama_bank"`
	AccountNumber   string `json:"no_rekening"`
	MaxPercent      string `json:"pct_max"`
	ReporterPIC     string `json:"pic_lapor"`
	ReporterEmail   string `json:"email_lapor"`
	ReceiverAddress string `json:"alamat_penerima"`

	ClientID   string `json:"id_client"`
	ClientName string `json:"nama_client"`

	// ClaimAllowed adalah kolom CLAIM_ALLOWED, dibaca apa adanya dari basis data.
	//
	// Ia dikirim meski layar tidak menyediakan isiannya: baris lama dapat bernilai
	// selain "1", dan nilai itu MENENTUKAN apakah klaim otomatis benar-benar terbentuk
	// (`GetReceiverClaimAsuransiKredit-SQL.xml` menyaring `claim_allowed = 1`).
	// Menyembunyikannya berarti petugas tidak punya cara mengetahui kenapa sebuah baris
	// yang tampak disetujui tidak pernah dipakai.
	ClaimAllowed string `json:"claim_allowed"`

	// Committee adalah kolom KOMITE — operator yang berwenang memutuskan baris ini.
	Committee string `json:"komite"`

	// Status adalah kolom APPROVAL: "0" menunggu, "1" disetujui, "2" ditolak.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status dalam bahasa yang dibaca pengguna, dihitung
	// server supaya layar tidak menyimpan salinan ketiga sandinya.
	StatusLabel string `json:"status_label"`

	// Usable menyatakan baris ini benar-benar dipakai pembuatan klaim otomatis.
	//
	// Dihitung server dari dua syarat — disetujui komite DAN claim_allowed "1" — supaya
	// keduanya tidak perlu diulang di setiap layar, dan supaya keduanya persis sama
	// dengan penyaring kueri hilirnya.
	Usable bool `json:"dapat_dipakai"`
}

// BusinessSourceDTO adalah satu pilihan pada lookup Sumber Bisnis.
type BusinessSourceDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ClientDTO adalah satu pilihan pada lookup Client.
type ClientDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// BankDTO adalah satu pilihan pada dropdown Bank Penerima.
type BankDTO struct {
	// Code dikirim meski TIDAK pernah disimpan ke master auto claim: layar memakainya
	// sebagai kunci pilihan pada dropdown, dan menampilkannya di sebelah nama supaya dua
	// bank bernama mirip dapat dibedakan.
	Code string `json:"kode"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/auto-claim.
type ListResponse struct {
	AutoClaim []AutoClaimDTO `json:"auto_claim"`

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
	AutoClaim AutoClaimDTO `json:"auto_claim"`
	Portal    string       `json:"portal"`
}

// BusinessSourceListResponse adalah jawaban GET /api/master/auto-claim/sumber-bisnis.
type BusinessSourceListResponse struct {
	BusinessSource []BusinessSourceDTO `json:"sumber_bisnis"`
	Portal         string              `json:"portal"`
}

// ClientListResponse adalah jawaban GET /api/master/auto-claim/client.
type ClientListResponse struct {
	Client []ClientDTO `json:"client"`
	Portal string      `json:"portal"`
}

// BankListResponse adalah jawaban GET /api/master/auto-claim/bank.
type BankListResponse struct {
	Bank   []BankDTO `json:"bank"`
	Portal string    `json:"portal"`
}

// CreateRequest adalah badan permintaan penambahan.
//
// Empat nilai yang ADA di tabel sengaja TIDAK ada di sini, karena keempatnya diturunkan
// server dan menerimanya dari peramban berarti mempercayai klien atas nilai yang bukan
// miliknya:
//
//	status         selalu "0" pada baris baru
//	claim_allowed  selalu "1" (keputusan Work Owner 2026-09-19)
//	komite         hasil lookup POOLDATA.EMAILKOMITE
//	diinput_oleh   operator yang sedang masuk
type CreateRequest struct {
	Initial      string `json:"inisial"`
	ReceiverName string `json:"nama_penerima"`

	BankName        string `json:"nama_bank"`
	AccountNumber   string `json:"no_rekening"`
	MaxPercent      string `json:"pct_max"`
	ReporterPIC     string `json:"pic_lapor"`
	ReporterEmail   string `json:"email_lapor"`
	ReceiverAddress string `json:"alamat_penerima"`

	ClientID   string `json:"id_client"`
	ClientName string `json:"nama_client"`
}

// SaveRequest adalah badan permintaan penyimpanan — sekaligus keputusan komite.
//
// # Kenapa satu bentuk untuk tiga tombol
//
// Karena di sistem lama memang satu. `Activity/UpdateMstAutoClaim_act` melayani tombol
// Update, Approve, dan Reject sekaligus; yang membedakannya hanya parameter
// `stsapprove`. Keputusan Work Owner 2026-09-19 mempertahankan bentuk itu.
//
// # Yang TIDAK ada di sini, dan kenapa
//
//	inisial        kunci baris, diambil dari jalur URL. Dua sumber untuk satu nilai
//	               berarti keduanya dapat berbeda, dan yang mana yang menang menjadi
//	               pertanyaan yang tidak perlu ada.
//	nama_penerima  tidak dapat diubah (keputusan Work Owner 2026-09-19); kueri lama pun
//	               tidak menyebut kolomnya.
//	komite         dibaca dari baris yang tersimpan, tidak pernah dari permintaan. Pega
//	               menulisnya dari isi form, dan form yang belum dimuat mengosongkannya
//	               — jalur itu tidak dibawa.
//	claim_allowed  selalu "1".
type SaveRequest struct {
	BankName        string `json:"nama_bank"`
	AccountNumber   string `json:"no_rekening"`
	MaxPercent      string `json:"pct_max"`
	ReporterPIC     string `json:"pic_lapor"`
	ReporterEmail   string `json:"email_lapor"`
	ReceiverAddress string `json:"alamat_penerima"`

	ClientID   string `json:"id_client"`
	ClientName string `json:"nama_client"`

	// Status adalah posisi persetujuan yang dikehendaki: "0", "1", atau "2".
	//
	//	tombol Simpan  "0" — perubahan mengembalikan baris ke antrean persetujuan
	//	tombol Approve "1"
	//	tombol Reject  "2"
	Status string `json:"status"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul auth — `{kode, pesan}` — ditambah `detail` untuk
// pelanggaran per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan
// mencocokkan teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
//
// Nama kuncinya `kolom`, mengikuti masterstatusprogres. Ketiga modul master belum
// sepakat menamainya — masterstatus memakai `field` — dan penyeragamannya adalah
// TKT-F1-004 yang masih terhalang. Frontend sudah menampung keduanya lewat
// `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(ac masterautoclaim.AutoClaim) AutoClaimDTO {
	return AutoClaimDTO{
		Initial:         ac.Initial,
		ReceiverName:    ac.ReceiverName,
		BankName:        ac.BankName,
		AccountNumber:   ac.AccountNumber,
		MaxPercent:      ac.MaxPercent,
		ReporterPIC:     ac.ReporterPIC,
		ReporterEmail:   ac.ReporterEmail,
		ReceiverAddress: ac.ReceiverAddress,
		ClientID:        ac.ClientID,
		ClientName:      ac.ClientName,
		ClaimAllowed:    ac.ClaimAllowed,
		Committee:       ac.Committee,
		Status:          string(ac.Status),
		StatusLabel:     ac.Status.Label(),
		Usable:          ac.Usable(),
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []masterautoclaim.AutoClaim) []AutoClaimDTO {
	result := make([]AutoClaimDTO, 0, len(list))
	for _, ac := range list {
		result = append(result, toDTO(ac))
	}
	return result
}

func toBusinessSourceListDTO(list []masterautoclaim.BusinessSource) []BusinessSourceDTO {
	result := make([]BusinessSourceDTO, 0, len(list))
	for _, s := range list {
		result = append(result, BusinessSourceDTO{ID: s.ID, Name: s.Name})
	}
	return result
}

func toClientListDTO(list []masterautoclaim.Client) []ClientDTO {
	result := make([]ClientDTO, 0, len(list))
	for _, c := range list {
		result = append(result, ClientDTO{ID: c.ID, Name: c.Name})
	}
	return result
}

func toBankListDTO(list []masterautoclaim.Bank) []BankDTO {
	result := make([]BankDTO, 0, len(list))
	for _, b := range list {
		result = append(result, BankDTO{Code: b.Code, Name: b.Name})
	}
	return result
}
