package docapi_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/adapter/docapi"
)

// klienPalsu merekam perintah yang diterima klien penyimpanan.
//
// Uji di sini memeriksa PEMETAANNYA, bukan bentuk HTTP-nya. Bentuk HTTP sudah dijaga
// `internal/dokumenpenunjang/storage/httpstorage` beserta ujinya sendiri — termasuk satu
// yang menjaga `Durasi` terkirim sebagai angka. Mengulanginya di sini akan menduplikasi
// pengetahuan yang sama di dua tempat, dan itu justru yang baru saja dibersihkan.
type klienPalsu struct {
	unggah     []dokumenpenunjang.PerintahUnggah
	tautan     []dokumenpenunjang.PerintahTautan
	hapus      []dokumenpenunjang.PerintahHapus
	hasil      dokumenpenunjang.HasilUnggah
	galat      error
	galatHapus error
}

func (k *klienPalsu) Upload(
	_ context.Context, p dokumenpenunjang.PerintahUnggah,
) (dokumenpenunjang.HasilUnggah, error) {
	k.unggah = append(k.unggah, p)
	return k.hasil, k.galat
}

func (k *klienPalsu) PerpanjangTautan(
	_ context.Context, p dokumenpenunjang.PerintahTautan,
) (dokumenpenunjang.HasilUnggah, error) {
	k.tautan = append(k.tautan, p)
	return k.hasil, k.galat
}

func (k *klienPalsu) Hapus(
	_ context.Context, p dokumenpenunjang.PerintahHapus,
) (string, error) {
	k.hapus = append(k.hapus, p)
	return "", k.galatHapus
}

func penyimpan(t *testing.T, klien docapi.Klien) *docapi.Penyimpan {
	t.Helper()
	p, err := docapi.New(docapi.Config{
		Klien: klien, App: "KLAIMPNC", Folder: "Doc/", AccessCode: "KODE-CONTOH",
	})
	require.NoError(t, err)
	return p
}

// Penerbitan tautan membawa kunci penyimpanan, pelaku, kode akses, dan Durasi bawaan.
func TestNewLinkMemetakanPerintah(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{
		hasil: dokumenpenunjang.HasilUnggah{URL: "https://penyimpanan.contoh/x"},
	}

	tautan, err := penyimpan(t, klien).NewLink(context.Background(),
		inboxcompliance.LinkRequest{
			UserInput: "ADMINCONTOH", StorageID: "IMG-1", FileName: "berkas.pdf",
		})
	require.NoError(t, err)
	require.Equal(t, "https://penyimpanan.contoh/x", tautan.URL)

	require.Len(t, klien.tautan, 1)
	require.Equal(t, "IMG-1", klien.tautan[0].ImageID)
	require.Equal(t, "ADMINCONTOH", klien.tautan[0].Pengunggah)
	require.Equal(t, "KLAIMPNC", klien.tautan[0].NamaAplikasi)
	require.Equal(t, "KODE-CONTOH", klien.tautan[0].KodeAkses)

	// Durasi bawaan 3600 — nilai jatuh-balik Pega ketika master kosong.
	require.Equal(t, 3600, klien.tautan[0].Durasi)
}

// Unggah MEMBERSIHKAN nama berkas dan memetakan tipe MIME.
//
// Keduanya aturan jalur Compliance (`InsertDokumenPNC` langkah 2 dan 16), bukan aturan
// layanan — karena itu ia diuji di sini, bukan di klien.
func TestUploadMembersihkanNamaDanMemetakanMime(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{hasil: dokumenpenunjang.HasilUnggah{ImageID: "IMG-99"}}

	hasil, err := penyimpan(t, klien).Upload(context.Background(),
		inboxcompliance.UploadRequest{
			UserInput: "ADMINCONTOH", ClaimNumber: "PNC-2114",
			FileName: "Surat-Ket 01.pdf", MimeTypeHint: "pdf",
			Content: []byte("halo"),
		})
	require.NoError(t, err)
	require.Equal(t, "IMG-99", hasil.StorageID)

	require.Len(t, klien.unggah, 1)
	// `[^a-zA-Z0-9]` membuang spasi, tanda hubung, DAN titik beserta ekstensinya.
	require.Equal(t, "SuratKet01pdf", klien.unggah[0].NamaBerkas)
	require.Equal(t, "application/pdf", klien.unggah[0].TipeMedia)
	require.Equal(t, []byte("halo"), klien.unggah[0].Isi)
	require.Equal(t, "PNC-2114", klien.unggah[0].NomorKlaim)
	require.Regexp(t, `^Doc/\d{4}/\d{2}/$`, klien.unggah[0].Folder)
}

