package memory

import (
	"context"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/money"
)

// SummaryRows mengembalikan baris sempit seluruh klaim outstanding satu cabang.
//
// Ia memanggil matching — penyaring yang sama dengan List — bukan menelusuri s.rows sendiri.
// Itu yang menjamin kartu angka panel selalu cocok dengan jumlah baris grid; penyaring kedua
// yang ditulis terpisah akan berselisih diam-diam begitu salah satunya disunting.
func (s *Store) SummaryRows(
	_ context.Context,
	query inboxosclaimpercabang.Query,
) ([]inboxosclaimpercabang.SummaryRow, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := []inboxosclaimpercabang.SummaryRow{}
	for _, r := range s.matching(query) {
		result = append(result, inboxosclaimpercabang.SummaryRow{
			RegisterDate:    r.item.RegisterDate,
			BusinessName:    r.item.BusinessName,
			BusinessSource:  r.item.BusinessSource,
			EstimationValue: r.item.EstimationValue,
		})
	}

	return result, nil
}

// TreatyOR mengembalikan total porsi treaty OR satu cabang.
//
// Pengisi memori menjumlahkan kolom treaty OR pada baris contohnya. Nilai kedua selalu true:
// tidak ada DB Link yang dapat padam di sini, dan memalsukan kegagalannya akan membuat uji
// lulus atas keadaan yang tidak pernah terjadi pada pengisi ini.
func (s *Store) TreatyOR(
	_ context.Context,
	query inboxosclaimpercabang.Query,
) (money.Money, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var total money.Money
	for _, r := range s.matching(query) {
		total += r.item.TreatyShares.OR
	}

	return total, true, nil
}
