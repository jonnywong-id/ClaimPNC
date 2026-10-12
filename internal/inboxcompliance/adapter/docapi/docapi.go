// Package docapi menyambungkan modul Inbox Compliance ke layanan penyimpanan dokumen.
//
// # Ia JEMBATAN, bukan klien
//
// Klien HTTP-nya sudah ada: `internal/dokumenpenunjang/storage/httpstorage`, yang memenuhi
// ketiga Connect REST yang sama —
//
//	UploadDokumenPNC    POST /api/v1/upload
//	NewLinkDokumenPNC   POST /api/v1/geturl
//	DeleteDokumenPNC    POST /api/v1/delete
//
// — dan sudah lebih lengkap daripada apa pun yang dapat ditulis ulang di sini: ia membawa
// `KodeString` (kode akses folder), membaca `appfolder` dari jawaban, menguraikan masa
// berlaku, dan punya uji tersendiri termasuk satu yang menjaga `Durasi` terkirim sebagai
// ANGKA, bukan teks.
//
// # Kenapa berkas ini pernah berisi klien kedua
//
// Karena saya menulisnya tanpa mencari lebih dulu. Klien kedua itu dibuang pada
// 2026-10-07; yang tersisa adalah pemetaan tipe. Dicatat di sini, bukan dihapus diam-diam,
// supaya alasannya terbaca bila ada yang tergoda menulis klien ketiga.
//
// # Harga yang dibayar
//
// Modul ini mengimpor tipe modul lain (`dokumenpenunjang`), yang selama ini dihindari
// antarmodul fitur. Pertukarannya disengaja: satu pengetahuan kontrak di satu tempat,
// dibanding dua klien yang dapat menyimpang diam-diam — dan penyimpangan itu baru terlihat
// ketika berkas nasabah sudah terkirim dengan muatan yang berbeda.
//
// Pemetaannya sempit dan satu arah, sehingga ketergantungan itu tidak menjalar: seluruh
// modul ini tetap bicara dalam tipe `inboxcompliance`.
package docapi

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/inboxcompliance"
)

// Penyimpan memenuhi `inboxcompliance.DocumentLinker` dan `DocumentStore` di atas klien
// `httpstorage`.
type Penyimpan struct {
	klien   Klien
	resolve func(ctx context.Context, pengunggah string) (Kredensial, error)

	// app dan folder mengisi `App` dan `Folder` pada muatan. Keduanya tetap sepanjang
	// umur satu portal, sehingga ia konfigurasi — bukan isian tiap panggilan.
	app    string
	folder string

	// accessCode adalah `KodeString`. Kosong berarti layanan membuat token sekali pakai
	// sendiri; diisi bila layanan menuntut kode yang didaftarkan pemilik folder.
	accessCode string
}

// Klien adalah bagian `httpstorage.Client` yang dipakai modul ini.
//
// Dideklarasikan DI SINI, bukan diimpor sebagai tipe konkret, supaya uji dapat
// memalsukannya tanpa server tiruan — dan supaya terbaca persis seberapa banyak dari
// klien itu yang benar-benar dipakai.
type Klien interface {
	Upload(context.Context, dokumenpenunjang.PerintahUnggah) (dokumenpenunjang.HasilUnggah, error)
	PerpanjangTautan(context.Context, dokumenpenunjang.PerintahTautan) (dokumenpenunjang.HasilUnggah, error)
	Hapus(context.Context, dokumenpenunjang.PerintahHapus) (string, error)
}

// Kredensial adalah nama folder aplikasi beserta kode aksesnya untuk SATU permintaan.
//
// Keduanya disatukan karena di Pega memang berasal dari satu rangkaian: nama foldernya
// dicari lebih dulu, lalu dipakai sebagai masukan penerbitan kode aksesnya.
type Kredensial struct {
	// NamaAplikasi mengisi `DocAPI.App`.
	//
	// Bukan konstanta `"KLAIMPNC"` melainkan NAMA FOLDER-nya, hasil
	// `select NAMA_FOLDER from general.T_FOLDER_STORAGE where APLIKASI = ?`.
	NamaAplikasi string

	// KodeAkses mengisi `DocAPI.KodeString`.
	KodeAkses string
}

