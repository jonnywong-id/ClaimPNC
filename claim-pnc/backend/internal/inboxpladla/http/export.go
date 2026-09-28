package inboxpladlahttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/usecase"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
const exportChunk = inboxpladla.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// Berkas yang menyentuhnya diberi tanda di baris terakhir — bukan dipotong tanpa satu pun
// pemberitahuan, yang persis cacat sistem lama (`pyMaxRecords=500` pada 54 dari 56
// laporan Pega). Nilainya sama dengan modul inbox lain.
const exportLimit = 50_000

// Export menangani GET /api/inbox-pla-dla/ekspor — tombol "Export To Excel".
//
// # Apa yang diekspor
//
// Salinan daftar yang SEDANG DILIHAT, beserta pencariannya — dan itu memang yang
// dilakukan Pega: `Activity/ExportDataPLADLAReas-Act.xml` tidak menjalankan kueri
// tersendiri sama sekali, ia hanya mengubah senarai yang sudah dimuat menjadi CSV lewat
// `pxConvertResultsToCSV`.
//
// Satu-satunya perbedaan adalah cara barisnya dikumpulkan: di sana seluruh baris sudah ada
// di memori karena kuerinya tidak berpaginasi, di sini ia diambil sepotong demi sepotong
// dan ditulis langsung ke jawaban supaya memori tetap datar
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	filter := readFilter(r)
	page := inboxpladla.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — dan di layar ini galat yang paling
	// mungkin terjadi justru yang paling perlu terbaca: pemanggil yang bukan reasuradur
	// terdaftar.
	first, err := h.service.List(r.Context(), active.Alias, caller, filter, page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.exportGrid(w, r, active.Alias, caller, filter, first)
}

// exportGrid menulis berkas berisi salinan daftar yang sedang dilihat.
func (h *Handler) exportGrid(
	w http.ResponseWriter,
	r *http.Request,
	portalAlias string,
	caller inboxpladla.Caller,
	filter inboxpladla.QueryInput,
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

	page := inboxpladla.Pagination{Page: 1, Size: exportChunk}

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

// beginDownload memasang header unduhan.
//
// `no-store` bukan kehati-hatian berlebih: berkas ini memuat nama tertanggung dan nomor
// polis milik tertanggung Asuransi Sinar Mas, dan yang mengunduhnya adalah PIHAK LUAR.
// Berkas seperti itu tidak boleh mengendap di cache perantara mana pun.
func (h *Handler) beginDownload(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
}

// flush mendorong isi yang sudah tertulis keluar setiap potong.
func (h *Handler) flush(
	w http.ResponseWriter,
	writer *csv.Writer,
	r *http.Request,
) bool {
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
func columnTitles(columns []inboxpladla.Column) []string {
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, column.Title)
	}
	return header
}

// gridRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan columnTitles, dan keduanya dibangun dari senarai kolom yang
// SAMA — bukan dari dua daftar yang kebetulan sejalan.
func gridRow(tab inboxpladla.Tab, row inboxpladla.Row) []string {
	cells := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		cells = append(cells, gridCell(row, column.Key))
	}
	return cells
}

// gridCell mengambil isi satu sel menurut nama kolomnya.
//
// Kolom status mengambil LABEL-nya, dan jatuh ke kode hanya bila labelnya kosong. Itu
// sama dengan yang digambar layar — berkas yang berisi angka sementara layarnya berisi
// kata akan terbaca sebagai berkas yang salah.
func gridCell(row inboxpladla.Row, key string) string {
	switch key {
	case inboxpladla.FieldClaimNo:
		return row.ClaimNo
	case inboxpladla.FieldPolicyNo:
		return row.PolicyNo
	case inboxpladla.FieldInsured:
		return row.Insured
	case inboxpladla.FieldBusinessName:
		return row.BusinessName
	case inboxpladla.FieldRegisterDate:
		return row.RegisterDate
	case inboxpladla.FieldLossDate:
		return row.LossDate
	case inboxpladla.FieldPICTeknik:
		return row.PICTeknik
	case inboxpladla.FieldStatus:
		if row.StatusLabel != "" {
			return row.StatusLabel
		}
		return row.StatusCode
	case inboxpladla.FieldAdviceNo:
		return row.AdviceNo
	case inboxpladla.FieldCloseNote:
		return row.CloseNote
	default:
		return ""
	}
}

// exportFilename menyusun nama berkas yang menyebut daftar asalnya.
//
// Tanpa itu, unduhan dari ketiga tab menghasilkan berkas bernama sama di folder unduhan —
// dan yang berikutnya menimpa yang sebelumnya pada sebagian peramban. Ketiga tab di layar
// ini punya kolom yang IDENTIK, sehingga berkas yang tertimpa tidak dapat dikenali dari
// isinya.
func exportFilename(tab inboxpladla.Tab) string {
	if tab.Code == "" {
		return "inbox-pla-dla.csv"
	}
	return "inbox-pla-dla-" + tab.Code + ".csv"
}

// truncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func truncationNotice(width, total int) []string {
	notice := make([]string, width)
	if width == 0 {
		return notice
	}
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %s baris dari %s yang cocok. Persempit pencariannya. --",
		strconv.Itoa(exportLimit), strconv.Itoa(total),
	)
	return notice
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Error(
		"ekspor inbox PLA/DLA reasuradur terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
