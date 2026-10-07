// Package monitoringslinkojkhttp adalah lapisan transport modul Monitoring SLINK OJK.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya.
package monitoringslinkojkhttp

import (
	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/usecase"
	"claim-pnc/internal/platform/apierror"
)

// ColumnDTO adalah satu kolom katalog.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`.
type ColumnDTO struct {
	Key    string `json:"kunci"`
	Header string `json:"judul"`

	// Available menyatakan kolomnya punya sumber data.
	//
	// # Kenapa ini dikirim ke layar, dan bukan disembunyikan
	//
	// Segmen F06 punya 38 kolom dan hanya 8 di antaranya terisi. Tanpa penanda ini,
	// tiga puluh sel kosong terbaca sebagai "datanya memang kosong" — padahal artinya
	// "kolomnya belum punya sumber". Pada layar pemantauan laporan ke OJK, kedua
	// keadaan itu menuntut tindakan yang sama sekali berbeda.
	Available bool `json:"tersedia"`

	// Numeric menandai kolom bernilai angka, supaya layar meratakannya ke kanan.
	Numeric bool `json:"angka"`

	// LegacyProperty adalah nama properti Pega yang mengisinya; kosong bila tidak ada.
	//
	// Ia dikirim supaya keterangan "kolom ini dulunya terikat properti apa" dapat
	// ditampilkan sebagai tooltip pada kolom yang belum bersumber — jawaban atas
	// pertanyaan yang pasti muncul begitu pengguna melihat kolom kosong.
	LegacyProperty string `json:"properti_lama,omitempty"`
}

func toColumnDTO(column monitoringslinkojk.Column) ColumnDTO {
	return ColumnDTO{
		Key:            column.Key,
		Header:         column.Header,
		Available:      column.Source == monitoringslinkojk.SourceAvailable,
		Numeric:        column.Numeric,
		LegacyProperty: column.LegacyProperty,
	}
}

func toColumnDTOs(columns []monitoringslinkojk.Column) []ColumnDTO {
	// Senarai KOSONG, bukan nil: JSON `null` memaksa setiap layar memeriksa dua bentuk
	// "tidak ada", dan yang satu mudah terlupakan.
	out := make([]ColumnDTO, 0, len(columns))
	for _, column := range columns {
		out = append(out, toColumnDTO(column))
	}
	return out
}

// SegmentDTO adalah satu segmen beserta katalog kolomnya.
type SegmentDTO struct {
	Code   string `json:"kode"`
	Label  string `json:"label"`
	Column int    `json:"jumlah_kolom"`

	// AvailableColumn adalah banyaknya kolom yang benar-benar terisi. Lihat
	// ColumnDTO.Available.
	AvailableColumn int `json:"jumlah_kolom_tersedia"`

	Columns []ColumnDTO `json:"kolom"`
}

func toSegmentDTO(info usecase.SegmentInfo) SegmentDTO {
	return SegmentDTO{
		Code:            string(info.Segment),
		Label:           info.Label,
		Column:          len(info.Columns),
		AvailableColumn: info.AvailableCount,
		Columns:         toColumnDTOs(info.Columns),
	}
}

// ScopeDTO adalah satu pilihan dropdown "Business Name".
type ScopeDTO struct {
	Value string `json:"nilai"`
	Label string `json:"label"`

	// Note menjelaskan arti pilihannya.
	//
	// Ia ada karena satu pilihan di layar ini MENIADAKAN alih-alih memilih: "SURETY
	// BOND" menyaring seluruh lini SELAIN Asuransi Kredit, bukan lini Surety Bond saja.
	// Tanpa keterangan, pengguna yang memilihnya akan membaca hasilnya sebagai daftar
	// Surety Bond — dan salah membaca laporan regulator.
	Note string `json:"keterangan,omitempty"`
}

