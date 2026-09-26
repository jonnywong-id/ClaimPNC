package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/mastersurveyors"
)

// Batas jumlah baris satu halaman.
//
// defaultLimit dipakai bila pemanggil tidak menyebut batas. maxLimit adalah pagar keras:
// permintaan yang lebih besar DIPOTONG, bukan ditolak, karena memotong halaman tidak
// pernah merusak apa pun sedangkan menolaknya membuat layar kosong.
//
// Angka 500 bukan pilihan bebas — ia sama dengan `pyMaxRecords` pada
// `Report Definition/BrowseVDSurveyors_RD-RD.xml`, sehingga satu halaman di sistem baru
// tidak pernah lebih besar daripada yang pernah dilayani sistem lama.
const (
	defaultLimit = 50
	maxLimit     = 500
)

// Nama objek basis data yang dipakai menerjemahkan galat tulis menjadi galat domain.
//
// Keduanya konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam.
//
//   - NameIndexName dan LoginIndexName dibuat migrasi 0004. Selama migrasi itu belum
//     dijalankan, tidak ada galat yang cocok dengan keduanya — dan itu berarti nama dan
//     login ganda hanya dicegah pemeriksaan di usecase, yang tidak tahan terhadap dua
//     permintaan yang tiba bersamaan.
//   - PrimaryKeyName BELUM DIVERIFIKASI ke ALL_CONSTRAINTS; namanya diduga mengikuti pola
//     saudaranya yang sudah diperiksa (`M_SURVEYORS_PK`). Bila dugaan ini salah, akibatnya
//     TERBATAS dan tidak berbahaya: bentrok kunci utama muncul sebagai galat 500 alih-alih
//     pesan yang rapi. Ia tidak pernah menyebabkan data salah tulis.
const (
	NameIndexName  = "UX_D_SURVEYORS_NAME"
	LoginIndexName = "UX_D_SURVEYORS_LOGIN"
	PrimaryKeyName = "D_SURVEYORS_PK"
)

// Repo membaca dan menulis POOLDATA.D_SURVEYORS.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar Detail Surveyors adalah
// SATU-SATUNYA penulis D_SURVEYORS di sistem lama, dan itu diperiksa ke seluruh export,
// bukan diandaikan:
//
//	Database/PEGA_D_SURVEYORS.prc            satu-satunya berkas yang memuat INSERT/UPDATE
//	RDB List/UpdateDetailSurveyors-SQL.xml   satu-satunya pemanggil procedure itu
//	Activity/CNMInsertDetailSurveyors_act    satu-satunya pemanggil rule itu
//
// Pembacanya banyak — ketiga `BrowseSurveyorType*-SQL.xml` dan setiap layar penugasan
// surveyor — tetapi pembaca tidak memindahkan kepemilikan. Memindahkan layar penulisnya
// ke sini memindahkan kepemilikan tabelnya secara utuh.
//
// Sama seperti pada M_SURVEYORS, `V_D_SURVEYORS` membaca KOLOM dan bukan JSON_DATA
// (ditegaskan Work Owner 2026-09-20) — sehingga tulisan Go langsung terlihat oleh rule
// Pega yang membacanya, tanpa perubahan view apa pun. Catatan lengkapnya di kepala
// mastersurveyors.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca surveyor yang cocok dengan filter beserta jumlah seluruh baris yang cocok.
func (r *Repo) List(ctx context.Context, f mastersurveyors.Filter) ([]mastersurveyors.Surveyor, int, error) {
	filters := filterArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("surveyor_count"), filters...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("mastersurveyors/sqlstore: menghitung surveyor: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	skip := f.Offset
	if skip < 0 {
		skip = 0
	}

	args := append(append([]any(nil), filters...), skip, limit)
	rows, err := r.db.QueryContext(ctx, getQuery("surveyor_list"), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mastersurveyors/sqlstore: membaca daftar surveyor: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastersurveyors.Surveyor, 0, limit)
	for rows.Next() {
		surveyor, err := scanSurveyor(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, surveyor)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("mastersurveyors/sqlstore: menelusuri daftar surveyor: %w", err)
	}
	return result, total, nil
}

// Get membaca satu surveyor.
func (r *Repo) Get(ctx context.Context, id string) (mastersurveyors.Surveyor, error) {
	surveyor, err := scanSurveyor(r.db.QueryRowContext(ctx, getQuery("surveyor_get"), strings.TrimSpace(id)))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return mastersurveyors.Surveyor{}, mastersurveyors.ErrNotFound
	case err != nil:
		return mastersurveyors.Surveyor{}, err
	}
	return surveyor, nil
}

