package gateway

import (
	"context"
	"encoding/json"
	"sync"

	"claim-pnc/internal/archivedokumenklaim"
)

// Recorder adalah pengisi seam Gateway yang MEREKAM pengiriman alih-alih mengirimkannya.
//
// Ia dipakai pengujian dan mode pengembangan lokal. Keberadaannya membuat modul ini dapat
// dikerjakan dan diuji penuh sebelum baris alamat layanan Arsip ada di
// POOLDATA.GCNM_CONNECT_REST — pola yang sama dengan adapter fake `F-3` terhadap kontrak
// HCC/HCQ yang belum tiba.
//
// Konsekuensinya disadari dan disebut terang: jalur yang dipakai saat pengembangan bukan
// jalur yang dipakai di produksi, sehingga kelas cacat integrasi baru muncul terlambat.
type Recorder struct {
	mu   sync.Mutex
	sent []archivedokumenklaim.Shipment

	// Code dan Note adalah jawaban yang dikembalikan. Keduanya dapat diatur pengujian
	// untuk menggambarkan jawaban layanan yang berbeda.
	Code string
	Note string

	// Err, bila diisi, dikembalikan alih-alih jawaban. Ia yang membuat jalur kegagalan
	// dapat diuji tanpa mematikan jaringan.
	Err error
}

// NewRecorder membentuk perekam dengan jawaban berhasil sebagai bawaannya.
func NewRecorder() *Recorder {
	return &Recorder{
		Code: "200",
		Note: "Diterima perekam pengembangan; TIDAK dikirim ke sistem Arsip",
	}
}

// Send merekam satu pengiriman dan mengembalikan jawaban yang sudah disiapkan.
func (r *Recorder) Send(
	_ context.Context,
	_ string,
	shipment archivedokumenklaim.Shipment,
) (archivedokumenklaim.Receipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Err != nil {
		return archivedokumenklaim.Receipt{}, r.Err
	}

	r.sent = append(r.sent, shipment)

	// Badan permintaan disusun dengan bentuk yang SAMA dengan klien nyata. Kalau
	// perekamnya menyusun bentuk lain, isi kolom HITARCHIVE di pengembangan tidak akan
	// menyerupai isinya di produksi — dan justru kolom itulah satu-satunya bukti yang
	// tersisa bila kelak pengiriman dipersoalkan.
	body, _ := json.Marshal(requestBody{
		DocumentNumber: shipment.DocumentNumber(),
		RequestedOn:    shipment.RequestedAt.Format("02/01/2006"),
	})

	return archivedokumenklaim.Receipt{
		ID:      shipment.ID,
		Code:    r.Code,
		Note:    r.Note,
		Request: string(body),
		SentAt:  shipment.RequestedAt,
	}, nil
}

// Sent mengembalikan salinan seluruh pengiriman yang terekam.
func (r *Recorder) Sent() []archivedokumenklaim.Shipment {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]archivedokumenklaim.Shipment{}, r.sent...)
}

// Recorder wajib memenuhi seam modul.
var _ archivedokumenklaim.Gateway = (*Recorder)(nil)