// DescribeResponse adalah jawaban keterangan layar: kedua segmen beserta katalognya.
type DescribeResponse struct {
	Segments []SegmentDTO `json:"segmen"`
	Scopes   []ScopeDTO   `json:"business_name"`

	// RowKey adalah nama field kunci baris di dalam setiap baris. Dikirim supaya layar
	// tidak perlu menuliskannya ulang sebagai konstanta yang dapat menyimpang.
	RowKey string `json:"kunci_baris"`
}

func toDescribeResponse(segments []usecase.SegmentInfo) DescribeResponse {
	out := DescribeResponse{
		Segments: make([]SegmentDTO, 0, len(segments)),
		Scopes:   make([]ScopeDTO, 0, 2),
		RowKey:   monitoringslinkojk.RowKeyColumn,
	}
	for _, info := range segments {
		out.Segments = append(out.Segments, toSegmentDTO(info))
	}
	for _, scope := range monitoringslinkojk.BusinessScopes() {
		item := ScopeDTO{Value: string(scope), Label: string(scope)}
		if scope == monitoringslinkojk.ScopeSuretyBond {
			item.Note = "Menampilkan seluruh lini SELAIN Asuransi Kredit."
		}
		out.Scopes = append(out.Scopes, item)
	}
	return out
}

// PageMetaDTO adalah keterangan halaman.
type PageMetaDTO struct {
	Page      int `json:"halaman"`
	Size      int `json:"ukuran"`
	Total     int `json:"total"`
	TotalPage int `json:"total_halaman"`
}

// SearchResponse adalah jawaban tombol "Cari Data".
type SearchResponse struct {
	Segment SegmentDTO  `json:"segmen"`
	Rows    []RowDTO    `json:"baris"`
	Meta    PageMetaDTO `json:"meta"`
}

// RowDTO adalah satu baris grid.
//
// Peta `kunci kolom -> teks`, bukan struct: kolomnya berupa katalog, dan 58 field yang
// harus diulang di sini akan menjadi tempat keempat yang harus berubah setiap kali satu
// kolom bergeser.
type RowDTO map[string]string

func toSearchResponse(
	info usecase.SegmentInfo,
	page monitoringslinkojk.Page,
	filter monitoringslinkojk.Filter,
) SearchResponse {
	rows := make([]RowDTO, 0, len(page.Rows))
	for _, row := range page.Rows {
		rows = append(rows, toRowDTO(info.Columns, row))
	}

	totalPage := 0
	if filter.Size > 0 {
		totalPage = (page.Total + filter.Size - 1) / filter.Size
	}

	return SearchResponse{
		Segment: toSegmentDTO(info),
		Rows:    rows,
		Meta: PageMetaDTO{
			Page:      filter.Page,
			Size:      filter.Size,
			Total:     page.Total,
			TotalPage: totalPage,
		},
	}
}

// toRowDTO menyalin satu baris, MEMASTIKAN setiap kolom katalog punya kuncinya.
//
// Kolom tanpa sumber dikirim sebagai teks kosong, bukan dihilangkan. Alasannya di sisi
// layar: tabel yang menggambar 38 kolom dari katalog akan menampilkan `undefined` pada
// kunci yang tidak ada, dan `undefined` di sel laporan regulator terbaca seperti cacat
// sistem alih-alih seperti kolom yang memang belum bersumber.
//
// Kunci barisnya ikut dibawa meski bukan kolom — lihat monitoringslinkojk.RowKeyColumn.
func toRowDTO(columns []monitoringslinkojk.Column, row monitoringslinkojk.Row) RowDTO {
	out := make(RowDTO, len(columns)+1)
	out[monitoringslinkojk.RowKeyColumn] = row.Get(monitoringslinkojk.RowKeyColumn)
	for _, column := range columns {
		out[column.Key] = row.Get(column.Key)
	}
	return out
}

// ViolationDTO adalah satu pelanggaran validasi.
type ViolationDTO = apierror.FieldError

// ErrorResponse adalah bentuk baku galat modul ini.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}
