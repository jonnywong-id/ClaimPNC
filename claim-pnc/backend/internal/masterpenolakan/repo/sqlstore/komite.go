package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpenolakan"
)

// RepoKomite membaca dan menulis POOLDATA.MST_REJECTED_KOMITE.
//
// Ia repo tersendiri, bukan method tambahan pada Repo, karena tabelnya tidak sekerabat
// dengan kedua tabel Penolakan Klaim: kuncinya bertipe angka, dan tidak ada satu pun
// kolom yang menghubungkannya. Yang dipakai bersama hanyalah pemuat kueri di
// masterpenolakan.go — satu berkas embed untuk satu folder.
type RepoKomite struct {
	db *sql.DB
}

// NewRepoKomite membentuk repo; db wajib sudah terhubung ke basis data portal yang
// dimaksud.
func NewRepoKomite(db *sql.DB) *RepoKomite { return &RepoKomite{db: db} }

// List membaca seluruh penolakan komite.
func (r *RepoKomite) List(ctx context.Context) ([]masterpenolakan.CommitteeRejection, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("committee_rejection_list"))
	if err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: membaca daftar penolakan komite: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpenolakan.CommitteeRejection
	for rows.Next() {
		rejection, err := scanKomite(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rejection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: menelusuri penolakan komite: %w", err)
	}
	return result, nil
}

// Get membaca satu penolakan komite berdasarkan IDMASTER-nya.
func (r *RepoKomite) Get(ctx context.Context, id string) (masterpenolakan.CommitteeRejection, error) {
	rejection, err := scanKomite(r.db.QueryRowContext(ctx, getQuery("committee_rejection_get"), strings.TrimSpace(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return masterpenolakan.CommitteeRejection{}, masterpenolakan.ErrKomiteNotFound
	}
	if err != nil {
		return masterpenolakan.CommitteeRejection{}, fmt.Errorf("masterpenolakan/sqlstore: membaca penolakan komite %q: %w", id, err)
	}
	return rejection, nil
}

// InsertNew menurunkan IDMASTER dari isi tabel lalu menyisipkan barisnya.
//
// Keduanya berjalan di dalam SATU transaksi, dengan alasan yang sama seperti
// Repo.InsertNew: nomor baru diturunkan dari isi tabel itu sendiri, sehingga membaca dan
// menulisnya tidak dapat dipisahkan tanpa membuka kembali lubang balapan yang justru
// sedang ditutup.
func (r *RepoKomite) InsertNew(ctx context.Context, input masterpenolakan.InputKomite) (masterpenolakan.CommitteeRejection, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterpenolakan.CommitteeRejection{}, fmt.Errorf("masterpenolakan/sqlstore: memulai transaksi komite: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	used, err := lockedIDs(ctx, tx, "committee_rejection_list_id_locked")
	if err != nil {
		return masterpenolakan.CommitteeRejection{}, err
	}

	fresh := masterpenolakan.CommitteeRejection{
		ID:   nextKomiteID(used),
		Note: input.Note,
	}
	if _, err := tx.ExecContext(ctx, getQuery("committee_rejection_insert"), fresh.ID, fresh.Note); err != nil {
		return masterpenolakan.CommitteeRejection{}, fmt.Errorf("masterpenolakan/sqlstore: menyisipkan penolakan komite %q: %w", fresh.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return masterpenolakan.CommitteeRejection{}, fmt.Errorf("masterpenolakan/sqlstore: menutup transaksi sisip komite: %w", err)
	}
	return fresh, nil
}

// Update menyimpan perubahan catatan pada baris yang sudah ada.
//
// Barisnya TIDAK dibaca ulang lebih dulu seperti pada Repo.Update, karena di sini tidak
// ada kolom lain yang perlu dibawa — tabelnya hanya dua kolom, dan salah satunya kunci.
// Yang membedakan "tidak ditemukan" dari "berhasil tanpa perubahan" adalah RowsAffected.
func (r *RepoKomite) Update(ctx context.Context, id string, input masterpenolakan.InputKomite) (masterpenolakan.CommitteeRejection, error) {
	clean := strings.TrimSpace(id)

	result, err := r.db.ExecContext(ctx, getQuery("committee_rejection_update"), input.Note, clean)
	if err != nil {
		return masterpenolakan.CommitteeRejection{}, fmt.Errorf("masterpenolakan/sqlstore: memperbarui penolakan komite %q: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat melaporkan jumlah baris tidak boleh diartikan sebagai
		// kegagalan: pernyataannya sendiri sudah berhasil.
		return masterpenolakan.CommitteeRejection{ID: clean, Note: input.Note}, nil
	}
	if affected == 0 {
		return masterpenolakan.CommitteeRejection{}, masterpenolakan.ErrKomiteNotFound
	}
	return masterpenolakan.CommitteeRejection{ID: clean, Note: input.Note}, nil
}

// CheckTable memastikan tabelnya ada dan dapat dibaca akun aplikasi.
func (r *RepoKomite) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("committee_rejection_check_table"))
	if err != nil {
		return fmt.Errorf("masterpenolakan/sqlstore: POOLDATA.MST_REJECTED_KOMITE tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// nextKomiteID menyusun IDMASTER baru yang belum dipakai.
//
// Ia TIDAK memakai masterpenolakan.NextSequence begitu saja, karena tabel ini punya satu
// aturan yang tidak dimiliki tabel mana pun di modul ini: baris pertama pada tabel yang
// masih kosong bernomor 111, bukan 1.
//
//	Database/INSERTMASTERREJECTEDKOMITE.prc:6-12
//	    select count(1) into count_data from POOLDATA.MST_REJECTED_KOMITE;
//	    if count_data=0 then count_data:=111;
//	    else select max(IDMASTER)+1 into count_data from ...;
//
// Tidak ada keterangan apa pun tentang asal angka itu. Ia direplikasi karena `P-5`
// menuntut perilaku dipertahankan lebih dulu — dan karena keadaannya hanya terjadi sekali
// seumur tabel, mengubahnya tidak memberi manfaat apa pun yang sepadan dengan selisih
// yang akan ditimbulkannya.
func nextKomiteID(used []string) string {
	if len(used) == 0 {
		return masterpenolakan.FormatIDKomite(masterpenolakan.FirstIDKomite)
	}
	return masterpenolakan.NextSequence(used, masterpenolakan.FormatIDKomite)
}

// scanKomite membaca satu baris hasil kueri menjadi CommitteeRejection.
//
// IDMASTER bertipe NUMBER, tetapi tetap dibaca sebagai teks: driver Oracle memulangkan
// NUMBER tanpa skala sebagai string apa adanya, dan membacanya sebagai angka lalu
// memformatnya kembali hanya menambah satu tempat yang dapat salah format. Nilai yang
// ditampilkan dan nilai yang dikirim balik saat menyunting karena itu identik.
func scanKomite(p scanner) (masterpenolakan.CommitteeRejection, error) {
	var id, note sql.NullString
	if err := p.Scan(&id, &note); err != nil {
		return masterpenolakan.CommitteeRejection{}, err
	}
	return masterpenolakan.CommitteeRejection{
		ID:   strings.TrimSpace(id.String),
		Note: strings.TrimSpace(note.String),
	}, nil
}

var _ masterpenolakan.RepoKomite = (*RepoKomite)(nil)
