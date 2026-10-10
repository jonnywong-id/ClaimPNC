// Package dokumenlink menghubungkan seam masterbengkel.DocumentUploader ke modul dokumen
// penunjang — jalur `InsertDokumenPNC`: konversi, izin akses, unggah ke layanan penyimpanan
// internal, dan metadata GENERAL.T_STORAGE_IMAGE (`D-16`).
//
// Modul Master Bengkel tidak mengimpor modul lain secara langsung; adapter inilah
// satu-satunya titik sambungnya — pola yang sama dengan mastersparepart/repo/dokumenlink,
// masterpanel/repo/dokumenlink, dan registrasi/repo/dokumenlink. Galat modul dokumen
// penunjang diterjemahkan di sini menjadi masterbengkel.DocumentUploadError.
//
// # Kenapa menyambung, bukan membangun ulang
//
// `Flow Action/UploadDocument-FA.xml` yang dipakai tombol "Upload Document" Master Bengkel
// berkelas `@baseclass` dan dipanggil 13 section lintas modul — layar Bengkel memakai rule
// yang SAMA dengan layar Sparepart dan Panel. Membangun tiruannya kedua kali di sini berarti
// dua salinan aturan Pega yang sama, dan keduanya akan menyimpang pada perbaikan pertama
// yang hanya diterapkan di salah satunya.
package dokumenlink

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/dokumenpenunjang"
	dokumenpenunjangusecase "claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/masterbengkel"
)

// Uploader mengisi masterbengkel.DocumentUploader.
type Uploader struct {
	service *dokumenpenunjangusecase.Service
}

// New membentuk Uploader di atas layanan dokumen penunjang.
func New(service *dokumenpenunjangusecase.Service) *Uploader { return &Uploader{service: service} }

// Upload mengunggah satu berkas dan mengembalikan IMAGEID-nya.
//
// `ClaimNumber` sengaja dikosongkan: dokumen ini menempel pada BENGKEL, bukan pada klaim.
// Modul dokumen penunjang menggantinya menjadi `TanpaKlaim` ("-"), dan itu memang yang
// benar — memasukkan ID bengkel ke sana akan membuatnya terbaca sebagai nomor klaim oleh
// setiap layar yang menampilkan dokumen per klaim.
func (u *Uploader) Upload(ctx context.Context, f masterbengkel.DocumentFile) (string, error) {
	if u == nil || u.service == nil {
		return "", &masterbengkel.DocumentUploadError{
			Kind:    masterbengkel.UploadMisconfigured,
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
// Pesannya dijaga SAMA PERSIS dengan Master Sparepart, Master Panel, Open Protection, dan
// registrasi — hanya kata "sparepart" yang berganti "bengkel" pada kalimat yang menyebut
// barisnya.
func translate(err error) error {
	fail := func(kind masterbengkel.UploadFailure, message string) error {
		return &masterbengkel.DocumentUploadError{Kind: kind, Message: message, Err: err}
	}
	switch {
	case errors.Is(err, dokumenpenunjang.ErrBerkasTerlaluBesar):
		return fail(masterbengkel.UploadTooLarge, fmt.Sprintf(
			"Ukuran berkas melebihi batas %d MB. Perkecil berkasnya lalu unggah ulang.",
			dokumenpenunjang.BatasUkuranBerkas>>20))
	case errors.Is(err, dokumenpenunjang.ErrBerkasKosong):
		return fail(masterbengkel.UploadInvalid,
			"Berkas kosong. Pilih berkas yang berisi lalu unggah ulang.")
	case errors.Is(err, dokumenpenunjang.ErrNamaBerkasKosong):
		return fail(masterbengkel.UploadInvalid,
			"Nama berkas harus memuat huruf atau angka. Ganti namanya lalu unggah ulang.")
	case errors.Is(err, dokumenpenunjang.ErrPengunggahKosong):
		return fail(masterbengkel.UploadInvalid,
			"Sesi Anda tidak dikenali. Masuk kembali lalu ulangi.")
	case errors.Is(err, dokumenpenunjang.ErrFolderAplikasiTidakAda):
		return fail(masterbengkel.UploadMisconfigured,
			"Folder penyimpanan dokumen belum terdaftar. Laporkan ke administrator — "+
				"unggahan tidak dapat diproses.")
	case errors.Is(err, dokumenpenunjang.ErrLinkTakTerjangkau):
		return fail(masterbengkel.UploadUnavailable,
			"Basis data penyimpanan dokumen (DB link ASMD) sedang tidak dapat dihubungi. "+
				"Berkas belum tersimpan; laporkan ke DBA, lalu coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrKonversiGagal):
		return fail(masterbengkel.UploadUnavailable,
			"Layanan konversi gambar sedang tidak dapat dihubungi, sehingga berkas PNG, JPG, "+
				"dan PDF belum dapat diunggah. Berkas belum tersimpan; silakan coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrUnggahGagal):
		return fail(masterbengkel.UploadUnavailable,
			"Layanan penyimpanan dokumen sedang tidak dapat dihubungi. Berkas belum "+
				"tersimpan; silakan coba lagi.")
	case errors.Is(err, dokumenpenunjang.ErrMetadataGagal):
		return fail(masterbengkel.UploadHalfDone,
			"Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal disimpan, sehingga "+
				"belum tertaut ke bengkel. JANGAN unggah ulang — laporkan ke administrator.")
	}
	return err
}

var _ masterbengkel.DocumentUploader = (*Uploader)(nil)
