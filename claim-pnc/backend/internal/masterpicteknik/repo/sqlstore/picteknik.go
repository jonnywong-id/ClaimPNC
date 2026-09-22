package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpicteknik"
)

// Repo membaca dan menulis POOLDATA.MST_USER_TEKNIK.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel —
// bukan bahwa tabel lama tidak boleh ditulis sama sekali. Layar Master PIC Teknik adalah
// satu-satunya penulis MST_USER_TEKNIK di sistem lama: seluruh penulisan melewati
// `UpdateMasterUserTeknis` → `PEGA_MST_USER_TEKNIS`, dan pemanggilnya hanya
// `CNMInsertMstUserTeknis_act`. Memindahkan layar itu ke sini memindahkan kepemilikan
// tabelnya secara utuh; rule Pega selebihnya hanya MEMBACA.
//
// Konsekuensinya mengikat rollout: layar Master PIC Teknik di Pega wajib dimatikan pada
// saat modul ini dinyalakan, bukan sesudahnya.
type Repo struct {
	db *sql.DB
}

// RepoBaru membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// Daftar membaca seluruh PIC teknik.
func (r *Repo) List(ctx context.Context) ([]masterpicteknik.PICTeknik, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("pic_teknik_list"))
	if err != nil {
		return nil, fmt.Errorf("masterpicteknik/sqlstore: membaca daftar PIC: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpicteknik.PICTeknik
	for rows.Next() {
		p, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterpicteknik/sqlstore: membaca baris PIC: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpicteknik/sqlstore: menelusuri daftar PIC: %w", err)
	}
	return result, nil
}

// Ambil membaca satu PIC teknik.
func (r *Repo) Get(ctx context.Context, operatorID string) (masterpicteknik.PICTeknik, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("pic_teknik_get"), operatorID)

	p, err := scanRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrNotFound
	case err != nil:
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/sqlstore: membaca PIC %q: %w", operatorID, err)
	}
	return p, nil
}

