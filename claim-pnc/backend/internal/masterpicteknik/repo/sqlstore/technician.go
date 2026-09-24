package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpicteknik"
)

// uniqueViolation adalah kode galat Oracle untuk pelanggaran batasan unik.
//
// Dicocokkan lewat KODE, bukan nama constraint-nya: nama batasan pada MST_USER_TEKNIK
// tidak terbaca dari export, dan mengarang namanya akan menghasilkan pencocokan yang
// selalu meleset — kegagalan yang jauh lebih buruk karena ia diam.
const uniqueViolation = "ORA-00001"

// Repo membaca dan menulis master PIC teknik.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar Master PIC Teknik adalah
// SATU-SATUNYA penulis MST_USER_TEKNIK di sistem lama, dan itu diperiksa ke seluruh
// export, bukan diandaikan:
//
//	Database/PEGA_MST_USER_TEKNIS.prc            satu-satunya berkas ber-INSERT/UPDATE
//	RDB List/UpdateMasterUserTeknis-SQL.xml      satu-satunya pemanggil procedure itu
//	Activity/CNMInsertMstUserTeknis_act-Act.xml  satu-satunya pemanggil rule itu
//
// Memindahkan layar itu ke sini karena itu memindahkan kepemilikan tabelnya secara utuh.
// Pega berubah menjadi pembaca saja lewat V_MST_USER_TEKNIS, dan tidak ada data kembar.
//
// # Konsekuensi yang mengikat rollout
//
// Layar Master PIC Teknik di Pega wajib dimatikan pada saat modul ini dinyalakan, bukan
// sesudahnya. Selama keduanya hidup, `P-1` benar-benar dilanggar — dan pelanggarannya
// tidak akan terlihat sebagai galat, melainkan sebagai dua baris yang saling menimpa.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca petugas AKTIF dari view.
//
// Sandi aktif dikirim sebagai parameter, bukan ditulis di dalam teks SQL, supaya ia hidup
// di satu tempat saja — masterpicteknik.ActiveCode.
func (r *Repo) List(ctx context.Context) ([]masterpicteknik.Technician, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("technician_list"), masterpicteknik.ActiveCode)
	if err != nil {
		return nil, fmt.Errorf("masterpicteknik/sqlstore: membaca daftar PIC teknik: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpicteknik.Technician
	for rows.Next() {
		technician, err := scanListRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterpicteknik/sqlstore: membaca baris PIC teknik: %w", err)
		}
		result = append(result, technician)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpicteknik/sqlstore: menelusuri daftar PIC teknik: %w", err)
	}
	return result, nil
}

// Get membaca satu petugas dari tabel, aktif maupun tidak.
func (r *Repo) Get(ctx context.Context, operatorID string) (masterpicteknik.Technician, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("technician_get"), operatorID)

	technician, err := scanTableRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterpicteknik.Technician{}, masterpicteknik.ErrNotFound
	case err != nil:
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/sqlstore: membaca PIC teknik: %w", err)
	}
	return technician, nil
}

