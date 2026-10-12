package inboxrclpuclhttp

import (
	"encoding/json"
	"errors"
	"net/http"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/usecase"
)

// sendReminderRequest adalah badan permintaan tombol "Kirim Reminder".
//
// Ia memuat NOMOR KLAIM saja. Alamat surel dan nama tertanggung sengaja TIDAK diterima dari
// layar: keduanya dibaca ulang di peladen dari basis data. Ini tindakan yang mengirim
// sesuatu keluar, dan alamat yang datang dari badan permintaan dapat ditulis siapa pun yang
// dapat memanggil API.
type sendReminderRequest struct {
	ClaimNumbers []string `json:"nomor_klaim"`
}

// ReminderOutcomeDTO adalah hasil satu klaim.
type ReminderOutcomeDTO struct {
	ClaimNumber string `json:"nomor_klaim"`

	// Status salah satu dari `terkirim`, `tanpa-email`, `tidak-memenuhi-syarat`, `gagal`.
	// Nilainya kontrak, bukan kalimat — layar yang menyusun kalimatnya.
	Status string `json:"status"`

	// Reason hanya terisi pada status `gagal`, dan isinya kalimat yang layak dibaca
	// petugas. Galat mentah peladen SMTP TIDAK diteruskan: ia dapat memuat nama host
	// internal.
	Reason string `json:"alasan,omitempty"`
}

// SendReminderResponse adalah jawaban tombol "Kirim Reminder".
//
// Ia selalu melaporkan PER KLAIM. Pengiriman sebagian adalah hasil yang lazim di sini —
// sebagian klaim punya alamat surel, sebagian belum — dan satu kalimat "berhasil" atau
// "gagal" untuk seluruh permintaan menyembunyikan tepat hal yang perlu ditindak.
type SendReminderResponse struct {
	Hasil []ReminderOutcomeDTO `json:"hasil"`

	// Ringkasan dihitung di PELADEN, bukan di layar.
	//
	// Dua tempat yang menghitung hal yang sama dapat berselisih, dan selisihnya muncul
	// sebagai kalimat "3 terkirim" di atas daftar yang memuat empat baris terkirim.
	Ringkasan ReminderSummaryDTO `json:"ringkasan"`
}

// ReminderSummaryDTO adalah cacah per status.
type ReminderSummaryDTO struct {
	Terkirim            int `json:"terkirim"`
	TanpaEmail          int `json:"tanpa_email"`
	TidakMemenuhiSyarat int `json:"tidak_memenuhi_syarat"`
	Gagal               int `json:"gagal"`
}

// SendReminder menangani `POST /api/inbox-rcl-pucl/reminder`.
//
// # Kenapa POST ke alamat DAFTAR, bukan ke alamat satu klaim
//
// Karena satu permintaan menyangkut BANYAK klaim sekaligus — itulah bentuk modalnya di
// Pega, yang mencentang beberapa baris lalu menekan satu tombol. Memecahnya menjadi satu
// permintaan per klaim akan membuat layar menampilkan hasil separuh jadi saat salah satunya
// gagal, dan tidak ada satu tempat pun yang memegang ringkasannya.
func (h *Handler) SendReminder(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var body sendReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    "badan_tidak_terbaca",
			Message: "Daftar klaim yang dikirim tidak terbaca.",
		})
		return
	}

	hasil, err := h.service.SendReminder(r.Context(), active.Alias, caller, body.ClaimNumbers)
	if err != nil {
		h.writeReminderError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, SendReminderResponse{
		Hasil: reminderOutcomesOf(hasil),
		Ringkasan: ReminderSummaryDTO{
			Terkirim:            hasil.Count(inboxrclpucl.ReminderSent),
			TanpaEmail:          hasil.Count(inboxrclpucl.ReminderNoEmail),
			TidakMemenuhiSyarat: hasil.Count(inboxrclpucl.ReminderNotEligible),
			Gagal:               hasil.Count(inboxrclpucl.ReminderFailed),
		},
	})
}

// writeReminderError memetakan galat yang menyangkut SELURUH permintaan.
//
// Kegagalan PER KLAIM tidak sampai ke sini — ia dilaporkan sebagai baris hasil. Yang di
// bawah adalah kegagalan yang membuat permintaannya sendiri tidak dapat dijalankan.
func (h *Handler) writeReminderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, inboxrclpucl.ErrNoReminderRecipients):
		h.writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
			Code:    "tidak_ada_klaim_dipilih",
			Message: "Pilih dulu klaim yang akan dikirimi pengingat.",
		})

	case errors.Is(err, inboxrclpucl.ErrTooManyReminderRecipients):
		h.writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
			Code: "klaim_terlalu_banyak",
			Message: "Terlalu banyak klaim dipilih sekaligus. Kirim paling banyak " +
				"satu halaman penuh, lalu ulangi untuk sisanya.",
		})

	case errors.Is(err, usecase.ErrReminderUnavailable):
		// 503, bukan 500: ini keadaan lingkungan yang dapat berubah tanpa perbaikan kode,
		// dan pembedaan itu menentukan apakah petugas perlu melapor atau cukup menunggu.
		h.writeJSON(w, r, http.StatusServiceUnavailable, ErrorResponse{
			Code: "pengirim_belum_siap",
			Message: "Pengirim surel belum terkonfigurasi di lingkungan ini. " +
				"Laporkan ke tim teknis.",
		})

	default:
		h.writeError(w, r, err)
	}
}

// reminderOutcomesOf mengubah hasil domain menjadi DTO.
//
// Senarai KOSONG, bukan nil, supaya JSON-nya `[]` dan bukan `null` — layar membedakan
// "tidak ada hasil" dari "isiannya tidak ada".
func reminderOutcomesOf(result inboxrclpucl.ReminderResult) []ReminderOutcomeDTO {
	out := make([]ReminderOutcomeDTO, 0, len(result.Outcomes))
	for _, outcome := range result.Outcomes {
		out = append(out, ReminderOutcomeDTO{
			ClaimNumber: outcome.ClaimNumber,
			Status:      outcome.Status,
			Reason:      outcome.Reason,
		})
	}
	return out
}
