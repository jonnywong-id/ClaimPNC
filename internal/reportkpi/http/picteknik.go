package reportkpihttp

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"claim-pnc/internal/reportkpi"
)

// Berkas ini adalah lapisan transport tab **KPI PIC Teknik**.

// BusinessLineDTO adalah satu pilihan dropdown lini bisnis.
type BusinessLineDTO struct {
	Code  string `json:"kode"`
	Title string `json:"judul"`
	Note  string `json:"keterangan,omitempty"`
}

// PICComponentDTO adalah satu komponen penilaian PIC Teknik.
type PICComponentDTO struct {
	Code  string `json:"kode"`
	Label string `json:"judul"`

	// Descending menyatakan tangga nilainya MENURUN — makin kecil persentasenya, makin
	// tinggi nilainya.
	//
	// Ia dikirim sebagai data supaya layar dapat menandai baris itu. Tanpa penanda, pembaca
	// yang melihat "95% → nilai 1" akan menyimpulkan angkanya rusak — padahal itulah yang
	// sedang berjalan di Pega hari ini.
	Descending bool `json:"tangga_menurun,omitempty"`

	// Weighted menyatakan komponen ini punya nilai berbobot terpisah, berskala 0–15.
	Weighted bool `json:"berbobot,omitempty"`
}

// PICRowDTO adalah satu baris penilaian pada kartu skor.
type PICRowDTO struct {
	Component string `json:"komponen"`
	Label     string `json:"judul"`

	Total    float64 `json:"total"`
	Achieved float64 `json:"tercapai"`

	// Percent dan Value `null` berarti tidak dapat dihitung, BUKAN nol.
	Percent *float64 `json:"persentase"`
	Value   *float64 `json:"nilai"`
}

// PICScorecardDTO adalah kartu skor satu PIC.
type PICScorecardDTO struct {
	PIC    string `json:"pic"`
	Leader bool   `json:"leader"`

	Rows []PICRowDTO `json:"baris"`

	// Weighted adalah nilai berbobot komponen Progress, berskala 0–15.
	//
	// Ia BUKAN pengganti nilai pita pada baris Progress — keduanya disimpan terpisah di
	// sistem lama dan keduanya ditampilkan.
	Weighted *float64 `json:"nilai_berbobot"`
}

// PICFilterDTO adalah penyaring yang benar-benar dipakai menjawab permintaan.
type PICFilterDTO struct {
	Line string `json:"lini_bisnis"`
	From string `json:"dari"`
	To   string `json:"sampai"`
}

// PICTeknikResponse adalah jawaban GET /api/report-kpi/pic-teknik.
type PICTeknikResponse struct {
	Scorecards []PICScorecardDTO `json:"kartu_skor"`

	// Leader adalah rekapitulasi seluruh PIC.
	Leader PICScorecardDTO `json:"rekapitulasi"`

	Filter PICFilterDTO `json:"penyaring"`
	Portal string       `json:"portal"`
}

// PICTeknik melayani GET /api/report-kpi/pic-teknik.
func (h *Handler) PICTeknik(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	scored, err := h.service.PICTeknik(r.Context(), active.Alias, readPICFilter(r), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, PICTeknikResponse{
		Scorecards: toPICScorecards(scored.Result.Scorecards),
		Leader:     toPICScorecard(scored.Result.Leader),
		Filter:     toPICFilterDTO(scored.Filter),
		Portal:     scored.Portal,
	})
}

