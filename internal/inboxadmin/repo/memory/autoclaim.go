package memory

import (
	"context"
	"time"

	"claim-pnc/internal/inboxadmin"
)

var _ inboxadmin.AutoClaimRepo = (*Store)(nil)

// AutoClaimResults mengembalikan daftar kosong.
//
// Data contoh tidak memuat batch Auto Claim: tabelnya hasil proses batch kredit, bukan
// antrean kerja, dan menirunya di memori tidak menguji apa pun selain tiruannya sendiri.
// Tombolnya tetap dapat dicoba — berkasnya berisi baris judul saja.
func (s *Store) AutoClaimResults(context.Context, string, time.Time, time.Time) ([]inboxadmin.AutoClaimResult, error) {
	return []inboxadmin.AutoClaimResult{}, nil
}

// AutoClaimFailures mengembalikan daftar kosong, dengan alasan yang sama.
func (s *Store) AutoClaimFailures(context.Context) ([]inboxadmin.AutoClaimFailure, error) {
	return []inboxadmin.AutoClaimFailure{}, nil
}
