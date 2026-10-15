// Package masterpenolakanhttp adalah lapisan transport modul Master Penolakan Klaim.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterpenolakanhttp` supaya tidak menutupi `net/http`.
package masterpenolakanhttp

import (
	"time"

	"claim-pnc/internal/masterpenolakan"
)

// RejectionDTO adalah bentuk satu baris Status Penolakan 2 yang dikirim ke peramban.
//
// Terpisah dari masterpenolakan.RejectionStatus2 supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
type RejectionDTO struct {
	// ID adalah kolom ID_ND. Diterbitkan server; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// Name adalah kolom NOTE_ND — Status Penolakan 2.
	Name string `json:"nama"`

	// ParentID adalah kolom ID_ST — Status Penolakan 1 yang menaungi baris ini.
	ParentID string `json:"id_status_1"`

	// ParentName adalah kolom NOTE_ST: SALINAN nama induk saat baris ini disimpan.
	ParentName string `json:"nama_status_1"`

	// Status adalah kolom STATUS apa adanya — "0", "1", atau "2".
	//
	// Ia dikirim bersama labelnya supaya layar dapat menandai barisnya berdasarkan nilai
	// yang tetap, bukan dengan mencocokkan teks yang sewaktu-waktu berubah.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status yang dibaca pengguna — MENUNGGU, APPROVED, atau
	// REJECTED. Teksnya sama persis dengan derivasi kueri lama (`D-13`).
	StatusLabel string `json:"status_label"`

	// SubmittedBy adalah kolom USER_INPUT.
	SubmittedBy string `json:"diajukan_oleh"`

	// SubmittedAt adalah kolom TANGGALKIRIM dalam RFC 3339 UTC; kosong bila tidak terisi.
	//
	// Waktu dikirim sebagai UTC dan diubah ke WIB oleh layar, bukan diubah di sini —
	// konversi zona waktu terjadi di satu tempat saja (`08-TECHNICAL-STRATEGY.md` §4.4).
	SubmittedAt string `json:"diajukan_pada"`

	// ApprovedBy adalah kolom APPROVEBY — diisi layar Inbox Manager, baca-saja di sini.
	ApprovedBy string `json:"disetujui_oleh"`

	// ApprovedAt adalah kolom TANGGAL_APPROVE; kosong bila belum pernah diputuskan.
	ApprovedAt string `json:"disetujui_pada"`

	// ApprovalNote adalah kolom NOTEAPPROVED — catatan checker, baca-saja di sini.
	ApprovalNote string `json:"catatan_persetujuan"`
}

// ParentDTO adalah satu pilihan pada daftar "Status Penolakan 1".
//
// Ia memuat ID dan nama saja — persis kedua kolom yang dimiliki
// POOLDATA.MST_PENOLAKAN_KLAIM_1.
type ParentDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/penolakan-klaim.
type ListResponse struct {
	Rejection []RejectionDTO `json:"penolakan_klaim"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang
	// melayani empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban penambahan dan penyuntingan.
type SingleResponse struct {
	Rejection RejectionDTO `json:"penolakan_klaim"`
	Portal    string       `json:"portal"`
}

// ParentListResponse adalah jawaban GET /api/master/penolakan-klaim/status-1.
type ParentListResponse struct {
	// Parent BERBEDA antarentitas — ia dibaca dari tabel milik portal yang bersangkutan,
	// bukan daftar tetap milik aplikasi seperti halnya posisi klaim pada modul Master
	// Status Progres.
	Parent []ParentDTO `json:"status_1"`
	Portal string      `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan dan penyuntingan.
//
// ID tidak ada di sini, dan itu disengaja. Pada penambahan ia diturunkan dari isi tabel;
// pada penyuntingan ia diambil dari jalur, bukan dari badan — dua sumber untuk satu nilai
// berarti keduanya dapat berbeda, dan yang mana yang menang menjadi pertanyaan yang tidak
// perlu ada.
//
// Begitu pula status, pelaku, dan waktunya: ketiganya diterbitkan server. Menerimanya
// dari peramban berarti siapa pun dapat menyetujui catatannya sendiri.
type SaveRequest struct {
	// Name adalah Status Penolakan 2 yang diketik pengguna.
	Name string `json:"nama"`

	// ParentID adalah Status Penolakan 1 yang dipilih dari daftar.
	//
	// Kosong berarti pengguna membuat induk baru, dan ParentName yang dipakai. Keduanya
	// terisi bersamaan DITOLAK terang-terangan — lihat masterpenolakan.Input.Check.
	ParentID string `json:"id_status_1"`

	// ParentName adalah Status Penolakan 1 baru yang diketik pengguna.
	ParentName string `json:"nama_status_1"`
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
// Nama kuncinya `kolom`, mengikuti modul Master Status Progres. Ketidakseragaman dengan
// modul Master Status Klaim yang memakai `field` sudah dikenali dan masuk TKT-F1-004;
// frontend menyatukan keduanya di `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(rejection masterpenolakan.RejectionStatus2) RejectionDTO {
	return RejectionDTO{
		ID:           rejection.ID,
		Name:         rejection.Name,
		ParentID:     rejection.ParentID,
		ParentName:   rejection.ParentName,
		Status:       string(rejection.Status),
		StatusLabel:  rejection.Status.Label(),
		SubmittedBy:  rejection.SubmittedBy,
		SubmittedAt:  formatTime(rejection.SubmittedAt),
		ApprovedBy:   rejection.ApprovedBy,
		ApprovedAt:   formatTimePointer(rejection.ApprovedAt),
		ApprovalNote: rejection.ApprovalNote,
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []masterpenolakan.RejectionStatus2) []RejectionDTO {
	result := make([]RejectionDTO, 0, len(list))
	for _, rejection := range list {
		result = append(result, toDTO(rejection))
	}
	return result
}

// toParentListDTO mengubah baris tingkat 1 menjadi pilihan di layar.
func toParentListDTO(list []masterpenolakan.RejectionStatus) []ParentDTO {
	result := make([]ParentDTO, 0, len(list))
	for _, parent := range list {
		result = append(result, ParentDTO{ID: parent.ID, Name: parent.Name})
	}
	return result
}

// formatTime menuliskan waktu sebagai RFC 3339 UTC, atau string kosong bila tidak terisi.
//
// Kosong, bukan "0001-01-01T00:00:00Z": kolomnya memang boleh NULL, dan mengirim tanggal
// tahun satu akan membuat layar menampilkannya sebagai tanggal yang sungguhan.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func formatTimePointer(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}
