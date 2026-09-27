package archivedokumenklaimhttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claim-pnc/internal/archivedokumenklaim"

	portalhttp "claim-pnc/internal/portal/http"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = archivedokumenklaim.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas, padahal sistem lama tidak punya
//
// Justru karena sistem lama tidak punya. `pyMaxRecords=500` terpasang pada 54 dari 56
// laporan Pega, sehingga kebutuhan ekspor bervolume besar BELUM PERNAH benar-benar
// dilayani — dan berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka
// `ADR-0011` yang belum dijawab.
//
// Angka ini karena itu bukan aturan bisnis melainkan penjaga. Ia dipasang jauh di atas 500
// supaya tidak diam-diam mengulang pemotongan lama, dan berkas yang menyentuhnya diberi
// tanda di baris terakhir — bukan dipotong tanpa satu pun pemberitahuan, yang persis cacat
// sistem lama.
const exportLimit = 50_000

// Export menangani GET /arsip-dokumen/ekspor — tombol "Export To Excel".
//
// # Kenapa CSV, padahal tombolnya berbunyi Excel
//
// Karena begitulah sistem lama bekerja: `Activity/SearchDataArchiveFilling-Act.xml`
// langkah 17 memanggil `pxConvertResultsToCSV`, bukan pembuat berkas Excel. Labelnya
// menyebut Excel karena berkas CSV memang dibuka dengan Excel — dan label itu
// dipertahankan apa adanya (`D-13`), sementara isinya tetap CSV seperti aslinya.
//
// Membuat berkas `.xlsx` sungguhan akan menambah satu dependensi dan mengubah bentuk
// keluaran terhadap sistem lama. Keduanya keputusan tersendiri, bukan bagian dari menutup
// selisih tombol.
//
// # Penyaringnya sama dengan daftar yang sedang dilihat
//
// Ekspor yang mengabaikan penyaring akan mengeluarkan berkas yang isinya tidak dapat
// dicocokkan dengan apa pun di layar.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, _, ok := h.begin(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	from, readable := parseDate(query.Get("tanggal_dari"))
	if !readable {
		h.writeBadDate(w, r, archivedokumenklaim.FieldFrom)
		return
	}

	until, readable := parseDate(query.Get("tanggal_sampai"))
	if !readable {
		h.writeBadDate(w, r, archivedokumenklaim.FieldTo)
		return
	}

	input := archivedokumenklaim.CriteriaInput{
		Mode:    query.Get("mode"),
		Keyword: query.Get("kata_kunci"),
		From:    from,
		To:      until,
	}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas CSV separuh jadi tanpa satu pun keterangan.
	page := archivedokumenklaim.Pagination{Page: 1, Size: exportChunk}

	first, err := h.service.Search(r.Context(), active.Alias, input, page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	filename := fmt.Sprintf("archive-dokumen-klaim-%s.csv", time.Now().Format("20060102"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// Berkas ini memuat nama tertanggung dan nomor polis; ia tidak boleh mengendap di
	// cache perantara.
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(exportHeader); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	current := first

	for {
		for _, file := range current.Page.Files {
			if written >= exportLimit {
				// Batas tercapai. Barisnya diberi tanda supaya pembaca berkas tahu isinya
				// tidak lengkap.
				_ = writer.Write(exportTruncationNotice(current.Page.Total))
				return
			}
			if err := writer.Write(exportRow(file)); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		// Isi yang sudah tertulis didorong keluar setiap potong, bukan ditahan sampai
		// akhir. Itulah yang membuat unduhan besar mulai mengalir segera dan memori tidak
		// menumpuk.
		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logExportFailure(r, err)
			return
		}
		if flusher, able := w.(http.Flusher); able {
			flusher.Flush()
		}

		if written >= current.Page.Total || len(current.Page.Files) == 0 {
			return
		}

		page.Page++
		current, err = h.service.Search(r.Context(), active.Alias, input, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
	}
}

// exportHeader adalah baris judul berkas CSV.
//
// Judulnya memakai nama kolom yang dibaca petugas di layar, bukan nama kolom basis data —
// berkas ini dibuka orang, bukan mesin. Urutannya pun sama dengan grid, supaya keduanya
// dapat dibandingkan berdampingan.
var exportHeader = []string{
	"NO KLAIM",
	"NO POLIS",
	"TERTANGGUNG",
	"DOL",
	"PIC Teknis",
	"Tgl Terima Dokumen",
	"TGL INPUT",
	"Jumlah Lembar",
	"Tipe Dokumen",
	"Jenis Dokumen",
	"Nama BOX",
	"Kode Filling",
	"USER INPUT",
	"Tanggal Kirim Dok",
}

// exportRow menuliskan satu baris berkas arsip.
//
// Tanggal ditulis `YYYY-MM-DD`, bukan `dd/mm/yyyy` seperti di layar: berkas ini sering
// diurutkan di Excel, dan bentuk hari-di-depan akan terurut sebagai teks yang salah —
// persis cacat `TO_CHAR` yang `09-DATABASE-STRATEGY.md` §3.2 hapus dari SQL.
func exportRow(file archivedokumenklaim.ArchiveFile) []string {
	return []string{
		file.ClaimNumber,
		file.PolicyNumber,
		file.InsuredName,
		exportDate(file.LossDate),
		file.TechnicalPIC,
		exportDate(file.DocumentReceivedDate),
		exportDate(file.InputDate),
		strconv.Itoa(file.SheetCount),

		// Nama dokumen kosong bila kodenya tidak ada di master; kodenya dipakai sebagai
		// gantinya, sama seperti di grid. Sel kosong pada berkas yang dibuka orang lain
		// hanya memberi tahu bahwa datanya hilang.
		firstNonEmpty(file.DocumentTypeName, file.DocumentTypeCode),
		firstNonEmpty(file.DocumentKindName, file.DocumentKindCode),

		file.BoxName,
		file.FillingCode,
		file.InputUser,
		exportDate(file.SentDate),
	}
}

// exportTruncationNotice adalah baris terakhir berkas yang isinya terpotong.
func exportTruncationNotice(total int) []string {
	notice := make([]string, len(exportHeader))
	notice[0] = fmt.Sprintf(
		"-- terpotong pada %d baris dari %d; persempit pencariannya lalu ekspor lagi --",
		exportLimit, total)
	return notice
}

// exportDate menuliskan tanggal, atau sel kosong bila tidak ada.
func exportDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(dateLayout)
}

// firstNonEmpty mengembalikan nilai pertama yang tidak kosong.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Ia hanya dapat dicatat, tidak dapat dijawab: status dan header sudah terlanjur
// dikirimkan, sehingga yang sampai ke pengguna adalah berkas yang terputus. Tanpa catatan
// ini, kegagalannya tidak meninggalkan jejak di mana pun.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	if h.logger == nil {
		return
	}

	alias := ""
	if active, exists := portalhttp.ActivePortalFrom(r.Context()); exists {
		alias = active.Alias
	}

	h.logger.Error("ekspor berkas arsip terputus di tengah jalan",
		slog.String("jalur", r.URL.Path),
		slog.String("portal", alias),
		slog.String("galat", err.Error()),
	)
}
