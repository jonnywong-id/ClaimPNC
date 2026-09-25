// Package reportkpihttp adalah lapisan transport modul Report KPI PNC.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: `reportkpi.Score` membawa penanda
// `Present` yang tidak punya arti apa pun bagi layar, sementara yang dibutuhkan layar
// justru kebalikannya, yaitu nilai yang boleh `null`.
package reportkpihttp

import (
	"claim-pnc/internal/reportkpi"
	"claim-pnc/internal/reportkpi/usecase"
)

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	// Key menyebut isian mana pada baris yang digambar.
	Key string `json:"kunci"`

	// Title adalah judul kolom yang dibaca pengguna.
	Title string `json:"judul"`

	// OnlyOnCombinedType menyatakan kolom ini HANYA digambar pada tipe report `ALL`.
	//
	// Layar yang menyaringnya, bukan server, karena metadata tidak tahu tipe report yang
	// sedang dipilih — ia keterangan layar, bukan jawaban permintaan. Berkas ekspor
	// disaring di server lewat `Grid.ColumnsFor`, dan keduanya bertanya kepada penanda
	// yang sama sehingga tidak dapat berselisih.
	OnlyOnCombinedType bool `json:"hanya_tipe_gabungan,omitempty"`

	// OnlyOnGroup membatasi kolom pada satu kelompok tab KPI Admin. Kosong berarti
	// berlaku di kedua kelompok.
	OnlyOnGroup string `json:"hanya_kelompok,omitempty"`
}

// ComponentDTO adalah satu komponen penilaian KPI.
type ComponentDTO struct {
	Code  string `json:"kode"`
	Title string `json:"judul"`

	// Column adalah nama kolom basis datanya.
	//
	// Ia DIKIRIM ke layar, dan itu disengaja: penguji gerbang 1 membandingkan angka di
	// layar ini dengan angka di Pega, dan yang pertama ditanyakannya selalu "ini kolom
	// yang mana". Menyebutnya di layar menghemat satu perjalanan ke dokumen.
	Column string `json:"kolom"`
}

// ReportTypeDTO adalah satu pilihan dropdown "Pilih Tipe Report".
type ReportTypeDTO struct {
	Code  string `json:"kode"`
	Title string `json:"judul"`
	Note  string `json:"keterangan,omitempty"`
}

// GridDTO adalah satu grid pada sebuah tab.
type GridDTO struct {
	Code  string `json:"kode"`
	Title string `json:"judul"`

	// Columns adalah kolom TETAP grid ini — belum termasuk kesembilan komponen.
	//
	// Layar merakit kolomnya dengan menyambung daftar ini dengan `komponen` pada akar
	// jawaban. Keduanya dipisah karena kesembilan komponen itu sama di kedua grid;
	// mengirimnya dua kali berarti dua daftar yang dapat berselisih.
	Columns []ColumnDTO `json:"kolom"`
}

