package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/mastertipesurveyors"
)

// Repo membaca dan menulis POOLDATA.M_SURVEYORS.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar Master Tipe Surveyors adalah
// SATU-SATUNYA penulis M_SURVEYORS di sistem lama, dan itu diperiksa ke seluruh export,
// bukan diandaikan:
//
//	Database/PEGA_M_SURVEYORS.prc      satu-satunya berkas yang memuat INSERT/UPDATE
//	RDB List/UpdateMSurveyors-SQL.xml  satu-satunya pemanggil procedure itu
//	Activity/CNMInsertSurveyors_act    satu-satunya pemanggil rule itu
//
// Memindahkan layar itu ke sini karena itu memindahkan kepemilikan tabelnya secara utuh.
// Pega berubah menjadi pembaca saja lewat V_M_SURVEYORS, dan tidak ada data yang kembar.
//
// Keputusan Work Owner 2026-09-19.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh tipe surveyor.
func (r *Repo) List(ctx context.Context) ([]mastertipesurveyors.SurveyorType, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("surveyor_type_list"))
	if err != nil {
		return nil, fmt.Errorf("mastertipesurveyors/sqlstore: membaca daftar tipe surveyor: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastertipesurveyors.SurveyorType
	for rows.Next() {
		surveyorType, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("mastertipesurveyors/sqlstore: membaca baris tipe surveyor: %w", err)
		}
		result = append(result, surveyorType)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastertipesurveyors/sqlstore: menelusuri daftar tipe surveyor: %w", err)
	}
	return result, nil
}

// Get membaca satu tipe surveyor.
func (r *Repo) Get(ctx context.Context, code string) (mastertipesurveyors.SurveyorType, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("surveyor_type_get"), code)

	surveyorType, err := scanRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return mastertipesurveyors.SurveyorType{}, mastertipesurveyors.ErrNotFound
	case err != nil:
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/sqlstore: membaca tipe surveyor: %w", err)
	}
	return surveyorType, nil
}

// Insert menyimpan tipe baru dengan kode yang dibentuk seperti procedure lama.
//
// Ketiga langkahnya berada dalam SATU transaksi. Ini memperbaiki cacat nyata sistem lama:
// `PEGA_M_SURVEYORS.prc` menjalankan COMMIT sendiri di dalam cabang INSERT (`:24`),
// sementara satu-satunya ROLLBACK cabang itu berada SESUDAH commit tersebut — sehingga
// tidak memulihkan apa pun. `D-68` menetapkan kepemilikan transaksi berpindah ke Go
// persis karena pola seperti itu.
func (r *Repo) Insert(ctx context.Context, description string) (mastertipesurveyors.SurveyorType, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = tx.Rollback() }()

	code, err := issueCode(ctx, tx)
	if err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}

	if _, err := tx.ExecContext(ctx, getQuery("surveyor_type_insert"), code, description); err != nil {
		return mastertipesurveyors.SurveyorType{}, translateWriteError(err, "menyisipkan tipe surveyor")
	}
	if err := tx.Commit(); err != nil {
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/sqlstore: menyimpan tipe surveyor baru: %w", err)
	}

	return mastertipesurveyors.SurveyorType{Code: code, Description: description}, nil
}

