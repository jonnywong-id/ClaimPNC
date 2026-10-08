package inboxinvestigatorhttp

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// dateLayout adalah bentuk tanggal pada parameter kueri ekspor.
//
// ISO `YYYY-MM-DD`, bukan `dd/mm/yyyy` yang dipakai kueri lama. Ia bentuk yang dihasilkan
// `<input type="date">` tanpa perantara, tidak bergantung pada bahasa peramban, dan tidak
// dapat dibaca terbalik — `03/04/2026` berarti dua tanggal berbeda bagi dua pembaca,
// `2026-04-03` hanya satu.
const dateLayout = "2006-01-02"

// Export menangani GET /inbox/investigator/ekspor — tombol "Export Data Investigation".
//
// # Asalnya
//
// `Activity/ExportDataInvestigator-Act.xml`, delapan langkah. Yang dibawa adalah BENTUK
// berkasnya — tiga belas kolom dengan judul dan urutan yang sama — bukan caranya.
//
// # Tiga hal tentang activity itu yang perlu diketahui
//
//  1. **Barisnya bukan dari antrean.** Langkah 5 menjalankan
//     `RDB List/ExportDatainvestigator-SQL.xml` atas `POOLDATA.T_CLAIM_PNC` menurut rentang
//     `INVESTIGATOR_TF_DATE`, bukan atas workbasket. Berkasnya karena itu memuat klaim yang
//     SUDAH SELESAI dan tidak lagi tampil di layar. Itu bukan cacat — itulah gunanya kedua
//     isian tanggal.
//
//  2. **Langkah Report Definition-nya tidak terpakai.** Langkah 2 menjalankan
//     `pxRetrieveReportData` berparameter `Operator = "InvestigatorPNC"`, lalu hasilnya tidak
//     pernah disentuh lagi. Langkah yang tidak dipakai tidak ditiru.
//
//  3. **Judul kolomnya ditulis sebagai satu teks tetap.** Baris 2332 activity itu memuat
//     ketiga belas judulnya persis seperti yang dibaca pengguna, dan itulah yang disalin ke
//     exportHeader — termasuk ejaan "Remaks".
//
// # Ia tidak dialirkan potong demi potong, dan itu pilihan sadar
//
// Modul ekspor lain membaca per halaman lalu menulisnya sambil jalan. Di sini seluruh baris
// diambil sekali, karena kueri ekspornya TIDAK berhalaman: ia satu pernyataan ber-FETCH
// FIRST, dan memecahnya menjadi halaman menuntut penyaring urutan yang tidak ada di sistem
// lama. Batas MaxExportRows yang menjaga memorinya tetap terbatas.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	filter, problem := readExportFilter(r)
	if problem != "" {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: problem,
		})
		return
	}

	// Seluruh baris diambil SEBELUM satu byte pun ditulis. Setelah header terkirim, galat
	// tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa berkas
	// separuh jadi tanpa satu pun keterangan.
	rows, err := h.service.Export(r.Context(), active.Alias, filter)
	if err != nil {
		if errors.Is(err, inboxinvestigator.ErrExportFilterInvalid) {
			h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
				Code:    CodeMalformedRequest,
				Message: exportFilterMessage(err),
			})
			return
		}
		h.writeModuleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+exportFilename(filter)+`"`)
	// Berkas ini memuat alamat rumah sakit dan nomor rekam medis; ia tidak boleh mengendap
	// di cache perantara mana pun (`FR-R2`).
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(exportHeader); err != nil {
		h.logExportFailure(r, err)
		return
	}

	for index, one := range rows {
		if index >= inboxinvestigator.MaxExportRows {
			// Batas tercapai. Barisnya diberi tanda supaya pembaca berkas tahu isinya
			// tidak lengkap — bukan dipotong dalam diam seperti `pyMaxRecords` warisan.
			_ = writer.Write(exportTruncationNotice())
			return
		}
		if err := writer.Write(exportRow(one)); err != nil {
			h.logExportFailure(r, err)
			return
		}
	}
}

