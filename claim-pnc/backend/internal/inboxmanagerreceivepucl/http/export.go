package inboxmanagerreceivepuclhttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxmanagerreceivepucl.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas
//
// Karena berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka `ADR-0011`
// yang belum dijawab: `pyMaxRecords=500` terpasang pada 54 dari 56 laporan Pega, sehingga
// kebutuhan ekspor bervolume besar belum pernah benar-benar dilayani.
//
// Angka ini karena itu bukan aturan bisnis melainkan penjaga: ia mencegah satu permintaan
// menarik puluhan juta baris (`D-10`) sebelum jawabannya ada. Berkas yang menyentuhnya
// diberi tanda di baris terakhir — bukan dipotong tanpa satu pun pemberitahuan, yang persis
// cacat sistem lama.
//
// Nilainya sama dengan modul inbox lain supaya tidak ada dua batas berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/inbox-manager-receive-pucl/ekspor — tombol ekspor di layar.
//
// # Ia KEMAMPUAN BARU, bukan pemindahan
//
// Layar lama TIDAK punya tombol ekspor. Penelusuran `ReceiveDoucument_Harness-Harness.xml`
// dan `InboxManagerReceive_Section-Section.xml` tidak menemukan satu pun activity ekspor,
// dan tidak ada rule `Generate*CSV` maupun `MSOGenerateExcelFile` yang dirujuk keduanya.
//
// Penambahannya diputuskan Work Owner 2026-09-22, dan dinyatakan ke pengguna lewat
// inboxmanagerreceivepucl.PlannedDifferences supaya tidak terbaca sebagai fitur yang
// "hilang lalu muncul kembali".
//
// # Judul kolomnya mengikuti TAB yang sedang terbuka
//
// Ketiga tab punya kolom yang berbeda — dua tab Receive berbagi sembilan kolom, tab RCL/PUCL
// punya sepuluh yang lain. Berkas ekspor karena itu tidak punya satu susunan tetap: ia
// menyalin susunan grid yang sedang dilihat, sehingga berkas dan layar menyebut hal yang
// sama dengan kata yang sama.
//
// # Penyaring yang berlaku
//
// Sama persis dengan daftar yang sedang dilihat — readFilter yang sama melayani keduanya.
// Ekspor yang mengabaikan penyaring akan mengeluarkan berkas yang isinya tidak dapat
// dicocokkan dengan apa pun di layar.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	filter := readFilter(r.URL.Query())
	page := inboxmanagerreceivepucl.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan. Tab yang tidak dikenal dan sesi yang
	// tidak lengkap karena itu tetap dijawab sebagai galat yang terbaca.
	first, err := h.service.List(r.Context(), active.Alias, caller, filter, page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	tab := first.Query.Tab
	header := exportHeader(tab)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+exportFilename(tab)+`"`)
	// Berkas ini memuat nomor polis dan nama tertanggung; ia tidak boleh mengendap di
	// cache perantara.
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		h.logExportFailure(r, err)
		return
	}

	written := 0
	current := first
	for {
		for _, item := range current.Page.Items {
			if written >= exportLimit {
				// Batas tercapai. Barisnya diberi tanda supaya pembaca berkas tahu isinya
				// tidak lengkap.
				_ = writer.Write(exportTruncationNotice(len(header), current.Page.Total))
				return
			}
			if err := writer.Write(exportRow(tab, item)); err != nil {
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

		if written >= current.Page.Total || len(current.Page.Items) == 0 {
			return
		}

		page.Page++
		current, err = h.service.List(r.Context(), active.Alias, caller, filter, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
	}
}

// exportHeader menyusun baris judul dari kolom tab yang sedang terbuka.
//
// Judulnya sama persis dengan judul kolom di grid, termasuk kolom yang isinya selalu kosong
// ("Jumlah Lembar Dokumen"). Menghilangkan kolom kosong dari berkas akan membuat berkas dan
// layar punya jumlah kolom berbeda — dan orang yang mencocokkan keduanya akan mengira ada
// kolom yang tergeser.
func exportHeader(tab inboxmanagerreceivepucl.Tab) []string {
	header := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		header = append(header, column.Title)
	}
	return header
}

// exportRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan exportHeader, dan keduanya dibangun dari senarai kolom yang
// SAMA — bukan dari dua daftar yang kebetulan sejalan. Dengan begitu penambahan kolom di
// tab.go tidak dapat menggeser isi berkas tanpa menggeser judulnya sekaligus.
func exportRow(
	tab inboxmanagerreceivepucl.Tab,
	item inboxmanagerreceivepucl.WorkItem,
) []string {
	row := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		row = append(row, cellValue(item, column.Key))
	}
	return row
}

