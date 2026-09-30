// Package memory adalah adapter kedua di balik seam Repo dan Storage.
//
// Ia bukan sekadar pelengkap uji: tanpa adapter kedua, seam-nya hipotetis. Dengan adapter
// ini, seluruh aturan unggah dapat diuji tanpa Oracle dan tanpa jaringan — dan itulah yang
// membuat aturan folder, pembersihan nama, dan pemetaan tipe media punya uji yang cepat.
package memory

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/dokumenpenunjang"
)

// Repo menyimpan metadata di memori.
type Repo struct {
	mu        sync.RWMutex
	folder    map[string]string
	dokumen   map[string]dokumenpenunjang.Document
	urutan    []string
	akses     []CatatanAkses
	galatSave error
}

// CatatanAkses merekam pemanggilan CatatAksesUnggah.
//
// Direkam, bukan diabaikan, supaya uji dapat menegaskan bahwa izin akses dicatat SEBELUM
// berkasnya dikirim. Di Pega urutan itu ditegakkan oleh urutan step; di sini tidak ada yang
// menegakkannya kecuali uji.
type CatatanAkses struct {
	Aplikasi   string
	Pengunggah string
}

// NewRepo membentuk repo kosong dengan folder aplikasi yang sudah terdaftar.
func NewRepo() *Repo {
	return &Repo{
		folder:  map[string]string{dokumenpenunjang.NamaAplikasi: "klaimpnc"},
		dokumen: map[string]dokumenpenunjang.Document{},
	}
}

// LupakanFolder menghapus pendaftaran folder, untuk menguji ErrFolderAplikasiTidakAda.
func (r *Repo) LupakanFolder(aplikasi string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.folder, aplikasi)
}

// GagalkanSimpan membuat Simpan selalu gagal, untuk menguji ErrMetadataGagal.
func (r *Repo) GagalkanSimpan(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.galatSave = err
}

// Akses mengembalikan catatan izin yang terekam.
func (r *Repo) Akses() []CatatanAkses {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]CatatanAkses(nil), r.akses...)
}

// NamaFolderAplikasi memenuhi dokumenpenunjang.Repo.
func (r *Repo) NamaFolderAplikasi(_ context.Context, aplikasi string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nama, ada := r.folder[aplikasi]
	if !ada {
		return "", fmt.Errorf("%w: %s", dokumenpenunjang.ErrFolderAplikasiTidakAda, aplikasi)
	}
	return nama, nil
}

// CatatAksesUnggah memenuhi dokumenpenunjang.Repo.
func (r *Repo) CatatAksesUnggah(_ context.Context, aplikasi, pengunggah string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.akses = append(r.akses, CatatanAkses{Aplikasi: aplikasi, Pengunggah: pengunggah})
	return fmt.Sprintf("KODE-%d", len(r.akses)), nil
}

// Simpan memenuhi dokumenpenunjang.Repo.
func (r *Repo) Simpan(_ context.Context, dokumen dokumenpenunjang.Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.galatSave != nil {
		return r.galatSave
	}
	if _, ada := r.dokumen[dokumen.ImageID]; !ada {
		r.urutan = append(r.urutan, dokumen.ImageID)
	}
	r.dokumen[dokumen.ImageID] = dokumen
	return nil
}

// PerKlaim memenuhi dokumenpenunjang.Repo; terbaru lebih dulu, seperti adapter Oracle.
func (r *Repo) PerKlaim(
	_ context.Context,
	nomorKlaim string,
) ([]dokumenpenunjang.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cari := strings.ToUpper(strings.TrimSpace(nomorKlaim))
	var hasil []dokumenpenunjang.Document
	for i := len(r.urutan) - 1; i >= 0; i-- {
		d := r.dokumen[r.urutan[i]]
		if strings.ToUpper(strings.TrimSpace(d.ClaimNumber)) == cari {
			hasil = append(hasil, d)
		}
	}
	return hasil, nil
}

// Ambil memenuhi dokumenpenunjang.Repo.
func (r *Repo) Ambil(_ context.Context, imageID string) (dokumenpenunjang.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ada := r.dokumen[strings.TrimSpace(imageID)]
	if !ada {
		return dokumenpenunjang.Document{}, dokumenpenunjang.ErrTidakDitemukan
	}
	return d, nil
}

