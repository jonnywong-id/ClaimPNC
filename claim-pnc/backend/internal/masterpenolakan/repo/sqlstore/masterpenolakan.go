// Package sqlstore memenuhi seam masterpenolakan.Repo dan masterpenolakan.RepoKomite
// dengan SQL.
//
// Satu instans repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpenolakan"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// Repo membaca dan menulis POOLDATA.MST_PENOLAKAN_KLAIM_1 dan _2.
//
// Kedua tabel dilayani satu repo karena penambahannya satu operasi yang tidak dapat
// dipecah; alasan lengkapnya ada pada doc comment masterpenolakan.Repo.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// ListParent membaca seluruh Status Penolakan 1.
func (r *Repo) ListParent(ctx context.Context) ([]masterpenolakan.RejectionStatus, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("rejection_parent_list"))
	if err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: membaca daftar status penolakan 1: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpenolakan.RejectionStatus
	for rows.Next() {
		parent, err := scanParent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, parent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: menelusuri status penolakan 1: %w", err)
	}
	return result, nil
}

// List membaca seluruh Status Penolakan 2.
func (r *Repo) List(ctx context.Context) ([]masterpenolakan.RejectionStatus2, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("rejection_list"))
	if err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpenolakan.RejectionStatus2
	for rows.Next() {
		rejection, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rejection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu Status Penolakan 2 berdasarkan ID_ND-nya.
func (r *Repo) Get(ctx context.Context, id string) (masterpenolakan.RejectionStatus2, error) {
	rejection, err := scanRow(r.db.QueryRowContext(ctx, getQuery("rejection_get"), strings.TrimSpace(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return masterpenolakan.RejectionStatus2{}, masterpenolakan.ErrNotFound
	}
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: membaca %q: %w", id, err)
	}
	return rejection, nil
}

// InsertNew menyimpan satu Status Penolakan 2 baru beserta induknya bila diminta.
//
// Seluruhnya berjalan di dalam SATU transaksi — pengecualian yang disadari terhadap
// §4.5 `08-TECHNICAL-STRATEGY.md` yang menempatkan batas transaksi di lapisan aplikasi.
// Dua alasan yang tidak dapat dipisahkan: nomor baru diturunkan dari isi tabel itu
// sendiri, dan induk yang baru harus lahir bersama anaknya — induk tanpa anak adalah
// baris menggantung, dan justru itu cacat yang sedang diperbaiki modul ini.
//
// Urutannya mengikuti `Activity/InsertMasterPenolakanNoteKlaim-Act.xml`:
//
//  1. selesaikan induk      -> MasterPenolakanKlaim1, atau pakai yang sudah ada
//  2. pakai ID induk        -> hasil langkah 1 menjadi ID_ST
//  3. turunkan ID_ND        -> MasterPenolakanKlaim2 langkah pertamanya
//  4. sisipkan barisnya
//
// Induk diselesaikan LEBIH DULU, sebelum baris tingkat 2 mana pun dikunci. Kegagalan yang
// paling mungkin terjadi — induk yang dipilih sudah tidak ada — diletakkan paling awal,
// supaya ia paling murah.
func (r *Repo) InsertNew(ctx context.Context, s masterpenolakan.Submission) (masterpenolakan.RejectionStatus2, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	parent, err := resolveParent(ctx, tx, s.Input)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	used, err := lockedIDs(ctx, tx, "rejection_list_id_locked")
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	fresh := masterpenolakan.RejectionStatus2{
		ID:   masterpenolakan.NextSequence(used, masterpenolakan.FormatID2),
		Name: s.Name,
		// Nama induk DISALIN ke kolom NOTE_ST, bukan dibiarkan kosong dan bukan dibaca
		// lewat join saat menampilkan — `MASTERPENOLAKANKLAIM2.prc:9` memang menuliskan
		// kedua kolom sekaligus, dan sistem lama membacanya dari sana.
		ParentID:    parent.ID,
		ParentName:  parent.Name,
		Status:      masterpenolakan.StatusPending,
		SubmittedBy: s.By,
		SubmittedAt: s.At.UTC(),
	}

	if _, err := tx.ExecContext(ctx, getQuery("rejection_insert"),
		fresh.ParentID, fresh.ParentName, fresh.ID, fresh.Name,
		fresh.SubmittedBy, fresh.SubmittedAt, string(fresh.Status),
	); err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: menyisipkan %q: %w", fresh.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: menutup transaksi sisip: %w", err)
	}
	return fresh, nil
}

