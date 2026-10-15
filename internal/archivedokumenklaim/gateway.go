package archivedokumenklaim

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// Shipment adalah satu berkas arsip yang hendak dikirim ke layanan Arsip.
//
// # Bentuknya ditentukan sistem penerima, bukan oleh kita
//
// `Activity/SendDataArchiveDOcumentByService-Act.xml` menyusun badan permintaannya
// sebagai satu JSON berisi dua isian saja:
//
//	NoDokumen   "<ID_ARCHIVE>/<NAMABOX>/<KODEFILLING>"
//	TglRequest  "dd/mm/yyyy"
//
// Ketiga bagian NoDokumen dirangkai dengan garis miring, dan itulah satu-satunya
// pengenal yang diterima sistem Arsip. Bentuk itu ditiru persis — mengubahnya berarti
// mengubah kontrak dengan sistem milik tim lain.
type Shipment struct {
	// ID adalah ID_ARCHIVE, bagian pertama NoDokumen.
	ID int64

	// BoxName adalah NAMABOX, bagian kedua.
	BoxName string

	// FillingCode adalah KODEFILLING, bagian ketiga.
	FillingCode string

	// RequestedAt adalah TglRequest.
	//
	// Sistem lama mengambilnya dari `@CurrentDateTime()` pada langkah pertama
	// `SENDDATACABANGKEARCHIVE`, yaitu waktu server — bukan tanggal yang dipilih
	// pengguna, dan memang tidak ada isian tanggal di layar itu.
	RequestedAt time.Time
}

// DocumentNumber merangkai bagian NoDokumen persis seperti sistem lama.
//
// Ia method, bukan perangkaian di tempat pemakaian, supaya bentuknya punya satu tempat —
// dan supaya satu uji dapat menahannya bila kelak seseorang menyisipkan pemisah lain.
func (s Shipment) DocumentNumber() string {
	return joinWithSlash(s.ID, s.BoxName, s.FillingCode)
}

// Receipt adalah jawaban layanan Arsip atas satu pengiriman, beserta permintaan yang
// menghasilkannya.
//
// Ketiganya tersimpan ke baris arsip lewat `UpdateDataArchiveKlaimSetelahService`:
//
//	KODESERVICE  ← Code
//	NOTESERVICE  ← Note
//	HITARCHIVE   ← Request
//
// Kolom ketiga menyimpan BADAN PERMINTAAN, bukan jawaban. Namanya menyebut "hit", dan
// isinya memang jejak apa yang dikirimkan — satu-satunya bukti yang tersisa bila kelak
// jawabannya dipersoalkan.
type Receipt struct {
	// ID adalah ID_ARCHIVE baris yang dikirim.
	ID int64

	// Code adalah `ResponseCode` dari layanan Arsip.
	Code string

	// Note adalah `ResponseMessage` dari layanan Arsip.
	Note string

	// Request adalah badan permintaan yang dikirimkan, apa adanya.
	Request string

	// SentAt adalah waktu pengiriman, pengisi TGLKIRIMDOK.
	//
	// Sistem lama TIDAK mengisi kolom itu di jalur mana pun yang terbaca di export,
	// padahal grid menampilkannya sebagai "Tanggal Kirim Dok". Mengisinya di sini adalah
	// perbedaan yang disengaja, dan alasannya disebut di docs/keputusan-implementasi.md:
	// kolom yang ditampilkan tetapi tidak pernah diisi hanya memberi tahu pengguna bahwa
	// datanya hilang.
	SentAt time.Time
}

// Gateway adalah seam ke layanan Arsip milik tim lain.
//
// Pengisinya ada di gateway/: satu klien HTTP nyata dan satu perekam untuk pengujian dan
// pengembangan lokal — dua adapter, sehingga seam ini nyata dan bukan hipotetis.
//
// # Kenapa antarmukanya bicara dalam istilah domain
//
// Ia menerima Shipment dan mengembalikan Receipt, bukan `Post(url, body)`. Dengan begitu
// alamat, bentuk JSON, dan penanganan kegagalannya menjadi urusan adapter — dan usecase
// tidak perlu berubah bila sistem Arsip kelak mengganti bentuk permintaannya.
//
// # Yang dituntut dari setiap pengisinya
//
//   - Batas waktu WAJIB ada. Tanpa itu, satu layanan yang menggantung menghabiskan
//     seluruh koneksi kita (`10-API-STRATEGY.md` §8.2).
//   - Ia TIDAK BOLEH dipanggil di dalam transaksi basis data. Kegagalan jaringan tidak
//     boleh menahan kunci baris.
type Gateway interface {
	// Send mengirim satu berkas arsip dan mengembalikan jawabannya.
	//
	// Galat yang dikembalikan dibungkus ErrServiceFailed atau ErrServiceAddress supaya
	// pemanggil dapat membedakan "layanannya bermasalah" dari "alamatnya belum ada" —
	// dua hal yang menuntut perbaikan oleh orang yang berbeda.
	Send(ctx context.Context, portalAlias string, shipment Shipment) (Receipt, error)
}

// joinWithSlash merangkai ketiga bagian NoDokumen.
//
// Bagian yang kosong tetap menyisakan pemisahnya, persis seperti perangkaian teks di
// sistem lama. Menghilangkannya akan mengubah bentuk pengenal yang diterima sistem Arsip
// tepat pada baris yang datanya tidak lengkap — yaitu baris yang paling mungkin
// dipersoalkan kemudian.
func joinWithSlash(id int64, boxName, fillingCode string) string {
	return strings.Join([]string{
		strconv.FormatInt(id, 10),
		strings.TrimSpace(boxName),
		strings.TrimSpace(fillingCode),
	}, "/")
}
