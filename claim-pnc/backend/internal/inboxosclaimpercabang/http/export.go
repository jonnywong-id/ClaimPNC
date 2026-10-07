package inboxosclaimpercabanghttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/money"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxosclaimpercabang.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas, padahal sistem lama tidak punya
//
// Justru karena sistem lama tidak punya. `ExportDataOSCabang` memanggil RDB-List tanpa
// `MaxRecords` sama sekali, dan berapa baris yang wajib dilayani satu ekspor adalah pertanyaan
// terbuka `ADR-0011` yang belum dijawab.
//
// Angka ini karena itu bukan aturan bisnis melainkan penjaga: ia mencegah satu permintaan
// menarik puluhan juta baris (`D-10`) sebelum jawabannya ada. Berkas yang menyentuhnya diberi
// tanda di baris terakhir — bukan dipotong tanpa satu pun pemberitahuan, yang persis cacat
// `pyMaxRecords=500` pada 54 dari 56 laporan lama.
//
// Nilainya sama dengan modul Pelaporan Klaim dan Treaty Non-Prop supaya ketiga ekspor tidak
// punya tiga batas berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/inbox-os-claim-per-cabang/ekspor — tombol "Export To Excel".
//
// # Yang dibawa dari `Activity/ExportDataOSCabang-Act.xml`, dan yang tidak
//
// Yang dibawa adalah SUSUNAN KOLOMNYA. Yang tidak dibawa adalah caranya: langkah terakhir
// activity itu memanggil `MSOGenerateExcelFile`, sehingga berkasnya .xlsx. Di sini yang
// dihasilkan CSV — ia dibuka Excel tanpa perantara, tidak menuntut pustaka pihak ketiga, dan
// dapat dialirkan potong demi potong. Yang ketiga tidak mungkin dilakukan penghasil Excel,
// dan justru itu yang membuat memori tetap datar.
//
// Nama berkas lamanya `SummaryOS-ddMMyyyy-HHmmss.xlsx`; bentuk itu dipertahankan, hanya
// ekstensinya yang berubah dan kode cabangnya ikut disebut — lihat exportFilename.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	// Sesi disiapkan SEBELUM satu byte pun ditulis. Setelah header terkirim, galat tidak
	// dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa berkas separuh
	// jadi tanpa satu pun keterangan. Cabang yang tidak diketahui dan sesi yang tidak lengkap
	// karena itu tetap dijawab sebagai galat yang terbaca.
	session, err := h.Service.BeginExport(r.Context(), active.Alias, caller)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	page := inboxosclaimpercabang.Pagination{Page: 1, Size: exportChunk}

	first, err := session.Page(r.Context(), page)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+exportFilename(session.Query)+`"`)
	// Berkas ini memuat nomor polis, nama tertanggung, dan nilai uang; ia tidak boleh
	// mengendap di cache perantara.
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(exportHeader()); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	current := first
	for {
		for _, item := range current.Items {
			if written >= exportLimit {
				// Batas tercapai. Barisnya diberi tanda supaya pembaca berkas tahu isinya
				// tidak lengkap.
				_ = writer.Write(exportTruncationNotice(current.Total))
				return
			}
			if err := writer.Write(exportRow(item)); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		// Isi yang sudah tertulis didorong keluar setiap potong, bukan ditahan sampai akhir.
		// Itulah yang membuat unduhan besar mulai mengalir segera dan memori tidak menumpuk.
		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logExportFailure(r, err)
			return
		}
		if flusher, able := w.(http.Flusher); able {
			flusher.Flush()
		}

		if written >= current.Total || len(current.Items) == 0 {
			return
		}

		page.Page++
		current, err = session.Page(r.Context(), page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
	}
}

// exportBaseHeader adalah judul kolom berkas SEBELUM ke-24 kolom treaty.
//
// Urutannya mengikuti `GetDataOutstandingperCabangExport-SQL.xml` apa adanya. Judulnya memakai
// teks yang dibaca pengguna di grid untuk kolom yang memang ada di grid, dan nama alias kueri
// untuk kolom yang hanya ada di berkas — karena hanya nama itulah yang pernah dilihat penerima
// berkas.
var exportBaseHeader = []string{
	"Cabang",
	"Sumbis",
	"COB",
	"Business Name",
	"Policy No",
	"Nama Insured",
	"Claim No",
	"Registration Date",
	"DOL",
	"Remark Recommendation",
	"Reserve Claim Full",
	"Reserve Claim ASM",
	"Coins",
}