// PICTeknikExport melayani GET /api/report-kpi/pic-teknik/ekspor.
//
// # Yang diekspor BUKAN kartu skor
//
// Sampai 2026-10-08 berkas ini berisi kartu skor — nama PIC dan nilainya. Itu tidak pernah
// dilakukan Pega. Pega mengekspor DATA KLAIM MENTAH, dan pilihan "Pilih Data KPI"
// menentukan kumpulan yang mana dari empat.
//
// Judul kolom berkasnya datang dari hasil kueri, bukan disusun di sini — lihat
// reportkpi.PICExportTable.
func (h *Handler) PICTeknikExport(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	kind, known := reportkpi.FindPICExportKind(r.URL.Query().Get("data_kpi"))
	if !known {
		h.writeError(w, r, reportkpi.NewValidationError([]reportkpi.Violation{{
			Field: "data_kpi",
			Message: "Pilihan data KPI tidak dikenal. Pilih salah satu dari daftar " +
				"\"Pilih Data KPI\".",
		}}))
		return
	}

	// Seluruh isinya diambil SEBELUM satu byte pun ditulis. Setelah header terkirim, galat
	// tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa berkas
	// separuh jadi tanpa satu pun keterangan.
	exported, err := h.service.PICTeknikExport(
		r.Context(), active.Alias, kind, readPICFilter(r), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.beginDownload(w, picExportFilename(exported.Filter, exported.Kind))

	writer := csv.NewWriter(w)
	if err := writer.Write(exported.Table.Header); err != nil {
		h.logExportFailure(r, err)
		return
	}
	for _, record := range exported.Table.Rows {
		if err := writer.Write(record); err != nil {
			h.logExportFailure(r, err)
			return
		}
	}

	h.flush(w, writer, r)
}

// readPICFilter membaca penyaring tab KPI PIC Teknik dari alamat permintaan.
func readPICFilter(r *http.Request) reportkpi.PICQueryInput {
	q := r.URL.Query()
	return reportkpi.PICQueryInput{
		Line: q.Get("lini_bisnis"),
		From: q.Get("dari"),
		To:   q.Get("sampai"),
	}
}

// toPICFilterDTO menyalin penyaring yang dipakai ke bentuk jawaban.
func toPICFilterDTO(query reportkpi.PICTeknikQuery) PICFilterDTO {
	return PICFilterDTO{
		Line: string(query.Line),
		From: query.Range.From,
		To:   query.Range.To,
	}
}

// toPICScorecards menyalin seluruh kartu skor.
func toPICScorecards(cards []reportkpi.PICScorecard) []PICScorecardDTO {
	// Selalu tatasusun kosong, tidak pernah `null`: layar memetakannya langsung, dan `null`
	// pada JSON menjadi kegagalan di sisi klien — bukan daftar kosong.
	out := make([]PICScorecardDTO, 0, len(cards))
	for _, card := range cards {
		out = append(out, toPICScorecard(card))
	}
	return out
}

// toPICScorecard menyalin satu kartu skor.
func toPICScorecard(card reportkpi.PICScorecard) PICScorecardDTO {
	rows := make([]PICRowDTO, 0, len(card.Rows))
	for _, row := range card.Rows {
		rows = append(rows, PICRowDTO{
			Component: row.Component,
			Label:     row.Label,
			Total:     row.Total,
			Achieved:  row.Achieved,
			Percent:   optionalNumber(row.Percent),
			Value:     optionalNumber(row.Value),
		})
	}
	return PICScorecardDTO{
		PIC:      card.PIC,
		Leader:   card.Leader,
		Rows:     rows,
		Weighted: optionalNumber(card.Weighted),
	}
}

// toBusinessLines menyalin pilihan lini bisnis ke bentuk jawaban.
func toBusinessLines(lines []reportkpi.BusinessLineOption) []BusinessLineDTO {
	out := make([]BusinessLineDTO, 0, len(lines))
	for _, line := range lines {
		out = append(out, BusinessLineDTO{
			Code:  string(line.Code),
			Title: line.Title,
			Note:  line.Note,
		})
	}
	return out
}

// toPICComponents menyalin keempat komponen ke bentuk jawaban.
func toPICComponents(components []reportkpi.PICComponent) []PICComponentDTO {
	out := make([]PICComponentDTO, 0, len(components))
	for _, component := range components {
		out = append(out, PICComponentDTO{
			Code:       component.Code,
			Label:      component.Label,
			Descending: component.DescendingBand,
			Weighted:   component.Weighted,
		})
	}
	return out
}

// picExportFilename menyusun nama berkas yang menyebut penyaringnya.
// picExportFilename menyusun nama berkas unduhan.
//
// Pilihan data KPI ikut masuk ke namanya. Keempat berkas dapat diunduh pada penyaring
// yang sama, dan tanpa pembeda itu keempatnya akan bernama sama persis di folder
// unduhan — lalu bertimpa diam-diam.
func picExportFilename(query reportkpi.PICTeknikQuery, kind reportkpi.PICExportKind) string {
	name := "kpi-pic-teknik-" + strings.ToLower(strings.ReplaceAll(kind.Label(), " ", "-")) +
		"-" + string(query.Line) +
		"-" + query.Range.From + "-sd-" + query.Range.To + ".csv"
	return url.PathEscape(name)
}

// PICTeknikLaporanExport melayani GET /api/report-kpi/pic-teknik/ekspor-laporan.
//
// # Ia BUKAN varian dari PICTeknikExport
//
// Tab KPI PIC Teknik di Pega punya DUA tombol ekspor, dan keduanya mengeluarkan berkas
// yang sama sekali berbeda:
//
//	Export Data KPI  di sebelah Cari            -> PENILAIANNYA, 6 kolom      <- yang ini
//	Export Data KPI  di sebelah Pilih Data KPI  -> data klaim mentah, 27–40 kolom
//
// Yang ini mengekspor persis isi grid di layar, sehingga ia memakai ulang `PICTeknik` —
// bukan kueri tersendiri. Dengan begitu angka di layar dan angka di berkas tidak dapat
// berselisih: keduanya hasil perhitungan yang sama.
func (h *Handler) PICTeknikLaporanExport(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	// Seluruh isinya dihitung SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON.
	scored, err := h.service.PICTeknik(r.Context(), active.Alias, readPICFilter(r), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	spec := reportkpi.ExportPICScorecard()
	h.beginDownload(w, picLaporanFilename(spec.FileLabel, scored.Filter))

	writer := csv.NewWriter(w)
	if err := writer.Write(spec.Headers); err != nil {
		h.logExportFailure(r, err)
		return
	}

	for _, card := range picExportCards(scored.Result) {
		for _, row := range card.Rows {
			cells := make([]string, 0, len(spec.Fields))
			for _, field := range spec.Fields {
				cells = append(cells, picScorecardCell(card, row, field))
			}
			if err := writer.Write(cells); err != nil {
				h.logExportFailure(r, err)
				return
			}
		}
	}

	h.flush(w, writer, r)
}

// picExportCards menyusun urutan kartu yang ditulis: tiap PIC, lalu baris rekapitulasi.
//
// Rekapitulasi DIBUANG ketika tidak ada satu pun petugas — ia rata-rata dari nol orang,
// dan menuliskannya menghasilkan baris "Leader" yang tidak merangkum apa pun. Layar pun
// menyembunyikannya pada keadaan yang sama.
func picExportCards(result reportkpi.PICTeknikResult) []reportkpi.PICScorecard {
	if len(result.Scorecards) == 0 {
		return nil
	}
	return append(append([]reportkpi.PICScorecard{}, result.Scorecards...), result.Leader)
}

// picScorecardCell mengambil satu sel berkas "Laporan KPI".
//
// Persentase dan nilai yang TIDAK dapat dihitung menjadi sel KOSONG, bukan "0" — sama
// seperti berkas ekspor lain. Nol yang dikarang akan ikut terhitung ketika pembacanya
// menjumlahkan kolomnya di Excel.
func picScorecardCell(
	card reportkpi.PICScorecard,
	row reportkpi.PICRow,
	field string,
) string {
	switch field {
	case reportkpi.FieldPICName:
		return card.PIC
	case reportkpi.FieldPICMetric:
		return row.Label
	case reportkpi.FieldPICTotal:
		return strconv.FormatFloat(row.Total, 'f', -1, 64)
	case reportkpi.FieldPICAchieved:
		return strconv.FormatFloat(row.Achieved, 'f', -1, 64)
	case reportkpi.FieldPICPercent:
		return scoreCell(row.Percent)
	case reportkpi.FieldPICValue:
		return scoreCell(row.Value)
	default:
		return ""
	}
}

// picLaporanFilename menyusun nama berkas "Laporan KPI" beserta periodenya.
//
// Pega menyusunnya `"Laporan KPI "+" , "+Param.awal+" - "+Param.akhir` — dua spasi sebelum
// koma, dan itu ditiru. Yang TIDAK dapat ditiru adalah bentuk `Param.awal`: badan langkah
// activity-nya tidak ikut ter-export, sehingga tidak diketahui apakah ia `dd/mm/yyyy`
// atau bentuk lain.
//
// Dipakai di sini bentuk `YYYY-MM-DD` apa adanya seperti yang dipilih pengguna. Alasannya
// bukan selera: `dd/mm/yyyy` memuat garis miring, dan garis miring tidak sah di dalam nama
// berkas Windows — bentuk itu karena itu mustahil menjadi yang dipakai Pega.
func picLaporanFilename(label string, query reportkpi.PICTeknikQuery) string {
	name := label + "  , " + query.Range.From + " - " + query.Range.To + ".csv"
	return url.PathEscape(name)
}