// TabDTO adalah satu tab layar.
type TabDTO struct {
	Code  string `json:"kode"`
	Title string `json:"judul"`

	Grids []GridDTO `json:"grid"`

	// Kedua isian berikut menyatakan tab yang digambar tetapi belum dapat diisi.
	//
	// Tab terhalang tetap DIKIRIM, bukan disembunyikan. Keputusan Work Owner 2026-09-18
	// untuk butir menu berlaku sama di sini: kemajuan migrasi terbaca langsung dari layar,
	// dan pengguna tidak melaporkan tab yang "hilang".
	Blocked       bool   `json:"terhalang"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
}

// MetadataResponse adalah jawaban GET /api/report-kpi/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	// Components adalah kesembilan komponen, dalam urutan kolom layar lama.
	Components []ComponentDTO `json:"komponen"`

	// ReportTypes adalah isi dropdown "Pilih Tipe Report".
	ReportTypes []ReportTypeDTO `json:"tipe_report"`

	// PlannedDifferences adalah selisih terencana tab KPI Adjuster.
	PlannedDifferences []string `json:"selisih_terencana"`

	// AdminGroups adalah isi dropdown "Pilih Data KPI" pada tab KPI Admin.
	AdminGroups []AdminGroupDTO `json:"kelompok_admin"`

	// AdminPlannedDifferences adalah selisih terencana tab KPI Admin.
	AdminPlannedDifferences []string `json:"selisih_terencana_admin"`

	// CoordinatorInQuery adalah nama koordinator sebagaimana ditulis di TEKS KUERI lama,
	// yang BERBEDA dari yang ditampilkan. Dikirim supaya layar dapat menjelaskan
	// selisihnya kepada penguji yang membandingkan layar ini dengan rule Pega.
	CoordinatorInQuery string `json:"koordinator_di_kueri"`

	// BusinessLines adalah isi dropdown lini bisnis pada tab KPI PIC Teknik.
	BusinessLines []BusinessLineDTO `json:"lini_bisnis"`

	// PICComponents adalah keempat komponen penilaian PIC Teknik.
	PICComponents []PICComponentDTO `json:"komponen_pic"`

	// PICTeknikPlannedDifferences adalah selisih terencana tab KPI PIC Teknik.
	PICTeknikPlannedDifferences []string `json:"selisih_terencana_pic"`

	// SLAExcludedPICs adalah petugas yang dikecualikan dari penilaian SLA di sistem lama.
	SLAExcludedPICs []string `json:"pic_dikecualikan_sla"`

	// SourceTable disebutkan supaya penguji tahu tabel mana yang dibandingkan.
	SourceTable string `json:"tabel_sumber"`

	// BandTable adalah tabel tangga nilai tab KPI PIC Teknik.
	//
	// Disebut terpisah dari SourceTable karena tabnya memang membaca tabel yang berbeda —
	// dan penguji yang membandingkan nilai 1–5 akan mencari tabel inilah, bukan yang lain.
	BandTable string `json:"tabel_tangga_nilai"`
}

// FilterDTO adalah penyaring yang BENAR-BENAR dipakai menjawab permintaan.
//
// Ia dikirim balik, bukan diasumsikan sama dengan yang dikirim layar: tipe report yang
// dikirim huruf kecil dibakukan menjadi huruf besar, dan layar harus menyatakan penyaring
// yang sebenarnya berlaku — bukan yang ia kira berlaku.
type FilterDTO struct {
	ReportType string `json:"tipe_report"`

	// Adjuster kosong berarti seluruh adjuster.
	Adjuster string `json:"adjuster"`

	From string `json:"dari"`
	To   string `json:"sampai"`
}

// SummaryRowDTO adalah satu baris grid Summary.
//
// Kesembilan nilainya berada di dalam `nilai`, dikunci kode komponen — bukan sebagai
// sembilan field tersendiri. Alasannya: layar menggambar kolomnya dari `komponen` pada
// metadata, sehingga menambah komponen kelak menjadi perubahan DATA, bukan perubahan
// kontrak yang menuntut kedua sisi ikut disunting.
type SummaryRowDTO struct {
	Adjuster   string `json:"adjuster"`
	ReportType string `json:"tipe"`

	// Scores memetakan kode komponen ke nilainya. `null` berarti komponen itu tidak punya
	// satu pun nilai yang terbaca — BUKAN bernilai nol.
	Scores map[string]*float64 `json:"nilai"`
}

// DetailRowDTO adalah satu baris grid Detail.
type DetailRowDTO struct {
	Adjuster   string `json:"adjuster"`
	CaseID     string `json:"no_case"`
	ReportType string `json:"tipe"`

	// ScoredOn berbentuk `YYYY-MM-DD`. Kosong berarti kolomnya kosong di basis data.
	ScoredOn string `json:"tanggal"`

	Scores map[string]*float64 `json:"nilai"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// SummaryResponse adalah jawaban GET /api/report-kpi/adjuster/ringkasan.
//
// Ia TIDAK punya paginasi, dan itu bukan kelalaian: grid Summary menghasilkan satu baris
// per adjuster — puluhan, bukan puluhan ribu — dan layar lama pun memuatnya sekaligus.
type SummaryResponse struct {
	Rows   []SummaryRowDTO `json:"baris"`
	Filter FilterDTO       `json:"penyaring"`
	Portal string          `json:"portal"`
}

// DetailResponse adalah jawaban GET /api/report-kpi/adjuster.
type DetailResponse struct {
	Rows       []DetailRowDTO `json:"baris"`
	Pagination PaginationDTO  `json:"paginasi"`
	Filter     FilterDTO      `json:"penyaring"`
	Portal     string         `json:"portal"`
}

// AdjusterListResponse adalah jawaban GET /api/report-kpi/adjuster/pilihan.
type AdjusterListResponse struct {
	Adjusters []string  `json:"adjuster"`
	Filter    FilterDTO `json:"penyaring"`
	Portal    string    `json:"portal"`
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

// toMetadataResponse menyusun jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		grids := make([]GridDTO, 0, len(tab.Grids))
		for _, grid := range tab.Grids {
			columns := make([]ColumnDTO, 0, len(grid.Columns))
			for _, column := range grid.Columns {
				columns = append(columns, ColumnDTO{
					Key:                column.Key,
					Title:              column.Title,
					OnlyOnCombinedType: column.OnlyOnCombinedType,
					OnlyOnGroup:        string(column.OnlyOnGroup),
				})
			}
			grids = append(grids, GridDTO{
				Code: grid.Code, Title: grid.Title, Columns: columns,
			})
		}

		tabs = append(tabs, TabDTO{
			Code:          tab.Code,
			Title:         tab.Title,
			Grids:         grids,
			Blocked:       tab.Blocked,
			BlockedReason: tab.BlockedReason,
		})
	}

	componentList := make([]ComponentDTO, 0, len(meta.Components))
	for _, c := range meta.Components {
		componentList = append(componentList, ComponentDTO{
			Code: c.Code, Title: c.Label, Column: c.Column,
		})
	}

	typeList := make([]ReportTypeDTO, 0, len(meta.ReportTypes))
	for _, t := range meta.ReportTypes {
		typeList = append(typeList, ReportTypeDTO{
			Code: string(t.Code), Title: t.Label, Note: t.Note,
		})
	}

	differences := make([]string, 0, len(meta.PlannedDifferences))
	differences = append(differences, meta.PlannedDifferences...)

	adminGroups := make([]AdminGroupDTO, 0, len(meta.AdminGroups))
	for _, g := range meta.AdminGroups {
		adminGroups = append(adminGroups, AdminGroupDTO{
			Code: string(g.Code), Title: g.Label, Note: g.Note,
		})
	}

	adminDifferences := make([]string, 0, len(meta.AdminPlannedDifferences))
	adminDifferences = append(adminDifferences, meta.AdminPlannedDifferences...)

	return MetadataResponse{
		Tabs:                    tabs,
		DefaultTab:              meta.DefaultTab,
		Components:              componentList,
		ReportTypes:             typeList,
		PlannedDifferences:      differences,
		AdminGroups:             adminGroups,
		AdminPlannedDifferences: adminDifferences,
		CoordinatorInQuery:      meta.CoordinatorInQuery,

		BusinessLines: toBusinessLines(meta.BusinessLines),
		PICComponents: toPICComponents(meta.PICComponents),
		PICTeknikPlannedDifferences: append(
			make([]string, 0, len(meta.PICTeknikPlannedDifferences)),
			meta.PICTeknikPlannedDifferences...,
		),
		SLAExcludedPICs: append(
			make([]string, 0, len(meta.SLAExcludedPICs)),
			meta.SLAExcludedPICs...,
		),

		SourceTable: reportkpi.SourceTable,
		BandTable:   reportkpi.BandTable,
	}
}

