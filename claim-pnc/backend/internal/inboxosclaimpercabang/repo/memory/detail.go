package memory

import (
	"context"
	"strings"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// FindDetail mengambil isi popup untuk satu klaim milik cabang pada query.
//
// # Penyaring cabangnya nyata, bukan formalitas
//
// Ia memanggil matching — penyaring yang sama dengan List — lalu mencari nomornya DI DALAM
// hasil itu. Mencarinya langsung di s.rows akan lulus uji apa pun yang hanya memeriksa isi
// popup, sementara kebocoran antarcabangnya lolos tanpa terlihat (`R-20`).
func (s *Store) FindDetail(
	_ context.Context,
	query inboxosclaimpercabang.Query,
	claimNumber string,
) (inboxosclaimpercabang.Detail, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	wanted := strings.TrimSpace(claimNumber)
	if wanted == "" {
		return inboxosclaimpercabang.Detail{}, false, nil
	}

	for _, r := range s.matching(query) {
		if r.item.ClaimNumber != wanted {
			continue
		}

		detail := inboxosclaimpercabang.Detail{
			ClaimNumber:          r.item.ClaimNumber,
			ClaimKey:             r.item.ClaimKey,
			BusinessName:         r.item.BusinessName,
			Occupation:           r.detail.Occupation,
			TotalSumInsured:      r.detail.TotalSumInsured,
			Chronology:           r.item.Chronology,
			EstimationValue:      r.item.EstimationValue,
			RegisterDate:         r.item.RegisterDate,
			RemarkRecommendation: r.item.RemarkRecommendation,
			ProgressNote:         r.item.ProgressNote,
			DominantFactors:      joinFactors(s.factors[r.item.ClaimKey]),
			Objects:              append([]inboxosclaimpercabang.DetailObject{}, r.detail.Objects...),
			ProgressHistory:      append([]inboxosclaimpercabang.DetailProgress{}, r.detail.ProgressHistory...),
			AdjusterMessages:     append([]inboxosclaimpercabang.DetailMessage{}, r.detail.AdjusterMessages...),
		}
		return detail, true, nil
	}

	return inboxosclaimpercabang.Detail{}, false, nil
}

// joinFactors merangkai nama faktor dominan, meniru bentuk kueri lama.
//
// `"-"` ketika kosong, bukan teks kosong: itu yang selama ini terbaca di layar.
func joinFactors(names []string) string {
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, ", ")
}
