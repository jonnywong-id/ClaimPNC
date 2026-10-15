package reportklaimhttp

import (
	"encoding/csv"
	"net/http"
	"strings"

	"claim-pnc/internal/reportklaim"
	"claim-pnc/internal/reportklaim/usecase"
)

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
// menarik puluhan juta baris (`D-10`) sebelum jawabannya ada. Ia dipasang jauh di atas
// 500 supaya tidak diam-diam mengulang pemotongan lama, dan berkas yang menyentuhnya
// diberi TANDA di baris terakhir — bukan dipotong tanpa satu pun pemberitahuan, yang
// persis cacat sistem lama.
const exportLimit = 200_000

// Export menangani GET /report-klaim/{kode}/ekspor — tombol Export pada satu kartu.
//
// # Kenapa berkasnya dialirkan, bukan disusun lebih dulu
//
// Laporan ini berjalan atas data historis puluhan juta baris (`D-10`). Menyusun seluruh
// berkas di memori sebelum mengirimnya membuat memori tumbuh sebanding dengan hasil —
// persis yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	filter, err := readFilter(r, active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	request := usecase.ExportRequest{
		PortalAlias: active.Alias,
		Caller:      caller,
		Code:        reportCode(r),
		Action:      strings.TrimSpace(r.URL.Query().Get("aksi")),
		Filter:      filter,
	}

	// Laporan dan penyaringnya diperiksa SEBELUM satu byte pun ditulis. Sesudah header
	// terkirim, galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna
	// akan berupa berkas CSV separuh jadi tanpa satu pun keterangan.
	report, err := reportklaim.Lookup(request.Code)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	if _, known := report.Action(request.Action); !known {
		h.writeModuleError(w, r, reportklaim.ErrUnknownAction)
		return
	}
	if violation := report.Validate(filter); len(violation) > 0 {
		h.writeModuleError(w, r, &reportklaim.ValidationError{Violation: violation})
		return
	}

	h.stream(w, r, report, request)
}

// stream menulis berkas CSV sambil membacanya dari basis data.
func (h *Handler) stream(
	w http.ResponseWriter,
	r *http.Request,
	report reportklaim.Report,
	request usecase.ExportRequest,
) {
	columns := report.Columns(request.Filter)
	field := reportklaim.Fields(columns)

	// Baris pertama ditulis SETELAH header HTTP dipasang, tetapi keduanya tetap terjadi
	// sebelum kueri berjalan. Kegagalan kueri karena itu masih dapat dijawab sebagai
	// galat — lihat percobaan pertama di bawah.
	var (
		writer  *csv.Writer
		started bool
		written int
		failure error
	)

	begin := func() error {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+report.FileName(request.Filter)+`"`)
		// Berkas ini memuat data nasabah LINTAS CABANG; ia tidak boleh mengendap di
		// cache perantara mana pun.
		w.Header().Set("Cache-Control", "no-store")

		writer = csv.NewWriter(w)
		started = true
		return writer.Write(reportklaim.Headers(columns))
	}

	_, _, err := h.service.Export(r.Context(), request, func(row reportklaim.Row) error {
		if !started {
			if err := begin(); err != nil {
				return err
			}
		}
		if written >= exportLimit {
			// Berhenti membaca, tetapi JANGAN diam. Baris penanda di bawah yang
			// memberitahu pembacanya bahwa berkas ini tidak utuh.
			return errLimitReached
		}

		record := make([]string, len(field))
		for i, f := range field {
			record[i] = row.Value(f)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
		written++
		return nil
	})

	switch {
	case err == errLimitReached:
		failure = nil
	case err != nil:
		failure = err
	}

	if failure != nil && !started {
		// Belum satu byte pun terkirim: galatnya masih dapat dijawab sebagai JSON.
		h.writeModuleError(w, r, failure)
		return
	}

	if !started {
		// Tidak ada satu baris pun. Berkasnya tetap dikirim — berisi judul kolom saja.
		//
		// Berkas berjudul kolom lebih baik daripada berkas kosong: ia membuktikan
		// laporannya berjalan dan memang tidak ada data pada penyaring itu, bukan
		// meninggalkan pengguna menebak antara "tidak ada data" dan "gagal diam-diam".
		if err := begin(); err != nil {
			h.logExportFailure(r, err)
			return
		}
	}

	if err == errLimitReached {
		_ = writer.Write(batasTercapai(len(field)))
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		h.logExportFailure(r, err)
		return
	}
	if failure != nil {
		h.logExportFailure(r, failure)
	}
}

// errLimitReached menghentikan pengaliran saat batas baris tercapai.
//
// Ia galat internal yang TIDAK pernah sampai ke pengguna sebagai galat: yang sampai
// adalah berkas beserta baris penandanya.
var errLimitReached = errPenanda("reportklaim/http: batas baris ekspor tercapai")

type errPenanda string

func (e errPenanda) Error() string { return string(e) }

// batasTercapai menyusun baris penanda di akhir berkas yang terpotong.
//
// Ia ditulis pada kolom PERTAMA supaya terbaca tanpa menggulir ke kanan pada berkas
// berkolom delapan puluh.
func batasTercapai(columns int) []string {
	record := make([]string, columns)
	record[0] = "-- BERKAS DIPOTONG: batas " +
		formatRibuan(exportLimit) +
		" baris tercapai. Persempit rentang tanggal atau lini bisnisnya. --"
	return record
}

// formatRibuan menulis angka dengan titik sebagai pemisah ribuan.
func formatRibuan(n int) string {
	digit := []byte{}
	for n > 0 {
		digit = append(digit, byte('0'+n%10))
		n /= 10
	}
	var b strings.Builder
	for i := len(digit) - 1; i >= 0; i-- {
		b.WriteByte(digit[i])
		if i > 0 && i%3 == 0 {
			b.WriteByte('.')
		}
	}
	return b.String()
}