// Nomor klaim kosong menjadi "-", meniru langkah 16.
func TestUploadTanpaNomorKlaimMengirimStrip(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{hasil: dokumenpenunjang.HasilUnggah{ImageID: "IMG-1"}}

	_, err := penyimpan(t, klien).Upload(context.Background(),
		inboxcompliance.UploadRequest{
			FileName: "a.pdf", MimeTypeHint: "pdf", Content: []byte("x"),
		})
	require.NoError(t, err)
	require.Equal(t, "-", klien.unggah[0].NomorKlaim)
}

// Ekstensi tak dikenal jatuh ke `application/<ekstensi>` — cabang terakhir Pega.
func TestUploadEkstensiTidakDikenal(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{hasil: dokumenpenunjang.HasilUnggah{ImageID: "IMG-1"}}

	_, err := penyimpan(t, klien).Upload(context.Background(),
		inboxcompliance.UploadRequest{
			FileName: "a.xyz", MimeTypeHint: "xyz", Content: []byte("x"),
		})
	require.NoError(t, err)
	require.Equal(t, "application/xyz", klien.unggah[0].TipeMedia)
}

// Jawaban berhasil TANPA ImageID adalah kegagalan.
//
// Baris tanpa kunci penyimpanan akan tampil di grid dengan tombol Lihat yang selalu
// gagal — dan kueri daftar justru menyaring baris seperti itu keluar.
func TestUploadTanpaImageIDAdalahGalat(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{hasil: dokumenpenunjang.HasilUnggah{}}

	_, err := penyimpan(t, klien).Upload(context.Background(),
		inboxcompliance.UploadRequest{
			FileName: "a.pdf", MimeTypeHint: "pdf", Content: []byte("x"),
		})
	require.Error(t, err)
}

// Berkas kosong ditolak sebelum layanan dihubungi.
func TestUploadBerkasKosongDitolak(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{}

	_, err := penyimpan(t, klien).Upload(context.Background(),
		inboxcompliance.UploadRequest{FileName: "a.pdf", MimeTypeHint: "pdf"})
	require.Error(t, err)
	require.Empty(t, klien.unggah, "layanan tidak boleh dihubungi untuk berkas kosong")
}

// Hapus memangkas awalan bucket tanpa perlu mengetahui namanya.
func TestDeleteMemangkasAwalanBucket(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{}

	require.NoError(t, penyimpan(t, klien).Delete(context.Background(),
		inboxcompliance.DeleteRequest{
			ObjectPath: "gs://ember-contoh/Doc/2026/10/berkas.pdf",
		}))

	require.Len(t, klien.hapus, 1)
	require.Equal(t, "Doc/2026/10/berkas.pdf", klien.hapus[0].Jalur)
	require.Equal(t, "KODE-CONTOH", klien.hapus[0].KodeAkses)
}

// Jalur tanpa awalan bucket dikirim apa adanya.
func TestDeleteTanpaAwalanBucket(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{}

	require.NoError(t, penyimpan(t, klien).Delete(context.Background(),
		inboxcompliance.DeleteRequest{ObjectPath: "Doc/2026/10/berkas.pdf"}))
	require.Equal(t, "Doc/2026/10/berkas.pdf", klien.hapus[0].Jalur)
}

// Kegagalan klien dibawa naik apa adanya, tidak disamarkan sebagai berhasil.
func TestKegagalanKlienDibawaNaik(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{galat: errors.New("layanan menjawab 503")}

	_, err := penyimpan(t, klien).Upload(context.Background(),
		inboxcompliance.UploadRequest{
			FileName: "a.pdf", MimeTypeHint: "pdf", Content: []byte("x"),
		})
	require.Error(t, err)
}

// Klien kosong ditolak saat pembentukan, bukan saat panggilan pertama pengguna.
func TestKlienWajibDiisi(t *testing.T) {
	t.Parallel()

	_, err := docapi.New(docapi.Config{App: "KLAIMPNC"})
	require.Error(t, err)
}

