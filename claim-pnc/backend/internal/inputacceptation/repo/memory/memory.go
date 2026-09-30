// Package memory memenuhi seam inputacceptation.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji berjalan cepat dan tidak menuntut
//     Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa isinya disimpan sebagai Detail jadi, bukan sebagai dokumen JSON
//
// Karena yang diuji di sini adalah perilaku MODUL — penyaring awalan nomor klaim, pembedaan
// "tidak ditemukan" dari "ditemukan tetapi kosong", dan penolakan tulis — bukan pembacaan
// dokumen. Penguraian dokumen diuji tersendiri di repo/sqlstore, terhadap dokumen contoh.
//
// Menyimpan dokumen JSON di sini akan menguji pengurai yang sama dua kali dan tidak menguji
// satu pun hal yang khas penyimpanan memori.
package memory

import (
	"context"
	"strings"

	"claim-pnc/internal/inputacceptation"
)

// Store adalah penyimpanan akseptasi di memori.
//
// Ia tidak dilindungi mutex karena tidak pernah berubah setelah dibentuk: satu-satunya operasi
// tulis di seam ini MENOLAK, sehingga tidak ada jalur yang mengubah isinya.
type Store struct {
	byClaimID map[string]inputacceptation.Detail
}

// NewStore membentuk penyimpanan berisi rincian yang diberikan.
func NewStore(details ...inputacceptation.Detail) *Store {
	byClaimID := make(map[string]inputacceptation.Detail, len(details))
	for _, detail := range details {
		byClaimID[strings.ToUpper(detail.ClaimID)] = detail
	}
	return &Store{byClaimID: byClaimID}
}

// NewSampleStore membentuk penyimpanan berisi contoh bawaan.
func NewSampleStore() *Store {
	return NewStore(SampleDetails()...)
}

// Find mengembalikan satu rincian akseptasi.
//
// Nomor klaim dicocokkan tanpa memandang huruf besar-kecil, meniru Oracle yang di basis data
// pengembangan menyimpan `PYID` dalam huruf besar sementara alamat layar dapat diketik
// bagaimana pun.
func (s *Store) Find(
	_ context.Context, q inputacceptation.Query,
) (inputacceptation.Detail, error) {
	detail, found := s.byClaimID[strings.ToUpper(strings.TrimSpace(q.ClaimID))]
	if !found {
		return inputacceptation.Detail{}, inputacceptation.ErrNotFound
	}
	return detail, nil
}

// Save menolak dengan alasan yang SAMA dengan penyimpanan SQL.
//
// Kesamaan itu disengaja. Kalau penyimpanan memori menerima Submit sementara penyimpanan SQL
// menolaknya, layar akan berperilaku berbeda di pengembangan dan di staging — dan yang
// pertama akan membuat orang mengira fiturnya sudah selesai.
//
// Penghalangnya bukan penyimpanannya melainkan kepemilikan tabel (`P-1`), dan itu berlaku sama
// di mana pun kodenya dijalankan.
func (s *Store) Save(_ context.Context, _ inputacceptation.SaveCommand) error {
	return inputacceptation.ErrWriteNotOwned
}
