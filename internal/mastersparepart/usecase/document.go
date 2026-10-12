package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/mastersparepart"
)

// UploadCommand adalah satu permintaan unggah dokumen sparepart.
type UploadCommand struct {
	PortalAlias string
	SparepartID string
	FileName    string
	Content     []byte
	Note        string
	By          Actor
}

// UploadAvailable menyatakan jalur unggah siap dipakai.
//
// Layar menanyakannya sebelum menggambar isiannya. Tanpa ini isiannya akan selalu tampak
// hidup dan baru gagal setelah pengguna memilih berkas — kegagalan paling menjengkelkan,
// karena ia terjadi sesudah pekerjaan, bukan sebelumnya.
func (l *Service) UploadAvailable() bool { return l.uploader != nil }

// UploadDocument mengunggah satu berkas dan menautkannya ke sparepart-nya.
//
// # Urutannya mengikuti Pega, dan urutan itu bukan kebetulan
//
// Berkas dikirim ke layanan penyimpanan LEBIH DULU, metadata ditulis SESUDAHNYA. Dibalik,
// barisnya akan menunjuk ke `IMAGEID` yang belum tentu ada. Dengan urutan ini, kegagalan di
// langkah kedua meninggalkan berkas yatim di layanan penyimpanan — tidak terlihat pengguna,
// tidak merusak apa pun, dan dapat dibersihkan terpisah. Itu pertukaran yang benar: berkas
// yatim jauh lebih ringan daripada baris yang menunjuk ke tempat kosong.
//
// Kegagalan langkah kedua karena itu digolongkan UploadHalfDone, bukan UploadUnavailable —
// yang membedakan keduanya adalah apakah pengguna boleh mengulang.
//
// # Unggahan mengganti dokumen yang ada
//
// Satu sparepart memegang satu dokumen (`SPAREPART_HE.DOKUMENID`). Ini dinyatakan di layar
// sebelum pengguna memilih berkas, bukan dilaporkan sesudahnya.
//
// # Mengunggah TIDAK memindahkan baris ke antrean persetujuan
//
// Berbeda dari menyimpan lewat form, yang `Activity/UpdateSparepartHE_act` setel
// `APPROVAL := "0"` tanpa syarat. Kueri `sparepart_document_link` hanya menyentuh kolom
// `DOKUMENID`. Pembedaan itu disengaja: melampirkan berkas bukan mengubah nilai sparepart,
// sehingga memaksanya melewati persetujuan ulang akan menahan baris yang isinya tidak
// berubah sama sekali.
func (l *Service) UploadDocument(
	ctx context.Context, cmd UploadCommand, logger *slog.Logger,
) (mastersparepart.SparepartDocument, error) {
	if l.uploader == nil {
		return mastersparepart.SparepartDocument{}, &mastersparepart.DocumentUploadError{
			Kind: mastersparepart.UploadMisconfigured,
			Message: "Layanan unggah dokumen belum terpasang pada lingkungan ini. " +
				"Laporkan ke administrator.",
		}
	}

	request := mastersparepart.DocumentUpload{
		SparepartID: cmd.SparepartID,
		Note:        cmd.Note,
		File: mastersparepart.DocumentFile{
			Portal:   cmd.PortalAlias,
			FileName: cmd.FileName,
			Content:  cmd.Content,
			By:       cmd.By.Login,
		},
	}.CleanDocument()

	if err := request.CheckDocument(); err != nil {
		return mastersparepart.SparepartDocument{}, err
	}

	store, err := l.repoSelector(cmd.PortalAlias)
	if err != nil {
		return mastersparepart.SparepartDocument{}, err
	}

	// Sparepart-nya dibaca LEBIH DULU, sebelum berkasnya dikirim ke mana pun. Mengunggah
	// dulu lalu menemukan barisnya tidak ada berarti satu berkas yatim di layanan
	// penyimpanan untuk kesalahan yang dapat diketahui dengan satu pembacaan murah.
	if _, err := store.Get(ctx, request.SparepartID); err != nil {
		return mastersparepart.SparepartDocument{}, err
	}

	imageID, err := l.uploader.Upload(ctx, request.File)
	if err != nil {
		return mastersparepart.SparepartDocument{}, err
	}

	// Waktunya diambil dari seam Clock, bukan time.Now() — sama seperti stempel
	// TGL_UPDATE_HARGA, dan karena alasan yang sama (`F-5`).
	now := l.clock.Now()
	doc := mastersparepart.SparepartDocument{
		ImageID:     imageID,
		Name:        request.File.FileName,
		Note:        request.Note,
		UploadedBy:  request.File.By,
		UploadedAt:  &now,
		SparepartID: request.SparepartID,
	}

	dataID, err := store.SaveDocument(ctx, doc)
	if err != nil {
		// Berkasnya SUDAH terkirim. Menyembunyikan itu dan melaporkan "gagal, coba lagi"
		// akan membuat pengguna mengunggah ulang dan menumpuk berkas ganda di layanan
		// penyimpanan yang tidak ada satu pun barisnya menunjuk ke sana.
		if logger != nil {
			logger.ErrorContext(ctx,
				"dokumen sparepart terunggah tetapi metadatanya gagal disimpan",
				slog.String("modul", "mastersparepart"),
				slog.String("sparepart", request.SparepartID),
				slog.String("image_id", imageID),
				slog.String("galat", err.Error()))
		}
		if errors.Is(err, mastersparepart.ErrNotFound) {
			return mastersparepart.SparepartDocument{}, err
		}
		return mastersparepart.SparepartDocument{}, &mastersparepart.DocumentUploadError{
			Kind: mastersparepart.UploadHalfDone,
			Message: "Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal disimpan, " +
				"sehingga belum tertaut ke sparepart. JANGAN unggah ulang — laporkan ke " +
				"administrator.",
			Err: err,
		}
	}
	doc.DataID = dataID

	if logger != nil {
		logger.InfoContext(ctx, "dokumen sparepart diunggah",
			slog.String("modul", "mastersparepart"),
			slog.String("sparepart", request.SparepartID),
			slog.String("data_id", dataID),
			slog.String("oleh", request.File.By))
	}
	return doc, nil
}

// DocumentOf mengembalikan dokumen sebuah sparepart.
//
// Ia TIDAK mengembalikan URL berkasnya. URL beserta masa berlakunya dimiliki modul dokumen
// penunjang dan dibaca dari sana saat benar-benar dibuka — menyalinnya ke sini akan membuat
// layar menampilkan tautan yang masa berlakunya sudah lewat tanpa ada yang tahu.
func (l *Service) DocumentOf(
	ctx context.Context, portalAlias, sparepartID string,
) (mastersparepart.SparepartDocument, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersparepart.SparepartDocument{}, err
	}
	doc, err := store.DocumentOf(ctx, sparepartID)
	if err != nil {
		return mastersparepart.SparepartDocument{}, fmt.Errorf("%w", err)
	}
	return doc, nil
}
