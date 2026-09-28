package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/registrasi"
)

// saveSettlement menuliskan Settlement Line satu coverage ke T_CLAIM_ADJUSTMENT.
func (r *ClaimStore) saveSettlement(
	ctx context.Context,
	exec executor,
	claimID, objectID string,
	coverageSeq int,
	lines []registrasi.SettlementLine,
) error {
	coverageID := strconv.Itoa(coverageSeq)
	for n, s := range lines {
		adjustmentID := strconv.Itoa(n + 1)
		exGratia := "0"
		if s.ExGratia {
			exGratia = "1"
		}
		values := []any{
			s.PaymentType, emptyTextAsNil(s.Currency), int64(s.Rate), int64(s.Propose),
			int64(s.LOC), int64(s.SalvageA), emptyTextAsNil(s.RiskType), int64(s.RiskPercent),
			int64(s.RiskValue), int64(s.Gross), int64(s.ShareASM), int64(s.Value),
			int64(s.Accepted), int64(s.Submitted), exGratia, emptyTextAsNil(s.Chronology),
			emptyTextAsNil(s.Notes), emptyTextAsNil(s.AcceptanceStatus), emptyTextAsNil(s.AcceptedNo),
			emptyTextAsNil(strings.TrimSpace(s.CommitteeCaseID)), timeOrNil(s.CommitteeTransferredAt),
			timeOrNil(s.CommitteeDecidedAt),
		}
		keys := []any{claimID, objectID, coverageID, adjustmentID}
		if err := upsert(ctx, exec,
			"adjustment_perbarui", append(append([]any{}, values...), keys...),
			"adjustment_sisip", append(append([]any{}, values...), keys...),
		); err != nil {
			return fmt.Errorf("adjustment %s: %w", adjustmentID, err)
		}
	}
	return nil
}

// loadSettlement membaca Settlement Line seluruh coverage klaim.
func loadSettlement(
	ctx context.Context,
	exec executor,
	claimID string,
	coverageAt func(objectID, coverageID string) *registrasi.Coverage,
) error {
	rows, err := exec.QueryContext(ctx, loadQuery("adjustment_daftar"), claimID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca adjustment: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			objectID, coverageID, adjustmentID, paymentType, currency, riskType sql.NullString
			exGratia, chronology, notes, status, acceptedNo, committeeCase      sql.NullString
			rate, propose, loc, salvageA, riskPercent, riskValue                sql.NullInt64
			gross, share, value, accepted, submitted                            sql.NullInt64
			transferredAt, decidedAt                                            sql.NullTime
		)
		if err := rows.Scan(&objectID, &coverageID, &adjustmentID, &paymentType, &currency,
			&rate, &propose, &loc, &salvageA, &riskType, &riskPercent, &riskValue, &gross,
			&share, &value, &accepted, &submitted, &exGratia, &chronology, &notes, &status,
			&acceptedNo, &committeeCase, &transferredAt, &decidedAt); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris adjustment: %w", err)
		}
		c := coverageAt(strings.TrimSpace(objectID.String), strings.TrimSpace(coverageID.String))
		if c == nil {
			continue
		}
		c.Settlement = append(c.Settlement, registrasi.SettlementLine{
			PaymentType:      strings.TrimSpace(paymentType.String),
			Currency:         strings.TrimSpace(currency.String),
			Rate:             registrasi.ExchangeRate(rate.Int64),
			Propose:          registrasi.Money(propose.Int64),
			LOC:              registrasi.Percent(loc.Int64),
			SalvageA:         registrasi.Money(salvageA.Int64),
			RiskType:         strings.TrimSpace(riskType.String),
			RiskPercent:      registrasi.Percent(riskPercent.Int64),
			RiskValue:        registrasi.Money(riskValue.Int64),
			Gross:            registrasi.Money(gross.Int64),
			ShareASM:         registrasi.Percent(share.Int64),
			Value:            registrasi.Money(value.Int64),
			Accepted:         registrasi.Money(accepted.Int64),
			Submitted:        registrasi.Money(submitted.Int64),
			ExGratia:         strings.TrimSpace(exGratia.String) == "1",
			Chronology:       chronology.String,
			Notes:            notes.String,
			AcceptanceStatus: strings.TrimSpace(status.String),
			AcceptedNo:       strings.TrimSpace(acceptedNo.String),

			CommitteeCaseID:        strings.TrimSpace(committeeCase.String),
			CommitteeTransferredAt: transferredAt.Time,
			CommitteeDecidedAt:     decidedAt.Time,
		})
	}
	return rows.Err()
}