// toFilterDTO menyusun keterangan penyaring yang dipakai.
func toFilterDTO(query reportkpi.Query) FilterDTO {
	return FilterDTO{
		ReportType: string(query.ReportType),
		Adjuster:   query.Adjuster,
		From:       query.Range.From,
		To:         query.Range.To,
	}
}

// toScoreDTO mengubah peta nilai domain menjadi peta yang boleh memuat `null`.
//
// Yang tidak ada menjadi `null`, BUKAN 0. Keduanya berbeda artinya pada laporan penilaian
// kinerja, dan hanya `null` yang dapat digambar layar sebagai tanda hubung.
//
// Seluruh kode komponen SELALU ada sebagai kunci, meski nilainya `null`. Layar menggambar
// kolomnya dari metadata, dan kunci yang hilang akan membuat sel-nya kosong tanpa dapat
// dibedakan dari komponen yang memang belum dinilai.
func toScoreDTO(scores map[string]reportkpi.Score) map[string]*float64 {
	result := make(map[string]*float64, len(reportkpi.ComponentCodes()))

	for _, code := range reportkpi.ComponentCodes() {
		score, exists := scores[code]
		if !exists || !score.Present {
			result[code] = nil
			continue
		}
		value := score.Value
		result[code] = &value
	}
	return result
}

// toSummaryRows mengubah baris ringkasan.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toSummaryRows(rows []reportkpi.AdjusterSummary) []SummaryRowDTO {
	result := make([]SummaryRowDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, SummaryRowDTO{
			Adjuster:   row.Adjuster,
			ReportType: string(row.ReportType),
			Scores:     toScoreDTO(row.Scores),
		})
	}
	return result
}

// toDetailRows mengubah baris rincian.
func toDetailRows(rows []reportkpi.AdjusterDetail) []DetailRowDTO {
	result := make([]DetailRowDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, DetailRowDTO{
			Adjuster:   row.Adjuster,
			CaseID:     row.CaseID,
			ReportType: string(row.ReportType),
			ScoredOn:   row.ScoredOn,
			Scores:     toScoreDTO(row.Scores),
		})
	}
	return result
}

// toPaginationDTO menyusun keterangan halaman.
func toPaginationDTO(page reportkpi.Pagination, total int) PaginationDTO {
	clean := page.Normalize()

	totalPages := 0
	if clean.Size > 0 {
		totalPages = (total + clean.Size - 1) / clean.Size
	}

	return PaginationDTO{
		Page:       clean.Page,
		Size:       clean.Size,
		Total:      total,
		TotalPages: totalPages,
	}
}