// Config adalah isian pembentuk Penyimpan.
type Config struct {
	// Klien WAJIB — biasanya `*httpstorage.Client`.
	Klien Klien

	// ResolveKredensial dipanggil SETIAP permintaan.
	//
	// # Kenapa seam, bukan nilai tetap
	//
	// Versi pertama Config menerima `App` dan `AccessCode` sebagai nilai tetap. Itu
	// keliru, dan kekeliruannya tidak akan ditolak siapa pun: nama folder aplikasi hidup
	// di `general.T_FOLDER_STORAGE` dan kode aksesnya diterbitkan
	// `GENERAL.GET_TOKEN_STORAGE` — keduanya pembacaan basis data, per portal. Layanan
	// penyimpanan di seberang tidak memeriksa kepemilikan apa pun
	// (`pyUseAuthentication=false`), sehingga nilai yang salah tidak menghasilkan galat;
	// berkasnya hanya mendarat di tempat yang salah.
	//
	// `pengunggah` diteruskan karena penerbitan kode akses mencatat atas nama siapa ia
	// dibuat. Pada penghapusan ia kosong — tidak ada yang mengunggah apa pun di sana.
	//
	// BOLEH nil: bila kosong, dipakai App dan AccessCode di bawah. Itu yang membuat uji
	// paket ini tidak perlu memalsukan basis data.
	ResolveKredensial func(ctx context.Context, pengunggah string) (Kredensial, error)

	// App dan AccessCode adalah nilai tetap, dipakai HANYA ketika ResolveKredensial nil.
	App        string
	AccessCode string

	// Folder mengisi `DocAPI.Folder` pada penerbitan tautan dan penghapusan. Pada
	// unggah, foldernya dibentuk dari tanggal — lihat Upload.
	Folder string
}

// New membentuk Penyimpan.
func New(config Config) (*Penyimpan, error) {
	if config.Klien == nil {
		return nil, fmt.Errorf(
			"docapi: klien layanan penyimpanan wajib diisi")
	}
	return &Penyimpan{
		klien:      config.Klien,
		resolve:    config.ResolveKredensial,
		app:        strings.TrimSpace(config.App),
		folder:     strings.TrimSpace(config.Folder),
		accessCode: config.AccessCode,
	}, nil
}

// kredensial menyerahkan nama folder aplikasi dan kode akses untuk satu permintaan.
func (p *Penyimpan) kredensial(ctx context.Context, pengunggah string) (Kredensial, error) {
	if p.resolve == nil {
		return Kredensial{NamaAplikasi: p.app, KodeAkses: p.accessCode}, nil
	}

	k, err := p.resolve(ctx, pengunggah)
	if err != nil {
		return Kredensial{}, fmt.Errorf("docapi: menyiapkan kredensial penyimpanan: %w", err)
	}
	if strings.TrimSpace(k.NamaAplikasi) == "" {
		// Nama folder kosong akan membuat berkas mendarat di tempat yang salah TANPA
		// galat dari layanan. Lebih baik gagal di sini, di mana sebabnya masih terbaca.
		return Kredensial{}, fmt.Errorf(
			"docapi: nama folder aplikasi kosong — periksa baris %q pada general.T_FOLDER_STORAGE",
			dokumenpenunjang.NamaAplikasi)
	}
	return k, nil
}

