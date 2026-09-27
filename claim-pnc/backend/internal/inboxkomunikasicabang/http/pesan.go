package inboxkomunikasicabanghttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// BranchOptionDTO adalah satu pilihan pada pemilih cabang.
type BranchOptionDTO struct {
	// Code adalah kode cabang — yang dikirim balik saat pesannya disimpan.
	Code string `json:"kode"`

	// Name adalah nama cabang yang dibaca pengguna.
	Name string `json:"nama"`
}

// BranchListResponse adalah jawaban GET /api/inbox-komunikasi-cabang/cabang.
//
// # Kenapa alamat surel cabang TIDAK ikut dikirim
//
// Karena layar tidak membutuhkannya, dan alamat surel yang dikirim ke peramban ikut tercatat
// di cache, log proxy, dan alat pengembang. Ia dibaca peladen untuk keperluan notifikasi —
// yang belum dibangun — dan tidak pernah meninggalkan peladen.
type BranchListResponse struct {
	Branches []BranchOptionDTO `json:"cabang"`

	// Destinations adalah kedua pilihan dropdown tujuan, apa adanya dari domain.
	//
	// Ia dikirim BERSAMA daftar cabang, bukan ditulis tetap di layar, dengan alasan yang
	// sama seperti kolom grid: keduanya hasil pembacaan export, dan tempat pembacaan itu
	// tercatat adalah peladen.
	Destinations []string `json:"tujuan"`

	Portal string `json:"portal"`
}

// NewMessageRequest adalah badan permintaan POST /api/inbox-komunikasi-cabang/pesan.
//
// Pengirimnya TIDAK ada di sini — ia diambil dari sesi. Begitu pula cabang ASAL pesan, yang
// diturunkan peladen dari login pengirim: keduanya menentukan siapa melihat percakapannya,
// dan nilai yang menentukan hak tidak pernah datang dari permintaan (`R-20`).
type NewMessageRequest struct {
	// Destination adalah "PUSAT" atau "CABANG".
	Destination string `json:"tujuan"`

	// Branch adalah kode cabang tujuan, diisi hanya bila tujuannya "CABANG".
	Branch string `json:"cabang"`

	// Message adalah isi pesannya.
	Message string `json:"pesan"`
}

// Branches menangani GET /api/inbox-komunikasi-cabang/cabang.
//
// GET, dan ia tidak mengubah apa pun. Ia terpisah dari `/tab` meski keduanya memasok bentuk
// layar: `/tab` dibaca dari kode dan tidak pernah berubah selama aplikasi hidup, sementara
// daftar cabang datang dari basis data dan dapat bertambah. Menyatukannya akan memaksa
// keduanya punya masa cache yang sama.
func (h *Handler) Branches(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	branches, err := h.service.Branches(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	options := make([]BranchOptionDTO, 0, len(branches))
	for _, branch := range branches {
		options = append(options, BranchOptionDTO{Code: branch.Code, Name: branch.Name})
	}

	destinations := make([]string, 0, 2)
	for _, destination := range inboxkomunikasicabang.Destinations() {
		destinations = append(destinations, string(destination))
	}

	h.writeJSON(w, r, http.StatusOK, BranchListResponse{
		Branches:     options,
		Destinations: destinations,
		Portal:       active.Alias,
	})
}

// SendMessage menangani POST /api/inbox-komunikasi-cabang/pesan.
//
// Ia MEMBUAT percakapan baru, sehingga jawabannya 201 — berbeda dari balas dan tutup, yang
// mengubah percakapan yang alamatnya sudah ada.
//
// Jalurnya TIDAK bersarang di bawah nomor percakapan, dan itu konsekuensi langsung dari apa
// yang dilakukannya: nomornya belum ada sampai permintaan ini selesai.
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var body NewMessageRequest

	decoder := json.NewDecoder(io.LimitReader(r.Body, maxReplyBodyBytes))

	// Isian yang tidak dikenal ditolak, bukan diabaikan diam-diam — alasannya sama dengan
	// pada balasan, dan di sini akibatnya lebih berat: isian `cabang` yang salah nama akan
	// mengirim pesan ke kantor pusat padahal penggunanya memilih sebuah cabang.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		h.writeError(w, r, inboxkomunikasicabang.NewValidationError(
			[]inboxkomunikasicabang.Violation{{
				Field:   inboxkomunikasicabang.FieldMessageBody,
				Message: "Isi pesan tidak dapat dibaca dari permintaan.",
			}},
		))
		return
	}

	id, err := h.service.SendMessage(
		r.Context(), active.Alias, caller,
		inboxkomunikasicabang.NewMessageInput{
			Destination: body.Destination,
			BranchCode:  body.Branch,
			Message:     body.Message,
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusCreated, ActionResponse{
		ID: id,
		Message: "Pesan terkirim. Ia muncul di tab \"Belum Dijawab\" milik " +
			recipientLabel(body.Destination) + " sampai dibalas.",
		Portal: active.Alias,
	})
}

// recipientLabel menyusun sebutan penerima untuk kalimat jawaban.
//
// Ia menyebut PUSAT atau CABANG, bukan nama cabangnya. Menyebut namanya menuntut pembacaan
// ulang daftar cabang hanya untuk menyusun satu kalimat — dan layar sudah tahu nama cabang
// yang barusan dipilihnya sendiri.
func recipientLabel(destination string) string {
	if strings.EqualFold(strings.TrimSpace(destination),
		string(inboxkomunikasicabang.DestinationBranch)) {
		return "cabang tujuan"
	}
	return "kantor pusat"
}
