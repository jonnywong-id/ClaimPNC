package masterpanelhttp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/masterpanel"
	masterpanelusecase "claim-pnc/internal/masterpanel/usecase"
	portalhttp "claim-pnc/internal/portal/http"
)

// Kode galat jalur unggah CSV.
//
// Dipisahkan dari kode jalur unggah dokumen: yang gagal di sini adalah BERKASNYA sebagai
// keseluruhan, bukan penyimpanannya, dan yang dapat dilakukan pengguna pun berbeda —
// memperbaiki berkasnya, bukan mengulang unggahannya.
const (
	// CodeCSVInvalid: berkasnya tidak dapat dibaca sebagai CSV modul ini.
	CodeCSVInvalid = "csv_tidak_sah"
	// CodeCSVTooManyRows: barisnya melebihi batas satu unggahan.
	CodeCSVTooManyRows = "csv_terlalu_banyak_baris"
)

// ImportPanelCSV menerima berkas master panel.
//
// # Jawabannya 200, bukan 201, bahkan saat ada baris yang gagal
//
// Permintaannya sendiri BERHASIL diproses: setiap baris dibaca, dan hasilnya dilaporkan satu
// per satu. Menjawab 4xx karena sebagian baris ditolak akan membuat layar memperlakukan
// seluruh unggahan sebagai gagal, padahal sebagian datanya sudah masuk — dan pengguna yang
// mengunggah ulang karenanya akan menimpa baris yang sudah benar.
//
// Yang menjadikannya 4xx hanyalah berkas yang tidak dapat dibaca sama sekali.
func (h *Handler) ImportPanelCSV(w http.ResponseWriter, r *http.Request) {
	h.importCSV(w, r, h.service.ImportPanelCSV)
}

// ImportLocationCSV menerima berkas lokasi panel.
func (h *Handler) ImportLocationCSV(w http.ResponseWriter, r *http.Request) {
	// Nama berkasnya diabaikan jalur lokasi: Pega tidak melampirkan berkas CSV lokasi ke
	// panel mana pun — `PNCUploadLokasiSisiPanel_Act` justru MEMPERTAHANKAN CoverID yang
	// sudah ada (`TempLokasiPanel.CoverID := TempPanelHE.pxResults(1).CoverID`).
	h.importCSV(w, r, func(
		ctx context.Context, portalAlias string, content []byte, _ string,
		by masterpanelusecase.Actor, logger *slog.Logger,
	) (masterpanelusecase.ImportReport, error) {
		return h.service.ImportLocationCSV(ctx, portalAlias, content, by, logger)
	})
}

// importer adalah bentuk kedua jalur unggah CSV.
type importer func(
	ctx context.Context, portalAlias string, content []byte, fileName string,
	by masterpanelusecase.Actor, logger *slog.Logger,
) (masterpanelusecase.ImportReport, error)

// importCSV melayani kedua jalur unggah CSV.
//
// Disatukan karena keduanya BENAR-BENAR sama sampai ke pemanggilan layanannya: portal,
// batas ukuran, nama bagian multipart, identitas pemanggil, dan bentuk jawabannya tidak
// berbeda satu pun. Menyalinnya dua kali akan membuat salah satunya tertinggal pada
// perbaikan berikutnya — dan yang tertinggal itu biasanya pemeriksaan portal.
func (h *Handler) importCSV(w http.ResponseWriter, r *http.Request, run importer) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists || strings.TrimSpace(active.Alias) == "" {
		h.writeModuleError(w, r, errors.New("masterpanel/http: portal aktif tidak dikenali"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)

	file, header, err := r.FormFile("berkas")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeModuleError(w, r, tooLarge(err))
			return
		}
		h.writeModuleError(w, r, &masterpanel.DocumentUploadError{
			Kind:    masterpanel.UploadInvalid,
			Message: `Berkas tidak ditemukan pada permintaan. Sertakan bagian bernama "berkas".`,
			Err:     err,
		})
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeModuleError(w, r, tooLarge(err))
			return
		}
		h.writeModuleError(w, r, err)
		return
	}

	by, known := h.actor(r)
	if !known {
		h.writeError(w, r, errors.New("masterpanel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	report, err := run(r.Context(), active.Alias, content, baseName(header.Filename), by, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ImportResponse{Data: toImportDTO(report)})
}

// mapCSVError memetakan kegagalan membaca berkas menjadi status dan badan respons.
func mapCSVError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, masterpanel.ErrCSVEmpty):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeCSVInvalid,
			Message: "Berkas tidak memuat satu baris data pun.",
			Detail:  []ViolationDTO{{Field: "berkas", Message: "Berkas kosong."}},
		}, true

	case errors.Is(err, masterpanel.ErrCSVHeaderMissing):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodeCSVInvalid,
			// Nama header yang kurang ikut disebut: tanpa itu pengguna harus menebak
			// kolom mana yang salah ketik di antara sepuluh.
			Message: "Header berkas tidak lengkap. " + headerHint(err),
			Detail:  []ViolationDTO{{Field: "berkas", Message: headerHint(err)}},
		}, true

	case errors.Is(err, masterpanel.ErrCSVTooManyRows):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodeCSVTooManyRows,
			Message: "Berkas memuat lebih dari " +
				strconv.Itoa(masterpanel.MaxCSVRows) +
				" baris. Pecah berkasnya lalu unggah bergantian.",
			Detail: []ViolationDTO{{Field: "berkas", Message: "Terlalu banyak baris."}},
		}, true
	}
	return 0, ErrorResponse{}, false
}

// headerHint mengambil daftar header yang kurang dari pesan galatnya.
//
// Galat domainnya berbentuk `...: NAME, STS_PECAH`, dan yang dibutuhkan layar hanya bagian
// setelah titik dua.
func headerHint(err error) string {
	message := err.Error()
	if cut := strings.LastIndex(message, ": "); cut >= 0 {
		return "Kolom yang kurang: " + message[cut+2:] + "."
	}
	return "Periksa baris pertama berkasnya."
}

// ImportRowDTO adalah hasil satu baris.
type ImportRowDTO struct {
	Baris int    `json:"baris"`
	Nama  string `json:"nama_panel"`
	ID    string `json:"id_panel"`
	Hasil string `json:"hasil"`
	Pesan string `json:"pesan"`
}

// ImportReportDTO meringkas satu unggahan.
type ImportReportDTO struct {
	Total      int            `json:"total"`
	Baru       int            `json:"baru"`
	Diperbarui int            `json:"diperbarui"`
	Gagal      int            `json:"gagal"`
	Baris      []ImportRowDTO `json:"baris"`
}

// ImportResponse membungkus laporan unggahan.
type ImportResponse struct {
	Data ImportReportDTO `json:"data"`
}

// toImportDTO memindahkan laporan domain ke bentuk kawat.
func toImportDTO(report masterpanelusecase.ImportReport) ImportReportDTO {
	rows := make([]ImportRowDTO, 0, len(report.Rows))
	for _, row := range report.Rows {
		rows = append(rows, ImportRowDTO{
			Baris: row.Line,
			Nama:  row.Name,
			ID:    row.ID,
			Hasil: string(row.Outcome),
			Pesan: row.Message,
		})
	}
	return ImportReportDTO{
		Total:      report.Total,
		Baru:       report.Created,
		Diperbarui: report.Updated,
		Gagal:      report.Failed,
		Baris:      rows,
	}
}
