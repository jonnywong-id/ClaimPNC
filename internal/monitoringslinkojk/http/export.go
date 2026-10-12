package monitoringslinkojkhttp

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/usecase"
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

// errLimitReached menghentikan pembacaan begitu batas baris tercapai.
//
// Ia galat internal yang TIDAK pernah sampai ke klien: pemanggil mengenalinya lalu
// menutup berkas dengan baris penanda.
var errLimitReached = errors.New("monitoringslinkojkhttp: batas baris ekspor tercapai")

// Export menangani GET /monitoring-slink-ojk/ekspor — tombol "Export Data".
//
// # Kenapa berkasnya dialirkan, bukan disusun lebih dulu
//
// Ekspor ini berjalan atas data historis puluhan juta baris (`D-10`). Menyusun seluruh
// berkas di memori sebelum mengirimnya membuat memori tumbuh sebanding dengan hasil —
// persis yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	alias, caller, ready := h.context(w, r)
	if !ready {
		return
	}

	request, err := readRequest(r, alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Segmen dan penyaringnya diperiksa SEBELUM satu byte pun ditulis. Sesudah header
	// terkirim, galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna
	// akan berupa berkas CSV separuh jadi tanpa satu pun keterangan.
	if _, err := h.service.Describe(request.Segment); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.stream(w, r, request)
}

// Template menangani GET /monitoring-slink-ojk/format — tombol "Format File".
//
// Ia mengunduh berkas CONTOH berisi satu baris, yang menunjukkan bentuk berkas unggahan
// yang diharapkan. Tidak menyentuh basis data dan tidak menuntut portal-nya siap: isinya
// tetap sama untuk entitas mana pun.
//
// Pemeriksaan sesi tetap berlaku. Berkas ini memang tidak memuat data nasabah, tetapi ia
// memperlihatkan bentuk laporan yang dikirim ke OJK — dan tidak ada alasan membukanya
// bagi siapa pun yang belum masuk.
func (h *Handler) Template(w http.ResponseWriter, r *http.Request) {
	if _, _, ready := h.context(w, r); !ready {
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+monitoringslinkojk.TemplateFileName+`.csv"`)
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	if err := writer.Write(monitoringslinkojk.TemplateColumns); err != nil {
		h.logExportFailure(r, err)
		return
	}
	if err := writer.Write(monitoringslinkojk.TemplateSample); err != nil {
		h.logExportFailure(r, err)
		return
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		h.logExportFailure(r, err)
	}
}

// stream menulis berkas CSV sambil membacanya dari basis data.
func (h *Handler) stream(w http.ResponseWriter, r *http.Request, request usecase.Request) {
	segment := request.Segment

	// Judul dan kolom data diambil TERPISAH, dan jumlahnya boleh berbeda.
	//
	// Pada segmen F06 memang berbeda — 38 judul, 34 kolom data — karena berkas lamanya
	// begitu, dan Work Owner memutuskan itu direplikasi (2026-09-26). Menyatukan keduanya
	// menjadi satu daftar akan memaksa jumlahnya sama, dan itu justru MENGHAPUS
	// pergeseran yang sedang direplikasi. Lihat monitoringslinkojk.ExportSlots.
	headers := monitoringslinkojk.ExportHeaders(segment)
	slots := monitoringslinkojk.ExportSlots(segment)

	var (
		writer  *csv.Writer
		started bool
		written int
	)

	begin := func() error {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition",
			`attachment; filename="`+monitoringslinkojk.FileName(segment)+`.csv"`)
		// Berkas ini memuat data debitur; ia tidak boleh mengendap di cache perantara
		// mana pun.
		w.Header().Set("Cache-Control", "no-store")

		writer = csv.NewWriter(w)
		started = true
		return writer.Write(headers)
	}

	err := h.service.Export(r.Context(), request, func(row monitoringslinkojk.Row) error {
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

		// Satu sel per SLOT, bukan per judul. Slot tanpa kunci menghasilkan sel kosong —
		// itulah 31 dari 34 slot pada segmen F06.
		record := make([]string, len(slots))
		for i, slot := range slots {
			record[i] = row.Get(slot.Key)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
		written++
		return nil
	})

	switch {
	case err == nil, errors.Is(err, errLimitReached):
		// Jalur normal; ditangani di bawah.
	default:
		if !started {
			// Belum satu byte pun terkirim — galatnya masih dapat dijawab sebagai JSON.
			h.writeError(w, r, err)
			return
		}
		// Header sudah terkirim. Yang tersisa hanyalah menandai berkasnya tidak utuh
		// dan mencatat sebabnya; mengirim JSON di sini akan menghasilkan berkas campuran
		// yang tidak dapat dibuka aplikasi mana pun.
		h.logExportFailure(r, err)
		writer.Write([]string{"BERKAS TIDAK LENGKAP — terjadi kesalahan saat membaca data."})
		writer.Flush()
		return
	}

	if !started {
		// Nol baris. Berkasnya tetap dikirim dengan kepala kolomnya, bukan dijawab 204:
		// pelapor yang menekan Export dan tidak menerima apa pun tidak punya cara
		// membedakan "tidak ada data" dari "tombolnya rusak".
		if err := begin(); err != nil {
			h.logExportFailure(r, err)
			return
		}
	}

	if errors.Is(err, errLimitReached) {
		writer.Write([]string{
			"BERKAS DIPOTONG — hanya " + strconv.Itoa(exportLimit) +
				" baris pertama yang disertakan. Persempit rentang tanggalnya lalu ekspor ulang.",
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		h.logExportFailure(r, err)
	}
}

// logExportFailure mencatat kegagalan yang tidak dapat lagi dijawab sebagai galat HTTP.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	if h.logger == nil {
		return
	}
	h.logger.Error("ekspor Monitoring SLINK OJK gagal",
		"jalur", r.URL.Path,
		"galat", err.Error())
}
