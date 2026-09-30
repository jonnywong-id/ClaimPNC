// Package outstandingclaimhttp adalah lapisan transport modul Outstanding Claim.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe domain
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya — dan
// di modul ini bocornya akan nyata: Field dan Grid membawa `Path`, jalur di dalam dokumen
// klaim Pega, yang tidak ada urusannya dengan layar.
package outstandingclaimhttp

import (
	"claim-pnc/internal/outstandingclaim"
	"claim-pnc/internal/outstandingclaim/usecase"
)

// FieldDTO adalah satu isian skalar pada layar rincian.
//
// `Path` TIDAK ikut dikirim. Ia jalur di dalam dokumen klaim Pega — pengetahuan penyimpanan,
// bukan pengetahuan layar — dan mengirimkannya berarti bentuk penyimpanan lama ikut menjadi
// kontrak yang harus dipertahankan.
type FieldDTO struct {
	Key   string `json:"kunci"`
	Title string `json:"judul"`

	// Blocked menyatakan isian ini digambar tetapi belum dapat diisi.
	//
	// Ia dikirim sebagai DATA supaya penandanya hilang dengan sendirinya begitu
	// penghalangnya hilang — tanpa menyunting frontend.
	Blocked bool `json:"terhalang,omitempty"`
}

// GridColumnDTO adalah satu kolom grid.
type GridColumnDTO struct {
	Key   string `json:"kunci"`
	Title string `json:"judul"`
}

// GridDTO adalah susunan satu tabel pada layar rincian.
type GridDTO struct {
	Code    string          `json:"kode"`
	Title   string          `json:"judul"`
	Columns []GridColumnDTO `json:"kolom"`

	Blocked       bool   `json:"terhalang,omitempty"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
	BlockedOwner  string `json:"pemilik_penghalang,omitempty"`
}

// GroupDTO adalah satu kelompok isian beserta grid yang digambar sesudahnya.
type GroupDTO struct {
	Code   string     `json:"kode"`
	Title  string     `json:"judul"`
	Fields []FieldDTO `json:"isian"`

	// Grids menyebut KODE grid, bukan susunannya. Susunan lengkapnya ada sekali saja di
	// LayoutResponse.Grids — menyalinnya ke tiap kelompok membuat satu grid yang dipakai
	// dua kelompok punya dua susunan yang dapat berselisih.
	Grids []string `json:"grid"`
}

// LayoutResponse adalah jawaban GET /api/outstanding-claim/tata-letak.
type LayoutResponse struct {
	Groups []GroupDTO `json:"kelompok"`
	Grids  []GridDTO  `json:"grid"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal yang
	// sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`
}

// DetailResponse adalah jawaban GET /api/outstanding-claim/{no_klaim}.
type DetailResponse struct {
	// Keadaan objek kerja.
	ClaimID            string `json:"no_klaim"`
	StatusWork         string `json:"status_kerja"`
	LastUpdateOperator string `json:"operator_pengubah"`

	// Values memuat isian skalar, dikunci nama isian pada FieldDTO.
	//
	// Isian yang TIDAK ada kuncinya di sini berbeda artinya dari isian yang kuncinya ada
	// tetapi kosong: yang pertama berarti jalurnya tidak ditemukan di dokumen klaim, yang
	// kedua berarti datanya memang belum diisi. Layar menggambar keduanya sebagai tanda
	// pisah, tetapi perbedaannya terbawa supaya dapat diperiksa dari respons.
	Values map[string]string `json:"isian"`

	// Rows memuat baris setiap grid, dikunci kode grid.
	Rows map[string][]map[string]string `json:"baris"`

	Portal string `json:"portal"`
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

// toLayoutResponse merakit jawaban susunan layar.
func toLayoutResponse(layout usecase.Layout, portalAlias string) LayoutResponse {
	groups := make([]GroupDTO, 0, len(layout.Groups))
	for _, group := range layout.Groups {
		fields := make([]FieldDTO, 0, len(group.Fields))
		for _, field := range group.Fields {
			fields = append(fields, FieldDTO{
				Key:     field.Key,
				Title:   field.Title,
				Blocked: field.Blocked,
			})
		}

		gridCodes := make([]string, 0, len(group.Grids))
		gridCodes = append(gridCodes, group.Grids...)

		groups = append(groups, GroupDTO{
			Code:   group.Code,
			Title:  group.Title,
			Fields: fields,
			Grids:  gridCodes,
		})
	}

	grids := make([]GridDTO, 0, len(layout.Grids))
	for _, grid := range layout.Grids {
		columns := make([]GridColumnDTO, 0, len(grid.Columns))
		for _, column := range grid.Columns {
			columns = append(columns, GridColumnDTO{Key: column.Key, Title: column.Title})
		}

		grids = append(grids, GridDTO{
			Code:          grid.Code,
			Title:         grid.Title,
			Columns:       columns,
			Blocked:       grid.Blocked,
			BlockedReason: grid.BlockedReason,
			BlockedOwner:  grid.BlockedOwner,
		})
	}

	return LayoutResponse{Groups: groups, Grids: grids, Portal: portalAlias}
}

// toDetailResponse merakit jawaban isi satu klaim.
//
// Peta dan senarai selalu dibentuk, tidak pernah nil: `{}` dan `null` ditangani berbeda oleh
// klien, dan yang kedua memaksa setiap layar memeriksanya lebih dulu.
func toDetailResponse(
	detail outstandingclaim.Detail,
	portalAlias string,
) DetailResponse {
	values := make(map[string]string, len(detail.Values))
	for key, value := range detail.Values {
		values[key] = value
	}

	rows := make(map[string][]map[string]string, len(detail.Grids))
	for code, list := range detail.Grids {
		converted := make([]map[string]string, 0, len(list))
		for _, row := range list {
			cells := make(map[string]string, len(row))
			for key, value := range row {
				cells[key] = value
			}
			converted = append(converted, cells)
		}
		rows[code] = converted
	}

	return DetailResponse{
		ClaimID:            detail.ClaimID,
		StatusWork:         detail.StatusWork,
		LastUpdateOperator: detail.LastUpdateOperator,
		Values:             values,
		Rows:               rows,
		Portal:             portalAlias,
	}
}