// exportTailHeader adalah judul kolom berkas SESUDAH ke-24 kolom treaty.
var exportTailHeader = []string{
	"Tgl Update Progress Terakhir",
	"Status Progress 1",
	"Status Progress2",
	"PIC",
	"Keterangan",
	"Aging (Hari)",
	"Kode Cabang",
	"Adjuster",
	"COL",
	"Dominan Factor",
	"Kronologis",
}

// exportHeader menyusun baris judul berkas: dasar, lalu ke-24 treaty, lalu ekor.
//
// Ia fungsi, bukan variabel, supaya judul treaty selalu diambil dari satu sumber —
// inboxosclaimpercabang.ExportTreatyColumns — dan tidak dapat berselisih dengan nilai yang
// ditulis exportRow.
func exportHeader() []string {
	header := make([]string, 0,
		len(exportBaseHeader)+len(inboxosclaimpercabang.ExportTreatyColumns)+len(exportTailHeader))

	header = append(header, exportBaseHeader...)
	header = append(header, inboxosclaimpercabang.ExportTreatyColumns...)
	header = append(header, exportTailHeader...)
	return header
}

// exportRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan exportHeader. Keduanya dijaga uji di http/export_test.go, dan
// penjaganya memeriksa PANJANG serta posisi kolom treaty — bukan hanya bahwa keduanya ada.
func exportRow(item inboxosclaimpercabang.ExportRow) []string {
	row := []string{
		item.BranchName,
		item.BusinessSource,
		item.BusinessName,
		item.PolicyBusinessName,
		item.PolicyNumber,
		item.InsuredName,
		item.ClaimNumber,
		isoDate(item.RegisterDate),
		isoDate(item.LossDate),
		item.RemarkRecommendation,
		amount(item.ReserveClaimFull),
		amount(item.ReserveClaimASM),
		amount(item.Coinsurance),
	}

	for _, value := range item.TreatyShares.TreatyValues() {
		row = append(row, amount(value))
	}

	return append(row,
		isoDate(item.LastProgressAt),
		item.ProgressStatus1,
		item.ProgressStatus2,
		item.TechnicalPIC,
		item.ProgressNote,
		strconv.Itoa(item.AgingDays),
		item.BranchCode,
		item.AdjusterName,
		item.CauseOfLoss,
		item.DominantFactors,
		item.Chronology,
	)
}

// amount menuliskan nilai uang untuk berkas.
//
// Ia memakai bentuk kanonik money.Money apa adanya — desimal dua angka, tanpa pemisah ribuan
// dan tanpa lambang mata uang. Itu tepat yang dibutuhkan berkas: ia dibuka di lembar kerja,
// dan angka berpemisah akan terbaca sebagai TEKS di sana sehingga penjumlahan kolomnya
// berhenti bekerja.
//
// Pemformatan untuk mata manusia — "Rp 1.000.000,00" — adalah urusan layar, bukan urusan
// berkas.
func amount(value money.Money) string {
	return value.String()
}

// exportFilename menyusun nama berkas yang menyebut cabangnya.
//
// Sistem lama menamainya `SummaryOS-<cap waktu>.xlsx` — seragam untuk setiap cabang, sehingga
// dua unduhan dari dua cabang menghasilkan dua berkas yang tidak dapat dibedakan tanpa
// membukanya. Kode cabang karena itu ikut disebut.
//
// Cap waktunya TIDAK dibawa: peramban sudah menambahkan penanda sendiri pada berkas bernama
// sama, dan nama yang berubah setiap detik membuat unduhan berulang tidak dapat saling
// menimpa — yang justru menumpuk berkas di folder unduhan pengguna.
func exportFilename(query inboxosclaimpercabang.Query) string {
	code := query.Branch.Code
	if code == "" {
		code = "tanpa-cabang"
	}
	return "SummaryOS-" + code + ".csv"
}

// exportTruncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func exportTruncationNotice(total int) []string {
	notice := make([]string, len(exportHeader()))
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %s baris dari %s yang cocok. Hubungi Tim IT bila seluruhnya "+
			"dibutuhkan. --",
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
	logging.From(r.Context(), h.Logger).Error("ekspor OS klaim per cabang terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
