package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// DiagnosisDirectory membaca master diagnosa ASMD (sm.m_diagnosis).
type DiagnosisDirectory struct {
	db *sql.DB
}

// NewDiagnosisDirectory membentuk pembaca master diagnosa.
func NewDiagnosisDirectory(db *sql.DB) *DiagnosisDirectory { return &DiagnosisDirectory{db: db} }

// SearchDiagnosis mencari diagnosa menurut kode persis atau sebagian deskripsi.
func (d *DiagnosisDirectory) SearchDiagnosis(ctx context.Context, text string) ([]registrasi.DiagnosisOption, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	escaped := strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToUpper(text))
	rows, err := d.db.QueryContext(ctx, loadQuery("diagnosa_cari"), text, "%"+escaped+"%")
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: mencari diagnosa: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.DiagnosisOption
	for rows.Next() {
		var code, desc sql.NullString
		if err := rows.Scan(&code, &desc); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris diagnosa: %w", err)
		}
		result = append(result, registrasi.DiagnosisOption{
			Code: strings.TrimSpace(code.String), Description: strings.TrimSpace(desc.String),
		})
	}
	return result, rows.Err()
}

var _ registrasi.DiagnosisDirectory = (*DiagnosisDirectory)(nil)
