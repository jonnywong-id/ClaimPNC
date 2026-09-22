package inboxlaporanklaimhttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban
// setiap kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun
// jumlah barisnya (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxlaporanklaim.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas, padahal sistem lama tidak punya
//
// Justru karena sistem lama tidak punya. `pyMaxRecords=500` terpasang pada 54 dari 56
// laporan Pega, sehingga kebutuhan ekspor bervolume besar BELUM PERNAH benar-benar
// dilayani — dan berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka
// `ADR-0011` yang belum dijawab.
//
// Angka ini karena itu bukan aturan bisnis melainkan penjaga: ia mencegah satu
// permintaan menarik puluhan juta baris (`D-10`) sebelum jawabannya ada. Ia dipasang
// jauh di atas 500 supaya tidak diam-diam mengulang pemotongan lama, dan berkas yang
// menyentuhnya diberi tanda di baris terakhir — bukan dipotong tanpa satu pun
// pemberitahuan, yang persis cacat sistem lama.
const exportLimit = 50_000

// Export menangani GET /inbox/laporan-klaim/ekspor — tombol "Export Data".
//
// Asal: `Activity/ExportNotTransferRCV-Act.xml`, yang memanggil `pxConvertResultsToCSV`
// atas halaman hasil yang sedang tampil. Yang dibawa adalah BENTUKNYA — berkas CSV berisi
// kolom yang sama dengan tabel di layar — bukan caranya: Pega mengubah satu halaman
// klipboard, sedangkan di sini barisnya dialirkan potong demi potong dari basis data.
//
// Penyaring yang berlaku sama persis dengan daftar yang sedang dilihat. Ekspor yang
// mengabaikan penyaring akan mengeluarkan berkas yang isinya tidak dapat dicocokkan
// dengan apa pun di layar.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := readQuery(r)
	query.Pagination = inboxlaporanklaim.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas CSV separuh jadi tanpa satu pun keterangan.
	first, err := h.service.List(r.Context(), active.Alias, caller, query)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	now := h.service.Now()
	filename := fmt.Sprintf("laporan-klaim-%s-%s.csv", query.Category, now.Format("20060102"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// Berkas ini memuat data nasabah; ia tidak boleh mengendap di cache perantara.
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(exportHeader); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	page := first
	for {
		for _, report := range page.Page.Report {
			if written >= exportLimit {
				// Batas tercapai. Barisnya diberi tanda supaya pembaca berkas tahu
				// isinya tidak lengkap.
				_ = writer.Write(exportTruncationNotice(page.Page.Total, exportLimit))
				return
			}
			if err := writer.Write(exportRow(report, now)); err != nil {
				h.logExportFailure(r, err)
				return
			}
			written++
		}

		// Isi yang sudah tertulis didorong keluar setiap potong, bukan ditahan sampai
		// akhir. Itulah yang membuat unduhan besar mulai mengalir segera dan memori
		// tidak menumpuk.
		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logExportFailure(r, err)
			return
		}
		if flusher, able := w.(http.Flusher); able {
			flusher.Flush()
		}

		if written >= page.Page.Total || len(page.Page.Report) == 0 {
			return
		}

		query.Pagination.Page++
		page, err = h.service.List(r.Context(), active.Alias, caller, query)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
	}
}

// exportHeader adalah baris judul berkas CSV.
//
// Judulnya memakai nama kolom yang dibaca petugas di layar, bukan nama kolom basis data
// — berkas ini dibuka orang, bukan mesin.
var exportHeader = []string{
	"Case ID",
	"Case PNC",
	"Polis no",
	"Insured Name",
	"Business Name",
	"Reference no",
	"Date of loss",
	"Input Date",
	"Creator",
	"Cabang Klaim",
	"Aging",
	"Total Aging",
	"Position",
	"Alasan",
	"Subject Email",
	"Asal data",
}

func exportRow(r inboxlaporanklaim.ClaimReport, now time.Time) []string {
	return []string{
		r.ID,
		r.ClaimNumber,
		r.PolicyNumber,
		r.InsuredName,
		r.BusinessName,
		r.ReferenceNumber,
		dateText(r.DateOfLoss),
		dateText(r.CreatedAt),
		r.CreatedBy,
		r.BranchName,
		dateText(r.AgingAt),
		strconv.Itoa(r.AgingDays(now)),
		string(r.Position),
		r.Reason,
		r.EmailSubject,
		string(r.Origin),
	}
}

// exportTruncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func exportTruncationNotice(total, limit int) []string {
	notice := make([]string, len(exportHeader))
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %d baris dari %d yang cocok. Persempit penyaringnya. --",
		limit, total,
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
	logging.From(r.Context(), h.logger).Error("ekspor laporan klaim terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
