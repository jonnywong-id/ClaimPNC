package inboxrclpuclhttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/usecase"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxrclpucl.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas
//
// Karena berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka `ADR-0011`
// yang belum dijawab: `pyMaxRecords=500` terpasang pada 54 dari 56 laporan Pega, sehingga
// kebutuhan ekspor bervolume besar belum pernah benar-benar dilayani.
//
// Di layar ini batas itu lebih berguna daripada di layar lain, dan alasannya khas: rentang
// tanggal laporan harian ditentukan PENGGUNA, dan tidak ada satu pun penjaga di layar lama
// yang mencegahnya memilih sepuluh tahun sekaligus.
//
// Berkas yang menyentuhnya diberi tanda di baris terakhir — bukan dipotong tanpa satu pun
// pemberitahuan, yang persis cacat sistem lama. Nilainya sama dengan modul inbox lain
// supaya tidak ada dua batas berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/inbox-rcl-pucl/ekspor — tombol "Export To Excel" di layar.
//
// # SATU TOMBOL, DUA PERILAKU — dan itu memang begitu di Pega
//
// Ketiga tab punya tombol ekspor, tetapi yang dijalankannya tidak sama:
//
//	tab Cetak Surat          Activity/ExportCetakSurat_act      -> GetDataPUCLRCLForDailyReport
//	tab Kelengkapan Dokumen  Activity/ExportKelengkapanDoc_act  -> InboxPUCLCetakSurat_RD
//	tab Klaim MSIG           Activity/ExportAJSMSIGDoc_act      -> InboxMISG_RD
//
// Dua yang terakhir mengekspor GRID-nya sendiri. Yang pertama menjalankan kueri yang
// BERBEDA — laporan harian berbasis rentang tanggal, dengan kolom yang berbeda dan isi yang
// berbeda. Lihat inboxrclpucl.DailyReportRow.
//
// Percabangan itu ditentukan `Tab.HasDateRangeReport`, bukan oleh kode tab yang ditulis
// tetap di sini: menaruh nomor tab di lapisan transport akan membuat penambahan tab kelak
// menuntut suntingan di dua tempat.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	page := inboxrclpucl.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan. Tab yang tidak dikenal, sesi yang
	// tidak lengkap, dan rentang tanggal yang salah karena itu tetap dijawab sebagai galat
	// yang terbaca.
	first, err := h.service.List(r.Context(), active.Alias, caller, readFilter(query), page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if first.Query.Tab.HasDateRangeReport {
		h.exportDailyReport(w, r, active.Alias, caller)
		return
	}

	h.exportGrid(w, r, active.Alias, caller, first)
}

