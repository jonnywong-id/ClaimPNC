// Package dokumenlink menghubungkan seam registrasi.DocumentUploader ke modul dokumen
// penunjang — jalur `InsertDokumenPNC`: konversi, izin akses, unggah ke layanan penyimpanan
// internal, dan metadata GENERAL.T_STORAGE_IMAGE (`D-16`).
//
// Modul registrasi tidak mengimpor modul lain secara langsung; adapter inilah satu-satunya
// titik sambungnya, seperti komitelink untuk penjenjangan komite. Galat modul dokumen
// penunjang diterjemahkan di sini ke registrasi.DocumentUploadError, dengan pesan yang
// sama seperti layar Open Protection.
package dokumenlink

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/dokumenpenunjang"
	dokumenpenunjangusecase "claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/registrasi"
)

// Uploader mengisi registrasi.DocumentUploader.
type Uploader struct{ service *dokumenpenunjangusecase.Service }

// New membentuk Uploader di atas layanan dokumen penunjang.
func New(service *dokumenpenunjangusecase.Service) *Uploader { return &Uploader{service: service} }

// Upload mengunggah satu berkas dan mengembalikan IMAGEID-nya.
func (u *Uploader) Upload(ctx context.Context, f registrasi.DocumentFile) (string, error) {
	if u == nil || u.service == nil {
		return "", &registrasi.DocumentUploadError{Kind: registrasi.UploadMisconfigured,
			Message: "Layanan unggah dokumen belum terpasang. Laporkan ke administrator."}
	}
	doc, err := u.service.Upload(ctx, dokumenpenunjangusecase.UploadCommand{
		PortalAlias: f.Portal,
		Request: dokumenpenunjang.UploadRequest{
			ClaimNumber: f.ClaimNumber,
			FileName:    f.FileName,
			Content:     f.Content,
			By:          f.By,
		},
	})
	if err != nil {
		return "", translate(err)
	}
	return doc.ImageID, nil
}

// translate memetakan galat modul dokumen penunjang.
func translate(err error) error {
	fail := func(kind registrasi.UploadFailure, message string) error {
		return &registrasi.DocumentUploadError{Kind: kind, Message: message, Err: err}
	}
	switch {
	case errors.Is(err, dokumenpenunjang.ErrBerkasTerlaluBesar):
		return fail(registrasi.UploadTooLarge, fmt.Sprintf(
			"Ukuran berkas melebihi batas %d MB. Perkecil berkasnya lalu unggah ulang.",
			dokumenpenunjang.BatasUkuranBerkas>>20))
	case errors.Is(err, dokumenpenunjang.ErrBerkasKosong):
		return fail(registrasi.UploadInvalid, "Berkas kosong. Pilih berkas yang berisi lalu unggah ulang.")
	case errors.Is(err, dokumenpenunjang.ErrNamaBerkasKosong):
		return fail(registrasi.UploadInvalid,
			"Nama berkas harus memuat huruf atau angka. Ganti namanya lalu unggah ulang.")
	case errors.Is(err, dokumenpenunjang.ErrPengunggahKosong):
		return fail(registrasi.UploadInvalid, "Sesi Anda tidak dikenali. Masuk kembali lalu ulangi.")
	case errors.Is(err, dokumenpenunjang.ErrFolderAplikasiTidakAda):
		return fail(registrasi.UploadMisconfigured,
			"Folder penyimpanan dokumen belum terdaftar. Laporkan ke administrator — unggahan tidak dapat diproses.")
	case errors.Is(err, dokumenpenunjang.ErrKonversiGagal):
		return fail(registrasi.UploadUnavailable,
			"Layanan konversi gambar sedang tidak dapat dihubungi, sehingga berkas PNG, JPG, dan PDF "+
				"belum dapat diunggah. Berkas belum tersimpan; silakan coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrUnggahGagal):
		return fail(registrasi.UploadUnavailable,
			"Layanan penyimpanan dokumen sedang tidak dapat dihubungi. Berkas belum tersimpan; silakan coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrMetadataGagal):
		return fail(registrasi.UploadHalfDone,
			"Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal disimpan, sehingga belum muncul "+
				"di daftar. JANGAN unggah ulang — laporkan ke administrator.")
	}
	return err
}

var _ registrasi.DocumentUploader = (*Uploader)(nil)
