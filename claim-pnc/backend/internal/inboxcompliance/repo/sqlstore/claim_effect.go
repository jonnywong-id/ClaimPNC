package sqlstore

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// ApplyDecisionToClaim menulis akibat keputusan Compliance ke `POOLDATA.T_CLAIM_PNC`.
//
// # Kenapa TIDAK satu transaksi dengan SaveDecision
//
// Karena seam Repo modul ini belum memiliki kepemilikan transaksi di lapisan aplikasi
// (`08-TECHNICAL-STRATEGY.md` §4.5) — kekurangan yang sama dengan pasangan
// keputusan→Post Audit, dan dicatat di tempat yang sama.
//
// Urutannya disengaja: keputusan disimpan lebih dulu, klaim menyusul. Bila yang kedua
// gagal, yang tersimpan adalah keputusan tanpa klaim berpindah status — keadaan yang
// terlihat petugas dan dapat diulang karena form masih terbuka. Kebalikannya lebih buruk:
// klaim berpindah status tanpa jejak siapa yang memutuskan.
func (r *Repo) ApplyDecisionToClaim(
	ctx context.Context, effect inboxcompliance.ClaimEffect,
) error {
	if _, err := r.db.ExecContext(
		ctx, query("apply_decision_to_claim"),
		effect.StatusClaim,
		nullableTime(effect.ValidatedAt),
		nullableTime(effect.SentToPostAuditAt),
		effect.Reference,
	); err != nil {
		return fmt.Errorf(
			"menulis akibat keputusan Compliance ke POOLDATA.T_CLAIM_PNC "+
				"(STATUSCLAIM, CPLVALID_DATE, POSTAUDIT_TF_ANALYSTDATE): %w",
			storeMissing(err))
	}

	// Baris yang tidak tersentuh TIDAK dianggap galat, dan itu disengaja.
	//
	// Klaim yang ada di antrean tetapi belum punya baris di `T_CLAIM_PNC` mungkin saja
	// terjadi: tabel itu diisi procedure konversi yang berjalan terpisah dari Pega.
	// Menggagalkan penyimpanan karena itu akan membuang keputusan yang sudah sah hanya
	// karena konversinya belum mengejar.
	return nil
}
