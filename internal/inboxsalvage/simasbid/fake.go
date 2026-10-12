package simasbid

import (
	"context"
	"sync"

	"claim-pnc/internal/inboxsalvage"
)

// Recorder adalah pengisi seam inboxsalvage.AuctionHouse yang MEREKAM, bukan mengirim.
//
// Ia adapter kedua yang membuat seam AuctionHouse nyata, bukan hipotetis — dan sekaligus
// yang membuat jalur Submit dapat diuji tanpa menembak sistem di luar aplikasi ini.
type Recorder struct {
	mutex       sync.Mutex
	submissions []inboxsalvage.AuctionSubmission
	receipt     inboxsalvage.AuctionReceipt
	failure     error
	attempts    int
}

// NewRecorder membentuk perekam yang menjawab "Sukses".
//
// Jawaban bawaannya sengaja yang BERHASIL, karena itulah jalur yang paling banyak diuji.
// Jalur gagal disetel eksplisit lewat SetReceipt atau SetError, sehingga uji yang
// mengujinya terbaca sebagai uji jalur gagal.
func NewRecorder() *Recorder {
	return &Recorder{receipt: inboxsalvage.AuctionReceipt{Message: "Sukses"}}
}

// SetReceipt menyetel jawaban balai lelang.
func (r *Recorder) SetReceipt(receipt inboxsalvage.AuctionReceipt) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.receipt = receipt
}

// SetError membuat perekam menjawab dengan galat, untuk menguji jalur gagal hubung.
//
// Ia DIBEDAKAN dari jawaban yang menolak: yang satu berarti pengajuannya tidak pernah
// sampai, yang lain berarti sampai lalu ditolak. Keduanya menuntut tindakan yang berbeda
// dari petugas.
func (r *Recorder) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// SendSalvage merekam pengajuan, atau mengembalikan galat yang disetel.
//
// Percobaannya dicacah SEBELUM galat diperiksa, sehingga uji dapat membuktikan pengiriman
// benar-benar dicoba meski gagal — berbeda dari keadaan AuctionHouse yang memang tidak
// dipasang, yang tidak pernah sampai ke sini sama sekali.
func (r *Recorder) SendSalvage(
	_ context.Context,
	submission inboxsalvage.AuctionSubmission,
) (inboxsalvage.AuctionReceipt, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.attempts++
	if r.failure != nil {
		return inboxsalvage.AuctionReceipt{}, r.failure
	}
	r.submissions = append(r.submissions, submission)
	return r.receipt, nil
}

// Submissions mengembalikan salinan seluruh pengajuan yang berhasil direkam.
func (r *Recorder) Submissions() []inboxsalvage.AuctionSubmission {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	result := make([]inboxsalvage.AuctionSubmission, len(r.submissions))
	copy(result, r.submissions)
	return result
}

// Attempts mengembalikan berapa kali pengiriman dicoba, berhasil maupun tidak.
func (r *Recorder) Attempts() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.attempts
}

var _ inboxsalvage.AuctionHouse = (*Recorder)(nil)
