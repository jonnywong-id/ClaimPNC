// Package mastercolsimasonlinehttp adalah lapisan transport modul Master COL Simas
// Online.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `mastercolsimasonlinehttp` supaya tidak menutupi `net/http`.
package mastercolsimasonlinehttp

import "claim-pnc/internal/mastercolsimasonline"

// CauseOfLossDTO adalah bentuk satu baris master yang dikirim ke peramban.
//
// Terpisah dari mastercolsimasonline.CauseOfLoss supaya perubahan internal tidak bocor
// ke klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field berbahasa Indonesia karena ia KONTRAK API, bukan nama internal
// (`D-80`).
type CauseOfLossDTO struct {
	// ID adalah M_COL_ID, berlabel "ID" di layar.
	ID string `json:"id"`

	// Name adalah COL_DESC, berlabel "Nama Cause of loss" di layar.
	Name string `json:"nama"`

	// MasterCode adalah MST_COL_ID, berlabel "ID Master Kerugian" di layar.
	MasterCode string `json:"id_master_kerugian"`

	// Businesses adalah daftar bisnis yang memakai penyebab kerugian ini.
	//
	// Pada jawaban DAFTAR ia selalu kosong, dan itu disengaja: grid layar hanya
	// menampilkan ID dan nama (`Section/Online_GridCauseOfLoss-Section.xml`), sehingga
	// menariknya untuk seluruh baris berarti satu kueri yang hasilnya tidak pernah
	// dilihat siapa pun. Layar memuatnya saat baris dibuka untuk disunting.
	Businesses []BusinessDTO `json:"bisnis"`
}

// BusinessDTO adalah satu lini bisnis.
type BusinessDTO struct {
	// ID BOLEH KOSONG pada pemetaan yang namanya diketik bebas dan tidak ada di master
	// — perilaku Pega yang dipertahankan (`pyAllowFreeFormInput=true`). Layar harus
	// menyiapkan keadaan itu; ia bukan tanda data rusak.
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/col-simas-online.
type ListResponse struct {
	CauseOfLoss []CauseOfLossDTO `json:"cause_of_loss"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang
	// melayani empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan
	// (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban pengambilan satu baris, penambahan, dan penyuntingan.
type SingleResponse struct {
	CauseOfLoss CauseOfLossDTO `json:"cause_of_loss"`
	Portal      string         `json:"portal"`
}

// BusinessListResponse adalah jawaban GET /api/master/bisnis.
type BusinessListResponse struct {
	Business []BusinessDTO `json:"bisnis"`
	Portal   string        `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan dan penyuntingan.
//
// ID tidak ada di sini, dan itu disengaja. Pada penambahan ia diterbitkan penyimpanan;
// pada penyuntingan ia diambil dari jalur, bukan dari badan — dua sumber untuk satu
// nilai berarti keduanya dapat berbeda, dan yang mana yang menang menjadi pertanyaan
// yang tidak perlu ada.
type SaveRequest struct {
	Name string `json:"nama"`

	// MasterCode adalah `id` cause of loss lain yang menjadi induk; kosong berarti tanpa
	// induk. Lihat mastercolsimasonline.CauseOfLoss.MasterCode — ia rujukan-diri, bukan
	// kode dari sistem sebelah.
	MasterCode string `json:"id_master_kerugian"`

	// Businesses berisi NAMA bisnis, bukan ID-nya.
	//
	// Nama yang dikirim karena itulah yang diketik dan dilihat petugas di layar Pega
	// (`pyValue = .Note`), dan karena nama yang diketik bebas memang tidak punya ID.
	// Server yang menyelesaikannya menjadi ID dengan mencocokkan ke master — nama yang
	// tidak cocok tetap diterima dan disimpan tanpa ID.
	Businesses []string `json:"bisnis"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul master lainnya — `{kode, pesan}` — ditambah `detail`
// untuk pelanggaran per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah
// dengan mencocokkan teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(row mastercolsimasonline.CauseOfLoss) CauseOfLossDTO {
	return CauseOfLossDTO{
		ID:         row.Code,
		Name:       row.Description,
		MasterCode: row.MasterCode,
		Businesses: toBusinessListDTO(row.Businesses),
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []mastercolsimasonline.CauseOfLoss) []CauseOfLossDTO {
	result := make([]CauseOfLossDTO, 0, len(list))
	for _, row := range list {
		result = append(result, toDTO(row))
	}
	return result
}

func toBusinessListDTO(list []mastercolsimasonline.Business) []BusinessDTO {
	result := make([]BusinessDTO, 0, len(list))
	for _, b := range list {
		result = append(result, BusinessDTO{ID: b.ID, Name: b.Name})
	}
	return result
}
