package kasir

import (
	"context"
	"sync"

	"claim-pnc/internal/masterrekening"
)

// Tiruan memenuhi seam Kasir tanpa jaringan sama sekali.
//
// Ia dipakai di lingkungan development dan di seluruh pengujian. Keberadaannya bukan
// kenyamanan: adapter keduanyalah yang membuat seam Kasir menjadi seam nyata, bukan
// seam hipotetis (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.4).
//
// Ia MENOLAK dipakai di produksi — pemeriksaannya ada di cmd/claimpnc, tempat seluruh
// penolakan lingkungan lain berada.
type Tiruan struct {
	mu sync.Mutex

	// Jawaban menentukan apa yang dikembalikan panggilan berikutnya. Nol-nilainya
	// adalah pendaftaran yang berhasil.
	Jawaban masterrekening.HasilKasir

	// Galat, bila terisi, dikembalikan sebagai kegagalan menghubungi Kasir.
	Galat error

	// Didaftarkan dan Diperbarui mencatat rekening yang dikirim, supaya pengujian
	// dapat memeriksa jalur mana yang dipakai.
	Didaftarkan []masterrekening.Rekening
	Diperbarui  []masterrekening.Rekening
}

// TiruanBaru membentuk tiruan yang selalu menjawab berhasil.
func TiruanBaru() *Tiruan {
	return &Tiruan{
		Jawaban: masterrekening.HasilKasir{
			Berhasil:   true,
			Kode:       "0",
			Pesan:      "[TIRUAN] Rekening diterima sistem Kasir.",
			IDRekening: "TIRUAN-0001",
		},
	}
}

// Daftarkan mencatat rekening dan mengembalikan jawaban yang sudah disiapkan.
func (t *Tiruan) Daftarkan(_ context.Context, r masterrekening.Rekening) (masterrekening.HasilKasir, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Didaftarkan = append(t.Didaftarkan, r)
	if t.Galat != nil {
		return masterrekening.HasilKasir{}, t.Galat
	}
	return t.Jawaban, nil
}

// Perbarui mencatat rekening dan mengembalikan jawaban yang sudah disiapkan.
func (t *Tiruan) Perbarui(_ context.Context, r masterrekening.Rekening) (masterrekening.HasilKasir, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Diperbarui = append(t.Diperbarui, r)
	if t.Galat != nil {
		return masterrekening.HasilKasir{}, t.Galat
	}
	return t.Jawaban, nil
}

var _ masterrekening.Kasir = (*Tiruan)(nil)