// Insert menyimpan petugas baru.
//
// # Kenapa memeriksa lebih dulu, bukan mengandalkan batasan unik saja
//
// Keberadaan batasan unik pada OPERATOR_ID TIDAK dapat dipastikan dari export — DDL
// tabelnya tidak ada di sana (`R-08`). Mengandalkannya saja berarti bertaruh: bila
// ternyata tidak ada, dua baris untuk satu petugas akan tersimpan diam-diam, dan
// penugasan klaim menjadi tidak dapat ditebak.
//
// Pemeriksaan di bawah karena itu bukan pengganti batasan unik melainkan lapisan yang
// bekerja tanpa mengandalkannya. Keduanya di dalam SATU transaksi supaya dua permintaan
// yang tiba bersamaan tidak sama-sama lolos. Bila batasannya ternyata ADA, galatnya tetap
// diterjemahkan — lihat translateWriteError.
func (r *Repo) Insert(ctx context.Context, t masterpicteknik.Technician) (masterpicteknik.Technician, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = tx.Rollback() }()

	var existing string
	err = tx.QueryRowContext(ctx, getQuery("technician_get"), t.OperatorID).Scan(
		&existing, new(sql.NullString), new(sql.NullString), new(sql.NullString),
		new(sql.NullString), new(sql.NullString), new(sql.NullInt64), new(sql.NullInt64),
		new(sql.NullString), new(sql.NullString),
	)
	switch {
	case err == nil:
		return masterpicteknik.Technician{}, masterpicteknik.ErrAlreadyExists
	case !errors.Is(err, sql.ErrNoRows):
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/sqlstore: memeriksa ID operator: %w", err)
	}

	if _, err := tx.ExecContext(ctx, getQuery("technician_insert"),
		t.OperatorID,
		t.Quota,
		nullable(t.BusinessLine),
		nullable(t.Email),
		activeCode(t.Active),
		nullable(t.Group),
		nullable(t.Supervisor),
		t.ExternalQuota,
		nullable(t.Name),
	); err != nil {
		return masterpicteknik.Technician{}, translateWriteError(err, "menambah PIC teknik")
	}

	if err := tx.Commit(); err != nil {
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/sqlstore: menyimpan PIC teknik: %w", err)
	}
	return t, nil
}

// Update mengubah petugas yang sudah ada.
//
// Baris yang tidak tersentuh dijawab ErrNotFound, bukan dianggap berhasil. Tanpa
// pemeriksaan itu, mengubah petugas yang sudah dihapus orang lain akan dijawab "tersimpan"
// padahal tidak ada yang tersimpan.
func (r *Repo) Update(ctx context.Context, t masterpicteknik.Technician) (masterpicteknik.Technician, error) {
	result, err := r.db.ExecContext(ctx, getQuery("technician_update"),
		nullable(t.Name),
		nullable(t.Email),
		nullable(t.BusinessLine),
		nullable(t.Group),
		nullable(t.Supervisor),
		t.Quota,
		t.ExternalQuota,
		activeCode(t.Active),
		t.OperatorID,
	)
	if err != nil {
		return masterpicteknik.Technician{}, translateWriteError(err, "mengubah PIC teknik")
	}

	changed, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak melaporkan jumlah baris tidak boleh membuat modul ini gagal:
		// pernyataannya sudah berhasil dijalankan. Yang hilang hanyalah kemampuan
		// membedakan "tidak ada barisnya", dan itu sudah diperiksa usecase lewat Get
		// sebelum sampai ke sini.
		return t, nil
	}
	if changed == 0 {
		return masterpicteknik.Technician{}, masterpicteknik.ErrNotFound
	}
	return t, nil
}

// CheckTable memastikan tabel yang DITULIS modul ini dapat dibaca akun aplikasi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("technician_check_table"))
	if err != nil {
		return fmt.Errorf("masterpicteknik/sqlstore: POOLDATA.MST_USER_TEKNIK tidak dapat dibaca: %w", err)
	}
	return rows.Close()
}

// CheckView memastikan view yang DIBACA daftar dapat dibaca akun aplikasi.
//
// Ia terpisah dari CheckTable karena keduanya dapat gagal sendiri-sendiri, dan sebabnya
// berbeda: nama kolom view belum dapat diverifikasi dari export, sedangkan tabelnya sudah.
// Memisahkannya membuat mode periksa dapat menunjuk yang mana yang bermasalah.
func (r *Repo) CheckView(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("technician_check_view"))
	if err != nil {
		return fmt.Errorf("masterpicteknik/sqlstore: POOLDATA.V_MST_USER_TEKNIS tidak dapat dibaca: %w", err)
	}
	return rows.Close()
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi antarmuka apa pun di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

