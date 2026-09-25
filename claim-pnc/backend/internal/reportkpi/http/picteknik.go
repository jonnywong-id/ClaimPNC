package reportkpihttp

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strconv"

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
// Berkasnya MERATAKAN kartu skor menjadi tabel: satu baris per PIC per komponen, dengan nama
// PIC diulang di setiap baris. Bentuk kartu tidak dapat dipindahkan apa adanya ke CSV, dan
// mengulang namanya membuat berkasnya dapat disaring dan diurutkan di penampil lembar kerja
// — yang justru alasan orang mengunduhnya.
func (h *Handler) PICTeknikExport(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	// Seluruh isinya diambil SEBELUM satu byte pun ditulis. Setelah header terkirim, galat
	// tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa berkas
	// separuh jadi tanpa satu pun keterangan.
	scored, err := h.service.PICTeknik(r.Context(), active.Alias, readPICFilter(r), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	grid := reportkpi.PICTeknikGrid()
	header := make([]string, 0, len(grid.Columns))
	for _, column := range grid.Columns {
		header = append(header, column.Title)
	}

	h.beginDownload(w, picExportFilename(scored.Filter))

	writer := csv.NewWriter(w)
	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	cards := append([]reportkpi.PICScorecard{}, scored.Result.Scorecards...)
	cards = append(cards, scored.Result.Leader)

	for _, card := range cards {
		for _, row := range card.Rows {
			record := []string{
				card.PIC,
				row.Label,
				strconv.FormatFloat(row.Total, 'f', -1, 64),
				strconv.FormatFloat(row.Achieved, 'f', -1, 64),
				numberText(row.Percent),
				numberText(row.Value),
			}
			if err := writer.Write(record); err != nil {
				h.logExportFailure(r, err)
				return
			}
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
func picExportFilename(query reportkpi.PICTeknikQuery) string {
	name := "kpi-pic-teknik-" + string(query.Line) +
		"-" + query.Range.From + "-sd-" + query.Range.To + ".csv"
	return url.PathEscape(name)
}
