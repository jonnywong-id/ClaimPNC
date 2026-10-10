// Package inputacceptationhttp adalah lapisan transport modul Acceptation Claim.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe domain
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya — dan
// di modul ini bocornya akan nyata: Detail membawa Reference, kunci teknis Pega yang tidak
// pernah digambar dan tidak boleh diterima balik dari klien.
package inputacceptationhttp

import (
	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/inputacceptation/usecase"
)

// FieldDTO adalah satu isian skalar beserta nilainya.
//
// Bentuk layar dan ISINYA dikirim bersama, bukan lewat dua endpoint. Layar ini selalu dibuka
// untuk satu klaim tertentu; memisahkannya berarti dua perjalanan untuk satu layar, dan
// kemungkinan keduanya menjawab keadaan yang berbeda.
type FieldDTO struct {
	Key   string `json:"kunci"`
	Title string `json:"judul"`

	// Value adalah isi yang digambar. Kosong bila isiannya tidak ada di dokumen klaim.
	Value string `json:"nilai"`

	// Editable menyatakan isian ini dapat diubah petugas dan boleh ikut dikirim saat Submit.
	Editable bool `json:"dapat_diubah"`

	// Ketiga isian berikut menyatakan isian yang digambar tetapi belum dapat diisi.
	//
	// Ia dikirim sebagai DATA, bukan ditulis tetap di layar, supaya hilang dengan sendirinya
	// begitu penghalangnya hilang — tanpa menyunting frontend.
	Blocked       bool   `json:"terhalang"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
	BlockedOwner  string `json:"pemilik_penghalang,omitempty"`
}

// GridColumnDTO adalah satu kolom grid.
type GridColumnDTO struct {
	Key      string `json:"kunci"`
	Title    string `json:"judul"`
	Editable bool   `json:"dapat_diubah"`
}

// GridDTO adalah satu tabel beserta barisnya.
type GridDTO struct {
	Code    string          `json:"kode"`
	Title   string          `json:"judul"`
	Columns []GridColumnDTO `json:"kolom"`

	// Rows adalah barisnya, berurutan. Senarai KOSONG, bukan null: `[]` dan `null` ditangani
	// berbeda oleh klien, dan yang kedua memaksa setiap layar memeriksanya lebih dulu.
	Rows []map[string]string `json:"baris"`

	Blocked       bool   `json:"terhalang"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
	BlockedOwner  string `json:"pemilik_penghalang,omitempty"`
}

// GroupDTO adalah satu kelompok isian beserta grid yang digambar sesudahnya.
type GroupDTO struct {
	Code   string     `json:"kode"`
	Title  string     `json:"judul"`
	Fields []FieldDTO `json:"isian"`
	Grids  []GridDTO  `json:"tabel"`
}

// DetailResponse adalah jawaban GET /api/input-acceptation/{no_klaim}.
type DetailResponse struct {
	// ClaimID adalah nomor klaim yang dibaca pengguna.
	ClaimID string `json:"no_klaim"`

	// StatusWork adalah status ALUR KERJA Pega, bukan Status Klaim berkode 1134–1166.
	StatusWork string `json:"status_kerja"`

	// LastUpdateOperator adalah petugas yang terakhir mengubah objek kerjanya.
	LastUpdateOperator string `json:"operator_pengubah"`

	Groups []GroupDTO `json:"kelompok"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal yang
	// sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`
}

// SubmitRequest adalah muatan POST /api/input-acceptation/{no_klaim}.
//
// # Kunci teknis TIDAK diterima dari klien
//
// Yang menunjuk objek kerja adalah nomor klaim di alamat, dan kunci teknisnya dibaca ulang
// server. Kunci teknis yang datang dari klien adalah kunci yang dapat ditukar klien — dan pada
// layar yang menetapkan Nomor Akseptasi, menukarnya berarti menulisi klaim yang lain.
type SubmitRequest struct {
	// Values memuat isian skalar yang diubah, dikunci nama isian.
	Values map[string]string `json:"isian"`

	// Grids memuat baris grid yang diubah, dikunci kode grid.
	//
	// Baris dikirim UTUH per grid — bukan sebagai selisih — karena urutan baris bermakna di
	// layar ini, dan selisih tanpa urutan tidak dapat diterapkan kembali dengan pasti.
	Grids map[string][]map[string]string `json:"tabel"`
}

