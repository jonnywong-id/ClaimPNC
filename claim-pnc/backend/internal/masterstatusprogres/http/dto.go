// Package masterstatusprogreshttp adalah lapisan transport modul Master Status Progres.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterstatusprogreshttp` supaya tidak menutupi `net/http`.
package masterstatusprogreshttp

import "claim-pnc/internal/masterstatusprogres"

// ProgressStatusDTO adalah bentuk satu baris master yang dikirim ke peramban.
//
// Terpisah dari masterstatusprogres.ProgressStatus supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
type ProgressStatusDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`

	// KodePosisi adalah nilai yang tersimpan, dipakai saat menyunting.
	PositionCode string `json:"kode_posisi"`

	// PositionName adalah label yang dibaca pengguna.
	//
	// Ia dikirim bersama kodenya supaya layar tidak perlu memetakan sendiri — dan
	// karena itu tidak perlu menyimpan salinan keempat posisi di frontend. Satu daftar,
	// satu tempat.
	PositionName string `json:"nama_posisi"`
}

// PositionDTO adalah satu pilihan pada dropdown Posisi.
type PositionDTO struct {
	Code string `json:"kode"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/status-progres-1.
type ListResponse struct {
	ProgressStatus []ProgressStatusDTO `json:"status_progres"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang
	// melayani empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban penambahan dan penyuntingan.
type SingleResponse struct {
	ProgressStatus ProgressStatusDTO `json:"status_progres"`
	Portal         string            `json:"portal"`
}

// PositionListResponse adalah jawaban GET /api/master/posisi-klaim.
type PositionListResponse struct {
	Position []PositionDTO `json:"posisi"`
}

// SaveRequest adalah badan permintaan penambahan dan penyuntingan.
//
// ID tidak ada di sini, dan itu disengaja. Pada penambahan ia diturunkan dari isi tabel;
// pada penyuntingan ia diambil dari jalur, bukan dari badan — dua sumber untuk satu
// nilai berarti keduanya dapat berbeda, dan yang mana yang menang menjadi pertanyaan
// yang tidak perlu ada.
type SaveRequest struct {
	Name         string `json:"nama"`
	PositionCode string `json:"kode_posisi"`
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
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(sp masterstatusprogres.ProgressStatus) ProgressStatusDTO {
	return ProgressStatusDTO{
		ID:           sp.ID,
		Name:         sp.Name,
		PositionCode: sp.PositionCode,
		PositionName: masterstatusprogres.PositionName(sp.PositionCode),
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []masterstatusprogres.ProgressStatus) []ProgressStatusDTO {
	result := make([]ProgressStatusDTO, 0, len(list))
	for _, sp := range list {
		result = append(result, toDTO(sp))
	}
	return result
}
