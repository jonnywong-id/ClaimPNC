package inboxclaimtreatynonprophttp

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/platform/logging"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja sama dengan batas halaman biasa: hasilnya DITULIS langsung ke jawaban setiap
// kali satu potong selesai dibaca, sehingga memori tetap datar berapa pun jumlah barisnya
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
const exportChunk = inboxclaimtreatynonprop.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas, padahal sistem lama tidak punya
//
// Justru karena sistem lama tidak punya. `pyMaxRecords=500` terpasang pada 54 dari 56
// laporan Pega, sehingga kebutuhan ekspor bervolume besar BELUM PERNAH benar-benar
// dilayani — dan berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka
// `ADR-0011` yang belum dijawab.
//
// Angka ini karena itu bukan aturan bisnis melainkan penjaga: ia mencegah satu permintaan
// menarik puluhan juta baris (`D-10`) sebelum jawabannya ada. Ia dipasang jauh di atas 500
// supaya tidak diam-diam mengulang pemotongan lama, dan berkas yang menyentuhnya diberi
// tanda di baris terakhir — bukan dipotong tanpa satu pun pemberitahuan, yang persis cacat
// sistem lama.
//
// Nilainya sama dengan modul Pelaporan Klaim supaya kedua ekspor tidak punya dua batas
// berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/inbox-claim-treaty-non-prop/ekspor — tombol ekspor di layar.
//
// # Asalnya
//
// `Activity/GenerateClaimNonPropCSV-Act.xml`. Yang dibawa adalah BENTUKNYA — berkas berisi
// tujuh kolom yang sama dengan urutan yang sama — bukan caranya.
//
// # Tiga hal tentang activity itu yang perlu diketahui
//
// 1. **Namanya menyebut CSV, keluarannya bukan.** Langkah terakhirnya memanggil
// `MSOGenerateExcelFile` atas halaman `TempData`, sehingga berkas yang benar-benar diunduh
// pengguna adalah berkas Excel. Di sini yang dihasilkan CSV sungguhan: ia dibuka Excel
// tanpa perantara, tidak menuntut pustaka pihak ketiga, dan dapat dialirkan potong demi
// potong — yang ketiga itu tidak mungkin dilakukan penghasil Excel.
//
// 2. **Baris judulnya tidak terbaca.** `MSOGenerateExcelFile` memakai nama properti sebagai
// judul kolom, dan properti di halaman `TempData` bernama `CARI1`…`CARI7`. Berkas lamanya
// karena itu berjudul kolom `CARI1`, `CARI2`, dan seterusnya. Di sini judulnya memakai teks
// yang dibaca pengguna di grid; isi dan urutan kolomnya tidak berubah sedikit pun. Selisih
// ini dinyatakan lewat inboxclaimtreatynonprop.PlannedDifferences.
//
// 3. **Report Definition-nya hilang dari export.** Langkah 4 activity itu menjalankan
// `InboxKlaimNonPropAdmin` atas kelas `Assign-Worklist`, dan rule itu TIDAK ADA di export.
// Itu tidak menghalangi: susunan kolomnya terbaca utuh dari langkah-langkah `Property-Set`
// sesudahnya, yang menyalin `.CARI11`, `.CARI15`, `.CARI14`, `.CARI16`, `.CARI17`,
// `.CARI22`, dan `.CARI12` ke `TempData` secara berurutan. Sumber barisnya pun sama dengan
// grid yang sedang tampil.
//
// # Penyaring yang berlaku
//
// Sama persis dengan daftar yang sedang dilihat, termasuk tab dan kedua checkbox — readFilter
// yang sama melayani keduanya. Ekspor yang mengabaikan penyaring akan mengeluarkan berkas
// yang isinya tidak dapat dicocokkan dengan apa pun di layar.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	filter := readFilter(r.URL.Query())
	page := inboxclaimtreatynonprop.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan. Tab terhalang dan sesi yang tidak
	// lengkap karena itu tetap dijawab sebagai galat yang terbaca.
	first, err := h.service.List(r.Context(), active.Alias, caller, filter, page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	filename := exportFilename(first.Query)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// Berkas ini memuat nama tertanggung dan nama Ceding Co; ia tidak boleh mengendap di
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
				// Batas tercapai. Barisnya diberi tanda supaya pembaca berkas tahu
				// isinya tidak lengkap.
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

// exportHeader adalah baris judul berkas ekspor.
//
// Ketujuh kolom dan URUTANNYA diambil dari `Activity/GenerateClaimNonPropCSV-Act.xml`, yang
// menyalin properti grid ke `TempData.pxResults` dalam urutan ini:
//
//	CARI1 <- .CARI11   nomor klaim
//	CARI2 <- .CARI15   nama bisnis
//	CARI3 <- .CARI14   sumber bisnis
//	CARI4 <- .CARI16   ceding co
//	CARI5 <- .CARI17   nama tertanggung
//	CARI6 <- .CARI22   tanggal kejadian
//	CARI7 <- .CARI12   operator pembuat
//
// Judulnya yang berbeda, bukan isinya — lihat catatan (2) pada Export. Judul yang dipakai
// adalah judul kolom yang sama di grid, sehingga berkas dan layar menyebut hal yang sama
// dengan kata yang sama.
//
// Perhatikan yang TIDAK ikut: nomor polis, kedua master id, status, aging, dan operator
// pengubah. Kelimanya ada di grid tetapi tidak pernah ada di berkas ekspor sistem lama, dan
// menambahkannya di sini adalah kemampuan baru — bukan pemindahan.
var exportHeader = []string{
	"No Klaim",
	"Business Name",
	"Source of Business",
	"Ceding Co Name",
	"Insured Name",
	"Date of Loss",
	"Create Operator",
}

// exportRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan exportHeader. Keduanya dijaga uji di http/export_test.go.
func exportRow(item inboxclaimtreatynonprop.WorkItem) []string {
	return []string{
		item.ClaimID,
		item.BusinessName,
		item.BusinessSource,
		item.CedingCompany,
		item.InsuredName,
		item.LossDate,
		item.CreateOperator,
	}
}

// exportFilename menyusun nama berkas yang menyebut tab dan penyaringnya.
//
// Sistem lama menamai berkasnya seragam, sehingga dua unduhan dengan penyaring berbeda
// menghasilkan dua berkas bernama sama di folder unduhan — dan yang kedua menimpa yang
// pertama pada sebagian peramban. Nama di sini menyebut apa yang membedakan keduanya.
func exportFilename(q inboxclaimtreatynonprop.Query) string {
	name := "klaim-treaty-non-prop-tab" + q.Tab.Code
	if q.SeeAll {
		name += "-semua"
	}
	if q.TBAOnly {
		name += "-tba"
	}
	return name + ".csv"
}

// exportTruncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func exportTruncationNotice(total int) []string {
	notice := make([]string, len(exportHeader))
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
	logging.From(r.Context(), h.logger).Error("ekspor klaim treaty non-prop terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
