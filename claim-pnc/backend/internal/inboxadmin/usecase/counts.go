package usecase

import (
	"context"
	"log/slog"
	"sync"

	"claim-pnc/internal/inboxadmin"
)

// TabCount adalah satu baris daftar "Status Register" — nama tab beserta jumlah barisnya.
type TabCount struct {
	Tab inboxadmin.Tab

	// Count adalah jumlah baris tab itu dengan penyaring lini bisnis yang dipakai.
	Count int

	// Failed menyatakan kueri tab itu gagal, sehingga Count tidak bermakna. Satu tab yang
	// gagal tidak menggagalkan daftar: layar menandai barisnya, bukan mengosongkan semuanya.
	Failed bool
}

// Counts menghitung jumlah baris SETIAP tab — daftar "Status Register" layar Pega, yang
// disusun activity `GetReportClaimRegistList`.
//
// # Kenapa dihitung dengan kueri isi tab, bukan kueri COUNT tersendiri
//
// Supaya angka di daftar dan banyaknya baris saat tab dibuka TIDAK PERNAH berbeda. Pega
// menghitungnya dengan kueri `CountClaimRegist*` yang terpisah dari kueri isinya, sehingga
// keduanya dapat berselisih tanpa ketahuan. Harganya: setiap kueri menarik seluruh baris,
// sama seperti saat tabnya dibuka (keputusan Work Owner 2026-09-20 tentang paginasi).
// Bila kelak terasa berat, kueri COUNT dapat ditambahkan di balik metode ini tanpa
// mengubah layarnya.
//
// Kata kunci pencarian TIDAK ikut: angka di daftar adalah ukuran antrean, bukan hasil
// pencarian. Penyaring lini bisnis ikut, karena di Pega ia penyaring tingkat layar.
func (s *Service) Counts(
	ctx context.Context,
	portalAlias string,
	caller inboxadmin.Caller,
	business string,
	region string,
) ([]TabCount, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	// Batas data dihitung SEKALI untuk kedelapan tab — ia milik pemanggil, bukan tab.
	clean := caller.Clean()
	if clean.Login == "" {
		return nil, inboxadmin.ErrCallerUnknown
	}
	scope, _, err := s.resolveScope(ctx, repo, clean, region)
	if err != nil {
		return nil, err
	}

	tabs := inboxadmin.Tabs()
	result := make([]TabCount, len(tabs))
	var wait sync.WaitGroup
	for i, tab := range tabs {
		result[i].Tab = tab
		query, err := inboxadmin.NewQuery(inboxadmin.QueryInput{Tab: tab.Code, Business: business}, caller)
		if err != nil {
			// Galat bentuk permintaan berlaku sama untuk seluruh tab — mis. lini bisnis
			// yang tidak dikenal — sehingga ia dijawab sebagai satu galat validasi.
			return nil, err
		}
		query.Scope = scope
		wait.Add(1)
		go func(i int, query inboxadmin.Query) {
			defer wait.Done()
			rows, err := repo.List(ctx, query)
			if err != nil {
				result[i].Failed = true
				if s.logger != nil {
					s.logger.Warn("menghitung isi tab Inbox Admin gagal",
						slog.String("tab", query.Tab.Code), slog.Any("galat", err))
				}
				return
			}
			result[i].Count = len(rows)
		}(i, query)
	}
	wait.Wait()
	return result, nil
}
