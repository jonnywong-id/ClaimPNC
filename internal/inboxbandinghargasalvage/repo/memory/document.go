package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// DocumentRecord adalah satu baris `POOLDATA.SALAVAGEDOCUMENT` beserta lampirannya.
//
// Kedua tabel digabung menjadi satu catatan di sini, karena kueri sungguhannya pun
// menggabungkan keduanya dengan JOIN. Memisahkannya hanya menambah bentuk yang tidak
// menirukan apa pun.
type DocumentRecord struct {
	// ID adalah `IDDOC`, yang sekaligus `DATA_ATTACHFILE.DATAID`.
	ID string

	// DetailObject adalah nilai kolom `NOKLAIM` pada tabel dokumen.
	//
	// Namanya sengaja BUKAN ClaimNo: kolom itu dibandingkan dengan IDDETAILSALVAGE, dan
	// menamainya menurut kolomnya akan menularkan kekeliruan nama itu ke sini.
	DetailObject string

	SalvageID string

	// Name, MIMEType, dan Content berasal dari tabel lampiran.
	Name     string
	MIMEType string
	Content  []byte

	// UploadedAt adalah `TGLINS`. Nil berarti kolomnya kosong.
	UploadedAt *time.Time

	// Rejected menirukan `IDBALAILELANG` yang sudah terisi — dokumen yang sudah ditandai
	// ditolak tidak lagi muncul di daftar.
	Rejected bool
}

// DocumentStore memenuhi seam inboxbandinghargasalvage.DocumentReader di memori.
//
// Ia memuat rujukan ke Store agar penyaring KEPEMILIKAN dapat ditirukan: daftar dokumen
// menuntut banding itu memang ditangani komite pemanggil, dan jawabannya ada di baris
// checker — bukan di tabel dokumen.
type DocumentStore struct {
	store     *Store
	documents []DocumentRecord
}

// NewDocumentStore membentuk pembaca dokumen di atas satu penyimpanan banding.
func NewDocumentStore(store *Store, documents ...DocumentRecord) *DocumentStore {
	return &DocumentStore{store: store, documents: documents}
}

// ListDocuments menyerahkan dokumen banding satu barang.
func (d *DocumentStore) ListDocuments(
	_ context.Context,
	q inboxbandinghargasalvage.DocumentQuery,
) ([]inboxbandinghargasalvage.DocumentRow, error) {
	rows := []inboxbandinghargasalvage.DocumentRow{}

	if !d.dimilikiKomite(q) {
		return rows, nil
	}

	for _, record := range d.documents {
		if !record.cocok(q) || record.Rejected {
			continue
		}
		rows = append(rows, inboxbandinghargasalvage.DocumentRow{
			ID:         record.ID,
			Name:       record.Name,
			UploadedAt: record.UploadedAt,
		})
	}

	// Terbaru di atas, sama seperti `ORDER BY TGLINS DESC, IDDOC DESC`. Dokumen tanpa
	// tanggal diperlakukan paling lama, sehingga ia turun ke bawah alih-alih melompat ke
	// atas karena tanggal nol.
	sort.SliceStable(rows, func(i, j int) bool {
		kiri, kanan := rows[i].UploadedAt, rows[j].UploadedAt
		switch {
		case kiri == nil && kanan == nil:
			return rows[i].ID > rows[j].ID
		case kiri == nil:
			return false
		case kanan == nil:
			return true
		case kiri.Equal(*kanan):
			return rows[i].ID > rows[j].ID
		default:
			return kiri.After(*kanan)
		}
	})

	return rows, nil
}

// DocumentContent menyerahkan isi satu dokumen.
func (d *DocumentStore) DocumentContent(
	_ context.Context,
	documentID string,
	q inboxbandinghargasalvage.DocumentQuery,
) (inboxbandinghargasalvage.DocumentContent, error) {
	var empty inboxbandinghargasalvage.DocumentContent

	if !d.dimilikiKomite(q) {
		return empty, inboxbandinghargasalvage.ErrDocumentNotFound
	}

	for _, record := range d.documents {
		if record.ID != strings.TrimSpace(documentID) || !record.cocok(q) {
			continue
		}
		return inboxbandinghargasalvage.DocumentContent{
			Name:     record.Name,
			MIMEType: record.MIMEType,
			Content:  record.Content,
		}, nil
	}

	return empty, inboxbandinghargasalvage.ErrDocumentNotFound
}

// dimilikiKomite menirukan klausa EXISTS ke tabel checker.
//
// Tanpa ini, seorang komite dapat membaca dokumen banding komite lain hanya dengan mengirim
// sepasang id — penyaring yang TIDAK ada di sistem lama, dan sengaja ditambahkan.
func (d *DocumentStore) dimilikiKomite(q inboxbandinghargasalvage.DocumentQuery) bool {
	komite := normal(q.Reviewer.Name)

	for _, record := range d.store.checkers {
		if record.DetailObject != q.DetailObject {
			continue
		}
		if strings.TrimSpace(record.SalvageID) != strings.TrimSpace(q.SalvageID) {
			continue
		}
		if normal(record.CommitteeName) == komite {
			return true
		}
	}
	return false
}

// cocok menirukan ketiga penyaring pada tabel dokumen.
//
// `TIPEDOCSALVAGE` tidak ikut diperiksa: penyimpanan ini hanya memuat dokumen banding, dan
// memodelkan jenis dokumen lain hanya menambah kepercayaan palsu pada ujinya.
func (r DocumentRecord) cocok(q inboxbandinghargasalvage.DocumentQuery) bool {
	return r.DetailObject == q.DetailObject &&
		strings.TrimSpace(r.SalvageID) == strings.TrimSpace(q.SalvageID)
}
