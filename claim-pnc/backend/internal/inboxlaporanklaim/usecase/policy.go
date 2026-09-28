package usecase

import (
	"context"
	"strings"

	"claim-pnc/internal/inboxlaporanklaim"
)

// PolicyLookup adalah jawaban pencarian polis dari form Input Receive Document.
type PolicyLookup struct {
	// Number adalah nomor yang sudah dirapikan — layar menggantikan isiannya dengan
	// nilai ini, seperti langkah 12 activity lama.
	Number string

	Found  bool
	Policy inboxlaporanklaim.Policy
	Notice []inboxlaporanklaim.PolicyNotice
}

// LookupPolicy menjalankan pengganti `PolisReceiveInternalExternal`: merapikan nomor
// polis, membaca datanya, lalu menurunkan pesan Syariah dan bukan-PNC.
//
// # Yang TIDAK ditiru
//
// Langkah 3 sampai 6 activity lama mengonversi polis yang belum ada di JSON_POLIS
// dengan memanggil dua stored procedure, `GLADMIN.updatepolisjson` dan
// `POOLDATA.INSERTPOLISTOJSON`. Keduanya tidak dipanggil: `D-02` melarang aplikasi
// memanggil stored procedure, dan JSON_POLIS milik sistem polis (`D-03`, `D-04`).
// Akibatnya, polis yang ada di T_GENERAL tetapi belum ada di JSON_POLIS tetap mengisi
// form, tetapi Register Klaim menolaknya dengan "Nomor Polis tidak ditemukan".
func (s *Service) LookupPolicy(
	ctx context.Context,
	portalAlias, number string,
) (PolicyLookup, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return PolicyLookup{}, err
	}

	clean := inboxlaporanklaim.NormalizePolicyNumber(number)
	if clean == "" {
		return PolicyLookup{Number: clean}, nil
	}

	policy, found, err := repo.FindPolicy(ctx, clean)
	if err != nil {
		return PolicyLookup{}, err
	}
	return PolicyLookup{
		Number: clean,
		Found:  found,
		Policy: policy,
		Notice: inboxlaporanklaim.Notices(policy, found),
	}, nil
}

// checkPolicy menolak penyimpanan berkas untuk polis Syariah atau bukan PNC.
//
// Di layar lama penolakan itu hanya berupa tombol yang dimatikan. Di sini ia juga
// ditegakkan server, supaya permintaan yang tidak lewat layar pun tertolak.
// Polis yang tidak ditemukan TIDAK ditolak — lihat inboxlaporanklaim.Notices.
func checkPolicy(ctx context.Context, repo inboxlaporanklaim.Repo, number string) error {
	clean := inboxlaporanklaim.NormalizePolicyNumber(number)
	if clean == "" {
		return nil
	}
	policy, found, err := repo.FindPolicy(ctx, clean)
	if err != nil {
		return err
	}
	var violation []inboxlaporanklaim.Violation
	for _, n := range inboxlaporanklaim.Notices(policy, found) {
		if n.Blocking {
			violation = append(violation, inboxlaporanklaim.Violation{
				Field:   "nomor_polis",
				Message: strings.TrimSpace(n.Message),
			})
		}
	}
	if len(violation) > 0 {
		return &inboxlaporanklaim.ValidationError{Violation: violation}
	}
	return nil
}
