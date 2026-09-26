package laporanhasilaihttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claim-pnc/internal/laporanhasilai"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = laporanhasilai.MaxPageSize

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

// csvDateLayout adalah bentuk tanggal di dalam berkas CSV: dd/mm/yyyy.
//
// # Kenapa BERBEDA dari tanggal di layar
//
// Karena layar lamanya pun berbeda, dan keduanya disengaja di sana:
//
//   - Grid menggambar `.TanggalAI` dan `.TanggalComitee` APA ADANYA — teks waktu Pega
//     seperti `20260903T000000.000 GMT`. Itu artefak layar yang belum selesai, bukan
//     bentuk yang dimaksudkan siapa pun.
//   - Berkas CSV menuliskan `.DISC` dan `.DISC2`, yaitu kedua tanggal yang sudah disusun
//     ulang activity menjadi `dd/mm/yyyy`:
//
//     .DISC  := @substring(.TanggalAI,6,8)+"/"+@substring(.TanggalAI,4,6)+"/"+
//     @substring(.TanggalAI,0,4)
//
// Yang ditiru di sini adalah bentuk CSV-nya, karena itulah satu-satunya bentuk yang
// benar-benar dimaksudkan. Di layar, tanggalnya mengikuti pemformatan baku aplikasi
// (`components/format.ts`) supaya tidak ada dua gaya tanggal dalam satu aplikasi.
const csvDateLayout = "02/01/2006"

// exportHeader adalah baris judul berkas CSV.
//
// Kesepuluhnya disalin PERSIS dari `CSVPropHeaders` pada
// `Activity/SearchDataLaporanAI-Act.xml`, termasuk perbedaannya dengan judul kolom di
// layar — dan perbedaan itu nyata:
//
//	judul di layar       judul di berkas CSV
//	Object Name          Nama Object
//	Note AI Terima       Note Terima
//	Note AI Tolak        Note Tolak
//	Coverage Final       Coverage AI Final
//
// Menyeragamkannya akan terasa lebih rapi dan sekaligus mengubah berkas yang sudah dipakai
// orang: nama kolom di CSV ikut terbawa ke dalam formula Excel yang menunjuknya.
var exportHeader = []string{
	"No Klaim",
	"Nama Object",
	"Komite Status",
	"Tanggal Komite",
	"AI Status",
	"Tanggal AI",
	"Note Terima",
	"Note Tolak",
	"Coverage AI Final",
	"Kategori Kronologi",
}

// Export menangani GET /api/laporan-hasil-ai/ekspor.
//
// # Di Pega ia BUKAN tombol yang berbeda
//
// Tombol "Export To Excel" pada layar lama memanggil activity yang SAMA PERSIS dengan
// tombol "Cari Data" — `SearchDataLaporanAI(flagss=2)` — hanya dibuka di jendela baru
// (`pyAction = openUrlInWindow`). Activity itu mengisi grid, lalu langkah
// `Call pxConvertResultsToCSV` mengubah `DatasearchLaporan.pxResults` menjadi berkas.
//
// Di sini keduanya dipisahkan menjadi dua rute karena keluarannya memang dua hal berbeda —
// JSON dan CSV — tetapi keduanya menempuh penyaring dan pembacaan yang SAMA. Ekspor yang
// membaca dengan cara berbeda akan menghasilkan berkas yang isinya tidak dapat dicocokkan
// dengan apa pun di layar.
//
// # Ekspor mengambil SELURUH baris, bukan halaman yang sedang terbuka
//
// Layar lama pun begitu: grid-nya memuat semuanya sekaligus, dan CSV-nya dibuat dari
// muatan itu. Paginasi di sini adalah cara membacanya potong demi potong, bukan
// pembatasan isinya.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	filter := readFilter(r.URL.Query())
	page := laporanhasilai.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan. Kedua tanggal yang belum diisi karena
	// itu tetap dijawab sebagai galat yang terbaca.
	first, err := h.service.ListForExport(r.Context(), active.Alias, filter, page)
	if err != nil {
		h.reportError(w, r, err)
		return
	}

	h.beginDownload(w, exportFilename(filter))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(exportHeader); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	current := first
	for {
		for _, row := range current.Rows {
			if written >= exportLimit {
				_ = writer.Write(truncationNotice(len(exportHeader), current.Total))
				return
			}
			if err := writer.Write(exportRow(row)); err != nil {
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
		next, err := h.service.ListForExport(r.Context(), active.Alias, filter, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
		current = next
	}
}

// exportRow menyusun satu baris berkas, pada urutan exportHeader.
//
// Kelima sel yang SELALU kosong tetap ditulis sebagai sel kosong, bukan dilewati:
// melewatinya akan menggeser seluruh kolom di sebelah kanannya dan membuat berkasnya tidak
// cocok dengan judulnya sendiri.
func exportRow(row laporanhasilai.Row) []string {
	return []string{
		row.ClaimNumber,
		row.ObjectName,
		row.CommitteeStatus,
		csvDate(row.CommitteeDate),
		row.AIStatus,
		csvDate(row.AIDate),
		row.AcceptNote,
		row.RejectNote,
		row.CoverageFinal,
		row.ChronologyCategory,
	}
}

// csvDate menuliskan tanggal sebagai dd/mm/yyyy; nol menjadi sel kosong.
func csvDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(csvDateLayout)
}

// beginDownload memasang header unduhan.
func (h *Handler) beginDownload(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
}

// flush mendorong isi yang sudah tertulis keluar setiap potong, bukan menahannya sampai
// akhir. Itulah yang membuat unduhan besar mulai mengalir segera dan memori tidak
// menumpuk.
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

// exportFilename menyusun nama berkas beserta rentang tanggal yang menghasilkannya.
//
// Rentangnya masuk ke dalam nama dengan sengaja: laporan yang sama diunduh berulang kali
// dengan rentang berbeda, dan berkas yang namanya sama membuat dua rentang tidak dapat
// dibedakan setelah tersimpan — pada sebagian peramban yang berikutnya bahkan menimpa
// yang sebelumnya.
func exportFilename(filter laporanhasilai.Filter) string {
	name := "laporan-hasil-ai"
	clean := filter.Clean()
	if !clean.From.IsZero() && !clean.To.IsZero() {
		name += "-" + clean.From.Format(dateLayout) + "-sd-" + clean.To.Format(dateLayout)
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
		"-- Terpotong pada %s baris dari %s yang cocok. Persempit rentang tanggalnya. --",
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
	logging.From(r.Context(), h.logger).Error("ekspor Laporan Hasil AI terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
