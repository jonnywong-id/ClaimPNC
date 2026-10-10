package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// SettlementHistory adalah dua grid di bawah rincian satu baris Adjustment pada
// Section/InputAdjustment_sect.xml: "Status Penerimaan Komite" (.ComiteeClaim) dan "Histori
// Transfer Kasir" (TempDataLogKasir).
type SettlementHistory struct {
	// Committee adalah anggota kasus komite baris itu, urut jenjang; kosong bila baris belum
	// ditransfer ke komite.
	Committee []registrasi.CommitteeMember
	// Cashier adalah riwayat POOLDATA.TRF_KASIR_LOG nomor akseptasi baris itu; kosong bila
	// baris belum diakseptasi.
	Cashier []registrasi.CashierHistoryEntry
}

// SettlementHistory membaca kedua riwayat satu baris Adjustment. Baris dipilih dari klaim
// itu sendiri — nomor kasus komite dan nomor akseptasinya tidak pernah diterima dari
// pemanggil, sehingga endpoint ini tidak dapat dipakai membaca riwayat klaim lain.
//
// Grid komite di XML hanya tampil untuk Bonding Interim (`IsBondingInterim && IsBonding`);
// Work Owner 2026-10-07 menetapkan ia tampil di SEMUA lini. Kolomnya tetap mengikuti XML.
func (l *Service) SettlementHistory(ctx context.Context, claimID string, object, coverage, adjustment int) (SettlementHistory, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return SettlementHistory{}, err
	}
	if object < 1 || object > len(claim.InsuredItem) ||
		coverage < 1 || coverage > len(claim.InsuredItem[object-1].Coverage) {
		return SettlementHistory{}, fmt.Errorf("%w: objek %d jaminan %d tidak ada", registrasi.ErrInvalidAction, object, coverage)
	}
	lines := claim.InsuredItem[object-1].Coverage[coverage-1].Settlement
	if adjustment < 1 || adjustment > len(lines) {
		return SettlementHistory{}, fmt.Errorf("%w: adjustment %d tidak ada", registrasi.ErrInvalidAction, adjustment)
	}
	line := lines[adjustment-1]

	var out SettlementHistory
	if id := strings.TrimSpace(line.CommitteeCaseID); id != "" {
		c, err := l.committees.Get(ctx, id)
		switch {
		case errors.Is(err, registrasi.ErrCommitteeNotFound):
			// Baris lama dapat menunjuk kasus komite Pega yang tidak punya kepala di
			// TC_PNC_KOMITE; gridnya kosong, bukan galat.
		case err != nil:
			return SettlementHistory{}, err
		default:
			out.Committee = c.Members
		}
	}
	if no := strings.TrimSpace(line.AcceptedNo); no != "" {
		out.Cashier, err = l.cashier.CashierHistory(ctx, no)
		if err != nil {
			return SettlementHistory{}, err
		}
	}
	return out, nil
}
