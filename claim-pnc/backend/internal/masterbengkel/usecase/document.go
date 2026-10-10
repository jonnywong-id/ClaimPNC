package usecase

import (
	"context"
	"strings"
	"time"

	"claim-pnc/internal/masterbengkel"
)

// Unggah dan lihat dokumen lampiran sebuah bengkel.
//
// Urutannya mengikuti sistem lama, dan tiap langkah menyebut asalnya:
//
//	GCNMUploadResult64            baca berkas dari permintaan, ambil nama dan isinya
//	PNCSaveAttachmentToDB         terbitkan DATAID lalu catat lampirannya
//	UpdateBengkelHE-SQL           taruh DATAID di BENGKEL_HE.DOKUMENID
//
// Membacanya kembali menempuh jalur `GetDetailDocument`:
//
//	GetIDDokumenBengkel           DOKUMENID milik satu bengkel
//	GetAttachmentFromDB_Sql       barisnya di DATA_ATTACHFILE
//
// Satu langkah DISISIPKAN sebelum pencatatan, dan alasannya ada di document.go: isi berkas
// dikirim ke layanan penyimpanan internal lewat seam DocumentUploader, lalu `IMAGEID`-nya
// yang disimpan di barisnya. Di Pega kolom itu disediakan prosedurnya tetapi tidak pernah
// diisi jalur Master Bengkel, sehingga berkasnya tidak tersimpan di mana pun.
//
// `CATEGORY` dan `SUB_CATEGORY` tetap TIDAK dibawa — layarnya tidak punya isiannya.

// systemClock adalah jam bawaan, dipakai bila perakitan tidak menyetel jam sendiri.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// clock menjawab dengan jam yang disetel, atau jam sistem bila belum disetel.
func (l *Service) clockOrSystem() masterbengkel.Clock {
	if l.clock != nil {
		return l.clock
	}
	return systemClock{}
}

// UploadDocument melampirkan satu berkas pada sebuah bengkel.
//
// # Satu lampiran per bengkel, meniru bentuk datanya
//
// `BENGKEL_HE` hanya punya SATU kolom `DOKUMENID`, sehingga unggahan berikutnya
// MENGGANTIKAN tautan yang sebelumnya — persis seperti Pega. Baris lampiran yang lama
// tidak dihapus (`D-66`); ia hanya tidak lagi tertaut, dan tetap dapat ditemukan lewat
// DATAID-nya.
//
// # Kenapa pemeriksaan masukan terjadi sebelum portal dipilih
//
// Berkas yang terlalu besar atau berjenis tidak diizinkan ditolak tanpa menyentuh basis
// data sama sekali. Menyusun koneksi lebih dulu berarti membayar biaya untuk permintaan
// yang sudah pasti ditolak.
func (l *Service) UploadDocument(
	ctx context.Context,
	portalAlias string,
	actor Actor,
	input masterbengkel.UploadInput,
) (masterbengkel.Document, error) {
	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterbengkel.Document{}, err
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterbengkel.Document{}, err
	}

	// Bengkelnya dibaca lebih dulu supaya "bengkel tidak ditemukan" terjawab sebelum satu
	// pun baris lampiran tercatat. Tanpa ini, lampirannya tersimpan lalu penautannya
	// gagal — dan meski transaksinya membatalkan keduanya, nomor urut DATAID sudah
	// telanjur terpakai.
	if _, err := store.Get(ctx, clean.WorkshopID); err != nil {
		return masterbengkel.Document{}, err
	}

	// Isi berkas pergi lebih dulu. Urutannya mengikuti `InsertDokumenPNC`, dan alasannya
	// bukan selera: nomor urut DATAID tidak dapat dikembalikan bila unggahannya gagal
	// sesudahnya, sehingga mengambil nomor lebih dulu berarti membuang satu nomor setiap
	// kali layanan penyimpanan sedang bermasalah.
	imageID, err := l.uploadFile(ctx, portalAlias, actor, clean)
	if err != nil {
		return masterbengkel.Document{}, err
	}

	documentID, err := store.NextDocumentID(ctx)
	if err != nil {
		return masterbengkel.Document{}, err
	}

	document := masterbengkel.Document{
		ID:       documentID,
		ImageID:  imageID,
		Name:     clean.FileName,
		MimeType: masterbengkel.DocumentMimeType(clean.FileName),

		// ATTACHNOTE dibiarkan kosong: `Section/UploadDocument` tidak punya isian
		// catatan, dan mengisinya dengan nama berkas — seperti yang dilakukan
		// `InsertDokumentHistoriKlaimPNC` pada jalur LAIN — berarti mengarang untuk
		// jalur ini.
		Note: "",

		UploadedBy: strings.TrimSpace(actor.Login),
		UploadedAt: l.clockOrSystem().Now(),
	}

	if err := store.SaveDocument(ctx, clean.WorkshopID, document); err != nil {
		// Berkasnya SUDAH sampai di layanan penyimpanan; yang gagal hanyalah catatannya.
		// Mengembalikan galat basis data apa adanya akan membuat layar menyarankan
		// "coba lagi" — dan pengulangan di sini menumpuk berkas ganda di penyimpanan,
		// sebab unggahan yang pertama tidak dapat ditarik kembali.
		return masterbengkel.Document{}, &masterbengkel.DocumentUploadError{
			Kind: masterbengkel.UploadHalfDone,
			Message: "Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal " +
				"disimpan, sehingga belum tertaut ke bengkel. JANGAN unggah ulang — " +
				"laporkan ke administrator.",
			Err: err,
		}
	}
	return document, nil
}