// FindByNameKey membaca seluruh surveyor yang namanya sama menurut NameKey.
func (r *Repo) FindByNameKey(ctx context.Context, key string) ([]mastersurveyors.Surveyor, error) {
	return r.queryMany(ctx, "surveyor_by_name_key", mastersurveyors.NameKey(key), "mencari surveyor menurut nama")
}

// FindByAppLogin membaca seluruh surveyor dengan nama login tertentu.
//
// Login kosong SELALU mengembalikan daftar kosong tanpa menyentuh basis data: surveyor
// eksternal boleh tidak punya login, dan dua-duanya kosong bukan bentrok.
func (r *Repo) FindByAppLogin(ctx context.Context, login string) ([]mastersurveyors.Surveyor, error) {
	clean := strings.ToUpper(strings.TrimSpace(login))
	if clean == "" {
		return nil, nil
	}
	return r.queryMany(ctx, "surveyor_by_login", clean, "mencari surveyor menurut login aplikasi")
}

// queryMany menjalankan kueri berparameter tunggal yang mengembalikan banyak baris.
func (r *Repo) queryMany(ctx context.Context, name, arg, activity string) ([]mastersurveyors.Surveyor, error) {
	rows, err := r.db.QueryContext(ctx, getQuery(name), arg)
	if err != nil {
		return nil, fmt.Errorf("mastersurveyors/sqlstore: %s: %w", activity, err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastersurveyors.Surveyor
	for rows.Next() {
		surveyor, err := scanSurveyor(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, surveyor)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersurveyors/sqlstore: menelusuri hasil %s: %w", activity, err)
	}
	return result, nil
}

// Insert menyimpan surveyor baru dan mengembalikannya lengkap dengan ID yang dibentuk.
//
// Seluruhnya dalam SATU transaksi — pembacaan kode situs, pengambilan nomor urut, dan
// penyisipannya. Alasannya bukan kerapian: bila penyisipan gagal setelah nomor urut
// diambil, nomor itu hangus. Transaksi tidak mengembalikan nomor urut yang sudah diambil
// (NEXTVAL memang tidak dapat dibatalkan), tetapi ia menjamin tidak ada baris setengah
// jadi yang tertinggal.
func (r *Repo) Insert(ctx context.Context, s mastersurveyors.Surveyor) (mastersurveyors.Surveyor, error) {
	clean := s.Clean()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mastersurveyors.Surveyor{}, fmt.Errorf("mastersurveyors/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = tx.Rollback() }()

	id, err := newID(ctx, tx)
	if err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	clean.ID = id

	_, err = tx.ExecContext(ctx, getQuery("surveyor_insert"),
		clean.ID,
		clean.TypeCode,
		clean.Name,
		nullIfEmpty(clean.Address),
		nullIfEmpty(clean.PostalCode),
		nullIfEmpty(clean.State),
		nullIfEmpty(clean.Phone),
		nullIfEmpty(clean.Fax),
		nullIfEmpty(clean.Email),
		nullIfEmpty(clean.OtherContact),
		nullIfEmpty(clean.BranchCode),
		nullIfEmpty(clean.BranchName),
		nullIfEmpty(clean.AppLogin),
		nullIfEmpty(clean.DocumentID),
		string(clean.Status),
		nullIfEmpty(clean.Committee),
		nullIfEmpty(clean.CommitteeTransferred),
		// CreatedBy sengaja TIDAK diikat: kolom USER_INPUT belum ada di tabelnya.
	)
	if err != nil {
		return mastersurveyors.Surveyor{}, translateWriteError(err, "menyimpan surveyor")
	}
	if err := tx.Commit(); err != nil {
		return mastersurveyors.Surveyor{}, fmt.Errorf("mastersurveyors/sqlstore: menyelesaikan transaksi: %w", err)
	}
	return clean, nil
}

// Update menulis ulang surveyor yang sudah ada.
//
// Baris yang tidak ditemukan menghasilkan ErrNotFound, bukan keberhasilan diam-diam.
// UPDATE yang tidak mengenai satu baris pun TIDAK dianggap galat oleh basis data, dan
// tanpa pemeriksaan ini pengguna akan melihat "berhasil disimpan" atas perubahan yang
// tidak pernah tersimpan.
func (r *Repo) Update(ctx context.Context, s mastersurveyors.Surveyor) error {
	clean := s.Clean()

	result, err := r.db.ExecContext(ctx, getQuery("surveyor_update"),
		clean.TypeCode,
		clean.Name,
		nullIfEmpty(clean.Address),
		nullIfEmpty(clean.PostalCode),
		nullIfEmpty(clean.State),
		nullIfEmpty(clean.Phone),
		nullIfEmpty(clean.Fax),
		nullIfEmpty(clean.Email),
		nullIfEmpty(clean.OtherContact),
		nullIfEmpty(clean.BranchCode),
		nullIfEmpty(clean.BranchName),
		nullIfEmpty(clean.AppLogin),
		nullIfEmpty(clean.DocumentID),
		string(clean.Status),
		nullIfEmpty(clean.CommitteeTransferred),
		// DecidedAt, Note, dan UpdatedBy sengaja TIDAK diikat: kolomnya belum ada di
		// POOLDATA.D_SURVEYORS, dan Pega pun tidak punya ketiganya. Lihat catatan pada
		// `surveyor_update` di mastersurveyors.sql.
		clean.ID,
	)
	if err != nil {
		return translateWriteError(err, "mengubah surveyor")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat melaporkan jumlah baris bukan alasan menyatakan gagal:
		// pernyataannya sendiri sudah berhasil. Yang hilang hanya kemampuan membedakan
		// "tidak ada barisnya" dari "berhasil", dan itu dinyatakan apa adanya.
		return nil
	}
	if affected == 0 {
		return mastersurveyors.ErrNotFound
	}
	return nil
}

// CheckTable memastikan tabel dan seluruh kolomnya dapat dibaca akun aplikasi.
//
// Dipakai mode periksa untuk membedakan "migrasi 0004 belum dijalankan" dari "tidak punya
// hak baca" — dua sebab yang gejalanya sama tetapi tindak lanjutnya berbeda jauh.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("surveyor_check_table"))
	if err != nil {
		return fmt.Errorf("mastersurveyors/sqlstore: memeriksa POOLDATA.D_SURVEYORS: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// newID membentuk D_SURVEY_ID baru: kode situs ditambah nomor urut enam digit.
//
// Meniru `Database/PEGA_D_SURVEYORS.prc:19` persis:
//
//	id_surv_ins := id_site || lpad(to_Char(D_SURVEYORS_SEQ.nextval),6,'0')
//
// Pembentukannya dilakukan di Go, bukan di SQL, karena LPAD dan perangkaian teks bukan
// hal yang sama di Oracle dan PostgreSQL — dan `D-20` menetapkan satu set SQL untuk
// keduanya.
func newID(ctx context.Context, tx *sql.Tx) (string, error) {
	var site string
	err := tx.QueryRowContext(ctx, getQuery("surveyor_site")).Scan(&site)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", mastersurveyors.ErrNoSite
	case err != nil:
		return "", fmt.Errorf("mastersurveyors/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("surveyor_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("mastersurveyors/sqlstore: mengambil nomor urut surveyor: %w", err)
	}

	return strings.TrimSpace(site) + pad6(sequence), nil
}

// pad6 menuliskan nomor urut dalam enam digit dengan nol di depan.
//
// Nomor yang MELEBIHI enam digit ditulis apa adanya, tidak dipotong. Memotongnya akan
// menghasilkan ID yang bertabrakan dengan ID lama tanpa satu pun pesan; membiarkannya
// panjang akan ditolak basis data dengan galat yang menyebut kolomnya — dan galat yang
// terlihat selalu lebih baik daripada tabrakan yang senyap.
func pad6(n int64) string {
	text := fmt.Sprintf("%d", n)
	if len(text) >= 6 {
		return text
	}
	return strings.Repeat("0", 6-len(text)) + text
}

// filterArgs menyusun sepuluh argumen saringan dalam urutan yang dituntut kedua kueri.
//
// Setiap saringan muncul DUA KALI di dalam SQL — sekali pada pemeriksaan IS NULL, sekali
// pada perbandingannya — sehingga nilainya dikirim dua kali pula. Nomor bind pada berkas
// .sql harus tetap sejalan dengan urutan di sini; keduanya diuji bersama di query_test.go.
func filterArgs(f mastersurveyors.Filter) []any {
	status := nullIfEmpty(string(f.Status))
	name := nullIfEmpty(f.Name)
	login := nullIfEmpty(f.AppLogin)
	typeCode := nullIfEmpty(f.TypeCode)

	// Antrean komite hanya menyaring bila DIMINTA dan identitasnya diketahui. Identitas
	// kosong dengan MyCommitteeOnly menyala akan menyaring ke "komite bernama kosong" dan
	// mengembalikan daftar kosong — diam-diam, tanpa pesan. Lebih baik tidak menyaring.
	var committee any
	if f.MyCommitteeOnly {
		committee = nullIfEmpty(f.CommitteeIdentity)
	}

	return []any{
		status, status,
		name, name,
		login, login,
		typeCode, typeCode,
		committee, committee,
	}
}

// scanRow adalah yang dipenuhi *sql.Row maupun *sql.Rows, sehingga satu fungsi pemindai
// melayani pembacaan satu baris dan pembacaan daftar.
type scanRow interface {
	Scan(dest ...any) error
}

// scanSurveyor memindai satu baris menjadi nilai domain.
//
// Seluruh kolom teks dipindai sebagai sql.NullString: kolom yang NULL adalah keadaan yang
// sah di tabel ini — surveyor eksternal tanpa login, surveyor tanpa faksimile — dan
// memindainya langsung ke string akan gagal dengan galat konversi, bukan menghasilkan
// string kosong.
func scanSurveyor(row scanRow) (mastersurveyors.Surveyor, error) {
	var (
		id, legacyID, typeCode, typeDesc          sql.NullString
		name, address, postalCode, state          sql.NullString
		phone, fax, email, otherContact           sql.NullString
		branchCode, branchName, appLogin, docID   sql.NullString
		approval, committee, committeeTransferred sql.NullString
		note, createdBy, updatedBy                sql.NullString
		decidedAt, createdAt                      sql.NullTime
	)

	err := row.Scan(
		&id, &legacyID, &typeCode, &typeDesc,
		&name, &address, &postalCode, &state,
		&phone, &fax, &email, &otherContact,
		&branchCode, &branchName, &appLogin, &docID,
		&approval, &committee, &committeeTransferred,
		&decidedAt, &note, &createdBy, &createdAt, &updatedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mastersurveyors.Surveyor{}, err
		}
		return mastersurveyors.Surveyor{}, fmt.Errorf("mastersurveyors/sqlstore: memindai baris surveyor: %w", err)
	}

	surveyor := mastersurveyors.Surveyor{
		ID:                   id.String,
		LegacyID:             legacyID.String,
		TypeCode:             typeCode.String,
		TypeDescription:      typeDesc.String,
		Name:                 name.String,
		Address:              address.String,
		PostalCode:           postalCode.String,
		State:                state.String,
		Phone:                phone.String,
		Fax:                  fax.String,
		Email:                email.String,
		OtherContact:         otherContact.String,
		BranchCode:           branchCode.String,
		BranchName:           branchName.String,
		AppLogin:             appLogin.String,
		DocumentID:           docID.String,
		Status:               mastersurveyors.ApprovalStatus(strings.TrimSpace(approval.String)),
		Committee:            committee.String,
		CommitteeTransferred: committeeTransferred.String,
		Note:                 note.String,
		CreatedBy:            createdBy.String,
		UpdatedBy:            updatedBy.String,
	}
	if decidedAt.Valid {
		when := decidedAt.Time
		surveyor.DecidedAt = &when
	}
	if createdAt.Valid {
		surveyor.CreatedAt = createdAt.Time
	}
	return surveyor.Clean(), nil
}

// nullIfEmpty mengubah teks kosong menjadi NULL basis data.
//
// Perbedaannya nyata, bukan gaya: kolom kosong berisi "" dan kolom NULL diperlakukan
// berbeda oleh saringan `(:n IS NULL OR ...)`, dan menyimpan "" pada kolom opsional
// membuat baris itu ikut pada pencarian yang seharusnya melewatinya.
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

// nullTime mengubah penunjuk waktu nil menjadi NULL basis data.
func nullTime(when *time.Time) any {
	if when == nil {
		return nil
	}
	return *when
}

// translateWriteError mengubah pelanggaran indeks unik menjadi galat domain.
//
// Tanpa penerjemahan ini, bentrok nama akan sampai ke pengguna sebagai 500 beserta nomor
// galat Oracle. Yang dicocokkan adalah NAMA INDEKS yang kita buat sendiri di migrasi 0004,
// bukan nomor galat driver — nama itu milik kita dan tidak berubah saat driver berganti.
func translateWriteError(err error, activity string) error {
	message := strings.ToUpper(err.Error())
	switch {
	case strings.Contains(message, NameIndexName):
		return mastersurveyors.ErrNameTaken
	case strings.Contains(message, LoginIndexName):
		return mastersurveyors.ErrLoginTaken
	case strings.Contains(message, PrimaryKeyName):
		// Kunci utama bentrok berarti nomor urut mengeluarkan kode yang sudah dipakai.
		// Seharusnya mustahil; bila terjadi ia harus terlihat, bukan menimpa baris lain.
		return fmt.Errorf("mastersurveyors/sqlstore: %s: kode surveyor bentrok: %w", activity, err)
	default:
		return fmt.Errorf("mastersurveyors/sqlstore: %s: %w", activity, err)
	}
}
