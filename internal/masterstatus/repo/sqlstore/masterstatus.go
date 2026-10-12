package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterstatus"
)

// Repo membaca dan menulis POOLDATA.M_STS_CLAIM.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// P-1 menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar Master Status Klaim adalah
// SATU-SATUNYA penulis M_STS_CLAIM di sistem lama (RDB List/UpdateStsClaim-SQL.xml,
// pemanggil tunggal PEGA_M_STS_CLAIM), sehingga memindahkan layar itu ke sini
// memindahkan kepemilikan tabelnya secara utuh. Pega berubah menjadi pembaca saja lewat
// V_STS_CLAIM, dan tidak ada data yang kembar.
//
// Keputusan Work Owner 2026-09-17.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// Daftar membaca seluruh status klaim.
func (r *Repo) List(ctx context.Context) ([]masterstatus.ClaimStatus, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("claim_status_list"))
	if err != nil {
		return nil, fmt.Errorf("masterstatus/sqlstore: membaca daftar status: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterstatus.ClaimStatus
	for rows.Next() {
		status, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterstatus/sqlstore: membaca baris status: %w", err)
		}
		result = append(result, status)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterstatus/sqlstore: menelusuri daftar status: %w", err)
	}
	return result, nil
}

// Ambil membaca satu status klaim.
func (r *Repo) Get(ctx context.Context, code string) (masterstatus.ClaimStatus, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("claim_status_get"), code)

	status, err := scanRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterstatus.ClaimStatus{}, masterstatus.ErrNotFound
	case err != nil:
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/sqlstore: membaca status: %w", err)
	}
	return status, nil
}

// Sisip menyimpan status baru dengan kode yang dibentuk seperti procedure lama.
//
// Ketiga langkahnya berada dalam SATU transaksi. Ini memperbaiki cacat nyata sistem
// lama: PEGA_M_STS_CLAIM.prc menjalankan COMMIT sendiri di dalam cabang INSERT (:25),
// sementara satu-satunya ROLLBACK-nya berada di handler terluar yang berjalan SESUDAH
// commit itu — sehingga tidak memulihkan apa pun. D-68 menetapkan kepemilikan transaksi
// berpindah ke Go persis karena pola seperti itu.
func (r *Repo) Insert(ctx context.Context, label string) (masterstatus.ClaimStatus, error) {
	transaksi, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = transaksi.Rollback() }()

	code, err := issueCode(ctx, transaksi)
	if err != nil {
		return masterstatus.ClaimStatus{}, err
	}

	if _, err := transaksi.ExecContext(ctx, getQuery("claim_status_insert"), code, label); err != nil {
		return masterstatus.ClaimStatus{}, translateWriteError(err, "menyisipkan status")
	}
	if err := transaksi.Commit(); err != nil {
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/sqlstore: menyimpan status baru: %w", err)
	}

	return masterstatus.ClaimStatus{Code: code, Label: label}, nil
}

