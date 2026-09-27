package reportklaim

import (
	"errors"
	"strings"
)

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrUnknownReport: kode laporan yang diminta tidak ada di katalog.
	ErrUnknownReport = errors.New("reportklaim: laporan tidak dikenal")

	// ErrReportNotReady: laporannya ada di katalog, tetapi belum dapat dijalankan.
	//
	// Ia DIBEDAKAN dari ErrUnknownReport meski keduanya berakhir sebagai penolakan.
	// Sebab dan tindak lanjutnya berlainan:
	//
	//	ErrUnknownReport   kodenya salah ketik, atau layar memakai kode yang sudah tidak ada
	//	ErrReportNotReady  kodenya benar; yang belum ada adalah artefak Pega-nya (`R-16`)
	//
	// Menyatukan keduanya akan membuat penghalang migrasi terbaca sebagai salah ketik,
	// lalu dicari di tempat yang salah.
	ErrReportNotReady = errors.New("reportklaim: laporan belum dapat dijalankan")

	// ErrUnknownBusinessLine: pilihan pada dropdown "Treaty" tidak ada di daftar.
	ErrUnknownBusinessLine = errors.New("reportklaim: pilihan lini bisnis tidak dikenal")

	// ErrUnknownAction: tombol yang diminta tidak ada pada panel itu.
	//
	// Ia juga yang menjawab permintaan TANPA kode tombol pada panel bertombol dua.
	// Menebak salah satunya berarti memilihkan "Approve" atau "Rejected" untuk pengguna,
	// dan keduanya laporan yang berbeda isi.
	ErrUnknownAction = errors.New("reportklaim: tombol laporan tidak dikenal")

	// ErrCallerUnknown: identitas pemanggil tidak diketahui.
	//
	// Ia BUKAN galat sesi. Sesi sudah diperiksa middleware jauh sebelum sampai ke sini;
	// yang ini berarti rakitan di cmd tidak memasang jembatan ke konteks pemanggil, dan
	// itu cacat pemrograman yang harus terlihat.
	ErrCallerUnknown = errors.New("reportklaim: identitas pemanggil tidak diketahui")
)

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Kesetaraan perilaku, bukan selera (`P-5`): layar lama menampilkan seluruh pesan
// bersamaan (`11-CROSSCUTTING.md` §1.2 aturan 1).
type ValidationError struct {
	Violation []Violation
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Violation))
	for _, p := range e.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "reportklaim: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}
