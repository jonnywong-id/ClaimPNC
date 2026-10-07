package inboxmanageradminhttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxmanageradmin.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas
//
// Karena berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka `ADR-0011`
// yang belum dijawab: `pyMaxRecords=500` terpasang pada 54 dari 56 laporan Pega — termasuk
// Report Definition layar ini — sehingga kebutuhan ekspor bervolume besar belum pernah
// benar-benar dilayani.
//
// Angka ini karena itu bukan aturan bisnis melainkan penjaga: ia mencegah satu permintaan
// menarik puluhan juta baris (`D-10`) sebelum jawabannya ada. Berkas yang menyentuhnya
// diberi tanda di baris terakhir — bukan dipotong tanpa satu pun pemberitahuan, yang persis
// cacat sistem lama.
//
// Nilainya sama dengan modul inbox lain supaya tidak ada dua batas berbeda tanpa alasan.
const exportLimit = 50_000

// Judul kolom berkas ekspor, apa adanya dari sistem lama.
//
// # Kedelapannya disalin dari HeadersMap, bukan disusun ulang
//
// `Activity/ExportExcelManagerAdminPA-Act.xml` menyusun berkasnya lewat
// `pxConvertResultsToCSV` dengan `HeadersMap` berbunyi:
//
//	"ID,No Polis,Nama Tertanggung,Nama Bisnis,Nama Sumber Bisnis,Status Klaim,Tanggal Pendaftaran,AdminPNC"
//
// Work Owner memutuskan 2026-09-26 berkas ekspor mengikuti kedelapan kolom itu apa adanya.
// Dua akibatnya perlu dibaca sebelum ada yang "merapikannya":
//
//   - Berkas memuat **Status Klaim**, kolom yang tidak ada di grid mana pun pada layar yang
//     sama, dan tidak memuat **Lama Waktu Klaim**, kolom yang justru ada di grid.
//   - Judul terakhir berbunyi **"AdminPNC"** tanpa spasi, sedangkan judul kolom yang sama di
//     grid berbunyi "Admin PNC". Ejaan berkas dipertahankan apa adanya (`D-13`) — ia yang
//     selama ini dipakai orang yang mengolah berkasnya di Excel, dan judul kolom adalah hal
//     pertama yang dicocokkan rumus.
//
// Urutannya pun berbeda dari grid: Status Klaim berada SEBELUM Tanggal Pendaftaran.
var exportHeader = []string{
	"ID",
	"No Polis",
	"Nama Tertanggung",
	"Nama Bisnis",
	"Nama Sumber Bisnis",
	"Status Klaim",
	"Tanggal Pendaftaran",
	"AdminPNC",
}

// exportFields adalah isian yang mengisi setiap kolom exportHeader, berurutan.
//
// Keduanya dipisah tetapi WAJIB sejalan, dan uji menjaganya. Menyusunnya sebagai dua senarai
// yang kebetulan sejalan adalah cara paling mudah menggeser isi berkas tanpa menggeser
// judulnya.
var exportFields = []string{
	inboxmanageradmin.FieldCaseID,
	inboxmanageradmin.FieldPolicyNumber,
	inboxmanageradmin.FieldInsuredName,
	inboxmanageradmin.FieldBusinessName,
	inboxmanageradmin.FieldBusinessSource,
	inboxmanageradmin.FieldClaimStatus,
	inboxmanageradmin.FieldRegisteredAt,
	inboxmanageradmin.FieldAdminName,
}

// Export menangani GET /api/inbox-manager-admin/ekspor — tombol "Export To Excel".
//
// # Bentuk berkasnya CSV, meski tombolnya berbunyi Excel
//
// Karena begitu pula sistem lama: activity-nya memanggil `pxConvertResultsToCSV`, bukan
// penulis Excel mana pun. Judul tombolnya dipertahankan apa adanya (`D-13`) — itulah yang
// dikenal pengguna, dan mengubahnya menjadi "Export CSV" berarti mengajari ulang orang
// tentang tombol yang tidak berubah fungsinya.
//
// # Penyaring yang berlaku
//
// Sama persis dengan daftar yang sedang dilihat — readFilter yang sama melayani keduanya.
// Ekspor yang mengabaikan penyaring akan mengeluarkan berkas yang isinya tidak dapat
// dicocokkan dengan apa pun di layar.
//
// Kewenangan tab pun diperiksa dengan jalur yang sama: permintaan ekspor atas tab yang bukan
// hak pemanggil ditolak sebelum satu byte pun ditulis, karena NewQuery yang memutuskan —
// bukan pemeriksaan terpisah yang dapat tertinggal saat aturannya berubah.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	filter := readFilter(r.URL.Query())
	page := inboxmanageradmin.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan. Tab yang bukan haknya dan sesi yang
	// tidak lengkap karena itu tetap dijawab sebagai galat yang terbaca.
	first, err := h.Service.List(r.Context(), active.Alias, caller, filter, page)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	tab := first.Query.Tab

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+exportFilename(tab)+`"`)
	// Berkas ini memuat nomor polis dan nama tertanggung; ia tidak boleh mengendap di
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
		for _, item := range current.Page.Items {
			if written >= exportLimit {
				// Batas tercapai. Barisnya diberi tanda supaya pembaca berkas tahu isinya
				// tidak lengkap.
				_ = writer.Write(exportTruncationNotice(current.Page.Total))
				return
			}
			if err := writer.Write(exportRow(item)); err != nil {
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
		current, err = h.Service.List(r.Context(), active.Alias, caller, filter, page)
		if err != nil {
			h.logExportFailure(r, err)
			return
		}
	}
}

// exportRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan exportHeader, dan keduanya dibangun dari exportFields yang
// sama — bukan dari dua daftar yang kebetulan sejalan.
func exportRow(item inboxmanageradmin.WorkItem) []string {
	row := make([]string, 0, len(exportFields))
	for _, field := range exportFields {
		row = append(row, cellValue(item, field))
	}
	return row
}

// cellValue mengambil isi satu sel menurut nama isiannya.
//
// Ia memakai konstanta Field… yang sama dengan tab.go dan dengan nama field JSON di dto.go.
// Isian yang tidak dikenali menghasilkan teks kosong, bukan panik: berkas ekspor yang
// kehilangan satu kolom masih dapat dipakai, sedangkan permintaan yang gagal di tengah
// unduhan tidak.
func cellValue(item inboxmanageradmin.WorkItem, key string) string {
	switch key {
	case inboxmanageradmin.FieldCaseID:
		return item.CaseID
	case inboxmanageradmin.FieldPolicyNumber:
		return item.PolicyNumber
	case inboxmanageradmin.FieldInsuredName:
		return item.InsuredName
	case inboxmanageradmin.FieldBusinessName:
		return item.BusinessName
	case inboxmanageradmin.FieldBusinessSource:
		return item.BusinessSource
	case inboxmanageradmin.FieldClaimStatus:
		return item.ClaimStatus
	case inboxmanageradmin.FieldRegisteredAt:
		if item.RegisteredAt == nil {
			return ""
		}
		return item.RegisteredAt.UTC().Format(dateLayout)
	case inboxmanageradmin.FieldClaimElapsed:
		return item.ClaimElapsed
	case inboxmanageradmin.FieldAdminName:
		return item.AdminName
	default:
		return ""
	}
}

// exportFilename menyusun nama berkas yang menyebut tab asalnya.
//
// # Ini SELISIH TERENCANA terhadap sistem lama, dan bukan kelalaian
//
// Activity lama menamai berkasnya `"Data PA"` — satu nama tetap, untuk ketiga tab, dengan
// `AppendTimeStampToFileName=False`. Akibatnya tiga unduhan dari tiga tab menghasilkan tiga
// berkas bernama sama di folder unduhan, dan yang berikutnya menimpa yang sebelumnya pada
// sebagian peramban. Di layar ini akibatnya lebih buruk daripada biasa: ketiga berkas punya
// kolom yang IDENTIK, sehingga yang tertimpa tidak dapat dikenali dari isinya sama sekali.
//
// Namanya di sini menyebut unit organisasinya. Ia tidak mengubah satu pun isi berkas.
func exportFilename(tab inboxmanageradmin.Tab) string {
	return "Data " + tab.OrgUnit + ".csv"
}

// exportTruncationNotice menyusun baris terakhir berkas yang isinya terpotong.
//
// Ia ditulis di kolom PERTAMA, bukan disebar ke seluruh kolom: yang membacanya adalah orang
// yang membuka berkas di Excel, dan kolom pertama itu yang terlihat tanpa menggulir.
func exportTruncationNotice(total int) []string {
	notice := make([]string, len(exportHeader))
	notice[0] = fmt.Sprintf(
		"-- Berkas dipotong pada %s baris dari %s baris yang cocok. "+
			"Persempit tab atau minta bantuan tim teknis. --",
		strconv.Itoa(exportLimit), strconv.Itoa(total))
	return notice
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Galat seperti itu tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna adalah
// berkas yang berhenti di tengah. Catatan ini satu-satunya jejaknya, dan tanpa itu keluhan
// "berkasnya tidak lengkap" tidak dapat ditelusuri sama sekali.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	if h.Logger == nil {
		return
	}
	logging.From(r.Context(), h.Logger).Error(
		"ekspor Inbox Manager Admin terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("tab", strings.TrimSpace(r.URL.Query().Get("tab"))),
		slog.String("galat", err.Error()),
	)
}