// uploadFile mengirim isi berkas ke layanan penyimpanan dan mengembalikan IMAGEID-nya.
//
// Pengunggah yang belum terpasang ditolak DI SINI, bukan dibiarkan menjadi nil-panic di
// hilir, dan bukan pula dibiarkan lewat sebagai dokumen tanpa IMAGEID — yang terakhir itu
// persis cacat sistem lama yang modul ini hendak tutup.
func (l *Service) uploadFile(
	ctx context.Context,
	portalAlias string,
	actor Actor,
	clean masterbengkel.UploadInput,
) (string, error) {
	if l.uploader == nil {
		return "", &masterbengkel.DocumentUploadError{
			Kind: masterbengkel.UploadMisconfigured,
			Message: "Layanan penyimpanan dokumen belum terpasang pada lingkungan ini. " +
				"Laporkan ke administrator.",
		}
	}
	return l.uploader.Upload(ctx, masterbengkel.DocumentFile{
		Portal:   portalAlias,
		FileName: clean.FileName,
		Content:  clean.Content,
		By:       strings.TrimSpace(actor.Login),
	})
}

// UploadAvailable menyatakan apakah jalur unggah siap dipakai.
//
// Layar membutuhkannya SEBELUM pengguna memilih berkas: menonaktifkan tombolnya beserta
// sebabnya jauh lebih terbaca daripada membiarkan pengguna memilih berkas, menunggu
// unggahan, lalu menerima penolakan.
func (l *Service) UploadAvailable() bool { return l.uploader != nil }

// Document mengembalikan lampiran yang tertaut pada sebuah bengkel.
//
// Ia meniru `GetDetailDocument`: baca `DOKUMENID` dari barisnya, lalu ambil lampirannya.
// Bengkel yang belum pernah dilampiri menjawab ErrDocumentNotFound — dibedakan dari
// bengkel yang tidak ada, yang menjawab ErrNotFound.
func (l *Service) Document(
	ctx context.Context,
	portalAlias, workshopID string,
) (masterbengkel.Document, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterbengkel.Document{}, err
	}

	workshop, err := store.Get(ctx, strings.TrimSpace(workshopID))
	if err != nil {
		return masterbengkel.Document{}, err
	}

	documentID := strings.TrimSpace(workshop.DocumentID)
	if documentID == "" {
		return masterbengkel.Document{}, masterbengkel.ErrDocumentNotFound
	}
	return store.FindDocument(ctx, documentID)
}
