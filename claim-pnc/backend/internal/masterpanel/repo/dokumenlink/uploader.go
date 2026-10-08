// Package dokumenlink menghubungkan seam masterpanel.DocumentUploader ke modul dokumen
// penunjang — jalur `InsertDokumenPNC`: konversi, izin akses, unggah ke layanan penyimpanan
// internal, dan metadata GENERAL.T_STORAGE_IMAGE (`D-16`).
//
// Modul Master Panel tidak mengimpor modul lain secara langsung; adapter inilah satu-satunya
// titik sambungnya — pola yang sama dengan registrasi/repo/dokumenlink. Galat modul dokumen
// penunjang diterjemahkan di sini menjadi masterpanel.DocumentUploadError.
//
// # Kenapa menyambung, bukan membangun ulang
//
// Rantai penyimpanan yang dipakai Upload Document Master Panel SAMA PERSIS dengan yang
// dipakai Open Protection dan registrasi — `Flow Action/UploadDocument-FA.xml` berkelas
// `@baseclass` dan dipanggil 13 section lintas modul. Membangunnya ulang di sini berarti
// dua tiruan aturan Pega yang sama, dan keduanya akan menyimpang satu sama lain pada
// perbaikan pertama yang hanya diterapkan di salah satunya.
package dokumenlink

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/dokumenpenunjang"
	dokumenpenunjangusecase "claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/masterpanel"
)

// Uploader mengisi masterpanel.DocumentUploader.
type Uploader struct {
	service *dokumenpenunjangusecase.Service
}

// New membentuk Uploader di atas layanan dokumen penunjang.
func New(service *dokumenpenunjangusecase.Service) *Uploader { return &Uploader{service: service} }

// Upload mengunggah satu berkas dan mengembalikan IMAGEID-nya.
//
// `ClaimNumber` sengaja dikosongkan: dokumen ini menempel pada PANEL, bukan pada klaim.
// Modul dokumen penunjang menggantinya menjadi `TanpaKlaim` ("-"), dan itu memang yang
// benar — memasukkan ID panel ke sana akan membuatnya terbaca sebagai nomor klaim oleh
// setiap layar yang menampilkan dokumen per klaim.
func (u *Uploader) Upload(ctx context.Context, f masterpanel.DocumentFile) (string, error) {
	if u == nil || u.service == nil {
		return "", &masterpanel.DocumentUploadError{
			Kind:    masterpanel.UploadMisconfigured,
			Message: "Layanan unggah dokumen belum terpasang. Laporkan ke administrator.",
		}
	}
	doc, err := u.service.Upload(ctx, dokumenpenunjangusecase.UploadCommand{
		PortalAlias: f.Portal,
		Request: dokumenpenunjang.UploadRequest{
			FileName: f.FileName,
			Content:  f.Content,
			By:       f.By,
		},
	})
	if err != nil {
		return "", translate(err)
	}
	return doc.ImageID, nil
}

// translate memetakan galat modul dokumen penunjang menjadi golongan yang menyatakan apa
// yang boleh dilakukan pengguna.
//
// Pesannya dijaga sama dengan layar Open Protection dan registrasi: pengguna yang menemui
// kegagalan yang sama di dua layar tidak seharusnya membaca dua kalimat yang berbeda.
func translate(err error) error {
	fail := func(kind masterpanel.UploadFailure, message string) error {
		return &masterpanel.DocumentUploadError{Kind: kind, Message: message, Err: err}
	}
	switch {
	case errors.Is(err, dokumenpenunjang.ErrBerkasTerlaluBesar):
		return fail(masterpanel.UploadTooLarge, fmt.Sprintf(
			"Ukuran berkas melebihi batas %d MB. Perkecil berkasnya lalu unggah ulang.",
			dokumenpenunjang.BatasUkuranBerkas>>20))
	case errors.Is(err, dokumenpenunjang.ErrBerkasKosong):
		return fail(masterpanel.UploadInvalid,
			"Berkas kosong. Pilih berkas yang berisi lalu unggah ulang.")
	case errors.Is(err, dokumenpenunjang.ErrNamaBerkasKosong):
		return fail(masterpanel.UploadInvalid,
			"Nama berkas harus memuat huruf atau angka. Ganti namanya lalu unggah ulang.")
	case errors.Is(err, dokumenpenunjang.ErrPengunggahKosong):
		return fail(masterpanel.UploadInvalid,
			"Sesi Anda tidak dikenali. Masuk kembali lalu ulangi.")
	case errors.Is(err, dokumenpenunjang.ErrFolderAplikasiTidakAda):
		return fail(masterpanel.UploadMisconfigured,
			"Folder penyimpanan dokumen belum terdaftar. Laporkan ke administrator — "+
				"unggahan tidak dapat diproses.")
	case errors.Is(err, dokumenpenunjang.ErrLinkTakTerjangkau):
		return fail(masterpanel.UploadUnavailable,
			"Basis data penyimpanan dokumen (DB link ASMD) sedang tidak dapat dihubungi. "+
				"Berkas belum tersimpan; laporkan ke DBA, lalu coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrKonversiGagal):
		return fail(masterpanel.UploadUnavailable,
			"Layanan konversi gambar sedang tidak dapat dihubungi, sehingga berkas PNG, JPG, "+
				"dan PDF belum dapat diunggah. Berkas belum tersimpan; silakan coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrUnggahGagal):
		return fail(masterpanel.UploadUnavailable,
			"Layanan penyimpanan dokumen sedang tidak dapat dihubungi. Berkas belum "+
				"tersimpan; silakan coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrMetadataGagal):
		return fail(masterpanel.UploadHalfDone,
			"Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal disimpan, sehingga "+
				"belum tertaut ke panel. JANGAN unggah ulang — laporkan ke administrator.")
	}
	return err
}

var _ masterpanel.DocumentUploader = (*Uploader)(nil)
