package dokumenlink_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	dokumenpenunjangusecase "claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/dokumenlink"
)

// Adapter tanpa layanan menolak dengan sebab yang menyebut pemasangannya.
//
// Ia TIDAK boleh nil-panic: perakitan yang belum lengkap harus terbaca sebagai pesan, bukan
// sebagai 500 tanpa keterangan.
func TestUploaderWithoutAServiceIsMisconfigured(t *testing.T) {
	for nama, uploader := range map[string]*dokumenlink.Uploader{
		"nil":           nil,
		"tanpa layanan": dokumenlink.New(nil),
	} {
		t.Run(nama, func(t *testing.T) {
			_, err := uploader.Upload(context.Background(), masterbengkel.DocumentFile{})

			var upload *masterbengkel.DocumentUploadError
			require.ErrorAs(t, err, &upload)
			require.Equal(t, masterbengkel.UploadMisconfigured, upload.Kind)
		})
	}
}

// Setiap galat modul dokumen penunjang punya golongan yang menyatakan APA YANG BOLEH
// DILAKUKAN PENGGUNA.
//
// Uji ini menembak terjemahannya lewat Upload yang sungguhan — bukan fungsi internalnya —
// supaya ia tetap berlaku bila susunan di dalamnya berubah.
func TestEveryUpstreamFailureIsTranslated(t *testing.T) {
	cases := map[error]masterbengkel.UploadFailure{
		dokumenpenunjang.ErrBerkasTerlaluBesar:     masterbengkel.UploadTooLarge,
		dokumenpenunjang.ErrBerkasKosong:           masterbengkel.UploadInvalid,
		dokumenpenunjang.ErrNamaBerkasKosong:       masterbengkel.UploadInvalid,
		dokumenpenunjang.ErrPengunggahKosong:       masterbengkel.UploadInvalid,
		dokumenpenunjang.ErrFolderAplikasiTidakAda: masterbengkel.UploadMisconfigured,
		dokumenpenunjang.ErrLinkTakTerjangkau:      masterbengkel.UploadUnavailable,
		dokumenpenunjang.ErrKonversiGagal:          masterbengkel.UploadUnavailable,
		dokumenpenunjang.ErrUnggahGagal:            masterbengkel.UploadUnavailable,
		dokumenpenunjang.ErrMetadataGagal:          masterbengkel.UploadHalfDone,
	}

	for hulu, golongan := range cases {
		t.Run(hulu.Error(), func(t *testing.T) {
			err := terjemahkan(t, fmt.Errorf("terbungkus: %w", hulu))

			var upload *masterbengkel.DocumentUploadError
			require.ErrorAs(t, err, &upload)
			require.Equal(t, golongan, upload.Kind)
			require.NotEmpty(t, upload.Message, "pesannya harus siap tampil")

			// Galat aslinya tetap terbaca lewat rantai, supaya log tidak kehilangan
			// sebabnya saat layar hanya menampilkan pesan ramahnya.
			require.ErrorIs(t, err, hulu)
		})
	}
}

// Galat di luar daftar diteruskan APA ADANYA, bukan dipaksa masuk salah satu golongan.
//
// Memaksanya akan membuat kegagalan yang belum dikenali tampak seperti kegagalan yang sudah
// dipahami — dan pengguna disuruh mengulang sesuatu yang belum tentu aman diulang.
func TestUnknownFailuresArePassedThrough(t *testing.T) {
	asing := errors.New("sesuatu yang belum dikenali")
	err := terjemahkan(t, asing)

	var upload *masterbengkel.DocumentUploadError
	require.False(t, errors.As(err, &upload))
	require.ErrorIs(t, err, asing)
}

// terjemahkan menjalankan Upload terhadap layanan yang pasti gagal dengan galat yang
// diminta.
//
// Ia memakai Service modul dokumen penunjang yang sungguhan dengan penyimpanan yang
// menolak, sehingga yang diuji adalah jalur yang benar-benar ditempuh di produksi.
func terjemahkan(t *testing.T, hulu error) error {
	t.Helper()

	_, err := dokumenlink.New(serviceYangGagal(t, hulu)).
		Upload(context.Background(), masterbengkel.DocumentFile{
			Portal:   "ASM",
			FileName: "bukti.pdf",
			Content:  []byte("isi"),
			By:       "penguji",
		})
	require.Error(t, err)
	return err
}

// serviceYangGagal membentuk Service dokumen penunjang yang pasti gagal dengan galat yang
// diminta.
//
// Titik suntiknya `NamaFolderAplikasi`, dan itu dipilih karena ia langkah pertama yang
// menyentuh repo — sesudah pemeriksaan masukan, sebelum apa pun terkirim. Satu titik cukup
// untuk seluruh daftar, sebab terjemahannya memang pemetaan murni atas `errors.Is`.
func serviceYangGagal(t *testing.T, hulu error) *dokumenpenunjangusecase.Service {
	t.Helper()

	service, err := dokumenpenunjangusecase.NewService(dokumenpenunjangusecase.Options{
		Repos: func(string) (dokumenpenunjang.Repo, error) {
			return repoGagal{err: hulu}, nil
		},
		Storage:   penyimpananDiam{},
		Converter: konversiDiam{},
		Clock:     jamTetap{},

		// Konversi dilewati supaya jalurnya sampai ke repo tanpa bergantung pada layanan
		// konversi — yang sedang tidak diuji di sini.
		SkipConversion: true,
	})
	require.NoError(t, err)
	return service
}

type repoGagal struct{ err error }

func (r repoGagal) NamaFolderAplikasi(context.Context, string) (string, error) {
	return "", r.err
}

func (r repoGagal) CatatAksesUnggah(context.Context, string, string) (string, error) {
	return "", r.err
}

func (r repoGagal) Simpan(context.Context, dokumenpenunjang.Document) error { return r.err }

func (r repoGagal) PerKlaim(context.Context, string) ([]dokumenpenunjang.Document, error) {
	return nil, r.err
}

func (r repoGagal) Ambil(context.Context, string) (dokumenpenunjang.Document, error) {
	return dokumenpenunjang.Document{}, r.err
}

type penyimpananDiam struct{}

func (penyimpananDiam) Upload(
	context.Context,
	dokumenpenunjang.PerintahUnggah,
) (dokumenpenunjang.HasilUnggah, error) {
	return dokumenpenunjang.HasilUnggah{ImageID: "img", URL: "https://contoh"}, nil
}

type konversiDiam struct{}

func (konversiDiam) Convert(_ context.Context, isi []byte) ([]byte, error) { return isi, nil }

type jamTetap struct{}

func (jamTetap) Now() time.Time { return time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC) }
