package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"claim-pnc/internal/inboxadmin"
)

// Repo juga memenuhi inboxadmin.AutoClaimRepo — dijaga di sini supaya hilangnya satu
// metode ketahuan saat kompilasi, bukan saat tombol ekspor ditekan.
var _ inboxadmin.AutoClaimRepo = (*Repo)(nil)

// AutoClaimResults membaca baris batch Auto Claim yang berhasil.
func (r *Repo) AutoClaimResults(
	ctx context.Context, login string, from, to time.Time,
) ([]inboxadmin.AutoClaimResult, error) {
	rows, err := r.db.QueryContext(ctx, query("auto_claim_results"), from, to, login)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri auto_claim_results: %w", err)
	}
	defer rows.Close()

	result := []inboxadmin.AutoClaimResult{}
	for rows.Next() {
		var (
			initial, policy, claimID, acceptance, object, message sql.NullString
			value                                                 any
		)
		if err := rows.Scan(&initial, &policy, &claimID, &acceptance, &value, &object, &message); err != nil {
			return nil, fmt.Errorf("membaca baris kueri auto_claim_results: %w", err)
		}
		result = append(result, inboxadmin.AutoClaimResult{
			Initial: initial.String, PolicyNumber: policy.String, ClaimID: claimID.String,
			AcceptanceNumber: acceptance.String, ClaimValue: numberText(value),
			ObjectNumber: object.String, Message: message.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri auto_claim_results: %w", err)
	}
	return result, nil
}

// AutoClaimFailures membaca seluruh baris batch Auto Claim yang gagal.
func (r *Repo) AutoClaimFailures(ctx context.Context) ([]inboxadmin.AutoClaimFailure, error) {
	rows, err := r.db.QueryContext(ctx, query("auto_claim_failures"))
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri auto_claim_failures: %w", err)
	}
	defer rows.Close()

	result := []inboxadmin.AutoClaimFailure{}
	for rows.Next() {
		var (
			policy, insurance, claimType, currency, agent, message sql.NullString
			value                                                  any
		)
		if err := rows.Scan(&policy, &insurance, &value, &claimType, &currency, &agent, &message); err != nil {
			return nil, fmt.Errorf("membaca baris kueri auto_claim_failures: %w", err)
		}
		result = append(result, inboxadmin.AutoClaimFailure{
			PolicyNumber: policy.String, InsuranceNumber: insurance.String,
			ClaimValue: numberText(value), ClaimType: claimType.String,
			Currency: currency.String, AgentID: agent.String, Message: message.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri auto_claim_failures: %w", err)
	}
	return result, nil
}

// numberText menuliskan kolom NUMBER tanpa notasi ilmiah.
//
// Memindai NUMBER langsung ke string lewat database/sql memakai format `%g`, sehingga
// nilai klaim Rp 15.000.000 tertulis `1.5e+07` di berkas CSV — angka yang tidak dapat
// dibaca Excel sebagai nilai uang.
func numberText(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []byte:
		return string(v)
	case string:
		return v
	}
	return fmt.Sprint(value)
}