// Storage adalah layanan penyimpanan palsu yang MEREKAM apa yang dikirim kepadanya.
//
// Merekam, bukan sekadar menerima: hampir seluruh aturan modul ini — folder, nama bersih,
// tipe media, nomor klaim yang ditambal — hanya terlihat pada apa yang DIKIRIM. Adapter
// palsu yang membuang perintahnya membuat aturan-aturan itu tidak dapat diuji sama sekali.
type Storage struct {
	mu       sync.Mutex
	diterima []dokumenpenunjang.PerintahUnggah
	galat    error
	urutan   int
	masaURL  time.Duration
}

// NewStorage membentuk layanan palsu yang selalu berhasil.
func NewStorage() *Storage { return &Storage{masaURL: 24 * time.Hour} }

// Gagalkan membuat setiap unggahan berikutnya gagal.
func (s *Storage) Gagalkan(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.galat = err
}

// Diterima mengembalikan seluruh perintah yang pernah masuk.
func (s *Storage) Diterima() []dokumenpenunjang.PerintahUnggah {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]dokumenpenunjang.PerintahUnggah(nil), s.diterima...)
}

// Terakhir mengembalikan perintah terakhir; panik bila belum ada satu pun.
func (s *Storage) Terakhir() dokumenpenunjang.PerintahUnggah {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.diterima) == 0 {
		panic("dokumenpenunjang/memory: belum ada unggahan yang masuk")
	}
	return s.diterima[len(s.diterima)-1]
}

// Upload memenuhi dokumenpenunjang.Storage.
//
// `ImageID` dibentuk dari sidik isi berkas ditambah pencacah, meniru sifat yang penting dari
// layanan sungguhan: kuncinya DITERBITKAN layanan, bukan disusun pemanggil. Uji yang
// menebak kuncinya akan lulus terhadap yang palsu dan gagal terhadap yang sungguhan.
func (s *Storage) Upload(
	_ context.Context,
	perintah dokumenpenunjang.PerintahUnggah,
) (dokumenpenunjang.HasilUnggah, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.galat != nil {
		return dokumenpenunjang.HasilUnggah{}, s.galat
	}

	s.diterima = append(s.diterima, perintah)
	s.urutan++

	sidik := sha1.Sum(perintah.Isi)
	id := fmt.Sprintf("%s-%d", hex.EncodeToString(sidik[:6]), s.urutan)

	return dokumenpenunjang.HasilUnggah{
		ImageID: id,
		Folder:  perintah.Folder,
	}, nil
}

// Converter adalah layanan konversi palsu yang MEREKAM apa yang dikonversinya.
//
// Merekam, bukan sekadar mengembalikan: yang perlu diuji bukan hanya hasilnya, melainkan
// **berkas mana saja yang dikirim ke sana**. Empat ekstensi dikonversi dan sisanya tidak,
// dan aturan itu hanya terlihat dari apa yang masuk.
type Converter struct {
	mu       sync.Mutex
	diterima [][]byte
	galat    error
	kosong   bool
}

// NewConverter membentuk layanan konversi palsu yang selalu berhasil.
func NewConverter() *Converter { return &Converter{} }

// Gagalkan membuat setiap konversi berikutnya gagal.
func (c *Converter) Gagalkan(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.galat = err
}

// KembalikanKosong meniru layanan yang menjawab berhasil tanpa memberi isi.
func (c *Converter) KembalikanKosong() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kosong = true
}

// Diterima mengembalikan seluruh isi yang pernah dikonversi.
func (c *Converter) Diterima() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([][]byte(nil), c.diterima...)
}

// Convert memenuhi dokumenpenunjang.Converter.
//
// Hasilnya sengaja BERBEDA dari masukannya — diberi awalan penanda. Layanan palsu yang
// mengembalikan isi apa adanya membuat uji "isi yang terunggah adalah HASIL konversi"
// lulus meski konversinya dilewati sama sekali.
func (c *Converter) Convert(_ context.Context, isi []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.galat != nil {
		return nil, c.galat
	}
	c.diterima = append(c.diterima, append([]byte(nil), isi...))
	if c.kosong {
		return nil, nil
	}
	return append([]byte("AVIF:"), isi...), nil
}
