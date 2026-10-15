package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/dashboardclaim"
)

// GroupsOf membaca seluruh grup akses milik satu pengguna.
//
// `identity` adalah `Caller.User.Identity`, yang bagi pengguna `POOLDATA.M_LOGIN_PNC`
// sepadan dengan `LOGIN_ID` — lihat catatan di `otorisasi.sql`.
//
// Identitas kosong menjawab daftar kosong TANPA menyentuh basis data. Itu bukan penghematan
// melainkan pencegahan: `UPPER(TRIM(”))` cocok dengan baris mana pun yang `LOGIN_ID`-nya
// juga kosong, dan satu baris cacat di master akan memberi hak kepada setiap permintaan yang
// kehilangan identitasnya.
func (r *Repo) GroupsOf(ctx context.Context, identity string) ([]dashboardclaim.Grup, error) {
	if strings.TrimSpace(identity) == "" {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx, query("grup_pengguna"), identity)
	if err != nil {
		return nil, fmt.Errorf("dashboardclaim/sqlstore: membaca grup pengguna: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var keluar []dashboardclaim.Grup
	for rows.Next() {
		var grup string
		if err := rows.Scan(&grup); err != nil {
			return nil, fmt.Errorf("dashboardclaim/sqlstore: memindai grup pengguna: %w", err)
		}
		keluar = append(keluar, dashboardclaim.Grup(strings.TrimSpace(grup)))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dashboardclaim/sqlstore: menutup daftar grup: %w", err)
	}
	return keluar, nil
}

// CheckGroupsReadable membuktikan tabel grup terbaca akun aplikasi.
//
// Dipanggil `claimpnc -periksa`. Tanpa probe ini, tabel yang tidak terbaca akan membuat
// SETIAP pengguna jatuh ke baca-saja — gejala yang tidak dapat dibedakan dari "memang tidak
// berhak", dan karena itu tidak akan pernah dilaporkan sebagai cacat.
func (r *Repo) CheckGroupsReadable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("grup_pengguna_check")); err != nil {
		return fmt.Errorf(
			"grup akses tidak terbaca akun aplikasi — periksa "+
				"POOLDATA.M_LOGIN_GROUP_PNC(LOGIN_ID, GROUP_ID): %w", err)
	}
	return nil
}