// Update mengganti deskripsi tipe yang sudah ada.
func (r *Repo) Update(ctx context.Context, code, description string) (mastertipesurveyors.SurveyorType, error) {
	result, err := r.db.ExecContext(ctx, getQuery("surveyor_type_update"), description, code)
	if err != nil {
		return mastertipesurveyors.SurveyorType{}, translateWriteError(err, "mengubah tipe surveyor")
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap kode yang tidak ada
	// berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi.
	affected, err := result.RowsAffected()
	if err != nil {
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if affected == 0 {
		return mastertipesurveyors.SurveyorType{}, mastertipesurveyors.ErrNotFound
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang sebenarnya, termasuk
	// OLD_M_SURVEY_ID yang tidak ikut diubah dan tidak diketahui pemanggil.
	return r.Get(ctx, code)
}

// issueCode membentuk M_SURVEY_ID persis seperti `Database/PEGA_M_SURVEYORS.prc` baris 11
// dan 19: kode situs disambung nomor urut tiga digit.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR
// termasuk yang dilarang `docs/Steering/09-DATABASE-STRATEGY.md` §4 karena keduanya
// mengikat kueri pada dialek Oracle.
func issueCode(ctx context.Context, tx *sql.Tx) (string, error) {
	var site string
	if err := tx.QueryRowContext(ctx, getQuery("surveyor_type_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, kode tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu berhenti seolah tidak
			// terjadi apa-apa; di sini ia menjadi galat yang benar-benar galat.
			return "", mastertipesurveyors.ErrNoSite
		}
		return "", fmt.Errorf("mastertipesurveyors/sqlstore: membaca kode situs: %w", err)
	}

	var order int64
	if err := tx.QueryRowContext(ctx, getQuery("surveyor_type_next_sequence")).Scan(&order); err != nil {
		return "", fmt.Errorf("mastertipesurveyors/sqlstore: mengambil nomor urut: %w", err)
	}

	return strings.TrimSpace(site) + ThreeDigits(order), nil
}

// ThreeDigits meniru `lpad(to_char(seq), 3, '0')` pada procedure lama.
//
// # Batas yang nyata, bukan teoretis
//
// Bilangan di atas 999 dikembalikan apa adanya, sama seperti LPAD Oracle — dan kode
// ke-1000 karena itu menjadi LIMA karakter. Pemeriksaan katalog pada 2026-09-19
// membuktikan akibatnya bukan sekadar kode yang kepanjangan: kolom M_SURVEY_ID bertipe
// CHAR(4), sehingga penyisipannya akan DITOLAK basis data (ORA-12899), bukan diterima
// dengan kode aneh.
//
// Urutan POOLDATA.M_SURVEYORS_SEQ berada di 11 pada tanggal itu, sementara kode tertinggi
// yang terpakai baru 1004. Artinya sekitar 989 penambahan lagi sebelum skema ini mentok —
// jauh lebih lapang daripada Master Status Klaim yang tinggal ~806, dan untuk daftar yang
// isinya empat baris, praktis tidak akan tercapai.
//
// Dipotong menjadi tiga digit? Tidak. Itu akan menghasilkan kode GANDA, yang jauh lebih
// buruk daripada penyisipan yang gagal dengan pesan jelas. Perilakunya dibiarkan apa
// adanya, dan batasnya dicatat supaya keputusan memperlebar kolom diambil sebelum, bukan
// sesudah, penyisipan pertama yang gagal.
//
// Diekspor supaya perilaku ini dapat diuji.
func ThreeDigits(n int64) string {
	number := fmt.Sprintf("%d", n)
	for len(number) < 3 {
		number = "0" + number
	}
	return number
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

func scanRow(rows rowScanner) (mastertipesurveyors.SurveyorType, error) {
	var (
		code        string
		description sql.NullString
		legacyCode  sql.NullString
	)
	if err := rows.Scan(&code, &description, &legacyCode); err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}
	// Ketiganya dirapikan di satu tempat: M_SURVEY_ID dan OLD_M_SURVEY_ID keduanya CHAR(4)
	// sehingga Oracle memadatkannya dengan spasi, dan deskripsi warisan dapat membawa
	// spasi tepi.
	//
	// DESCRIPTION dan OLD_M_SURVEY_ID dibaca lewat sql.NullString karena keduanya NULLABLE
	// — dan OLD_M_SURVEY_ID memang NULL pada seluruh empat baris yang ada.
	return mastertipesurveyors.SurveyorType{
		Code:        code,
		Description: description.String,
		LegacyCode:  legacyCode.String,
	}.Clean(), nil
}

// Dua nama objek basis data yang dibaca saat menerjemahkan galat.
//
//   - DescriptionIndexName dibuat migrasi 0003; ia yang menegakkan keunikan deskripsi.
//     Selama migrasi itu belum dijalankan, tidak ada galat yang cocok dengannya — dan
//     itu berarti nama ganda hanya dicegah pemeriksaan di usecase.
//   - PrimaryKeyName SUDAH ADA di basis data. Namanya diverifikasi langsung dari
//     ALL_CONSTRAINTS pada 2026-09-19: `M_SURVEYORS_PK`.
//
// Keduanya konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam.
const (
	DescriptionIndexName = "UX_M_SURVEYORS_DESC"
	PrimaryKeyName       = "M_SURVEYORS_PK"
)

// translateWriteError mengubah pelanggaran indeks unik menjadi galat domain.
//
// Tanpa penerjemahan ini, bentrok deskripsi akan sampai ke pengguna sebagai 500 beserta
// nomor galat Oracle. Yang dicocokkan adalah NAMA INDEKS yang kita buat sendiri di
// migrasi 0003, bukan nomor galat driver — nama itu milik kita dan tidak berubah saat
// driver atau basis datanya berganti.
func translateWriteError(err error, activity string) error {
	message := strings.ToUpper(err.Error())
	switch {
	case strings.Contains(message, DescriptionIndexName):
		return mastertipesurveyors.ErrDescriptionTaken
	case strings.Contains(message, PrimaryKeyName):
		// Kunci utama bentrok berarti nomor urut mengeluarkan kode yang sudah dipakai.
		// Seharusnya mustahil; bila terjadi ia harus terlihat, bukan menimpa baris lain.
		return mastertipesurveyors.ErrCodeTaken
	default:
		return fmt.Errorf("mastertipesurveyors/sqlstore: %s: %w", activity, err)
	}
}

var _ mastertipesurveyors.Repo = (*Repo)(nil)

// CheckTable menguji apakah ketiga kolom yang dipakai modul ini ada dan dapat dibaca akun
// aplikasi, tanpa mengambil satu baris pun.
//
// Dipakai untuk membedakan dua sebab kegagalan yang tampak mirip tetapi perbaikannya
// berbeda jauh: tabelnya tidak ada di basis data portal itu, versus akun aplikasi tidak
// punya hak baca atas tabel warisan.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("surveyor_type_check_table"))
	if err != nil {
		return fmt.Errorf("mastertipesurveyors/sqlstore: POOLDATA.M_SURVEYORS tidak dapat dibaca: %w", err)
	}
	return rows.Close()
}
