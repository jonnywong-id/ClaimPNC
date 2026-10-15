package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxxol"
)

// InsertDolCol menyimpan isian modal "INSERT DOL DAN COL" dan mengembalikan JUMLAH BARIS
// yang tersimpan — satu per group business perjanjian, bukan satu per simpan.
//
// # Urutan pemeriksaannya, dan kenapa begitu
//
//  1. Identitas pemanggil. Tanpa itu tidak ada yang dapat dicatat sebagai pelaku, dan
//     `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang yang tersisa.
//  2. Isian. SELURUH pelanggaran dikembalikan sekaligus (`P-5`).
//  3. Perjanjian. Ia harus ADA — bukan sekadar terisi — karena kode group business-nyalah
//     yang menentukan baris mana yang ditulis.
//  4. Group business. Perjanjian tanpa satu pun group business menghasilkan NOL baris.
//     Sistem lama melakukannya tanpa pesan: tombolnya terlihat berhasil, gridnya tetap
//     kosong, dan tidak ada apa pun yang dapat ditelusuri. Di sini ia ditolak.
//
// Butir 4 adalah satu-satunya tempat perilakunya berbeda dari sistem lama, dan bedanya ke
// arah yang terbaca: penolakan menyebut sebabnya, sedangkan keberhasilan palsu tidak.
func (s *Service) InsertDolCol(
	ctx context.Context,
	portalAlias string,
	caller inboxxol.Caller,
	request inboxxol.DolColRequest,
) (int, error) {
	if strings.TrimSpace(caller.Login) == "" {
		return 0, inboxxol.ErrCallerUnknown
	}

	clean := request.Clean()
	if err := clean.Validate(); err != nil {
		return 0, err
	}

	repo, err := s.repoFor(portalAlias)
	if err != nil {
		return 0, err
	}

	master, err := findMaster(ctx, repo, clean.MasterID)
	if err != nil {
		return 0, err
	}

	rows := inboxxol.NewDolColRows(clean, master.BusinessGroupIDs())
	if len(rows) == 0 {
		return 0, inboxxol.NewValidationError([]inboxxol.Violation{{
			Field: inboxxol.FieldMasterID,
			Message: "Perjanjian XOL ini belum punya satu pun Group Business, " +
				"sehingga tidak ada baris yang dapat ditulis. Lengkapi dulu Group Business-nya.",
		}})
	}

	if err := repo.InsertDolCol(ctx, rows); err != nil {
		return 0, fmt.Errorf("inboxxol/usecase: insert DOL dan COL: %w", err)
	}
	return len(rows), nil
}
