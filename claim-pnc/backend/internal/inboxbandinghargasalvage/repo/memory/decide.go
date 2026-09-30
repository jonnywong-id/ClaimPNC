package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Writer menulis keputusan banding harga ke penyimpanan di memori.
//
// # Kenapa ia tipe tersendiri, dan bukan method pada Store
//
// Karena Store tidak pernah berubah setelah dibentuk — itulah yang membuatnya aman dipakai
// bersama tanpa mutex. Penulisan mengubahnya, sehingga ia memerlukan penguncian yang tidak
// dibutuhkan jalur baca mana pun.
//
// Keduanya berbagi baris yang SAMA, bukan salinan: keputusan yang ditulis Writer harus
// terlihat pada daftar dan panel rincian yang dibaca Store, sebagaimana di basis data.
type Writer struct {
	store *Store
	lock  sync.Mutex

	// now menyerahkan waktu putusan, menggantikan `CURRENT_TIMESTAMP`.
	now func() time.Time
}

// NewWriter membentuk penulis keputusan di atas satu penyimpanan.
func NewWriter(store *Store) *Writer {
	return &Writer{store: store, now: func() time.Time { return time.Now().UTC() }}
}

// WithNow menyerahkan penulis yang waktu putusannya tetap, untuk uji yang memeriksa
// tanggalnya.
func (w *Writer) WithNow(at time.Time) *Writer {
	w.now = func() time.Time { return at }
	return w
}

// Decide menirukan sqlstore.Writer.Decide, termasuk urutan dan prakondisi langkahnya.
//
// Penyaring `TGLAPPROVE IS NULL` ditiru pula, dan itu yang membuat penekanan tombol kedua
// menghasilkan ErrAlreadyDecided alih-alih menimpa putusan pertama.
func (w *Writer) Decide(
	_ context.Context,
	command inboxbandinghargasalvage.DecisionCommand,
) (inboxbandinghargasalvage.DecisionResult, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	var empty inboxbandinghargasalvage.DecisionResult

	at := w.now()
	komite := normal(command.Reviewer.Name)

	// Langkah pertama: catat putusannya. Bila tidak ada yang cocok, tidak satu pun langkah
	// berikutnya dijalankan — sama seperti transaksi yang dibatalkan.
	recorded := false
	for index := range w.store.checkers {
		record := &w.store.checkers[index]
		if normal(record.CommitteeName) != komite {
			continue
		}
		if record.DetailObject != command.DetailObject {
			continue
		}
		if record.ApprovedAt != nil {
			continue
		}

		record.ApprovalStatus = command.Status
		record.CheckerNote = command.Note
		record.ApprovedAt = &at
		recorded = true
	}
	if !recorded {
		return empty, inboxbandinghargasalvage.ErrAlreadyDecided
	}

	result := inboxbandinghargasalvage.DecisionResult{Recorded: true}

	if command.CascadeTo != "" {
		berikutnya := normal(command.CascadeTo)
		for index := range w.store.checkers {
			record := &w.store.checkers[index]
			if normal(record.CommitteeName) != berikutnya {
				continue
			}
			if record.DetailObject != command.DetailObject || record.ApprovedAt != nil {
				continue
			}
			// Catatan sengaja TIDAK ikut ditulis — pernyataan aslinya pun tidak.
			record.ApprovalStatus = command.Status
			record.ApprovedAt = &at
		}
	}

	if command.MarkDocument {
		// Penyimpanan ini tidak memodelkan SALAVAGEDOCUMENT: penyaring kueri aslinya
		// membandingkan kolom NOKLAIM dengan sebuah ID detail salvage, sehingga ia hampir
		// pasti tidak mencocokkan apa pun. Memodelkan tabel yang tidak pernah tersentuh
		// hanya menambah kepercayaan palsu pada uji.
		result.DocumentMarked = true
	}

	if command.ApplyPrice {
		for index := range w.store.checkers {
			record := &w.store.checkers[index]
			if record.DetailObject != command.DetailObject {
				continue
			}
			if strings.TrimSpace(record.SalvageID) != strings.TrimSpace(command.SalvageID) {
				continue
			}
			// Cerminan `DETAIL_PNC_SALVAGE.HARGAITEM`. Pada baris checker, harga barangnya
			// adalah ItemPrice — dan itulah yang ditimpa harga tandingan balai lelang.
			record.ItemPrice = command.RequestPrice
		}
		result.PriceApplied = true
	}

	return result, nil
}
