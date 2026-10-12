package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"claim-pnc/internal/masterpanel"
)

// UploadCommand adalah satu permintaan unggah dokumen panel.
type UploadCommand struct {
	PortalAlias string
	PanelID     string
	FileName    string
	Content     []byte
	Note        string
	By          Actor
}

// UploadAvailable menyatakan jalur unggah siap dipakai.
//
// Layar menanyakannya sebelum menggambar tombol. Tanpa ini tombolnya akan selalu tampak
// hidup dan baru gagal setelah pengguna memilih berkas — kegagalan paling menjengkelkan,
// karena ia terjadi sesudah pekerjaan, bukan sebelumnya.
func (l *Service) UploadAvailable() bool { return l.uploader != nil }

// UploadDocument mengunggah satu berkas dan menautkannya ke panelnya.
//
// # Urutannya mengikuti Pega, dan urutan itu bukan kebetulan
//
// Berkas dikirim ke layanan penyimpanan LEBIH DULU, metadata ditulis SESUDAHNYA. Dibalik,
// barisnya akan menunjuk ke `IMAGEID` yang belum tentu ada. Dengan urutan ini, kegagalan
// di langkah kedua meninggalkan berkas yatim di layanan penyimpanan — tidak terlihat
// pengguna, tidak merusak apa pun, dan dapat dibersihkan terpisah. Itu pertukaran yang
// benar: berkas yatim jauh lebih ringan daripada baris yang menunjuk ke tempat kosong.
//
// Kegagalan langkah kedua karena itu digolongkan UploadHalfDone, bukan UploadUnavailable —
// yang membedakan keduanya adalah apakah pengguna boleh mengulang.
//
// # Unggahan mengganti dokumen yang ada
//
// Satu panel memegang satu dokumen (`PANEL_HE.DOKUMENID`). Ini dinyatakan di layar
// sebelum pengguna memilih berkas, bukan dilaporkan sesudahnya.
func (l *Service) UploadDocument(
	ctx context.Context, cmd UploadCommand, logger *slog.Logger,
) (masterpanel.PanelDocument, error) {
	if l.uploader == nil {
		return masterpanel.PanelDocument{}, &masterpanel.DocumentUploadError{
			Kind: masterpanel.UploadMisconfigured,
			Message: "Layanan unggah dokumen belum terpasang pada lingkungan ini. " +
				"Laporkan ke administrator.",
		}
	}

	request := masterpanel.DocumentUpload{
		PanelID: cmd.PanelID,
		Note:    cmd.Note,
		File: masterpanel.DocumentFile{
			Portal:   cmd.PortalAlias,
			FileName: cmd.FileName,
			Content:  cmd.Content,
			By:       cmd.By.Login,
		},
	}.CleanDocument()

	if err := request.CheckDocument(); err != nil {
		return masterpanel.PanelDocument{}, err
	}

	store, err := l.repoSelector(cmd.PortalAlias)
	if err != nil {
		return masterpanel.PanelDocument{}, err
	}

	// Panelnya dibaca LEBIH DULU, sebelum berkasnya dikirim ke mana pun. Mengunggah dulu
	// lalu menemukan panelnya tidak ada berarti satu berkas yatim di layanan penyimpanan
	// untuk kesalahan yang dapat diketahui dengan satu pembacaan murah.
	if _, err := store.Get(ctx, request.PanelID); err != nil {
		return masterpanel.PanelDocument{}, err
	}

	imageID, err := l.uploader.Upload(ctx, request.File)
	if err != nil {
		return masterpanel.PanelDocument{}, err
	}

	now := time.Now()
	doc := masterpanel.PanelDocument{
		ImageID:    imageID,
		Name:       request.File.FileName,
		Note:       request.Note,
		UploadedBy: request.File.By,
		UploadedAt: &now,
		PanelID:    request.PanelID,
	}

	dataID, err := store.SaveDocument(ctx, doc)
	if err != nil {
		// Berkasnya SUDAH terkirim. Menyembunyikan itu dan melaporkan "gagal, coba lagi"
		// akan membuat pengguna mengunggah ulang dan menumpuk berkas ganda di layanan
		// penyimpanan yang tidak ada satu pun barisnya menunjuk ke sana.
		if logger != nil {
			logger.ErrorContext(ctx, "dokumen panel terunggah tetapi metadatanya gagal disimpan",
				slog.String("modul", "masterpanel"),
				slog.String("panel", request.PanelID),
				slog.String("image_id", imageID),
				slog.String("galat", err.Error()))
		}
		if errors.Is(err, masterpanel.ErrNotFound) {
			return masterpanel.PanelDocument{}, err
		}
		return masterpanel.PanelDocument{}, &masterpanel.DocumentUploadError{
			Kind: masterpanel.UploadHalfDone,
			Message: "Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal disimpan, " +
				"sehingga belum tertaut ke panel. JANGAN unggah ulang — laporkan ke administrator.",
			Err: err,
		}
	}
	doc.DataID = dataID

	if logger != nil {
		logger.InfoContext(ctx, "dokumen panel diunggah",
			slog.String("modul", "masterpanel"),
			slog.String("panel", request.PanelID),
			slog.String("data_id", dataID),
			slog.String("oleh", request.File.By))
	}
	return doc, nil
}

// DocumentOf mengembalikan dokumen sebuah panel.
//
// Ia TIDAK mengembalikan URL berkasnya. URL beserta masa berlakunya dimiliki modul dokumen
// penunjang dan dibaca dari sana saat benar-benar dibuka — menyalinnya ke sini akan
// membuat layar menampilkan tautan yang masa berlakunya sudah lewat tanpa ada yang tahu.
func (l *Service) DocumentOf(
	ctx context.Context, portalAlias, panelID string,
) (masterpanel.PanelDocument, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpanel.PanelDocument{}, err
	}
	doc, err := store.DocumentOf(ctx, panelID)
	if err != nil {
		return masterpanel.PanelDocument{}, fmt.Errorf("%w", err)
	}
	return doc, nil
}
