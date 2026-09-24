package inboxkomunikasicabanghttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxkomunikasicabang.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas
//
// Karena berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka `ADR-0011`
// yang belum dijawab: `pyMaxRecords=500` terpasang pada 54 dari 56 laporan Pega, sehingga
// kebutuhan ekspor bervolume besar belum pernah benar-benar dilayani.
//
// Berkas yang menyentuhnya diberi tanda di baris terakhir — bukan dipotong tanpa satu pun
// pemberitahuan, yang persis cacat sistem lama. Nilainya sama dengan modul inbox lain supaya
// tidak ada dua batas berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/inbox-komunikasi-cabang/ekspor.
//
// # SELURUH tombol ini adalah KEMAMPUAN BARU
//
// Layar lama TIDAK punya tombol ekspor sama sekali — tidak ada satu pun activity ekspor
// yang dirujuk `Section/InboxKomunikasi-Section.xml`, dan ketiga tombol yang ada di sana
// menulis, bukan mengunduh.
//
// Ia ditambahkan karena percakapan yang menumpuk tidak dapat ditelusuri lewat layar
// berhalaman, dan karena modul inbox lain sudah memilikinya. Kemampuan barunya dinyatakan
// lewat PlannedDifferences, bukan disamarkan sebagai paritas.
//
// # SATU berkas untuk kedua tab, dan itu keputusan
//
// Kolom berkasnya TIDAK mengikuti kolom tab yang sedang terbuka — ia selalu kedelapan kolom
// `ExportColumns`. Berkas yang kolomnya berubah-ubah menurut tab yang kebetulan terbuka
// tidak dapat digabungkan maupun dibandingkan oleh penerimanya, dan yang diekspor memang
// baris yang sama dari tabel yang sama.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	page := inboxkomunikasicabang.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim, galat
	// tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa berkas
	// separuh jadi tanpa satu pun keterangan. Tab yang tidak dikenal, sesi yang tidak
	// lengkap, dan sumber cabang yang mati karena itu tetap dijawab sebagai galat terbaca.
	first, err := h.service.List(r.Context(), active.Alias, caller, readFilter(query), page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	header := columnTitles(inboxkomunikasicabang.ExportColumns)

	h.beginDownload(w, exportFilename(first.Query.Tab, first.Query.Branch))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	filter := readFilter(query)
	written := 0
	current := first

	for {
		for _, item := range current.Page.Items {
			if written >= exportLimit {
				_ = writer.Write(truncationNotice(len(header), current.Page.Total))
				return
			}
			if err := writer.Write(exportRow(item)); err != nil {
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
		next, err := h.service.List(r.Context(), active.Alias, caller, filter, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
		current = next
	}
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

// columnTitles menyusun baris judul dari senarai kolom.
func columnTitles(columns []inboxkomunikasicabang.Column) []string {
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, column.Title)
	}
	return header
}

// exportRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan columnTitles, dan keduanya dibangun dari senarai kolom yang
// SAMA — bukan dari dua daftar yang kebetulan sejalan. Dengan begitu penambahan kolom di
// tab.go tidak dapat menggeser isi berkas tanpa menggeser judulnya sekaligus.
func exportRow(item inboxkomunikasicabang.Conversation) []string {
	cells := make([]string, 0, len(inboxkomunikasicabang.ExportColumns))
	for _, column := range inboxkomunikasicabang.ExportColumns {
		cells = append(cells, exportCell(item, column.Key))
	}
	return cells
}

// exportCell mengambil isi satu sel menurut nama kolomnya.
//
// Ia memakai konstanta Field… yang sama dengan tab.go dan dengan nama field JSON di dto.go.
// Kolom yang tidak dikenali menghasilkan teks kosong, bukan panik: berkas ekspor yang
// kehilangan satu kolom masih dapat dipakai, sedangkan permintaan yang gagal di tengah
// unduhan tidak.
//
// Kolom pengirim dan penjawab dirakit dengan fungsi yang SAMA dengan yang dipakai DTO,
// sehingga berkas dan layar menyebut hal yang sama dengan bentuk yang sama.
func exportCell(item inboxkomunikasicabang.Conversation, key string) string {
	switch key {
	case inboxkomunikasicabang.FieldCreatedAt:
		return item.CreatedAt
	case inboxkomunikasicabang.FieldSender:
		return joinWithParenthesis(item.SenderOrigin, item.SenderOperator)
	case inboxkomunikasicabang.FieldRecipient:
		return item.RecipientOrigin
	case inboxkomunikasicabang.FieldMessage:
		return item.Message
	case inboxkomunikasicabang.FieldReply:
		return item.Reply
	case inboxkomunikasicabang.FieldReplier:
		return joinWithParenthesis(item.ReplierName, item.RecipientOrigin)
	case inboxkomunikasicabang.FieldRepliedAt:
		return item.RepliedAt
	case inboxkomunikasicabang.FieldStatus:
		return item.Status
	default:
		return ""
	}
}

// exportFilename menyusun nama berkas yang menyebut tab DAN batas cabangnya.
//
// # Kenapa cabangnya ikut, berbeda dari modul inbox lain
//
// Karena isi berkas ini bergantung pada SIAPA yang mengunduhnya. Dua petugas yang mengunduh
// tab yang sama pada saat yang sama menerima berkas yang isinya berbeda — masing-masing
// hanya berisi percakapan cabangnya sendiri.
//
// Berkas yang tidak menyebut cabangnya karena itu tidak dapat dikenali setelah tersimpan,
// dan lebih buruk: dua berkas dari dua cabang akan bernama sama dan yang satu menimpa yang
// lain di folder unduhan.
func exportFilename(
	tab inboxkomunikasicabang.Tab, branch inboxkomunikasicabang.BranchFilter,
) string {
	scope := "cabang-" + branch.Code
	if branch.HeadOffice {
		scope = "pusat"
	}

	part := "belum-dijawab"
	if tab.Answered {
		part = "sudah-dijawab"
	}

	return "komunikasi-cabang-" + scope + "-" + part + ".csv"
}

// truncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func truncationNotice(width, total int) []string {
	notice := make([]string, width)
	if width == 0 {
		return notice
	}
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %s baris dari %s yang cocok. --",
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
	logging.From(r.Context(), h.logger).Error("ekspor inbox komunikasi cabang terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