// cellValue mengambil isi satu sel menurut nama kolomnya.
//
// Ia memakai konstanta Field… yang sama dengan tab.go dan dengan nama field JSON di dto.go.
// Kolom yang tidak dikenali menghasilkan teks kosong, bukan panik: berkas ekspor yang
// kehilangan satu kolom masih dapat dipakai, sedangkan permintaan yang gagal di tengah
// unduhan tidak.
func cellValue(item inboxmanagerreceivepucl.WorkItem, key string) string {
	switch key {
	case inboxmanagerreceivepucl.FieldCaseID:
		return item.CaseID
	case inboxmanagerreceivepucl.FieldPolicyNumber:
		return item.PolicyNumber
	case inboxmanagerreceivepucl.FieldClaimNumber:
		return item.ClaimNumber
	case inboxmanagerreceivepucl.FieldInsuredName:
		return item.InsuredName
	case inboxmanagerreceivepucl.FieldLossDate:
		return item.LossDate
	case inboxmanagerreceivepucl.FieldClaimType:
		return item.ClaimType
	case inboxmanagerreceivepucl.FieldSenderName:
		return item.SenderName
	case inboxmanagerreceivepucl.FieldDocumentReceivedDate:
		return item.DocumentReceivedDate
	case inboxmanagerreceivepucl.FieldDocumentSheetCount:
		return item.DocumentSheetCount
	case inboxmanagerreceivepucl.FieldInboxEntryAt:
		return item.InboxEntryAt
	case inboxmanagerreceivepucl.FieldAnalystNote:
		return item.AnalystNote
	case inboxmanagerreceivepucl.FieldTrack:
		return item.Track
	case inboxmanagerreceivepucl.FieldTrackStatus:
		return item.TrackStatus
	case inboxmanagerreceivepucl.FieldLetterPrintedAt:
		return item.LetterPrintedAt
	case inboxmanagerreceivepucl.FieldClaimAge:
		return item.ClaimAge
	case inboxmanagerreceivepucl.FieldExpiryStatus:
		return item.ExpiryStatus
	default:
		return ""
	}
}

// exportFilename menyusun nama berkas yang menyebut tab asalnya.
//
// Tanpa itu, tiga unduhan dari tiga tab menghasilkan tiga berkas bernama sama di folder
// unduhan — dan yang berikutnya menimpa yang sebelumnya pada sebagian peramban. Di layar ini
// akibatnya lebih buruk daripada biasa: ketiga berkas punya kolom yang berbeda, sehingga
// yang tertimpa tidak dapat dikenali dari isinya.
func exportFilename(tab inboxmanagerreceivepucl.Tab) string {
	switch tab.Code {
	case inboxmanagerreceivepucl.TabReceivePA:
		return "receive-pa.csv"
	case inboxmanagerreceivepucl.TabReceiveNonMBU:
		return "receive-nonmbu.csv"
	case inboxmanagerreceivepucl.TabRCLPUCL:
		return "rcl-pucl.csv"
	default:
		return "inbox-manager-receive-pucl.csv"
	}
}

// exportTruncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func exportTruncationNotice(width, total int) []string {
	notice := make([]string, width)
	if width == 0 {
		return notice
	}
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %s baris dari %s yang cocok. Persempit penyaringnya. --",
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
	logging.From(r.Context(), h.logger).Error("ekspor inbox manager receive/PUCL terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
