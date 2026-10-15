package notification

import (
	"context"
	"sync"

	"claim-pnc/internal/masterxol"
)

// Fake merekam pemberitahuan tanpa mengirim apa pun.
//
// Ia adapter kedua seam masterxol.Notifier — dan adapter kedua itulah yang membuat
// seam-nya benar-benar seam, bukan abstraksi hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Dua kegunaannya nyata, bukan sekadar pelengkap:
//
//   - **Pengujian** dapat membuktikan pemberitahuan benar-benar dilepaskan saat induk
//     disimpan, tanpa server SMTP dan tanpa jaringan.
//   - **Pengembangan dan konfigurasi yang belum lengkap** tetap dapat memakai layarnya.
//     Bila SMTP belum diisi, cmd memasang tiruan ini — sehingga menyimpan induk tetap
//     berhasil dan pemberitahuannya tercatat sebagai riwayat, bukan menggagalkan
//     penyimpanan yang sebenarnya sudah selesai.
type Fake struct {
	mutex sync.Mutex

	// sent menyimpan seluruh pemberitahuan yang dilepaskan, berurutan.
	sent []masterxol.CommitteeSubmission
}

// NewFake membentuk tiruan kosong.
func NewFake() *Fake { return &Fake{} }

// NotifyCommitteeSubmission mencatat pemberitahuan dan selalu berhasil.
func (f *Fake) NotifyCommitteeSubmission(_ context.Context, submission masterxol.CommitteeSubmission) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.sent = append(f.sent, submission)
	return nil
}

// Sent mengembalikan salinan seluruh pemberitahuan yang tercatat.
func (f *Fake) Sent() []masterxol.CommitteeSubmission {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return append([]masterxol.CommitteeSubmission(nil), f.sent...)
}

// Count mengembalikan jumlah pemberitahuan yang tercatat.
func (f *Fake) Count() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return len(f.sent)
}

var _ masterxol.Notifier = (*Fake)(nil)