// scanListRow membaca satu baris dari VIEW — sepuluh kolom, termasuk TOTAL_JOB.
func scanListRow(rows rowScanner) (masterpicteknik.Technician, error) {
	var (
		operatorID    string
		name          sql.NullString
		email         sql.NullString
		businessLine  sql.NullString
		group         sql.NullString
		supervisor    sql.NullString
		quota         sql.NullInt64
		externalQuota sql.NullInt64
		workload      sql.NullInt64
		active        sql.NullString
	)
	if err := rows.Scan(
		&operatorID, &name, &email, &businessLine, &group,
		&supervisor, &quota, &externalQuota, &workload, &active,
	); err != nil {
		return masterpicteknik.Technician{}, err
	}

	return masterpicteknik.Technician{
		OperatorID:    operatorID,
		Name:          name.String,
		Email:         email.String,
		BusinessLine:  businessLine.String,
		Group:         group.String,
		Supervisor:    supervisor.String,
		Quota:         int(quota.Int64),
		ExternalQuota: int(externalQuota.Int64),
		Workload:      int(workload.Int64),
		Active:        isActive(active),
	}.Clean(), nil
}

// scanTableRow membaca satu baris dari TABEL — sepuluh kolom, dengan GROUPPANEL
// menggantikan TOTAL_JOB yang tidak ada di sana.
func scanTableRow(rows rowScanner) (masterpicteknik.Technician, error) {
	var (
		operatorID    string
		name          sql.NullString
		email         sql.NullString
		businessLine  sql.NullString
		group         sql.NullString
		supervisor    sql.NullString
		quota         sql.NullInt64
		externalQuota sql.NullInt64
		panelGroup    sql.NullString
		active        sql.NullString
	)
	if err := rows.Scan(
		&operatorID, &name, &email, &businessLine, &group,
		&supervisor, &quota, &externalQuota, &panelGroup, &active,
	); err != nil {
		return masterpicteknik.Technician{}, err
	}

	return masterpicteknik.Technician{
		OperatorID:    operatorID,
		Name:          name.String,
		Email:         email.String,
		BusinessLine:  businessLine.String,
		Group:         group.String,
		Supervisor:    supervisor.String,
		Quota:         int(quota.Int64),
		ExternalQuota: int(externalQuota.Int64),
		PanelGroup:    panelGroup.String,
		Active:        isActive(active),
	}.Clean(), nil
}

// isActive menerjemahkan STS_AKTIF menjadi boolean.
//
// NULL diperlakukan TIDAK AKTIF, bukan aktif. Baris yang statusnya tidak dinyatakan tidak
// boleh diam-diam ikut menerima penugasan klaim — dan pada kueri daftar ia memang tidak
// akan terbaca sama sekali, karena `STS_AKTIF = '1'` tidak pernah benar untuk NULL.
//
// Spasi dibuang lebih dulu: kolom bertipe CHAR akan dipadatkan Oracle dengan spasi tanpa
// memberi tanda apa pun, dan "1 " tidak akan pernah sama dengan "1".
func isActive(value sql.NullString) bool {
	if !value.Valid {
		return false
	}
	return strings.TrimSpace(value.String) == masterpicteknik.ActiveCode
}

// activeCode menerjemahkan boolean kembali menjadi isi kolom STS_AKTIF.
func activeCode(active bool) string {
	if active {
		return masterpicteknik.ActiveCode
	}
	return masterpicteknik.InactiveCode
}

// nullable mengubah teks kosong menjadi NULL.
//
// Bukan kerapian: kolom-kolom ini NULLABLE dan barisnya yang ditulis Pega meninggalkan
// NULL untuk isian yang tidak diisi. Menuliskan string kosong akan membuat dua baris yang
// sama-sama "tidak diisi" tersimpan berbeda, dan setiap kueri `IS NULL` yang ditulis pihak
// lain melewatkan separuhnya.
func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// translateWriteError menerjemahkan galat penulisan menjadi galat domain.
//
// Ia jaring pengaman untuk perlombaan: pemeriksaan di dalam Insert sudah menolak ID yang
// sudah ada, tetapi bila basis data ternyata memang punya batasan unik dan dua permintaan
// tiba bersamaan, yang kalah akan dijawab lewat sini — bukan dengan galat mentah driver.
func translateWriteError(err error, activity string) error {
	if strings.Contains(strings.ToUpper(err.Error()), uniqueViolation) {
		return masterpicteknik.ErrAlreadyExists
	}
	return fmt.Errorf("masterpicteknik/sqlstore: %s: %w", activity, err)
}

var _ masterpicteknik.Repo = (*Repo)(nil)
