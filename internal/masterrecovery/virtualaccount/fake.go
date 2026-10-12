package virtualaccount

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"claim-pnc/internal/masterrecovery"
)

// Fake menerbitkan nomor rekening virtual tanpa menyentuh jaringan.
//
// # Kenapa ia ada, dan kenapa ia BUKAN dummy logic
//
// Aturan proyek melarang mengarang logika bila proses bisnis aslinya dapat dipelajari.
// Yang dikarang di sini bukan prosesnya — urutan penerbitan, pemeriksaan principal yang
// sudah punya VA, dan pencatatannya ke master seluruhnya ada di usecase dan sqlstore, dan
// seluruhnya ditiru dari sistem lama. Yang digantikan hanya SATU langkah: panggilan
// jaringan ke layanan milik pihak lain.
//
// Menggantikannya diperlukan, bukan sekadar nyaman:
//
//   - Alamat layanan yang terdaftar menunjuk **Pega dev**, dan menembaknya MENERBITKAN
//     REKENING SUNGGUHAN. Menjalankan pengujian terhadapnya berarti meninggalkan sampah
//     di sistem yang dipakai orang lain.
//   - Tanpa adapter kedua, seam VirtualAccountIssuer hanya hipotetis
//     (`04-FUTURE-ARCHITECTURE.md` §3).
//   - Pengembangan tanpa jaringan ke Pega menjadi mustahil.
//
// Ia MENOLAK berjalan di produksi — dipasang di cmd, sejalan dengan cara adapter
// identitas fake diperlakukan.
type Fake struct {
	mu     sync.Mutex
	issued map[string]string

	// failWith membuat penerbitan menjawab galat, untuk menguji jalur gagal tanpa
	// mematikan jaringan.
	failWith error
}

// NewFake membentuk penerbit tiruan.
func NewFake() *Fake { return &Fake{issued: map[string]string{}} }

// SetError membuat penerbitan berikutnya gagal dengan galat yang diberikan.
func (f *Fake) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failWith = err
}

// Prefix adalah awalan nomor yang diterbitkan penerbit tiruan.
//
// Ia sengaja BUKAN awalan bank mana pun, dan sengaja terbaca manusia. Nomor uji yang
// menyerupai nomor sungguhan adalah nomor yang cepat atau lambat akan dikira sungguhan —
// dan yang dipertaruhkan di sini adalah ke mana uang dikirim.
const Prefix = "9999"

// Issue menerbitkan nomor tiruan yang TETAP untuk principal yang sama.
//
// Ketetapan itu disengaja: penerbitan yang menghasilkan nomor berbeda setiap dipanggil
// membuat perilaku "principal ini sudah punya VA" tidak dapat dicoba sama sekali tanpa
// basis data.
func (f *Fake) Issue(
	_ context.Context,
	portalAlias string,
	subject masterrecovery.VirtualAccountRequest,
) (masterrecovery.VirtualAccount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return masterrecovery.VirtualAccount{}, f.failWith
	}

	key := strings.ToUpper(strings.TrimSpace(portalAlias)) + "\x00" +
		masterrecovery.PrincipalKey(subject.ClientID, subject.PrincipalName)

	number, already := f.issued[key]
	if !already {
		number = Prefix + fmt.Sprintf("%012d", digitsFrom(key))
		f.issued[key] = number
	}

	return masterrecovery.VirtualAccount{
		Number: number,
		Status: "OK",
		// Pesannya menyebut dirinya tiruan. Tanpa itu, tangkapan layar dari lingkungan
		// pengembangan tidak dapat dibedakan dari tangkapan layar produksi.
		Message: "Nomor virtual account tiruan — penerbit sungguhan tidak dipakai di lingkungan ini.",
		Reused:  false,
	}, nil
}

// digitsFrom membentuk dua belas angka yang tetap untuk satu kunci.
//
// Sidik ringkasnya dipakai hanya untuk membuat nomor yang BERBEDA antarprincipal dan TETAP
// bagi principal yang sama; ia tidak dipakai sebagai pengaman apa pun.
func digitsFrom(key string) uint64 {
	sum := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint64(sum[:8]) % 1_000_000_000_000
}

var _ masterrecovery.VirtualAccountIssuer = (*Fake)(nil)
