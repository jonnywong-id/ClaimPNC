package masterpenolakanhttp

import "claim-pnc/internal/masterpenolakan"

// CommitteeRejectionDTO adalah bentuk satu baris Master Penolakan Komite yang dikirim ke
// peramban.
type CommitteeRejectionDTO struct {
	// ID adalah kolom IDMASTER. Diterbitkan server; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// Note adalah kolom NOTEMASTER — di layar lama berlabel "Note Komite Reject".
	Note string `json:"catatan"`
}

// KomiteListResponse adalah jawaban GET /api/master/penolakan-komite.
type KomiteListResponse struct {
	CommitteeRejection []CommitteeRejectionDTO `json:"penolakan_komite"`
	Portal             string                  `json:"portal"`
}

// KomiteSingleResponse adalah jawaban penambahan dan penyuntingan.
type KomiteSingleResponse struct {
	CommitteeRejection CommitteeRejectionDTO `json:"penolakan_komite"`
	Portal             string                `json:"portal"`
}

// KomiteSaveRequest adalah badan permintaan penambahan dan penyuntingan.
//
// Satu isian saja, persis seperti layar lama. ID diturunkan dari isi tabel pada
// penambahan, dan diambil dari jalur pada penyuntingan.
type KomiteSaveRequest struct {
	Note string `json:"catatan"`
}

// toKomiteDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toKomiteDTO(rejection masterpenolakan.CommitteeRejection) CommitteeRejectionDTO {
	return CommitteeRejectionDTO{ID: rejection.ID, Note: rejection.Note}
}

// toKomiteListDTO mengubah sekumpulan baris domain; slice-nya tidak pernah nil.
func toKomiteListDTO(list []masterpenolakan.CommitteeRejection) []CommitteeRejectionDTO {
	result := make([]CommitteeRejectionDTO, 0, len(list))
	for _, rejection := range list {
		result = append(result, toKomiteDTO(rejection))
	}
	return result
}