// Perbarui mengganti label status yang sudah ada.
func (r *Repo) Update(ctx context.Context, code, label string) (masterstatus.ClaimStatus, error) {
	result, err := r.db.ExecContext(ctx, getQuery("claim_status_update"), label, code)
	if err != nil {
		return masterstatus.ClaimStatus{}, translateWriteError(err, "mengubah status")
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap kode yang tidak
	// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan"
	// atas perubahan yang tidak pernah terjadi.
	tersentuh, err := result.RowsAffected()
	if err != nil {
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if tersentuh == 0 {
		return masterstatus.ClaimStatus{}, masterstatus.ErrNotFound
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang sebenarnya, termasuk
	// OLD_LSC_ID yang tidak ikut diubah dan tidak diketahui pemanggil.
	return r.Get(ctx, code)
}

// issueCode membentuk LSC_ID persis seperti PEGA_M_STS_CLAIM.prc baris 11 dan 19:
// kode situs disambung nomor urut tiga digit.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR
// termasuk yang dilarang docs/Steering/09-DATABASE-STRATEGY.md §4 karena keduanya
// mengikat kueri pada dialek Oracle.
func issueCode(ctx context.Context, transaksi *sql.Tx) (string, error) {
	var site string
	if err := transaksi.QueryRowContext(ctx, getQuery("claim_status_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, kode tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu berhenti seolah tidak
			// terjadi apa-apa; di sini ia menjadi galat yang benar-benar galat.
			return "", errors.New("masterstatus/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
		}
		return "", fmt.Errorf("masterstatus/sqlstore: membaca kode situs: %w", err)
	}

	var order int64
	if err := transaksi.QueryRowContext(ctx, getQuery("claim_status_next_sequence")).Scan(&order); err != nil {
		return "", fmt.Errorf("masterstatus/sqlstore: mengambil nomor urut: %w", err)
	}

	return strings.TrimSpace(site) + ThreeDigits(order), nil
}

// ThreeDigits meniru lpad(to_char(seq), 3, '0') pada procedure lama.
//
// # Batas yang nyata, bukan teoretis
//
// Bilangan di atas 999 dikembalikan apa adanya, sama seperti LPAD Oracle — dan kode
// ke-1000 karena itu menjadi LIMA karakter. Pemeriksaan katalog pada 2026-09-17
// membuktikan akibatnya bukan sekadar kode yang kepanjangan: kolom LSC_ID bertipe
// CHAR(4), sehingga penyisipannya akan DITOLAK basis data (ORA-12899), bukan diterima
// dengan kode aneh.
//
// Urutan POOLDATA.M_STS_CLAIM_SEQ sudah berada di 193 pada tanggal itu, sementara kode
// tertinggi yang terpakai baru 1166. Artinya sekitar 806 penambahan lagi sebelum skema
// ini mentok — cukup lama untuk tidak mendesak, terlalu dekat untuk dilupakan.
//
// Dipotong menjadi tiga digit? Tidak. Itu akan menghasilkan kode GANDA, yang jauh lebih
// buruk daripada penyisipan yang gagal dengan pesan jelas. Perilakunya dibiarkan apa
// adanya, dan batasnya dicatat di README supaya keputusan memperlebar kolom diambil
// sebelum, bukan sesudah, penyisipan pertama yang gagal.
//
// Diekspor supaya perilaku ini dapat diuji.
func ThreeDigits(n int64) string {
	angka := fmt.Sprintf("%d", n)
	for len(angka) < 3 {
		angka = "0" + angka
	}
	return angka
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

func scanRow(rows rowScanner) (masterstatus.ClaimStatus, error) {
	var (
		code         string
		label        sql.NullString
		previousCode sql.NullString
	)
	if err := rows.Scan(&code, &label, &previousCode); err != nil {
		return masterstatus.ClaimStatus{}, err
	}
	// Ketiganya dirapikan di satu tempat: OLD_LSC_ID tersimpan sebagai CHAR berisi
	// padding, dan label warisan dapat membawa spasi tepi.
	return masterstatus.ClaimStatus{
		Code:       code,
		Label:      label.String,
		LegacyCode: previousCode.String,
	}.Clean(), nil
}

// Dua nama objek basis data yang dibaca saat menerjemahkan galat.
//
//   - LabelIndexName dibuat migrasi 0002; ia yang menegakkan keunikan label.
//   - PrimaryKeyName SUDAH ADA di basis data. Namanya diverifikasi langsung dari
//     ALL_CONSTRAINTS pada 2026-09-17: `M_STS_CLAIM_PK`, bukan `PK_M_STS_CLAIM` seperti
//     yang saya tulis mula-mula dengan menebak dari pola penamaan migrasi 0001.
//
// Keduanya konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam.
const (
	LabelIndexName = "UX_M_STS_CLAIM_LABEL"
	PrimaryKeyName = "M_STS_CLAIM_PK"
)

// translateWriteError mengubah pelanggaran indeks unik menjadi galat domain.
//
// Tanpa penerjemahan ini, bentrok label akan sampai ke pengguna sebagai 500 beserta
// nomor galat Oracle. Yang dicocokkan adalah NAMA INDEKS yang kita buat sendiri di
// migrasi 0002, bukan nomor galat driver — nama itu milik kita dan tidak berubah saat
// driver atau basis datanya berganti.
func translateWriteError(err error, kegiatan string) error {
	message := strings.ToUpper(err.Error())
	switch {
	case strings.Contains(message, LabelIndexName):
		return masterstatus.ErrLabelTaken
	case strings.Contains(message, PrimaryKeyName):
		// Kunci utama bentrok berarti nomor urut mengeluarkan kode yang sudah dipakai.
		// Seharusnya mustahil; bila terjadi ia harus terlihat, bukan menimpa baris lain.
		return masterstatus.ErrCodeTaken
	default:
		return fmt.Errorf("masterstatus/sqlstore: %s: %w", kegiatan, err)
	}
}

var _ masterstatus.Repo = (*Repo)(nil)

// CheckTable menguji apakah kedua kolom yang ditambahkan migrasi 0002 sudah ada dan
// dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: migrasi belum dijalankan DBA, versus akun aplikasi tidak
// punya hak baca atas tabel warisan.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("claim_status_check_table"))
	if err != nil {
		return err
	}
	return rows.Close()
}
