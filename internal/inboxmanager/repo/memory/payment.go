package memory

import (
	"context"
	"strings"

	"claim-pnc/internal/inboxmanager"
)

// SetPaymentDetail menetapkan detail satu nomor akseptasi yang sedang menunggu.
//
// Baris antrean Payment Klaim Akseptasi-nya ditetapkan TERPISAH lewat SetQueue — sama seperti
// di basis data, daftar antrean dan panel detail dibaca kueri yang berbeda.
func (s *Store) SetPaymentDetail(detail inboxmanager.PaymentDetail) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments[strings.TrimSpace(detail.AcceptedNo)] = detail
}

// SetPaymentDocumentContent menetapkan isi lampiran lama sebuah nomor akseptasi.
func (s *Store) SetPaymentDocumentContent(acceptedNo, key string, content inboxmanager.PaymentDocumentContent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paymentFiles[paymentFileKey(acceptedNo, key)] = content
}

// PaymentSaves mengembalikan seluruh SIMPAN yang mengubah baris, berurutan — untuk uji.
func (s *Store) PaymentSaves() []inboxmanager.PaymentSave {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]inboxmanager.PaymentSave, len(s.paymentSaves))
	copy(result, s.paymentSaves)
	return result
}

func paymentFileKey(acceptedNo, key string) string {
	return strings.TrimSpace(acceptedNo) + "|" + strings.TrimSpace(key)
}

// PaymentDetail membaca detail satu nomor akseptasi yang sedang menunggu.
func (s *Store) PaymentDetail(_ context.Context, acceptedNo string) (inboxmanager.PaymentDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	detail, found := s.payments[strings.TrimSpace(acceptedNo)]
	if !found {
		return inboxmanager.PaymentDetail{}, inboxmanager.ErrPaymentNotFound
	}
	detail.History = append([]inboxmanager.PaymentHistoryRow{}, detail.History...)
	detail.Documents = append([]inboxmanager.PaymentDocument{}, detail.Documents...)
	return detail, nil
}

// PaymentDocumentContent membaca isi satu lampiran lama.
func (s *Store) PaymentDocumentContent(
	_ context.Context, acceptedNo, key string,
) (inboxmanager.PaymentDocumentContent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	content, found := s.paymentFiles[paymentFileKey(acceptedNo, key)]
	if !found || len(content.Content) == 0 {
		return inboxmanager.PaymentDocumentContent{}, inboxmanager.ErrPaymentDocumentNotFound
	}
	return content, nil
}

// SavePayment menuliskan SIMPAN.
//
// Seperti pernyataan SQL-nya, hanya nomor yang sedang menunggu yang berubah. Status "Sedang
// Proses Komite" membiarkan barisnya tetap di antrean (STSAPP tetap 0); status lain
// mengeluarkannya, dan detailnya ikut hilang karena detail hanya dibaca untuk nomor yang
// menunggu.
func (s *Store) SavePayment(_ context.Context, save inboxmanager.PaymentSave) (inboxmanager.PaymentSaveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	no := strings.TrimSpace(save.AcceptedNo)
	rows := s.queues[inboxmanager.TabPaymentAkseptasi]
	kept := make([]inboxmanager.QueueRow, 0, len(rows))
	changed := 0
	for _, row := range rows {
		if strings.TrimSpace(row.Key) == no {
			changed++
			if save.Status == inboxmanager.PaymentInCommittee {
				kept = append(kept, row)
			}
			continue
		}
		kept = append(kept, row)
	}
	if changed == 0 {
		return inboxmanager.PaymentSaveResult{}, nil
	}

	s.queues[inboxmanager.TabPaymentAkseptasi] = kept
	detail, found := s.payments[no]
	if save.Status != inboxmanager.PaymentInCommittee {
		delete(s.payments, no)
	}
	s.paymentSaves = append(s.paymentSaves, save)

	return inboxmanager.PaymentSaveResult{
		Changed:          changed,
		AdjustmentMarked: found && detail.AdjustmentFound,
	}, nil
}
