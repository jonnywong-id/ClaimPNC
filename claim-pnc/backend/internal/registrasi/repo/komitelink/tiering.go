// Package komitelink menyambungkan seam registrasi.CommitteeTiering ke modul Komite (`B-7`).
//
// Penjenjangan — kumulatif atas LIMIT_BOTTOM, pita Non-MBU (`D-70`), pengaju dikecualikan —
// hidup di modul komite dan tidak disalin ke registrasi. Paket ini hanya menerjemahkan
// tipe keduanya.
package komitelink

import (
	"context"
	"strings"

	"claim-pnc/internal/komite"
	komiteusecase "claim-pnc/internal/komite/usecase"
	"claim-pnc/internal/platform/money"
	"claim-pnc/internal/registrasi"
)

// Tiering membaca penyetuju dari modul komite.
type Tiering struct{ service *komiteusecase.Service }

// New membentuk Tiering di atas layanan komite.
func New(service *komiteusecase.Service) Tiering { return Tiering{service: service} }

// Route mengembalikan pita dan penyetuju berurutan jenjang untuk nilai (dalam sen) pada lini
// itu.
func (t Tiering) Route(ctx context.Context, line string, value registrasi.Money, applicant string) (registrasi.CommitteeRoute, error) {
	result, err := t.service.TieringWith(ctx, money.FromMinorUnits(int64(value)), komite.BusinessLine(line), applicant)
	if err != nil {
		return registrasi.CommitteeRoute{}, err
	}
	out := registrasi.CommitteeRoute{Band: result.Band, Approvers: make([]registrasi.CommitteeApprover, 0, len(result.Approvers))}
	for _, a := range result.Approvers {
		out.Approvers = append(out.Approvers, registrasi.CommitteeApprover{
			OperatorID: strings.TrimSpace(a.OperatorID),
			Name:       strings.TrimSpace(a.Name),
		})
	}
	return out, nil
}