// Kredensial diselesaikan SETIAP permintaan, bukan sekali saat start.
//
// Nama folder aplikasi hidup di `general.T_FOLDER_STORAGE` dan kode aksesnya diterbitkan
// per unggahan. Membacanya sekali saat start membuat perubahan barisnya tidak pernah
// terbawa sampai aplikasi di-restart — dan layanan penyimpanan tidak akan menolak nilai
// basi, karena ia tidak memeriksa apa pun.
func TestKredensialDiselesaikanSetiapPermintaan(t *testing.T) {
	t.Parallel()

	var dipanggil int
	var pengunggahTerlihat []string

	klien := &klienPalsu{hasil: dokumenpenunjang.HasilUnggah{ImageID: "IMG-1"}}
	p, err := docapi.New(docapi.Config{
		Klien: klien,
		ResolveKredensial: func(
			_ context.Context, pengunggah string,
		) (docapi.Kredensial, error) {
			dipanggil++
			pengunggahTerlihat = append(pengunggahTerlihat, pengunggah)
			return docapi.Kredensial{
				NamaAplikasi: "folder-ke-" + strconv.Itoa(dipanggil),
				KodeAkses:    "kode-ke-" + strconv.Itoa(dipanggil),
			}, nil
		},
	})
	require.NoError(t, err)

	_, err = p.Upload(context.Background(), inboxcompliance.UploadRequest{
		UserInput: "PETUGAS1", FileName: "a.pdf", MimeTypeHint: "pdf",
		Content: []byte("isi"),
	})
	require.NoError(t, err)

	_, err = p.Upload(context.Background(), inboxcompliance.UploadRequest{
		UserInput: "PETUGAS2", FileName: "b.pdf", MimeTypeHint: "pdf",
		Content: []byte("isi"),
	})
	require.NoError(t, err)

	require.Equal(t, 2, dipanggil, "kredensial harus diselesaikan tiap permintaan")
	require.Equal(t, []string{"PETUGAS1", "PETUGAS2"}, pengunggahTerlihat,
		"pengunggah ikut diteruskan — penerbitan kode akses mencatat atas nama siapa")

	require.Len(t, klien.unggah, 2)
	require.Equal(t, "folder-ke-1", klien.unggah[0].NamaAplikasi)
	require.Equal(t, "kode-ke-1", klien.unggah[0].KodeAkses)
	require.Equal(t, "folder-ke-2", klien.unggah[1].NamaAplikasi)
	require.Equal(t, "kode-ke-2", klien.unggah[1].KodeAkses)
}

// Nama folder KOSONG ditolak di sini, bukan diteruskan ke layanan.
//
// Layanan penyimpanan tidak memeriksa kepemilikan apa pun, sehingga nama folder kosong
// tidak menghasilkan galat dari sana — berkasnya hanya mendarat di tempat yang salah.
// Gagal di sini membuat sebabnya masih terbaca.
func TestNamaFolderKosongDitolakSebelumBerkasTerkirim(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{hasil: dokumenpenunjang.HasilUnggah{ImageID: "IMG-1"}}
	p, err := docapi.New(docapi.Config{
		Klien: klien,
		ResolveKredensial: func(context.Context, string) (docapi.Kredensial, error) {
			return docapi.Kredensial{KodeAkses: "kode"}, nil // tanpa NamaAplikasi
		},
	})
	require.NoError(t, err)

	_, err = p.Upload(context.Background(), inboxcompliance.UploadRequest{
		FileName: "a.pdf", MimeTypeHint: "pdf", Content: []byte("isi"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "T_FOLDER_STORAGE",
		"pesannya harus menyebut di mana barisnya diperiksa")
	require.Empty(t, klien.unggah, "tidak boleh ada apa pun yang terkirim")
}

// Gagal menyelesaikan kredensial TIDAK mengirim apa pun.
func TestKredensialGagalTidakMengirimBerkas(t *testing.T) {
	t.Parallel()

	klien := &klienPalsu{}
	p, err := docapi.New(docapi.Config{
		Klien: klien,
		ResolveKredensial: func(context.Context, string) (docapi.Kredensial, error) {
			return docapi.Kredensial{}, errors.New("basis data tidak terjangkau")
		},
	})
	require.NoError(t, err)

	_, err = p.Upload(context.Background(), inboxcompliance.UploadRequest{
		FileName: "a.pdf", MimeTypeHint: "pdf", Content: []byte("isi"),
	})
	require.Error(t, err)
	require.Empty(t, klien.unggah)

	_, err = p.NewLink(context.Background(), inboxcompliance.LinkRequest{StorageID: "IMG-1"})
	require.Error(t, err)
	require.Empty(t, klien.tautan)

	require.Error(t, p.Delete(context.Background(), inboxcompliance.DeleteRequest{
		ObjectPath: "Doc/2026/10/a.pdf",
	}))
	require.Empty(t, klien.hapus)
}
