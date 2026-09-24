package registrasihttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/registrasi"
)

// Kode galat modul registrasi.
//
// Ia melanjutkan daftar milik modul auth, bukan menggantikannya: klien memakai satu
// mekanisme pembeda untuk seluruh aplikasi. Kontrak galat yang mengikat seluruh aplikasi
// adalah `TKT-F1-004`, yang masih terhalang keputusan Work Owner — sampai itu selesai,
// bentuk ini mengikuti bentuk yang sudah berjalan.
const (
	CodeValidationFailed     = "validasi_gagal"
	CodePolicyNotFound       = "polis_tidak_ditemukan"
	CodeClaimNotFound        = "klaim_tidak_ditemukan"
	CodeTaskNotFound         = "tugas_tidak_ditemukan"
	CodeTaskAlreadyClaimed   = "tugas_sudah_diambil"
	CodeNotTaskOwner         = "bukan_pemilik_tugas"
	CodeTaskAlreadyDone      = "tugas_sudah_selesai"
	CodeStageMismatch        = "tahap_tidak_bersesuai"
	CodeInvalidAction        = "tindakan_tidak_sah"
	CodeExchangeRateNotFound = "kurs_tidak_ditemukan"
	CodeMalformedRequest     = "permintaan_cacat"
	CodeInternalError        = "galat_internal"
)

// mapError memilih status HTTP dan badan respons untuk sebuah galat.
func mapError(err error) (int, ErrorResponse) {
	var validation *registrasi.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: badan permintaan terbaca dengan benar dan bentuknya sah —
		// yang ditolak adalah ISINYA menurut aturan bisnis. Membedakan keduanya
		// menentukan apakah layar menandai kolom atau melaporkan cacat pemrograman.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:      CodeValidationFailed,
			Message:   shortMessage(validation),
			Violation: violationsDTO(validation),
		}

	case errors.Is(err, registrasi.ErrPolicyNotFound):
		// 422, bukan 404: yang tidak ditemukan bukan alamat yang diminta melainkan ISI
		// permintaannya, dan petugas dapat memperbaikinya sendiri. Pesannya menyebut
		// tindakan yang mungkin, bukan sekadar menyatakan kegagalan.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code: CodePolicyNotFound,
			Message: "Nomor Polis tidak ditemukan. Periksa kembali nomornya, " +
				"atau pastikan polisnya sudah terbit di sistem polis.",
		}

	case errors.Is(err, registrasi.ErrClaimNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeClaimNotFound,
			Message: "Klaim tidak ditemukan.",
		}

	case errors.Is(err, registrasi.ErrTaskNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeTaskNotFound,
			Message: "Tugas tidak ditemukan.",
		}

	case errors.Is(err, registrasi.ErrTaskAlreadyClaimed):
		// 409: rekan kerja mendahului. Mengulang permintaan tidak akan menolong, dan
		// ini bukan kesalahan pemanggil.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeTaskAlreadyClaimed,
			Message: "Tugas ini sudah diambil pengguna lain.",
		}

	case errors.Is(err, registrasi.ErrNotTaskOwner):
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeNotTaskOwner,
			Message: "Tugas ini bukan milik Anda.",
		}

	case errors.Is(err, registrasi.ErrTaskAlreadyDone):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeTaskAlreadyDone,
			Message: "Tugas ini sudah selesai dikerjakan.",
		}

	case errors.Is(err, registrasi.ErrStageMismatch):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeStageMismatch,
			Message: "Klaim sudah berpindah tahap. Muat ulang layar sebelum menyimpan.",
		}

	case errors.Is(err, registrasi.ErrInvalidAction):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeInvalidAction,
			Message: "Tindakan itu tidak berlaku pada tahap ini.",
		}

	case errors.Is(err, registrasi.ErrExchangeRateNotFound):
		// Kurs yang belum diisi MENOLAK klaim; ia tidak diganti nilai bawaan
		// (`ADR-0015`). Pesannya menyebut apa yang harus dilakukan orang, bukan apa
		// yang gagal di dalam sistem.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeExchangeRateNotFound,
			Message: "Kurs mata uang pada tanggal kejadian belum tersedia. Hubungi bagian yang mengisi kurs.",
		}

	default:
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternalError,
			Message: "Terjadi kesalahan pada sistem.",
		}
	}
}

// shortMessage mengambil pelanggaran pertama sebagai ringkasan.
//
// Yang pertama bukan sembarang pertama: urutannya sama dengan urutan langkah pada
// `Activity/InputRegister_act-Act.xml`, sehingga pesan ringkas ini selalu sama dengan
// satu-satunya pesan yang ditampilkan Pega.
func shortMessage(g *registrasi.ValidationError) string {
	if p, ok := g.First(); ok {
		return p.Message
	}
	return "Data klaim belum memenuhi ketentuan."
}

func violationsDTO(g *registrasi.ValidationError) []ViolationDTO {
	result := make([]ViolationDTO, 0, len(g.Violation))
	for _, p := range g.Violation {
		result = append(result, ViolationDTO{
			Code:    string(p.Code),
			Field:   p.Field,
			Message: p.Message,
		})
	}
	return result
}
