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
// Dua hal yang TIDAK dibawa, keduanya sudah dijelaskan di document.go: `IMAGEID` yang di
// Pega tidak pernah terisi, dan `CATEGORY`/`SUB_CATEGORY` yang layarnya tidak punya
// isiannya.

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

	documentID, err := store.NextDocumentID(ctx)
	if err != nil {
		return masterbengkel.Document{}, err
	}

	document := masterbengkel.Document{
		ID:       documentID,
		Name:     clean.FileName,
		MimeType: masterbengkel.DocumentMimeType(clean.FileName),
		Content:  clean.Content,

		// ATTACHNOTE dibiarkan kosong: `Section/UploadDocument` tidak punya isian
		// catatan, dan mengisinya dengan nama berkas — seperti yang dilakukan
		// `InsertDokumentHistoriKlaimPNC` pada jalur LAIN — berarti mengarang untuk
		// jalur ini.
		Note: "",

		UploadedBy: strings.TrimSpace(actor.Login),
		UploadedAt: l.clockOrSystem().Now(),
	}

	if err := store.SaveDocument(ctx, clean.WorkshopID, document); err != nil {
		return masterbengkel.Document{}, err
	}
	return document, nil
}

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
