// Package memori adalah pengisi seam masterstatus.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master
// Status Klaim dapat dicoba lengkap dengan 33 baris yang benar sebelum DBA menjalankan
// migrasi 0002.
package memori

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterstatus"
)

// Repo menyimpan master status di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu     sync.RWMutex
	baris  map[string]masterstatus.StatusKlaim
	urutan int
	situs  string
	galat  error
}

// RepoBaru membentuk repo berisi daftar yang diberikan.
func RepoBaru(daftar ...masterstatus.StatusKlaim) *Repo {
	r := &Repo{
		baris: make(map[string]masterstatus.StatusKlaim, len(daftar)),
		// Nilai berikut meniru pembentukan kode di sistem lama: id_site = "1" dan
		// urutan tiga digit. Urutan dimulai dari kode tertinggi yang sudah ada supaya
		// penambahan pertama menghasilkan kode berikutnya yang wajar, bukan bentrok.
		situs: "1",
	}
	for _, s := range daftar {
		bersih := s.Bersih()
		r.baris[bersih.Kode] = bersih
		if n := urutanDariKode(bersih.Kode, r.situs); n > r.urutan {
			r.urutan = n
		}
	}
	return r
}

// SetGalat membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetGalat(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.galat = err
}

// Daftar mengembalikan seluruh status, terurut menurut kode.
func (r *Repo) Daftar(_ context.Context) ([]masterstatus.StatusKlaim, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.galat != nil {
		return nil, r.galat
	}

	hasil := make([]masterstatus.StatusKlaim, 0, len(r.baris))
	for _, s := range r.baris {
		hasil = append(hasil, s)
	}
	sort.Slice(hasil, func(i, j int) bool { return hasil[i].Kode < hasil[j].Kode })
	return hasil, nil
}

// Ambil mengembalikan satu status.
func (r *Repo) Ambil(_ context.Context, kode string) (masterstatus.StatusKlaim, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.galat != nil {
		return masterstatus.StatusKlaim{}, r.galat
	}

	s, ada := r.baris[strings.TrimSpace(kode)]
	if !ada {
		return masterstatus.StatusKlaim{}, masterstatus.ErrTidakDitemukan
	}
	return s, nil
}

// Sisip menyimpan status baru dengan kode yang dibentuk seperti sistem lama.
func (r *Repo) Sisip(_ context.Context, label string) (masterstatus.StatusKlaim, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.galat != nil {
		return masterstatus.StatusKlaim{}, r.galat
	}
	if err := r.pastikanLabelUnik(label, ""); err != nil {
		return masterstatus.StatusKlaim{}, err
	}

	r.urutan++
	kode := r.situs + tigaDigit(r.urutan)
	if _, bentrok := r.baris[kode]; bentrok {
		return masterstatus.StatusKlaim{}, masterstatus.ErrKodeSudahAda
	}

	// KodeLama sengaja dibiarkan kosong: penomoran `01`–`11` hanya melekat pada sebelas
	// kode pertama dan tidak pernah diberikan pada status baru.
	status := masterstatus.StatusKlaim{Kode: kode, Label: strings.TrimSpace(label)}
	r.baris[kode] = status
	return status, nil
}

// Perbarui mengganti label status yang sudah ada.
func (r *Repo) Perbarui(_ context.Context, kode, label string) (masterstatus.StatusKlaim, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.galat != nil {
		return masterstatus.StatusKlaim{}, r.galat
	}

	kode = strings.TrimSpace(kode)
	lama, ada := r.baris[kode]
	if !ada {
		return masterstatus.StatusKlaim{}, masterstatus.ErrTidakDitemukan
	}
	if err := r.pastikanLabelUnik(label, kode); err != nil {
		return masterstatus.StatusKlaim{}, err
	}

	// Hanya label yang berubah; kode dan kode lama dipertahankan apa adanya.
	lama.Label = strings.TrimSpace(label)
	r.baris[kode] = lama
	return lama, nil
}

// pastikanLabelUnik meniru indeks unik basis data, supaya jalur gagal yang sama dapat
// diuji tanpa Oracle. Pemanggil sudah memegang kunci.
func (r *Repo) pastikanLabelUnik(label, kecualiKode string) error {
	dicari := masterstatus.KunciLabel(label)
	for kode, s := range r.baris {
		if kode == kecualiKode {
			continue
		}
		if masterstatus.KunciLabel(s.Label) == dicari {
			return masterstatus.ErrLabelSudahAda
		}
	}
	return nil
}

// tigaDigit meniru `lpad(to_char(seq), 3, '0')` di PEGA_M_STS_CLAIM.prc:19.
//
// Bilangan di atas 999 dikembalikan APA ADANYA, tanpa dipotong — sama seperti LPAD
// Oracle. Akibatnya kode ke-1000 menjadi lima karakter, dan itu memang cacat skema
// warisan yang sengaja tidak ditutupi di sini: menutupinya akan menghasilkan kode ganda.
func tigaDigit(n int) string {
	s := ""
	for sisa := n; sisa > 0; sisa /= 10 {
		s = string(rune('0'+sisa%10)) + s
	}
	if s == "" {
		s = "0"
	}
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

// urutanDariKode membaca kembali nomor urut dari sebuah kode, supaya penambahan
// berikutnya melanjutkan dan tidak mengulang nomor yang sudah dipakai.
func urutanDariKode(kode, situs string) int {
	sisa := strings.TrimPrefix(kode, situs)
	if sisa == kode || sisa == "" {
		return 0
	}
	n := 0
	for _, r := range sisa {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

var _ masterstatus.Repo = (*Repo)(nil)