// NewLink menerbitkan tautan baru untuk satu berkas.
func (p *Penyimpan) NewLink(
	ctx context.Context, request inboxcompliance.LinkRequest,
) (inboxcompliance.DocumentLink, error) {
	durasi := request.DurationSeconds
	if durasi <= 0 {
		durasi = inboxcompliance.DefaultLinkDurationSeconds
	}

	kred, err := p.kredensial(ctx, request.UserInput)
	if err != nil {
		return inboxcompliance.DocumentLink{}, err
	}

	hasil, err := p.klien.PerpanjangTautan(ctx, dokumenpenunjang.PerintahTautan{
		NamaAplikasi: kred.NamaAplikasi,
		Pengunggah:   request.UserInput,
		KodeAkses:    kred.KodeAkses,
		ImageID:      request.StorageID,
		Folder:       p.pilihFolder(request.Folder),
		NamaBerkas:   request.FileName,
		Durasi:       durasi,
	})
	if err != nil {
		return inboxcompliance.DocumentLink{}, err
	}

	return inboxcompliance.DocumentLink{
		URL:       hasil.URL,
		ExpiresAt: hasil.ExpiresAt,
	}, nil
}

// Upload mengunggah satu berkas.
//
// Nama berkas DIBERSIHKAN dan tipe MIME dipetakan di sini, bukan di klien: keduanya
// perilaku `Activity/InsertDokumenPNC-Act.xml` langkah 2 dan 16, yakni aturan jalur
// Compliance — bukan aturan layanan penyimpanan.
func (p *Penyimpan) Upload(
	ctx context.Context, request inboxcompliance.UploadRequest,
) (inboxcompliance.UploadResult, error) {
	if len(request.Content) == 0 {
		return inboxcompliance.UploadResult{},
			fmt.Errorf("docapi: berkas yang diunggah kosong")
	}

	noKlaim := strings.TrimSpace(request.ClaimNumber)
	if noKlaim == "" {
		// Peniruan langkah 16: `@If(DocAPI.NoClaim=="", "-", DocAPI.NoClaim)`.
		noKlaim = "-"
	}

	kred, err := p.kredensial(ctx, request.UserInput)
	if err != nil {
		return inboxcompliance.UploadResult{}, err
	}

	hasil, err := p.klien.Upload(ctx, dokumenpenunjang.PerintahUnggah{
		NamaAplikasi: kred.NamaAplikasi,
		Pengunggah:   request.UserInput,
		NomorKlaim:   noKlaim,
		KodeAkses:    kred.KodeAkses,
		Folder:       folderUnggah(waktuSekarang()),
		NamaBerkas:   bukanAlfanumerik.ReplaceAllString(request.FileName, ""),
		TipeMedia:    mimeDari(request.MimeTypeHint),
		Isi:          request.Content,
	})
	if err != nil {
		return inboxcompliance.UploadResult{}, err
	}

	if strings.TrimSpace(hasil.ImageID) == "" {
		// Jawaban berhasil tanpa ImageID adalah kegagalan. Menyimpan baris tanpa kunci
		// penyimpanan menghasilkan dokumen yang tampil di grid tetapi tidak pernah dapat
		// dibuka — persis baris yang kueri daftar justru menyaringnya keluar.
		return inboxcompliance.UploadResult{},
			fmt.Errorf("layanan dokumen menjawab tanpa ImageID")
	}

	return inboxcompliance.UploadResult{StorageID: hasil.ImageID}, nil
}

// Delete menghapus satu berkas.
func (p *Penyimpan) Delete(
	ctx context.Context, request inboxcompliance.DeleteRequest,
) error {
	jalur := strings.TrimSpace(request.ObjectPath)
	if jalur == "" {
		return fmt.Errorf("docapi: jalur objek yang dihapus kosong")
	}

	kred, err := p.kredensial(ctx, "")
	if err != nil {
		return err
	}

	_, err = p.klien.Hapus(ctx, dokumenpenunjang.PerintahHapus{
		NamaAplikasi: kred.NamaAplikasi,
		KodeAkses:    kred.KodeAkses,
		Jalur:        pangkasBucket(jalur),
	})
	return err
}

// pilihFolder memakai folder permintaan bila ada, selainnya folder konfigurasi.
func (p *Penyimpan) pilihFolder(dariPermintaan string) string {
	if strings.TrimSpace(dariPermintaan) != "" {
		return dariPermintaan
	}
	return p.folder
}