// Sisip menyimpan PIC teknik baru.
//
// Keberadaan baris diperiksa lebih dulu supaya bentrok dijawab ErrSudahAda, bukan galat
// kunci ganda dari driver yang tidak dapat dibedakan pemanggil. Ia tetap bukan jaminan:
// dua permintaan yang tiba bersamaan dapat sama-sama lolos, dan yang menahannya adalah
// primary key di basis data — yang galatnya juga diterjemahkan di sini.
func (r *Repo) Insert(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	if _, err := r.Get(ctx, p.OperatorID); err == nil {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrAlreadyExists
	} else if !errors.Is(err, masterpicteknik.ErrNotFound) {
		return masterpicteknik.PICTeknik{}, err
	}

	_, err := r.db.ExecContext(ctx, getQuery("pic_teknik_insert"),
		p.OperatorID,
		p.Name,
		p.Email,
		p.BusinessLine,
		p.Group,
		p.Supervisor,
		p.Quota,
		p.ExternalQuota,
		activeCode(p.Active),
	)
	if err != nil {
		if duplicateKey(err) {
			return masterpicteknik.PICTeknik{}, masterpicteknik.ErrAlreadyExists
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/sqlstore: menyisipkan PIC: %w", err)
	}
	return r.Get(ctx, p.OperatorID)
}

// Perbarui mengubah PIC teknik yang sudah ada.
func (r *Repo) Update(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	result, err := r.db.ExecContext(ctx, getQuery("pic_teknik_update"),
		p.Name,
		p.Email,
		p.BusinessLine,
		p.Group,
		p.Supervisor,
		p.Quota,
		p.ExternalQuota,
		activeCode(p.Active),
		p.OperatorID,
	)
	if err != nil {
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/sqlstore: memperbarui PIC: %w", err)
	}

	// Sebagian driver tidak melaporkan jumlah baris yang tersentuh. Ketiadaan angka
	// BUKAN bukti tidak ada yang berubah, sehingga hanya angka nol yang sungguhan
	// dianggap "tidak ditemukan".
	if count, err := result.RowsAffected(); err == nil && count == 0 {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrNotFound
	}
	return r.Get(ctx, p.OperatorID)
}

// PeriksaTabel memastikan seluruh kolom dapat dibaca akun aplikasi, tanpa mengambil satu
// baris pun. Dipakai mode periksa pada binary.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("pic_teknik_check_table"))
	if err != nil {
		return fmt.Errorf("masterpicteknik/sqlstore: memeriksa tabel MST_USER_TEKNIK: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// DirektoriRepo memenuhi seam DirektoriOperator dengan DATAPEGA.PR_OPERATORS.
//
// Tabel itu milik Pega dan hanya DIBACA (ADR-0004). Ia dipisahkan dari Repo karena
// sumbernya memang tabel lain dengan pemilik lain — menyatukannya akan menyamarkan
// bahwa satu di antaranya kita tulis dan satu lagi tidak.
type DirectoryRepo struct {
	db *sql.DB
}

// DirektoriRepoBaru membentuk pembaca direktori operator.
func NewDirectoryRepo(db *sql.DB) *DirectoryRepo { return &DirectoryRepo{db: db} }

// NamaOperator mencari nama petugas di direktori operator.
func (d *DirectoryRepo) OperatorName(ctx context.Context, operatorID string) (string, error) {
	var name sql.NullString

	err := d.db.QueryRowContext(ctx, getQuery("operator_name"), operatorID).Scan(&name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", masterpicteknik.ErrUnknownOperator
	case err != nil:
		// Kegagalan kueri terhadap tabel milik pihak lain diperlakukan sebagai
		// direktori tidak terhubung, bukan sebagai "tidak terdaftar": keduanya menuntut
		// tindak lanjut yang berbeda, dan menyamakannya akan menyuruh pengguna
		// memperbaiki ID yang sebenarnya sudah benar.
		return "", fmt.Errorf("%w: %v", masterpicteknik.ErrDirectoryUnreachable, err)
	}

	clean := strings.TrimSpace(name.String)
	if clean == "" {
		// Baris ada tetapi namanya kosong. Sistem lama pun menolaknya — prasyaratnya
		// `TempDcol.MCL_NAME==""`, bukan "baris tidak ditemukan".
		return "", masterpicteknik.ErrUnknownOperator
	}
	return clean, nil
}

// pemindai menyatukan *sql.Row dan *sql.Rows sehingga satu fungsi pemindaian melayani
// keduanya. Tanpa ini, sepuluh kolom harus ditulis dua kali dan kedua salinannya harus
// diingat untuk diubah bersama-sama.
type rowScanner interface {
	Scan(target ...any) error
}

func scanRow(p rowScanner) (masterpicteknik.PICTeknik, error) {
	var (
		id, name, email, line, group, supervisor, groupPanel, active sql.NullString
		quota, externalQuota                                      sql.NullInt64
	)

	if err := p.Scan(&id, &name, &email, &line, &group, &supervisor, &quota, &externalQuota, &groupPanel, &active); err != nil {
		return masterpicteknik.PICTeknik{}, err
	}

	return masterpicteknik.PICTeknik{
		OperatorID: text(id),
		Name:       text(name),
		Email:      text(email),
		BusinessLine: text(line),
		Group:       text(group),
		Supervisor:     text(supervisor),
		Quota:      int(quota.Int64),
		ExternalQuota:  int(externalQuota.Int64),
		GrupPanel:  text(groupPanel),
		Active:      readActive(text(active)),
	}, nil
}

// sandiAktif memetakan status aktif ke isi kolom STS_AKTIF.
//
// "1" dan "0" — BUKAN "Ya"/"Tidak" seperti kolom bernama sama pada POOLDATA.LST_ACCOUNT.
// Dua tabel berbeda memakai sandi berbeda, dan itu justru alasan pemetaannya ditulis
// sebagai fungsi bernama di sini alih-alih diketik ulang di setiap pemanggilan.
func activeCode(active bool) string {
	if active {
		return masterpicteknik.ActiveCode
	}
	return "0"
}

// bacaAktif menafsirkan isi kolom STS_AKTIF.
//
// Menerima beberapa ejaan yang mungkin sudah telanjur ada di tabel. Menolak mengenali
// baris lama hanya karena ejaannya berbeda akan menampilkan petugas aktif sebagai
// nonaktif — dan petugas nonaktif tidak menerima penugasan, sehingga kesalahan itu
// langsung berakibat pada pembagian kerja.
func readActive(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "1", "Y", "YA", "AKTIF", "A":
		return true
	default:
		return false
	}
}

func text(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return strings.TrimSpace(n.String)
}

// kunciGanda mengenali galat pelanggaran kunci unik Oracle (ORA-00001).
//
// Pencocokan teks dipakai karena driver tidak memberi tipe galat khusus untuknya.
// Kodenya dicocokkan, bukan kalimatnya — pesan Oracle diterjemahkan mengikuti
// NLS_LANGUAGE dan dapat berbeda per instans, sedangkan kodenya tidak.
func duplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ORA-00001")
}

var (
	_ masterpicteknik.Repo              = (*Repo)(nil)
	_ masterpicteknik.OperatorDirectory = (*DirectoryRepo)(nil)
)