// Update menyimpan perubahan pada Status Penolakan 2 yang sudah ada.
//
// Ia MENGEMBALIKAN baris ke antrean persetujuan; alasannya beserta buktinya ada pada doc
// comment masterpenolakan.Repo.Update dan pada kueri rejection_update.
//
// Barisnya dibaca lebih dulu di dalam transaksi yang sama, dengan dua sebab. Pertama,
// supaya "baris tidak ada" dapat dibedakan dari "baris ada tetapi nilainya sama persis" —
// UPDATE yang mengenai nol baris tidak membedakan keduanya, dan menjawab "tidak
// ditemukan" untuk penyimpanan yang sebenarnya berhasil akan membuat pengguna menyimpan
// berulang kali. Kedua, ketiga kolom persetujuan yang tidak ikut di-SET dibaca dari sana
// supaya baris yang dikembalikan ke layar utuh tanpa pembacaan kedua.
func (r *Repo) Update(ctx context.Context, id string, s masterpenolakan.Submission) (masterpenolakan.RejectionStatus2, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	clean := strings.TrimSpace(id)
	existing, err := scanRow(tx.QueryRowContext(ctx, getQuery("rejection_get"), clean))
	if errors.Is(err, sql.ErrNoRows) {
		return masterpenolakan.RejectionStatus2{}, masterpenolakan.ErrNotFound
	}
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: membaca %q: %w", id, err)
	}

	parent, err := resolveParent(ctx, tx, s.Input)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	updated := masterpenolakan.RejectionStatus2{
		ID:          existing.ID,
		Name:        s.Name,
		ParentID:    parent.ID,
		ParentName:  parent.Name,
		Status:      masterpenolakan.StatusPending,
		SubmittedBy: s.By,
		SubmittedAt: s.At.UTC(),
		// Ketiga jejak persetujuan dibawa apa adanya: procedure lama pun tidak
		// membersihkannya. Ia jejak keputusan yang PERNAH ada, bukan keadaan yang berlaku.
		ApprovedBy:   existing.ApprovedBy,
		ApprovedAt:   existing.ApprovedAt,
		ApprovalNote: existing.ApprovalNote,
	}

	if _, err := tx.ExecContext(ctx, getQuery("rejection_update"),
		updated.ParentID, updated.ParentName, updated.Name,
		updated.SubmittedBy, updated.SubmittedAt, string(updated.Status),
		clean,
	); err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: memperbarui %q: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return masterpenolakan.RejectionStatus2{}, fmt.Errorf("masterpenolakan/sqlstore: menutup transaksi ubah: %w", err)
	}
	return updated, nil
}

// CheckTable memastikan kedua tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	for _, t := range []struct {
		query string
		table string
	}{
		{"rejection_parent_check_table", "POOLDATA.MST_PENOLAKAN_KLAIM_1"},
		{"rejection_check_table", "POOLDATA.MST_PENOLAKAN_KLAIM_2"},
	} {
		rows, err := r.db.QueryContext(ctx, getQuery(t.query))
		if err != nil {
			return fmt.Errorf("masterpenolakan/sqlstore: %s tidak dapat dibaca: %w", t.table, err)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return fmt.Errorf("masterpenolakan/sqlstore: %s tidak dapat dibaca: %w", t.table, err)
		}
	}
	return nil
}

// resolveParent menyediakan Status Penolakan 1 yang akan dirujuk baris tingkat 2.
//
// Dua jalur, dan keduanya berakhir pada satu baris yang PASTI ada di basis data:
//
//   - induk dipilih  -> dibaca; ErrParentNotFound bila sudah tidak ada
//   - induk baru     -> nomornya diturunkan, lalu barisnya disisipkan
//
// Inilah tempat perbaikan yang diputuskan Work Owner 2026-09-19 benar-benar berlaku.
// Procedure lama (`MASTERPENOLAKANKLAIM1.prc`) menyisipkan baris baru pada KEDUA cabang
// IF-nya, sehingga memilih induk yang sudah ada pun tetap menerbitkan duplikat. Di sini
// jalur "pilih" tidak menulis apa pun.
func resolveParent(ctx context.Context, tx *sql.Tx, input masterpenolakan.Input) (masterpenolakan.RejectionStatus, error) {
	if !input.WantsNewParent() {
		parent, err := scanParent(tx.QueryRowContext(ctx, getQuery("rejection_parent_get"), input.ParentID))
		if errors.Is(err, sql.ErrNoRows) {
			// Galat yang KHUSUS, bukan ErrNotFound: yang hilang bukan baris yang diminta
			// pengguna, melainkan induk yang ia pilih dari daftar. Layar menanganinya
			// berbeda — yang satu berarti "muat ulang daftar", yang lain berarti "pilih
			// induk lain".
			return masterpenolakan.RejectionStatus{}, fmt.Errorf("%w: %q", masterpenolakan.ErrParentNotFound, input.ParentID)
		}
		if err != nil {
			return masterpenolakan.RejectionStatus{}, fmt.Errorf("masterpenolakan/sqlstore: membaca induk %q: %w", input.ParentID, err)
		}
		return parent, nil
	}

	used, err := lockedIDs(ctx, tx, "rejection_parent_list_id_locked")
	if err != nil {
		return masterpenolakan.RejectionStatus{}, err
	}

	fresh := masterpenolakan.RejectionStatus{
		ID:   masterpenolakan.NextSequence(used, masterpenolakan.FormatID),
		Name: input.ParentName,
	}
	if _, err := tx.ExecContext(ctx, getQuery("rejection_parent_insert"), fresh.ID, fresh.Name); err != nil {
		return masterpenolakan.RejectionStatus{}, fmt.Errorf("masterpenolakan/sqlstore: menyisipkan induk %q: %w", fresh.ID, err)
	}
	return fresh, nil
}

