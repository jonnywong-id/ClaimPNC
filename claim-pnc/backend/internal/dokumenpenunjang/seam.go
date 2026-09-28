package dokumenpenunjang

import (
	"context"
	"time"
)

// Dua seam, dan keduanya NYATA — masing-masing punya sekurangnya dua adapter.
//
//	Storage   (1) layanan penyimpanan internal lewat HTTP   (2) palsu untuk pengujian
//	Repo      (1) Oracle lewat DB link                      (2) memori untuk pengujian
//
// Keduanya dipisahkan meski satu unggahan selalu menempuh keduanya, sebab **kegagalannya
// berbeda sifat**: layanan penyimpanan gagal karena jaringan dan dapat dicoba ulang;
// pencatatan metadata gagal karena data dan tidak boleh dicoba ulang begitu saja. Menyatukan
// keduanya di balik satu interface memaksa keduanya diperlakukan sama.

// Converter adalah seam ke layanan konversi gambar.
//
// # Kenapa ia seam TERSENDIRI, bukan bagian dari Storage
//
// Keduanya layanan HTTP tanpa otentikasi, jadi menggabungkannya tampak hemat. Yang menahan:
// **keduanya layanan yang BERBEDA, di alamat yang berbeda, dan kegagalannya berakibat
// berbeda.** Konversi yang gagal membatalkan unggahan sebelum apa pun terkirim; penyimpanan
// yang gagal terjadi sesudahnya.
//
// Menyatukannya juga akan membuat satu alamat mewakili dua layanan — dan memindahkan salah
// satunya menjadi mustahil tanpa memindahkan keduanya.
type Converter interface {
	// Convert mengembalikan isi berkas yang sudah dikonversi.
	Convert(ctx context.Context, isi []byte) ([]byte, error)
}

// Storage adalah seam ke layanan penyimpanan internal.
//
// Ia bicara dalam istilah domain — berkas sebuah klaim — bukan dalam istilah HTTP. Pemanggil
// tidak pernah tahu bahwa isinya dikirim sebagai base64 di dalam JSON, dan tidak pernah
// menyusun muatan itu sendiri.
type Storage interface {
	// Upload mengirim satu berkas, dan mengembalikan kunci yang DITERBITKAN layanan.
	Upload(ctx context.Context, perintah PerintahUnggah) (HasilUnggah, error)
}

// PerintahUnggah adalah bentuk yang diterima layanan penyimpanan, sesudah seluruh aturan
// domain diterapkan.
//
// Ia sengaja BUKAN UploadRequest: yang ini sudah bersih dan sudah lengkap — nama sudah
// dibersihkan, tipe media sudah dipetakan, folder sudah disusun, nomor klaim sudah ditambal.
// Memakai satu tipe untuk keduanya membuat adapter bebas menerapkan atau melewatkan aturan
// itu, dan tidak ada yang menahannya.
type PerintahUnggah struct {
	// NamaAplikasi adalah nama folder penyimpanan hasil pencarian master — BUKAN konstanta
	// `KLAIMPNC`. Lihat komentar NamaAplikasi.
	NamaAplikasi string

	Pengunggah  string
	NomorKlaim  string
	Folder      string
	NamaBerkas  string
	TipeMedia   string
	Isi         []byte
}

// HasilUnggah adalah yang dikembalikan layanan penyimpanan.
//
// `Durasi` tidak ada di sini: Pega selalu mengirimnya `0`
// (`Activity/InsertDokumenPNC-Act.xml:3786`) dan tidak pernah membaca apa pun darinya.
// Membawanya sebagai field akan mengundang seseorang menyetelnya.
type HasilUnggah struct {
	// ImageID adalah kunci berkas di layanan penyimpanan.
	ImageID string

	// URL dan ExpiresAt boleh kosong: Pega membacanya kembali dari tabel metadata
	// (`GetURLAndEXPDate`), bukan mengandalkan responsnya saja.
	URL       string
	ExpiresAt *time.Time

	// Folder adalah folder yang benar-benar dipakai layanan, bila ia melaporkannya.
	Folder string
}

// Repo adalah seam ke penyimpan metadata.
type Repo interface {
	// NamaFolderAplikasi mencari nama folder penyimpanan untuk sebuah aplikasi.
	//
	// Pega: `select NAMA_FOLDER AS CARI1 FROM general.T_FOLDER_STORAGE WHERE APLIKASI = ?`
	NamaFolderAplikasi(ctx context.Context, aplikasi string) (string, error)

	// CatatAksesUnggah merekam satu izin unggah sebelum berkasnya dikirim.
	//
	// Pega menempuhnya lewat `GENERAL.GET_TOKEN_STORAGE`, yang membuat token lalu
	// MENYISIPKAN barisnya ke `GENERAL.GCP_IMAGE`. Lihat catatan di adapter — tokennya
	// tidak pernah ikut terkirim.
	CatatAksesUnggah(ctx context.Context, aplikasi, pengunggah string) error

	// Simpan mencatat metadata satu dokumen yang sudah terunggah.
	Simpan(ctx context.Context, dokumen Document) error

	// PerKlaim mengembalikan dokumen sebuah klaim, terbaru lebih dulu.
	PerKlaim(ctx context.Context, nomorKlaim string) ([]Document, error)

	// Ambil mengembalikan satu dokumen menurut ImageID-nya.
	Ambil(ctx context.Context, imageID string) (Document, error)
}

// RepoSelector memilih repo menurut portal.
//
// `D-75` menetapkan satu basis data per entitas, sehingga dokumen sebuah klaim berada di
// basis data portalnya. Pemilihannya karena itu terjadi per permintaan, bukan sekali saat
// aplikasi start.
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam waktu (`F-5`).
//
// Waktu unggah menentukan FOLDER tujuan, sehingga ia bukan sekadar cap waktu: mengujinya
// menuntut jam yang dapat dipatok. Tanpa seam ini, uji folder hanya dapat menegaskan
// bulan yang sedang berjalan — dan lulus sepanjang bulan itu saja.
type Clock interface {
	Now() time.Time
}
