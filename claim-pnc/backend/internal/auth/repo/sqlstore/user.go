package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"claim-pnc/internal/auth"
)

// UserRepo memenuhi auth.UserRepo terhadap basis data relasional.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo membentuk repo; db wajib sudah terhubung.
func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

// GetByIdentity membaca satu catatan pengguna.
func (r *UserRepo) GetByIdentity(ctx context.Context, identity string) (auth.User, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("user_get_by_identity"), identity)

	var (
		p          auth.User
		kind       string
		login      sql.NullString
		email      sql.NullString
		perusahaan sql.NullString
		branch     sql.NullString
		branchCode sql.NullString
		jabatan    sql.NullString
		operatorID sql.NullString
		active     string
	)
	err := rows.Scan(&p.Identity, &kind, &p.Name, &login, &email, &perusahaan,
		&branch, &branchCode, &jabatan, &operatorID, &active, &p.CreatedAt, &p.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return auth.User{}, auth.ErrUserNotFound
	case err != nil:
		return auth.User{}, fmt.Errorf("sqlstore: membaca pengguna: %w", err)
	}

	p.Kind = auth.UserKind(kind)
	p.Login = login.String
	p.Email = email.String
	p.Company = perusahaan.String
	p.Branch = branch.String
	p.BranchCode = branchCode.String
	p.Position = jabatan.String
	p.OperatorID = operatorID.String
	p.Active = active == "Y"
	return p, nil
}

// Save menulis catatan pengguna berdasarkan Identity.
//
// Ditulis sebagai UPDATE lalu INSERT bila tidak ada baris yang terkena, bukan MERGE:
// MERGE pada Oracle menuntut FROM DUAL yang tidak ada di PostgreSQL, dan disiplin SQL
// portabel (D-20) lebih berharga daripada satu pernyataan yang lebih ringkas.
//
// Bila dua permintaan masuk bersamaan untuk identitas yang sama-sama baru, satu INSERT
// akan kalah pada kunci utama. Kekalahan itu ditangani dengan mencoba UPDATE sekali
// lagi, bukan diteruskan sebagai galat ke pengguna.
func (r *UserRepo) Save(ctx context.Context, p auth.User) error {
	terkena, err := r.update(ctx, p)
	if err != nil {
		return err
	}
	if terkena > 0 {
		return nil
	}

	active := "N"
	if p.Active {
		active = "Y"
	}
	_, err = r.db.ExecContext(ctx, getQuery("user_insert"),
		p.Identity, string(p.Kind), p.Name, emptyToNull(p.Login),
		emptyToNull(p.Email), emptyToNull(p.Company),
		emptyToNull(p.Branch), emptyToNull(p.BranchCode), emptyToNull(p.Position),
		emptyToNull(p.OperatorID), active,
		p.CreatedAt.UTC(), p.UpdatedAt.UTC())
	if err == nil {
		return nil
	}

	// Kalah lomba dengan permintaan lain yang menyisipkan identitas yang sama lebih dulu.
	if terkena, errUlang := r.update(ctx, p); errUlang == nil && terkena > 0 {
		return nil
	}
	return fmt.Errorf("sqlstore: menyisipkan pengguna: %w", err)
}

func (r *UserRepo) update(ctx context.Context, p auth.User) (int64, error) {
	result, err := r.db.ExecContext(ctx, getQuery("user_update"),
		string(p.Kind), p.Name, emptyToNull(p.Login), emptyToNull(p.Email),
		emptyToNull(p.Company), emptyToNull(p.Branch), emptyToNull(p.BranchCode), emptyToNull(p.Position),
		p.UpdatedAt.UTC(), p.Identity)
	if err != nil {
		return 0, fmt.Errorf("sqlstore: memperbarui pengguna: %w", err)
	}
	terkena, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("sqlstore: membaca jumlah baris terkena: %w", err)
	}
	return terkena, nil
}

// emptyToNull menuliskan NULL, bukan string kosong, untuk field opsional yang tidak
// terisi. Oracle memperlakukan string kosong sebagai NULL, tetapi PostgreSQL tidak —
// menegaskannya di sini menjaga perilaku keduanya tetap sama (D-20).
func emptyToNull(value string) any {
	if value == "" {
		return nil
	}
	return value
}
