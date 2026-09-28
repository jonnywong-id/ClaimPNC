package reportkpihttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/reportkpi"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor rincian.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = reportkpi.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// Berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka `ADR-0011` yang
// belum dijawab: `pyMaxRecords=500` terpasang pada 54 dari 56 laporan Pega, sehingga
// kebutuhan ekspor bervolume besar belum pernah benar-benar dilayani.
//
// Berkas yang menyentuh batas diberi tanda di baris terakhir — bukan dipotong tanpa satu
// pun pemberitahuan, yang persis cacat sistem lama. Nilainya sama dengan modul lain supaya
// tidak ada dua batas berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/report-kpi/adjuster/ekspor.
//
// # SATU TOMBOL DI LAYAR, DUA ISI BERKAS
//
// Layar lama punya satu tombol "Export Data" pada tab KPI Adjuster, sementara tabnya
// menggambar DUA grid. Di sini keduanya dapat diunduh, dan yang mana ditentukan parameter
// `grid` — bukan dua tombol yang terlihat berbeda:
//
//	grid=ringkasan  (bawaan)  satu baris per adjuster, nilainya rata-rata
//	grid=rincian              satu baris per kasus survei
//
// Keduanya memakai penyaring yang SAMA PERSIS dengan yang sedang terlihat di layar, dan
// itu bukan kebetulan: keduanya membaca parameter lewat readFilter yang sama dengan
// Summary dan Detail. Ekspor yang membaca penyaring dengan cara berbeda akan menghasilkan
// berkas yang isinya tidak dapat dicocokkan dengan apa pun di layar.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	if strings.TrimSpace(r.URL.Query().Get("grid")) == reportkpi.GridDetail {
		h.exportDetail(w, r, active.Alias, caller)
		return
	}
	h.exportSummary(w, r, active.Alias, caller)
}

// exportSummary menulis berkas berisi grid Summary.
//
// Ia TIDAK dipaginasi, sama dengan grid-nya: satu baris per adjuster, dan jumlah adjuster
// eksternal terhitung puluhan.
func (h *Handler) exportSummary(
	w http.ResponseWriter,
	r *http.Request,
	portalAlias string,
	caller reportkpi.Caller,
) {
	// Datanya diambil SEBELUM satu byte pun ditulis. Setelah header terkirim, galat tidak
	// dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa berkas separuh
	// jadi tanpa satu pun keterangan. Tipe report yang belum dipilih dan periode yang
	// kosong karena itu tetap dijawab sebagai galat yang terbaca.
	result, err := h.service.Summary(r.Context(), portalAlias, caller, readFilter(r.URL.Query()))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Kolomnya disaring menurut tipe report, sama seperti di layar: kolom TIPE hanya ada
	// pada tipe ALL, karena hanya kueri ALL di Pega yang mengembalikannya.
	columns := findGrid(reportkpi.GridSummary).ColumnsFor(result.Query.ReportType)
	header := exportHeader(columns)

	h.beginDownload(w, exportFilename("ringkasan-kpi-adjuster", result.Query))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	for i, row := range result.Rows {
		if i >= exportLimit {
			_ = writer.Write(truncationNotice(len(header), len(result.Rows)))
			return
		}

		cells := make([]string, 0, len(columns))
		for _, column := range columns {
			cells = append(cells, summaryCell(row, column.Key))
		}
		if err := writer.Write(append(cells, scoreCells(row.Scores)...)); err != nil {
			h.logExportFailure(r, err)
			return
		}
	}
}

// detailCell mengambil isi satu sel tetap grid Detail menurut nama kolomnya.
//
// Ia TERPISAH dari summaryCell karena kolomnya memang berbeda: grid Detail membawa nomor
// kasus dan tanggal penilaian, yang tidak ada di grid Summary — barisnya satu kasus, bukan
// satu adjuster.
func detailCell(row reportkpi.AdjusterDetail, key string) string {
	switch key {
	case reportkpi.FieldAdjusterName:
		return row.Adjuster
	case reportkpi.FieldCaseID:
		return row.CaseID
	case reportkpi.FieldType:
		return string(row.ReportType)
	case reportkpi.FieldScoredOn:
		return row.ScoredOn
	default:
		return ""
	}
}

// summaryCell mengambil isi satu sel tetap grid Summary menurut nama kolomnya.
//
// Ia memakai konstanta `Field…` yang sama dengan screen.go dan dengan nama field JSON di
// dto.go. Kolom yang tidak dikenali menghasilkan teks kosong, bukan panik: berkas ekspor
// yang kehilangan satu kolom masih dapat dipakai, sedangkan permintaan yang gagal di tengah
// unduhan tidak.
func summaryCell(row reportkpi.AdjusterSummary, key string) string {
	switch key {
	case reportkpi.FieldAdjusterName:
		return row.Adjuster
	case reportkpi.FieldType:
		return string(row.ReportType)
	default:
		return ""
	}
}

