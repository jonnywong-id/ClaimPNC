package inboxsalvagehttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/usecase"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxsalvage.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas
//
// Karena berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka `ADR-0011`
// yang belum dijawab: `pyMaxRecords=500` terpasang pada 54 dari 56 laporan Pega, sehingga
// kebutuhan ekspor bervolume besar belum pernah benar-benar dilayani.
//
// Berkas yang menyentuhnya diberi tanda di baris terakhir — bukan dipotong tanpa satu pun
// pemberitahuan, yang persis cacat sistem lama. Nilainya sama dengan modul inbox lain
// supaya tidak ada dua batas berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/inbox-salvage/ekspor — tombol "Export Data" di layar.
//
// # Apa yang diekspor
//
// Salinan daftar yang SEDANG DILIHAT, beserta pencariannya. Itu berbeda dari sistem lama,
// dan perbedaannya disengaja.
//
// `Activity/ExportDataSalvageForm_pncsalvage-Act.xml` menjalankan kueri yang BERBEDA dari
// kueri grid — `GetDataSalvagefromPNC_salvage` — yang menggabungkan `PNC_SALVAGE` dengan
// `DETAIL_PNC_SALVAGE` sehingga satu pengajuan menjadi BANYAK baris, satu per nama barang.
// Kolomnya pun berbeda: ia memuat nama barang, nama pemenang, status terjual lima keadaan,
// dan nama surveyor — tidak satu pun ada di grid.
//
// Ekspor rincian itu BELUM dibangun. Yang dibangun adalah ekspor grid, yang isinya sama
// persis dengan yang dilihat pengguna. Selisihnya dinyatakan di PlannedDifferences.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	filter := readFilter(query.Get("daftar"), query.Get("cari"))
	page := inboxsalvage.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan. Daftar yang tidak dikenal dan sesi
	// yang tidak lengkap karena itu tetap dijawab sebagai galat yang terbaca.
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
	caller inboxsalvage.Caller,
	filter inboxsalvage.QueryInput,
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

	page := inboxsalvage.Pagination{Page: 1, Size: exportChunk}

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
// `no-store` bukan kehati-hatian berlebih: berkas ini memuat nomor klaim dan nilai uang,
// dan ia tidak boleh mengendap di cache perantara mana pun.
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
// ("Catatan" pada kelima daftar yang menggambarnya). Menghilangkan kolom kosong dari berkas
// akan membuat berkas dan layar punya jumlah kolom berbeda — dan orang yang mencocokkan
// keduanya akan mengira ada kolom yang tergeser.
func columnTitles(columns []inboxsalvage.Column) []string {
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, column.Title)
	}
	return header
}

// gridRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan columnTitles, dan keduanya dibangun dari senarai kolom yang
// SAMA — bukan dari dua daftar yang kebetulan sejalan. Dengan begitu penambahan kolom di
// tab.go tidak dapat menggeser isi berkas tanpa menggeser judulnya sekaligus.
func gridRow(tab inboxsalvage.Tab, row inboxsalvage.Row) []string {
	cells := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		cells = append(cells, gridCell(row, column.Key))
	}
	return cells
}

// gridCell mengambil isi satu sel menurut nama kolomnya.
//
// Ia memakai konstanta Field… yang sama dengan tab.go dan dengan nama field JSON di dto.go.
// Kolom yang tidak dikenali menghasilkan teks kosong, bukan panik: berkas ekspor yang
// kehilangan satu kolom masih dapat dipakai, sedangkan permintaan yang gagal di tengah
// unduhan tidak.
func gridCell(row inboxsalvage.Row, key string) string {
	switch key {
	case inboxsalvage.FieldClaimNo:
		return row.ClaimNo
	case inboxsalvage.FieldSalvageID:
		return row.SalvageID
	case inboxsalvage.FieldInputDate:
		return row.InputDate
	case inboxsalvage.FieldLossDate:
		return row.LossDate
	case inboxsalvage.FieldPIC:
		return row.PIC
	case inboxsalvage.FieldBusinessName:
		return row.BusinessName
	case inboxsalvage.FieldObjectName:
		return row.ObjectName
	case inboxsalvage.FieldSalvageType:
		return row.SalvageType
	case inboxsalvage.FieldSalvageLocation:
		return row.SalvageLocation
	case inboxsalvage.FieldAuctionStatus:
		return row.AuctionStatus
	case inboxsalvage.FieldEstimateValue:
		return row.EstimateValue
	case inboxsalvage.FieldEmail:
		return row.Email
	case inboxsalvage.FieldRemark:
		return row.Remark
	case inboxsalvage.FieldRequestValue:
		return row.RequestValue
	case inboxsalvage.FieldSubmissionType:
		return row.SubmissionType
	case inboxsalvage.FieldAging:
		return row.Aging
	case inboxsalvage.FieldNote:
		return row.Note
	default:
		return ""
	}
}

// exportFilename menyusun nama berkas yang menyebut daftar asalnya.
//
// Tanpa itu, unduhan dari beberapa daftar menghasilkan berkas bernama sama di folder
// unduhan — dan yang berikutnya menimpa yang sebelumnya pada sebagian peramban. Di layar
// ini akibatnya lebih buruk daripada biasa: dua daftar punya kolom yang IDENTIK dan isi
// yang identik pula ("Checker" dan "Salvage Diterima"), sehingga berkas yang tertimpa tidak
// dapat dikenali dari isinya sama sekali.
//
// Kode tab dipakai apa adanya sebagai bagian nama berkas. Ia sudah berbentuk `kebab-case`
// dan tidak memuat satu pun karakter yang perlu dilepaskan pada nama berkas.
func exportFilename(tab inboxsalvage.Tab) string {
	if tab.Code == "" {
		return "inbox-salvage.csv"
	}
	return "inbox-salvage-" + tab.Code + ".csv"
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
//
// Ia tidak dapat lagi dijawab sebagai galat HTTP — status sudah 200 dan sebagian berkas
// sudah sampai ke pengguna. Yang dapat dilakukan hanyalah menghentikan penulisan dan
// meninggalkan jejak, supaya unduhan yang terpotong di sisi pengguna punya pasangan
// keterangan di sisi peladen.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Error("ekspor inbox salvage terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
