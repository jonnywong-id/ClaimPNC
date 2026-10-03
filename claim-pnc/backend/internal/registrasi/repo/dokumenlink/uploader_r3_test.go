package dokumenlink

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/registrasi"
)

// Unggahan tanpa layanan terpasang ditolak sebagai salah konfigurasi, bukan panik.
func TestUploadWithoutService(t *testing.T) {
	var nilUploader *Uploader
	for _, u := range []*Uploader{nilUploader, New(nil)} {
		_, err := u.Upload(context.Background(), registrasi.DocumentFile{})
		var upload *registrasi.DocumentUploadError
		require.True(t, errors.As(err, &upload))
		require.Equal(t, registrasi.UploadMisconfigured, upload.Kind)
	}
}

// Setiap galat modul dokumen penunjang diterjemahkan ke jenis kegagalan unggah registrasi,
// dengan galat asalnya tetap terbungkus.
func TestTranslateUploadErrors(t *testing.T) {
	cases := map[error]registrasi.UploadFailure{
		dokumenpenunjang.ErrBerkasTerlaluBesar:     registrasi.UploadTooLarge,
		dokumenpenunjang.ErrBerkasKosong:           registrasi.UploadInvalid,
		dokumenpenunjang.ErrNamaBerkasKosong:       registrasi.UploadInvalid,
		dokumenpenunjang.ErrPengunggahKosong:       registrasi.UploadInvalid,
		dokumenpenunjang.ErrFolderAplikasiTidakAda: registrasi.UploadMisconfigured,
		dokumenpenunjang.ErrLinkTakTerjangkau:      registrasi.UploadUnavailable,
		dokumenpenunjang.ErrKonversiGagal:          registrasi.UploadUnavailable,
		dokumenpenunjang.ErrUnggahGagal:            registrasi.UploadUnavailable,
		dokumenpenunjang.ErrMetadataGagal:          registrasi.UploadHalfDone,
	}
	for cause, kind := range cases {
		wrapped := fmt.Errorf("lapisan: %w", cause)
		var upload *registrasi.DocumentUploadError
		err := translate(wrapped)
		require.True(t, errors.As(err, &upload), "%v", cause)
		require.Equal(t, kind, upload.Kind, "%v", cause)
		require.NotEmpty(t, upload.Message)
		require.ErrorIs(t, err, cause)
	}
	other := errors.New("lain")
	require.Same(t, other, translate(other))
}
