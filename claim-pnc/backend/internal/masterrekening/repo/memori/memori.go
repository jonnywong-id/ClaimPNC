// Package memori adalah pengisi seam penyimpanan Master Rekening yang hidup di dalam
// memori.
//
// Ia ada supaya alur pengajuan dan keputusan komite dapat diuji tanpa basis data dan
// tanpa jaringan sama sekali — adapter kedua inilah yang membuat seam penyimpanan
// menjadi seam nyata, bukan seam hipotetis
// (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.1).
//
// Adapter ini TIDAK dipakai di produksi: master rekening yang hilang saat proses
// dijalankan ulang tidak ada gunanya.
package memori

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterrekening"
)

// Repo menyimpan master rekening di memori, dikunci NomorRekening+KodeBank.
type Repo struct {
	mu  sync.RWMutex
	isi map[masterrekening.Kunci]masterrekening.Rekening
}

// RepoBaru membentuk store berisi baris awal yang diberikan.
func RepoBaru(awal ...masterrekening.Rekening) *Repo {
	s := &Repo{isi: map[masterrekening.Kunci]masterrekening.Rekening{}}
	for _, r := range awal {
		s.isi[r.KunciDari()] = r
	}
	return s
}

// Daftar membaca rekening yang cocok dengan filter.
func (s *Repo) Daftar(_ context.Context, f masterrekening.Filter) ([]masterrekening.Rekening, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cocok := make([]masterrekening.Rekening, 0, len(s.isi))
	for _, r := range s.isi {
		if !lolosFilter(r, f) {
			continue
		}
		cocok = append(cocok, r)
	}

	// Urutan tetap: baris terbaru lebih dulu, lalu nomor rekening sebagai pemutus
	// seri. Tanpa pemutus seri, dua baris berwaktu sama dapat bertukar tempat di
	// antara dua permintaan dan membuat paginasi melewatkan baris.
	sort.Slice(cocok, func(i, j int) bool {
		if !cocok[i].DiinputPada.Equal(cocok[j].DiinputPada) {
			return cocok[i].DiinputPada.After(cocok[j].DiinputPada)
		}
		return cocok[i].NomorRekening < cocok[j].NomorRekening
	})

	jumlah := len(cocok)
	return potong(cocok, f), jumlah, nil
}

func potong(baris []masterrekening.Rekening, f masterrekening.Filter) []masterrekening.Rekening {
	mulai := f.Lewati
	if mulai < 0 {
		mulai = 0
	}
	if mulai >= len(baris) {
		return []masterrekening.Rekening{}
	}
	sisa := baris[mulai:]
	if f.Batas > 0 && f.Batas < len(sisa) {
		sisa = sisa[:f.Batas]
	}
	return append([]masterrekening.Rekening(nil), sisa...)
}

func lolosFilter(r masterrekening.Rekening, f masterrekening.Filter) bool {
	if f.Status != "" && r.Status != f.Status {
		return false
	}
	if !memuat(r.NomorRekening, f.NomorRekening) {
		return false
	}
	if !memuat(r.NamaPemilik, f.NamaPemilik) {
		return false
	}
	if !memuat(r.NamaBank, f.NamaBank) {
		return false
	}
	if f.HanyaKomiteSaya && f.IdentitasKomite != "" {
		if !samaTanpaHuruf(r.KomiteApproval, f.IdentitasKomite) {
			return false
		}
	}
	return true
}

func memuat(nilai, cari string) bool {
	cari = strings.TrimSpace(cari)
	if cari == "" {
		return true
	}
	return strings.Contains(strings.ToUpper(nilai), strings.ToUpper(cari))
}

func samaTanpaHuruf(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// Ambil membaca satu rekening.
func (s *Repo) Ambil(_ context.Context, k masterrekening.Kunci) (masterrekening.Rekening, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ada := s.isi[k]
	if !ada {
		return masterrekening.Rekening{}, masterrekening.ErrTidakDitemukan
	}
	return r, nil
}

// CariNomor membaca seluruh baris dengan nomor rekening tertentu, tanpa peduli banknya.
func (s *Repo) CariNomor(_ context.Context, nomor string) ([]masterrekening.Rekening, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nomor = strings.TrimSpace(nomor)
	var hasil []masterrekening.Rekening
	for _, r := range s.isi {
		if r.NomorRekening == nomor {
			hasil = append(hasil, r)
		}
	}
	sort.Slice(hasil, func(i, j int) bool { return hasil[i].KodeBank < hasil[j].KodeBank })
	return hasil, nil
}

// Simpan menyisipkan rekening baru.
func (s *Repo) Simpan(_ context.Context, r masterrekening.Rekening) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ada := s.isi[r.KunciDari()]; ada {
		return masterrekening.ErrSudahAda
	}
	s.isi[r.KunciDari()] = r
	return nil
}

// Perbarui menulis ulang rekening yang sudah ada.
func (s *Repo) Perbarui(_ context.Context, r masterrekening.Rekening) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ada := s.isi[r.KunciDari()]; !ada {
		return masterrekening.ErrTidakDitemukan
	}
	s.isi[r.KunciDari()] = r
	return nil
}

// HapusYangDitolak membuang baris bekas penolakan komite.
//
// Ia menolak menghapus baris yang statusnya bukan StatusDitolak. Syarat itu ditegakkan
// di sini, bukan dipercayakan kepada pemanggil: sebuah method bernama "hapus" yang mau
// menghapus apa saja cepat atau lambat akan dipanggil untuk apa saja.
func (s *Repo) HapusYangDitolak(_ context.Context, k masterrekening.Kunci) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ada := s.isi[k]
	if !ada {
		return masterrekening.ErrTidakDitemukan
	}
	if r.Status != masterrekening.StatusDitolak {
		return masterrekening.ErrSudahDiputuskan
	}
	delete(s.isi, k)
	return nil
}

// BankRepo menyimpan daftar bank di memori.
type BankRepo struct {
	daftar []masterrekening.Bank
}

// BankRepoBaru membentuk daftar bank.
func BankRepoBaru(daftar ...masterrekening.Bank) *BankRepo {
	return &BankRepo{daftar: daftar}
}

// Daftar mengembalikan seluruh bank.
func (s *BankRepo) Daftar(context.Context) ([]masterrekening.Bank, error) {
	return append([]masterrekening.Bank(nil), s.daftar...), nil
}

// DaftarBankContoh adalah beberapa bank untuk menjalankan aplikasi tanpa basis data.
//
// Kodenya adalah LBG_ID yang sesungguhnya dipakai GENERAL.LST_BANK_GROUP; namanya
// adalah nama bank umum di Indonesia — bukan data nasabah, bukan karangan yang
// menyesatkan.
func DaftarBankContoh() []masterrekening.Bank {
	return []masterrekening.Bank{
		{Kode: "002", Nama: "BANK BRI"},
		{Kode: "008", Nama: "BANK MANDIRI"},
		{Kode: "009", Nama: "BANK BNI"},
		{Kode: "014", Nama: "BANK BCA"},
		{Kode: "011", Nama: "BANK DANAMON"},
		{Kode: "013", Nama: "BANK PERMATA"},
		{Kode: "022", Nama: "BANK CIMB NIAGA"},
		{Kode: "016", Nama: "BANK MAYBANK INDONESIA"},
		{Kode: "153", Nama: "BANK SINARMAS"},
		{Kode: "451", Nama: "BANK SYARIAH INDONESIA"},
	}
}

var (
	_ masterrekening.Repo     = (*Repo)(nil)
	_ masterrekening.BankRepo = (*BankRepo)(nil)
)
