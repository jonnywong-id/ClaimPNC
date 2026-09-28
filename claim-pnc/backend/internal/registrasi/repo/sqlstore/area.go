package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// AreaDirectory membaca master wilayah untuk daftar pilihan bertingkat.
type AreaDirectory struct {
	db *sql.DB
}

// NewAreaDirectory membentuk pembaca master wilayah.
func NewAreaDirectory(db *sql.DB) *AreaDirectory { return &AreaDirectory{db: db} }

var areaQuery = map[registrasi.AreaLevel]string{
	registrasi.AreaCountry:  "wilayah_negara",
	registrasi.AreaProvince: "wilayah_provinsi",
	registrasi.AreaCity:     "wilayah_kota",
	registrasi.AreaDistrict: "wilayah_kabupaten",
	registrasi.AreaVillage:  "wilayah_kelurahan",
}

// Options membaca pilihan satu tingkat. Tingkat di bawah negara tanpa induk menghasilkan
// daftar kosong, bukan seluruh isi master: petugas memilih dari atas ke bawah.
func (d *AreaDirectory) Options(ctx context.Context, level registrasi.AreaLevel, parent string) ([]registrasi.AreaOption, error) {
	name, ok := areaQuery[level]
	if !ok {
		return nil, fmt.Errorf("%w: tingkat wilayah %q", registrasi.ErrUnknownAreaLevel, level)
	}
	parent = strings.TrimSpace(parent)

	var args []any
	if level != registrasi.AreaCountry {
		if parent == "" {
			return []registrasi.AreaOption{}, nil
		}
		args = append(args, parent)
	}

	rows, err := executorFrom(ctx, d.db).QueryContext(ctx, loadQuery(name), args...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca wilayah %s: %w", level, err)
	}
	defer func() { _ = rows.Close() }()

	out := []registrasi.AreaOption{}
	for rows.Next() {
		var id, nama, pos sql.NullString
		if err := rows.Scan(&id, &nama, &pos); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris wilayah %s: %w", level, err)
		}
		if strings.TrimSpace(nama.String) == "" {
			continue
		}
		out = append(out, registrasi.AreaOption{
			ID:         strings.TrimSpace(id.String),
			Name:       strings.TrimSpace(nama.String),
			PostalCode: strings.TrimSpace(pos.String),
		})
	}
	return out, rows.Err()
}

var _ registrasi.AreaDirectory = (*AreaDirectory)(nil)