// lockedIDs mengunci baris yang ada lalu mengembalikan seluruh kuncinya.
//
// Satu fungsi untuk kedua tabel: yang berbeda hanya nama kuerinya, dan menyalin
// perulangannya dua kali berarti dua tempat yang harus diingat bersamaan setiap kali
// penanganan NULL-nya berubah.
func lockedIDs(ctx context.Context, tx *sql.Tx, queryName string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, getQuery(queryName))
	if err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: mengunci daftar ID (%s): %w", queryName, err)
	}
	defer func() { _ = rows.Close() }()

	var used []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("masterpenolakan/sqlstore: membaca ID (%s): %w", queryName, err)
		}
		used = append(used, strings.TrimSpace(id.String))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpenolakan/sqlstore: menelusuri daftar ID (%s): %w", queryName, err)
	}
	return used, nil
}

type scanner interface {
	Scan(target ...any) error
}

// scanParent membaca satu baris Status Penolakan 1.
func scanParent(p scanner) (masterpenolakan.RejectionStatus, error) {
	var id, name sql.NullString
	if err := p.Scan(&id, &name); err != nil {
		return masterpenolakan.RejectionStatus{}, err
	}
	return masterpenolakan.RejectionStatus{
		ID:   strings.TrimSpace(id.String),
		Name: strings.TrimSpace(name.String),
	}, nil
}

// scanRow membaca satu baris Status Penolakan 2.
//
// Seluruh kolom teks dibaca lewat sql.NullString lalu dipangkas, dengan dua sebab: kolom
// bertipe CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa
// pun, dan baris lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL
// yang diketahui (R-08).
//
// Urutan kolomnya mengikuti berkas .sql: baris ini sendiri lebih dulu (ID_ND, NOTE_ND),
// lalu induknya, lalu keadaan persetujuannya — terbaca sebagai "apa ini, anak siapa, dan
// sudah sampai mana".
func scanRow(p scanner) (masterpenolakan.RejectionStatus2, error) {
	var (
		id, name, parentID, parentName  sql.NullString
		status, submittedBy, approvedBy sql.NullString
		approvalNote                    sql.NullString
		submittedAt, approvedAt         sql.NullTime
	)
	if err := p.Scan(
		&id, &name, &parentID, &parentName,
		&status, &submittedBy, &submittedAt,
		&approvedBy, &approvedAt, &approvalNote,
	); err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	rejection := masterpenolakan.RejectionStatus2{
		ID:           strings.TrimSpace(id.String),
		Name:         strings.TrimSpace(name.String),
		ParentID:     strings.TrimSpace(parentID.String),
		ParentName:   strings.TrimSpace(parentName.String),
		Status:       masterpenolakan.ApprovalStatus(strings.TrimSpace(status.String)),
		SubmittedBy:  strings.TrimSpace(submittedBy.String),
		ApprovedBy:   strings.TrimSpace(approvedBy.String),
		ApprovalNote: strings.TrimSpace(approvalNote.String),
	}
	if submittedAt.Valid {
		rejection.SubmittedAt = submittedAt.Time.UTC()
	}
	if approvedAt.Valid {
		// Disalin ke variabel lokal lebih dulu: mengambil alamat field sql.NullTime
		// berarti menahan seluruh struct hasil pemindaian, dan nilainya akan berubah
		// bila fungsi ini kelak dipakai ulang di dalam perulangan.
		decided := approvedAt.Time.UTC()
		rejection.ApprovedAt = &decided
	}
	return rejection, nil
}

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf("masterpenolakan/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterpenolakan/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("masterpenolakan/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("masterpenolakan/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		var statement []string
		for _, line := range body {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				continue
			}
			statement = append(statement, line)
		}
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, line := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, line)
	}
	save()
	return result
}

// Penegasan bahwa seam benar-benar dipenuhi. Bila sebuah method hilang atau tandanya
// berubah, kegagalannya muncul saat kompilasi — bukan saat permintaan pertama.
var _ masterpenolakan.Repo = (*Repo)(nil)