// exportDetail menulis berkas berisi grid Detail, potong demi potong.
func (h *Handler) exportDetail(
	w http.ResponseWriter,
	r *http.Request,
	portalAlias string,
	caller reportkpi.Caller,
) {
	filter := readFilter(r.URL.Query())
	page := reportkpi.Pagination{Page: 1, Size: exportChunk}

	first, err := h.service.Detail(r.Context(), portalAlias, caller, filter, page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Grid Detail tidak punya kolom yang muncul-hilang, tetapi ia tetap menempuh
	// ColumnsFor supaya kedua berkas ekspor disusun dengan cara yang sama.
	columns := findGrid(reportkpi.GridDetail).ColumnsFor(first.Query.ReportType)
	header := exportHeader(columns)

	h.beginDownload(w, exportFilename("rincian-kpi-adjuster", first.Query))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	current := first
	for {
		for _, row := range current.Page.Rows {
			if written >= exportLimit {
				_ = writer.Write(truncationNotice(len(header), current.Page.Total))
				return
			}

			cells := make([]string, 0, len(columns))
			for _, column := range columns {
				cells = append(cells, detailCell(row, column.Key))
			}
			if err := writer.Write(append(cells, scoreCells(row.Scores)...)); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		if !h.flush(w, writer, r) {
			return
		}
		if written >= current.Page.Total || len(current.Page.Rows) == 0 {
			return
		}

		page.Page++
		next, err := h.service.Detail(r.Context(), portalAlias, caller, filter, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
		current = next
	}
}

// findGrid mengambil keterangan satu grid pada tab KPI Adjuster.
func findGrid(code string) reportkpi.Grid {
	for _, grid := range reportkpi.AdjusterTab().Grids {
		if grid.Code == code {
			return grid
		}
	}
	return reportkpi.Grid{}
}

// exportHeader menyusun baris judul: kolom tetap yang berlaku, lalu kesembilan komponen.
//
// Urutannya WAJIB sama dengan urutan sel yang ditulis exportSummary dan exportDetail, dan
// keduanya dibangun dari senarai kolom yang SAMA — bukan dari dua daftar yang kebetulan
// sejalan. Dengan begitu kolom yang muncul-hilang menurut tipe report tidak dapat
// menggeser isi berkas tanpa menggeser judulnya sekaligus.
func exportHeader(columns []reportkpi.Column) []string {
	header := make([]string, 0, len(columns)+len(reportkpi.Components()))
	for _, column := range columns {
		header = append(header, column.Title)
	}
	for _, component := range reportkpi.Components() {
		header = append(header, component.Label)
	}
	return header
}

// scoreCells menyusun kesembilan sel nilai, dalam urutan komponen.
//
// Nilai yang tidak ada menjadi sel KOSONG, bukan "0". Berkas ekspor dibaca ulang di Excel
// dan sering dijumlahkan di sana; nol yang dikarang akan ikut terhitung dan menggeser
// rata-rata yang dihitung ulang pembacanya.
func scoreCells(scores map[string]reportkpi.Score) []string {
	codes := reportkpi.ComponentCodes()

	cells := make([]string, 0, len(codes))
	for _, code := range codes {
		score, exists := scores[code]
		if !exists || !score.Present {
			cells = append(cells, "")
			continue
		}
		cells = append(cells, strconv.FormatFloat(score.Value, 'f', -1, 64))
	}
	return cells
}

// beginDownload memasang header unduhan.
func (h *Handler) beginDownload(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
}

// flush mendorong isi yang sudah tertulis keluar setiap potong, bukan menahannya sampai
// akhir. Itulah yang membuat unduhan besar mulai mengalir segera dan memori tidak menumpuk.
func (h *Handler) flush(w http.ResponseWriter, writer *csv.Writer, r *http.Request) bool {
	writer.Flush()
	if err := writer.Error(); err != nil {
		h.logExportFailure(r, err)
		return false
	}
	if flusher, able := w.(http.Flusher); able {
		flusher.Flush()
	}
	return true
}

// exportFilename menyusun nama berkas beserta penyaring yang menghasilkannya.
//
// Tipe report dan periode masuk ke dalam nama dengan sengaja: laporan yang sama diunduh
// berulang kali dengan penyaring berbeda, dan berkas yang namanya sama membuat dua
// penyaring tidak dapat dibedakan setelah tersimpan — pada sebagian peramban yang
// berikutnya bahkan menimpa yang sebelumnya.
func exportFilename(prefix string, query reportkpi.Query) string {
	name := prefix + "-" + strings.ToLower(string(query.ReportType))
	if query.Range.From != "" && query.Range.To != "" {
		name += "-" + query.Range.From + "-sd-" + query.Range.To
	}
	return name + ".csv"
}

// truncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func truncationNotice(width, total int) []string {
	notice := make([]string, width)
	if width == 0 {
		return notice
	}
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %s baris dari %s yang cocok. Persempit periodenya. --",
		strconv.Itoa(exportLimit), strconv.Itoa(total),
	)
	return notice
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Ia tidak dapat lagi dijawab sebagai galat HTTP — status sudah 200 dan sebagian berkas
// sudah sampai ke pengguna. Yang dapat dilakukan hanyalah menghentikan penulisan dan
// meninggalkan jejak, supaya unduhan yang terpotong di sisi pengguna punya pasangan
// keterangan di sisi peladen.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Error("ekspor Report KPI terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