// readExportFilter membaca ketiga kendali ekspor dari parameter kueri.
//
// Pesan galatnya ditulis untuk DIBACA PENGGUNA, bukan pengembang: yang salah adalah isian
// di layar, dan kalimat yang menyebut nama parameter tidak menolong siapa pun yang sedang
// mengisinya.
func readExportFilter(r *http.Request) (inboxinvestigator.ExportFilter, string) {
	query := r.URL.Query()

	from, err := time.ParseInLocation(dateLayout, query.Get("dari"), time.UTC)
	if err != nil {
		return inboxinvestigator.ExportFilter{},
			`Tanggal "Dari" tidak dapat dibaca. Isi dengan bentuk YYYY-MM-DD.`
	}
	to, err := time.ParseInLocation(dateLayout, query.Get("sampai"), time.UTC)
	if err != nil {
		return inboxinvestigator.ExportFilter{},
			`Tanggal "Sampai" tidak dapat dibaca. Isi dengan bentuk YYYY-MM-DD.`
	}

	return inboxinvestigator.ExportFilter{
		From:         from,
		To:           to,
		Investigated: query.Get("investigasi"),
	}, ""
}

// exportFilterMessage mengubah galat penyaring menjadi kalimat untuk pengguna.
//
// Galat domainnya sudah ditulis sebagai kalimat; yang dilakukan di sini hanyalah membuang
// awalan teknisnya. Menulis ulang kalimatnya di sini akan membuat dua tempat memutuskan apa
// yang dibaca pengguna, dan keduanya akan berbeda pada perubahan berikutnya.
func exportFilterMessage(err error) string {
	message := err.Error()
	const prefix = "penyaring ekspor tidak sah: "
	if len(message) > len(prefix) && message[:len(prefix)] == prefix {
		return "Ekspor tidak dapat dijalankan: " + message[len(prefix):] + "."
	}
	return "Ekspor tidak dapat dijalankan: penyaringnya belum lengkap."
}

// exportHeader adalah baris judul berkas ekspor.
//
// Disalin APA ADANYA dari `Activity/ExportDataInvestigator-Act.xml:2332`, tempat ketiga
// belas judulnya ditulis sebagai satu teks tetap:
//
//	"Tanggal Investigasi,Alamat RS Klinik,Asuransi Lain,Pasien,Perusahaan,
//	 Tidak ada pembayaran,IsInvestigated,Konfirmasi Model Kwitansi,NoRekap Medis,
//	 NoTelp DiHubungi,Pasien Terdaftar,Remaks,Jenis Rumah Sakit"
//
// Dua di antaranya terbaca seperti salah ketik dan TETAP dipakai: "Remaks" dan
// "IsInvestigated". Keduanya judul yang dihafal pengguna yang mencocokkan berkas baru dengan
// berkas lama kolom per kolom — memperbaikinya di sini berarti menyulitkan orang yang justru
// sedang membuktikan kedua berkas sama.
var exportHeader = []string{
	"Tanggal Investigasi",
	"Alamat RS Klinik",
	"Asuransi Lain",
	"Pasien",
	"Perusahaan",
	"Tidak ada pembayaran",
	"IsInvestigated",
	"Konfirmasi Model Kwitansi",
	"NoRekap Medis",
	"NoTelp DiHubungi",
	"Pasien Terdaftar",
	"Remaks",
	"Jenis Rumah Sakit",
}

// exportRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan exportHeader, dan http/export_test.go yang menjaganya.
//
// Kesebelas nilai teks ditulis APA ADANYA — `"true"`/`"false"` pada kelompok pembayaran,
// `"1"`/`"0"` pada dua kolom lainnya. Sistem lama menyalin properti ke halaman ekspor tanpa
// satu pun transformasi, dan menerjemahkannya di sini akan membuat berkas baru tidak dapat
// dibandingkan baris per baris dengan berkas lama (`P-5`).
func exportRow(row inboxinvestigator.ExportRow) []string {
	return []string{
		formatExportDate(row.InvestigatedAt),
		row.HospitalAddress,
		row.PaidByOtherInsurer,
		row.PaidByPatient,
		row.PaidByCompany,
		row.NoPayment,
		row.Investigated,
		row.ReceiptConfirmation,
		row.MedicalRecordNumber,
		row.PhoneCalled,
		row.PatientRegistered,
		row.Remarks,
		row.HospitalKindLabel(),
	}
}

// formatExportDate menuliskan tanggal investigasi.
//
// Bentuknya `dd/mm/yyyy`, SAMA dengan `to_char(INVESTIGATOR_TF_DATE,'dd/mm/yyyy')` pada
// kueri lama — karena di sinilah kesetaraan berkas benar-benar terlihat: pengguna
// membandingkan kedua berkas sel per sel.
//
// Pemformatannya di Go, bukan di SQL. Itu bukan selera: `TO_CHAR` mengikat kueri pada
// Oracle, dan `D-20` menuntut satu set SQL yang berjalan sama di kedua basis data.
//
// # Zona waktunya WIB
//
// Waktu disimpan UTC (`DB-8`) dan dikonversi ke Asia/Jakarta hanya saat ditampilkan —
// termasuk di berkas. Tanpa konversi, klaim yang ditransfer pukul 07.00 WIB akan tertulis
// pada tanggal SEBELUMNYA, dan itu kelas kesalahan yang melahirkan 118 penyesuaian tujuh
// jam di sistem lama (`R-12`).
//
// Zona dimuat sekali di tingkat paket; bila basis datanya tidak punya pustaka zona, nilainya
// nil dan waktunya ditulis apa adanya — berkas yang tanggalnya bergeser satu hari masih
// jauh lebih berguna daripada berkas yang gagal terbit.
func formatExportDate(moment *time.Time) string {
	if moment == nil {
		return ""
	}
	if jakarta != nil {
		return moment.In(jakarta).Format("02/01/2006")
	}
	return moment.Format("02/01/2006")
}

// jakarta adalah zona waktu tampilan seluruh berkas ekspor modul ini.
var jakarta = func() *time.Location {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return nil
	}
	return location
}()

// exportFilename menyusun nama berkas yang menyebut penyaringnya.
//
// Sistem lama menamai berkasnya seragam, sehingga dua unduhan dengan rentang berbeda
// menghasilkan dua berkas bernama sama di folder unduhan — dan yang kedua menimpa yang
// pertama pada sebagian peramban. Nama di sini menyebut apa yang membedakan keduanya.
func exportFilename(filter inboxinvestigator.ExportFilter) string {
	status := "belum-investigasi"
	if filter.Investigated == inboxinvestigator.InvestigatedYes {
		status = "sudah-investigasi"
	}
	return fmt.Sprintf("data-investigasi-%s-%s-%s.csv",
		filter.From.Format(dateLayout), filter.To.Format(dateLayout), status)
}

// exportTruncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func exportTruncationNotice() []string {
	notice := make([]string, len(exportHeader))
	notice[0] = fmt.Sprintf(
		"-- Terpotong pada %s baris. Persempit rentang tanggalnya. --",
		strconv.Itoa(inboxinvestigator.MaxExportRows))
	return notice
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Ia tidak dapat lagi dijawab sebagai galat HTTP — status sudah 200 dan sebagian berkas
// sudah sampai ke pengguna. Yang dapat dilakukan hanyalah menghentikan penulisan dan
// meninggalkan jejak, supaya unduhan yang terpotong di sisi pengguna punya pasangan
// keterangan di sisi peladen.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Error("ekspor data investigasi terputus",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}