// SubmitResponse adalah jawaban Submit yang berhasil.
type SubmitResponse struct {
	ClaimID string `json:"no_klaim"`
	Message string `json:"pesan"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toDetailResponse merakit jawaban satu rincian.
//
// Bentuk layar datang dari katalog dan ISINYA dari rincian; keduanya dirakit di sini supaya
// layar tidak perlu memasangkannya sendiri — dan supaya isian yang ada di katalog tetapi tidak
// ada di dokumen tetap DIGAMBAR, dengan nilai kosong.
func toDetailResponse(
	meta usecase.Metadata,
	detail inputacceptation.Detail,
	portalAlias string,
) DetailResponse {
	gridByCode := map[string]inputacceptation.Grid{}
	for _, grid := range meta.Grids {
		gridByCode[grid.Code] = grid
	}

	groups := make([]GroupDTO, 0, len(meta.Groups))
	for _, group := range meta.Groups {
		fields := make([]FieldDTO, 0, len(group.Fields))
		for _, field := range group.Fields {
			fields = append(fields, FieldDTO{
				Key:           field.Key,
				Title:         field.Title,
				Value:         detail.Get(field.Key),
				Editable:      field.Editable,
				Blocked:       field.Blocked,
				BlockedReason: field.BlockedReason,
				BlockedOwner:  field.BlockedOwner,
			})
		}

		grids := make([]GridDTO, 0, len(group.Grids))
		for _, code := range group.Grids {
			grid, known := gridByCode[code]
			if !known {
				// Tidak mungkin terjadi: uji katalog menjaga setiap kode grid yang
				// disebut kelompok memang ada. Dilewati alih-alih panik supaya satu
				// kekeliruan katalog tidak menjatuhkan seluruh layar.
				continue
			}

			columns := make([]GridColumnDTO, 0, len(grid.Columns))
			for _, column := range grid.Columns {
				columns = append(columns, GridColumnDTO{
					Key:      column.Key,
					Title:    column.Title,
					Editable: column.Editable,
				})
			}

			rows := make([]map[string]string, 0)
			for _, row := range detail.Rows(code) {
				rows = append(rows, map[string]string(row))
			}

			grids = append(grids, GridDTO{
				Code:          grid.Code,
				Title:         grid.Title,
				Columns:       columns,
				Rows:          rows,
				Blocked:       grid.Blocked,
				BlockedReason: grid.BlockedReason,
				BlockedOwner:  grid.BlockedOwner,
			})
		}

		groups = append(groups, GroupDTO{
			Code:   group.Code,
			Title:  group.Title,
			Fields: fields,
			Grids:  grids,
		})
	}


	return DetailResponse{
		ClaimID:            detail.ClaimID,
		StatusWork:         detail.StatusWork,
		LastUpdateOperator: detail.LastUpdateOperator,
		Groups:             groups,
		Portal:             portalAlias,
	}
}

// toGridRows mengubah muatan Submit menjadi bentuk domain.
func toGridRows(raw map[string][]map[string]string) map[string][]inputacceptation.GridRow {
	if raw == nil {
		return nil
	}
	result := make(map[string][]inputacceptation.GridRow, len(raw))
	for code, rows := range raw {
		converted := make([]inputacceptation.GridRow, 0, len(rows))
		for _, row := range rows {
			converted = append(converted, inputacceptation.GridRow(row))
		}
		result[code] = converted
	}
	return result
}
