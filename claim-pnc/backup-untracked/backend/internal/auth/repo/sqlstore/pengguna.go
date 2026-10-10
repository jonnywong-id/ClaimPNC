package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"claim-pnc/internal/auth"
)

// PenggunaRepo memenuhi auth.PenggunaRepo terhadap basis data relasional.
type PenggunaRepo struct {
	db *sql.DB
}

// PenggunaRepoBaru membentuk repo; db wajib sudah terhubung.
func PenggunaRepoBaru(db *sql.DB) *PenggunaRepo { return &PenggunaRepo{db: db} }

// AmbilByIdentitas membaca satu catatan pengguna.
func (r *PenggunaRepo) AmbilByIdentitas(ctx context.Context, identitas string) (auth.Pengguna, error) {
	baris := r.db.QueryRowContext(ctx, ambilKueri("pengguna_ambil_by_identitas"), identitas)

	var (
		p          auth.Pengguna
		jenis      string
		login      sql.NullString
		email      sql.NullString
		perusahaan sql.NullString
		cabang     sql.NullString
		kodeCabang sql.NullString
		jabatan    sql.NullString
		operatorID sql.NullString
		aktif      string
	)
	err := baris.Scan(&p.Identitas, &jenis, &p.Nama, &login, &email, &perusahaan,
		&cabang, &kodeCabang, &jabatan, &operatorID, &aktif, &p.DibuatPada, &p.DiperbaruiPada)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return auth.Pengguna{}, auth.ErrPenggunaTidakDitemukan
	case err != nil:
		return auth.Pengguna{}, fmt.Errorf("sqlstore: membaca pengguna: %w", err)
	}

	p.Jenis = auth.JenisPengguna(jenis)
	p.Login = login.String
	p.Email = email.String
	p.Perusahaan = perusahaan.String
	p.Cabang = cabang.String
	p.KodeCabang = kodeCabang.String
	p.Jabatan = jabatan.String
	p.OperatorID = operatorID.String
	p.Aktif = aktif == "Y"
	return p, nil
}

// SimpanAtauPerbarui menulis catatan pengguna berdasarkan Identitas.
//
// Ditulis sebagai UPDATE lalu INSERT bila tidak ada baris yang terkena, bukan MERGE:
// MERGE pada Oracle menuntut FROM DUAL yang tidak ada di PostgreSQL, dan disiplin SQL
// portabel (D-20) lebih berharga daripada satu pernyataan yang lebih ringkas.
//
// Bila dua permintaan masuk bersamaan untuk identitas yang sama-sama baru, satu INSERT
// akan kalah pada kunci utama. Kekalahan itu ditangani dengan mencoba UPDATE sekali
// lagi, bukan diteruskan sebagai galat ke pengguna.
func (r *PenggunaRepo) SimpanAtauPerbarui(ctx context.Context, p auth.Pengguna) error {
	terkena, err := r.perbarui(ctx, p)
	if err != nil {
		return err
	}
	if terkena > 0 {
		return nil
	}

	aktif := "N"
	if p.Aktif {
		aktif = "Y"
	}
	_, err = r.db.ExecContext(ctx, ambilKueri("pengguna_sisip"),
		p.Identitas, string(p.Jenis), p.Nama, kosongJadiNull(p.Login),
		kosongJadiNull(p.Email), kosongJadiNull(p.Perusahaan),
		kosongJadiNull(p.Cabang), kosongJadiNull(p.KodeCabang), kosongJadiNull(p.Jabatan),
		kosongJadiNull(p.OperatorID), aktif,
		p.DibuatPada.UTC(), p.DiperbaruiPada.UTC())
	if err == nil {
		return nil
	}

	// Kalah lomba dengan permintaan lain yang menyisipkan identitas yang sama lebih dulu.
	if terkena, errUlang := r.perbarui(ctx, p); errUlang == nil && terkena > 0 {
		return nil
	}
	return fmt.Errorf("sqlstore: menyisipkan pengguna: %w", err)
}

func (r *PenggunaRepo) perbarui(ctx context.Context, p auth.Pengguna) (int64, error) {
	hasil, err := r.db.ExecContext(ctx, ambilKueri("pengguna_perbarui"),
		string(p.Jenis), p.Nama, kosongJadiNull(p.Login), kosongJadiNull(p.Email),
		kosongJadiNull(p.Perusahaan), kosongJadiNull(p.Cabang), kosongJadiNull(p.KodeCabang), kosongJadiNull(p.Jabatan),
		p.DiperbaruiPada.UTC(), p.Identitas)
	if err != nil {
		return 0, fmt.Errorf("sqlstore: memperbarui pengguna: %w", err)
	}
	terkena, err := hasil.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("sqlstore: membaca jumlah baris terkena: %w", err)
	}
	return terkena, nil
}

// kosongJadiNull menuliskan NULL, bukan string kosong, untuk field opsional yang tidak
// terisi. Oracle memperlakukan string kosong sebagai NULL, tetapi PostgreSQL tidak —
// menegaskannya di sini menjaga perilaku keduanya tetap sama (D-20).
func kosongJadiNull(nilai string) any {
	if nilai == "" {
		return nil
	}
	return nilai
}