// exportGrid menulis berkas berisi salinan grid yang sedang dilihat.
//
// Dipakai tab "Kelengkapan Dokumen" dan "Klaim MSIG", yang di Pega pun mengekspor Report
// Definition grid-nya sendiri.
func (h *Handler) exportGrid(
	w http.ResponseWriter,
	r *http.Request,
	portalAlias string,
	caller inboxrclpucl.Caller,
	first usecase.Listed,
) {
	tab := first.Query.Tab
	header := columnTitles(tab.Columns)

	h.beginDownload(w, exportFilename(tab))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	filter := readFilter(r.URL.Query())
	page := inboxrclpucl.Pagination{Page: 1, Size: exportChunk}

	written := 0
	current := first
	for {
		for _, item := range current.Page.Items {
			if written >= exportLimit {
				_ = writer.Write(truncationNotice(len(header), current.Page.Total))
				return
			}
			if err := writer.Write(gridRow(tab, item)); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		if !h.flush(w, writer, r) {
			return
		}
		if written >= current.Page.Total || len(current.Page.Items) == 0 {
			return
		}

		page.Page++
		next, err := h.service.List(r.Context(), portalAlias, caller, filter, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
		current = next
	}
}

// exportDailyReport menulis berkas LAPORAN HARIAN RCL/PUCL.
//
// Dipakai tab "Cetak Surat" saja. Isinya BUKAN salinan grid — lihat catatan pada Export dan
// pada inboxrclpucl.DailyReportRow.
//
// Rentang tanggalnya divalidasi di sini pula, sebelum satu byte pun ditulis: kedua tanggal
// wajib, dan pengguna yang mengosongkannya harus menerima pesan yang terbaca — bukan berkas
// kosong yang tampak berhasil.
func (h *Handler) exportDailyReport(
	w http.ResponseWriter,
	r *http.Request,
	portalAlias string,
	caller inboxrclpucl.Caller,
) {
	input := readReport(r.URL.Query())
	page := inboxrclpucl.Pagination{Page: 1, Size: exportChunk}

	first, err := h.service.DailyReport(r.Context(), portalAlias, caller, input, page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	header := columnTitles(inboxrclpucl.DailyReportColumns)

	h.beginDownload(w, reportFilename(first.Request.Range))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	current := first
	for {
		for _, row := range current.Rows {
			if written >= exportLimit {
				_ = writer.Write(truncationNotice(len(header), current.Total))
				return
			}
			if err := writer.Write(reportRow(row)); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		if !h.flush(w, writer, r) {
			return
		}
		if written >= current.Total || len(current.Rows) == 0 {
			return
		}

		page.Page++
		next, err := h.service.DailyReport(r.Context(), portalAlias, caller, input, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
		current = next
	}
}

// beginDownload memasang header unduhan.
//
// `no-store` bukan kehati-hatian berlebih: berkas ini memuat nomor polis dan nama
// tertanggung, dan ia tidak boleh mengendap di cache perantara mana pun.
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

// columnTitles menyusun baris judul dari senarai kolom.
//
// Judulnya sama persis dengan judul kolom di grid, termasuk kolom yang isinya selalu kosong
// ("Tanggal Cetak Surat" pada tab Cetak Surat). Menghilangkan kolom kosong dari berkas akan
// membuat berkas dan layar punya jumlah kolom berbeda — dan orang yang mencocokkan keduanya
// akan mengira ada kolom yang tergeser.
func columnTitles(columns []inboxrclpucl.Column) []string {
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, column.Title)
	}
	return header
}

// gridRow menyusun satu baris berkas ekspor grid.
//
// Urutannya WAJIB sama dengan columnTitles, dan keduanya dibangun dari senarai kolom yang
// SAMA — bukan dari dua daftar yang kebetulan sejalan. Dengan begitu penambahan kolom di
// tab.go tidak dapat menggeser isi berkas tanpa menggeser judulnya sekaligus.
func gridRow(tab inboxrclpucl.Tab, item inboxrclpucl.WorkItem) []string {
	row := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		row = append(row, gridCell(item, column.Key))
	}
	return row
}

// gridCell mengambil isi satu sel grid menurut nama kolomnya.
//
// Ia memakai konstanta Field… yang sama dengan tab.go dan dengan nama field JSON di dto.go.
// Kolom yang tidak dikenali menghasilkan teks kosong, bukan panik: berkas ekspor yang
// kehilangan satu kolom masih dapat dipakai, sedangkan permintaan yang gagal di tengah
// unduhan tidak.
func gridCell(item inboxrclpucl.WorkItem, key string) string {
	switch key {
	case inboxrclpucl.FieldCaseID:
		return item.CaseID
	case inboxrclpucl.FieldPolicyNumber:
		return item.PolicyNumber
	case inboxrclpucl.FieldInsuredName:
		return item.InsuredName
	case inboxrclpucl.FieldInboxEntryAt:
		return item.InboxEntryAt
	case inboxrclpucl.FieldAnalystNote:
		return item.AnalystNote
	case inboxrclpucl.FieldTrack:
		return item.Track
	case inboxrclpucl.FieldLetterPrintedAt:
		return item.LetterPrintedAt
	case inboxrclpucl.FieldClaimAge:
		return item.ClaimAge
	case inboxrclpucl.FieldExpiryStatus:
		return item.ExpiryStatus
	default:
		return ""
	}
}

// reportRow menyusun satu baris berkas laporan harian.
//
// Ia TERPISAH dari gridRow karena kolomnya memang berbeda: laporan punya "Status Klaim"
// yang tidak ada di grid, dan TIDAK punya "Lama Klaim" yang ada di setiap grid.
func reportRow(row inboxrclpucl.DailyReportRow) []string {
	cells := make([]string, 0, len(inboxrclpucl.DailyReportColumns))
	for _, column := range inboxrclpucl.DailyReportColumns {
		cells = append(cells, reportCell(row, column.Key))
	}
	return cells
}

// reportCell mengambil isi satu sel laporan menurut nama kolomnya.
func reportCell(row inboxrclpucl.DailyReportRow, key string) string {
	switch key {
	case inboxrclpucl.FieldCaseID:
		return row.CaseID
	case inboxrclpucl.FieldPolicyNumber:
		return row.PolicyNumber
	case inboxrclpucl.FieldInsuredName:
		return row.InsuredName
	case inboxrclpucl.FieldReportSentAt:
		return row.SentAt
	case inboxrclpucl.FieldAnalystNote:
		return row.AnalystNote
	case inboxrclpucl.FieldLetterPrintedAt:
		return row.LetterPrintedAt
	case inboxrclpucl.FieldTrack:
		return row.Track
	case inboxrclpucl.FieldReportClaimStatus:
		return row.ClaimStatus
	default:
		return ""
	}
}

// exportFilename menyusun nama berkas ekspor grid yang menyebut tab asalnya.
//
// Tanpa itu, unduhan dari beberapa tab menghasilkan berkas bernama sama di folder unduhan —
// dan yang berikutnya menimpa yang sebelumnya pada sebagian peramban. Di layar ini
// akibatnya lebih buruk daripada biasa: ketiga tab punya kolom yang IDENTIK, sehingga
// berkas yang tertimpa tidak dapat dikenali dari isinya sama sekali.
func exportFilename(tab inboxrclpucl.Tab) string {
	switch tab.Code {
	case inboxrclpucl.TabKelengkapanDokumen:
		return "rcl-pucl-kelengkapan-dokumen.csv"
	case inboxrclpucl.TabKlaimMSIG:
		return "rcl-pucl-klaim-msig.csv"
	default:
		return "rcl-pucl.csv"
	}
}

// reportFilename menyusun nama berkas laporan harian beserta rentang tanggalnya.
//
// Rentangnya masuk ke dalam nama berkas dengan sengaja: laporan yang sama diunduh berulang
// kali dengan rentang yang berbeda, dan berkas yang namanya sama membuat dua rentang tidak
// dapat dibedakan setelah tersimpan.
func reportFilename(rng inboxrclpucl.DateRange) string {
	if rng.IsZero() {
		return "laporan-harian-rcl-pucl.csv"
	}
	return "laporan-harian-rcl-pucl-" + rng.From + "-sd-" + rng.To + ".csv"
}

// truncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func truncationNotice(width, total int) []string {
	notice := make([]string, width)
	if width == 0 {
		return notice
	}
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %s baris dari %s yang cocok. Persempit rentangnya. --",
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
	logging.From(r.Context(), h.logger).Error("ekspor inbox RCL/PUCL terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
