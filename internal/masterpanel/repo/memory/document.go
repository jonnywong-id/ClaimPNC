package memory

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/masterpanel"
)

// dataIDSequenceWidth sama dengan lebar pada adapter SQL.
//
// Disalin, bukan dibagi lewat satu konstanta bersama, dan itu disengaja: kedua adapter
// harus dapat berbeda tanpa saling menarik, dan uji kesetaraan bentuk kunci di
// query_test.go yang menjaga keduanya tetap sepakat. Konstanta bersama akan membuat uji
// itu selalu lulus tanpa membuktikan apa pun.
const dataIDSequenceWidth = 10

// documents menyimpan dokumen per panel.
//
// Peta, bukan daftar, karena satu panel memegang SATU dokumen — bentuk yang sama dengan
// kolom `PANEL_HE.DOKUMENID` di basis data. Memakai daftar di sini akan membuat adapter
// memori menerima keadaan yang tidak mungkin terjadi di produksi, dan uji yang lulus
// atasnya tidak membuktikan apa pun.
type documents map[string]masterpanel.PanelDocument

// SaveDocument mencatat dokumen sebuah panel dan menautkannya.
func (r *Repo) SaveDocument(
	_ context.Context, doc masterpanel.PanelDocument,
) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return "", r.failure
	}

	id := strings.TrimSpace(doc.PanelID)
	baris := -1
	for i := range r.rows {
		if strings.EqualFold(strings.TrimSpace(r.rows[i].ID), id) {
			baris = i
			break
		}
	}
	if baris < 0 {
		return "", fmt.Errorf("%w: %q", masterpanel.ErrNotFound, id)
	}

	r.documentSequence++
	at := time.Now()
	if doc.UploadedAt != nil {
		at = *doc.UploadedAt
	}
	number := strconv.FormatInt(r.documentSequence, 10)
	if len(number) < dataIDSequenceWidth {
		number = strings.Repeat("0", dataIDSequenceWidth-len(number)) + number
	}
	doc.DataID = strconv.Itoa(at.Year()) + number
	doc.PanelID = id
	moment := at
	doc.UploadedAt = &moment

	if r.documents == nil {
		r.documents = documents{}
	}
	// Unggahan berikutnya MENGGANTI yang sebelumnya, persis seperti UPDATE pada
	// `PANEL_HE.DOKUMENID`. Barisnya yang lama tidak dihapus di adapter SQL (`D-66`),
	// tetapi tautannya hilang — dan tautan itulah satu-satunya yang terlihat dari layar,
	// jadi inilah yang ditiru di sini.
	r.documents[id] = doc
	// DOKUMENID ikut disetel pada baris panelnya — adapter SQL melakukannya lewat
	// `panel_document_link` di dalam transaksi yang sama. Tanpa ini, tiruan memorinya
	// berperilaku berbeda dari produksi, dan uji yang lulus atasnya tidak membuktikan apa pun.
	r.rows[baris].DocumentID = doc.DataID
	// Disimpan pula menurut DataID: unggah CSV master menautkan SATU dokumen ke BANYAK
	// panel, dan penautannya hanya membawa DataID — bukan dokumennya.
	if r.documentByID == nil {
		r.documentByID = map[string]masterpanel.PanelDocument{}
	}
	r.documentByID[doc.DataID] = doc
	return doc.DataID, nil
}

// DocumentOf mengembalikan dokumen sebuah panel.
func (r *Repo) DocumentOf(
	_ context.Context, panelID string,
) (masterpanel.PanelDocument, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return masterpanel.PanelDocument{}, r.failure
	}
	doc, ok := r.documents[strings.TrimSpace(panelID)]
	if !ok {
		return masterpanel.PanelDocument{}, fmt.Errorf(
			"%w: %q", masterpanel.ErrDocumentMissing, panelID)
	}
	return doc, nil
}

// FakeUploader mengisi masterpanel.DocumentUploader tanpa menyentuh jaringan.
//
// Ia adapter KEDUA seam itu, dan itulah yang menjadikan seam-nya nyata alih-alih
// hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Tanpa ini, setiap uji unggah menuntut
// layanan penyimpanan internal yang hidup — dan uji yang menuntut jaringan adalah uji yang
// akhirnya dimatikan orang.
type FakeUploader struct {
	// Failure, bila diisi, dikembalikan apa adanya. Dipakai menguji jalur galat.
	Failure error

	// ImageID adalah kunci yang dikembalikan. Kosong berarti kunci yang dibangkitkan dari
	// pencacah di bawah.
	ImageID string

	// Uploaded merekam setiap berkas yang masuk, supaya uji dapat memeriksa APA yang
	// dikirim — bukan sekadar bahwa pemanggilannya terjadi.
	Uploaded []masterpanel.DocumentFile

	counter int
}

// Upload merekam berkasnya dan mengembalikan kunci palsu.
func (f *FakeUploader) Upload(
	_ context.Context, file masterpanel.DocumentFile,
) (string, error) {
	if f.Failure != nil {
		return "", f.Failure
	}
	f.Uploaded = append(f.Uploaded, file)
	if f.ImageID != "" {
		return f.ImageID, nil
	}
	f.counter++
	return fmt.Sprintf("IMG%09d", f.counter), nil
}

var _ masterpanel.DocumentUploader = (*FakeUploader)(nil)

// LinkDocument menautkan dokumen yang sudah tercatat ke sebuah panel.
func (r *Repo) LinkDocument(_ context.Context, panelID, dataID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return r.failure
	}

	id := strings.TrimSpace(panelID)
	for i := range r.rows {
		if strings.EqualFold(strings.TrimSpace(r.rows[i].ID), id) {
			r.rows[i].DocumentID = dataID
			// Dokumen yang sama ditautkan ke banyak panel: salinannya disimpan per panel,
			// persis seperti kolom DOKUMENID yang berisi DATAID yang sama di banyak baris.
			if source, ada := r.documentByID[dataID]; ada {
				if r.documents == nil {
					r.documents = documents{}
				}
				salinan := source
				salinan.PanelID = id
				r.documents[id] = salinan
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %q", masterpanel.ErrNotFound, id)
}
