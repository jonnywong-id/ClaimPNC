package notification

import (
	"context"
	"sync"

	"claim-pnc/internal/inboxreceivetka"
)

// Recorder adalah pengisi seam inboxreceivetka.Notifier yang MEREKAM, bukan mengirim.
//
// Ia adapter kedua yang membuat seam Notifier nyata, bukan hipotetis — dan sekaligus yang
// membuat jalur Submit dapat diuji tanpa server surel.
//
// # Ia merekam PERISTIWA, bukan teks surel
//
// Yang disimpan adalah inboxreceivetka.Notice apa adanya. Uji karena itu memeriksa bahwa
// peristiwa yang benar dilepaskan dengan nilai yang benar — bukan bahwa sebuah surel
// tersusun dengan kalimat tertentu. Susunan surelnya diuji terpisah lewat HTMLBody dan
// Subject, yang keduanya fungsi murni.
type Recorder struct {
	mutex    sync.Mutex
	notices  []inboxreceivetka.Notice
	failure  error
	attempts int
}

// NewRecorder membentuk perekam kosong.
func NewRecorder() *Recorder { return &Recorder{} }

// SetError membuat perekam menjawab dengan galat, untuk menguji jalur gagal kirim.
//
// Jalur itu perlu diuji justru karena akibatnya BUKAN kegagalan: penyimpanan sudah
// dikomitkan, dan yang gagal hanya pemberitahuannya. Lihat usecase.Complete.
func (r *Recorder) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// NotifyDocumentCompleted merekam peristiwa, atau mengembalikan galat yang disetel.
//
// Percobaannya dicacah SEBELUM galat diperiksa, sehingga uji dapat membuktikan pengiriman
// benar-benar dicoba meski gagal — berbeda dari keadaan Notifier yang memang tidak dipasang,
// yang tidak pernah sampai ke sini sama sekali.
func (r *Recorder) NotifyDocumentCompleted(
	_ context.Context,
	notice inboxreceivetka.Notice,
) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.attempts++
	if r.failure != nil {
		return r.failure
	}
	r.notices = append(r.notices, notice)
	return nil
}

// Notices mengembalikan salinan seluruh peristiwa yang berhasil direkam.
//
// Salinan, bukan senarai aslinya: mengembalikan yang asli membuat pemanggil dapat mengubah
// isi perekam tanpa menyadarinya, dan uji yang saling memengaruhi adalah uji yang tidak
// dapat dipercaya.
func (r *Recorder) Notices() []inboxreceivetka.Notice {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	result := make([]inboxreceivetka.Notice, len(r.notices))
	copy(result, r.notices)
	return result
}

// Attempts mengembalikan berapa kali pengiriman dicoba, berhasil maupun tidak.
func (r *Recorder) Attempts() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.attempts
}

var _ inboxreceivetka.Notifier = (*Recorder)(nil)
