package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpenyebabkerugian"
)

// Repo membaca dan menulis POOLDATA.M_CAUSE_OF_LOSS.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali.
//
// # Satu hal yang membedakannya dari Master Status Klaim, dan harus diketahui
//
// Di sana, layar yang dipindahkan adalah SATU-SATUNYA penulis tabelnya, sehingga
// memindahkan layar memindahkan kepemilikan tabel secara utuh. **Di sini tidak.** Dua
// layar Pega menulis tabel ini lewat `RDB List/UpdateMCauseOfLoss-SQL.xml`:
//
//	CauseOfLossInbox             MENU_ID 20  digantikan modul ini
//	CauseOfLossInboxSimasOnline  MENU_ID 21  BELUM digantikan
//
// Selama layar kedua masih hidup, `P-1` belum terpenuhi utuh dan baris yang ditulisnya
// hanya mengisi JSON_DATA. Akibatnya beserta cara memantaunya dicatat di berkas migrasi
// 0005 dan pada kueri `cause_of_loss_count_pending_json`.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh golongan penyebab kerugian.
func (r *Repo) List(ctx context.Context) ([]masterpenyebabkerugian.CauseOfLoss, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("cause_of_loss_list"))
	if err != nil {
		return nil, fmt.Errorf("masterpenyebabkerugian/sqlstore: membaca daftar penyebab kerugian: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterpenyebabkerugian.CauseOfLoss
	for rows.Next() {
		cause, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterpenyebabkerugian/sqlstore: membaca baris penyebab kerugian: %w", err)
		}
		result = append(result, cause)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterpenyebabkerugian/sqlstore: menelusuri daftar penyebab kerugian: %w", err)
	}

	// Diurutkan di sini, bukan di SQL: ORDER BY atas kolom CHAR berpadding berperilaku
	// berbeda antar-basis-data, sementara `D-20` menuntut satu set SQL yang berjalan di
	// Oracle maupun PostgreSQL. Alasan lengkapnya di masterpenyebabkerugian.SortByID.
	masterpenyebabkerugian.SortByID(result)
	return result, nil
}

// Get membaca satu golongan penyebab kerugian.
func (r *Repo) Get(ctx context.Context, id string) (masterpenyebabkerugian.CauseOfLoss, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("cause_of_loss_get"), id)

	cause, err := scanRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterpenyebabkerugian.CauseOfLoss{}, masterpenyebabkerugian.ErrNotFound
	case err != nil:
		return masterpenyebabkerugian.CauseOfLoss{}, fmt.Errorf("masterpenyebabkerugian/sqlstore: membaca penyebab kerugian: %w", err)
	}
	return cause, nil
}

// Insert menyimpan golongan baru dengan ID yang dibentuk seperti procedure lama.
//
// Ketiga langkahnya berada dalam SATU transaksi. Ini memperbaiki cacat nyata sistem lama:
// `PEGA_M_CAUSE_OF_LOSS.prc` menjalankan COMMIT sendiri di dalam cabang INSERT,
// sementara ROLLBACK-nya berada di handler yang berjalan SESUDAH commit itu — sehingga
// tidak memulihkan apa pun. `D-68` menetapkan kepemilikan transaksi berpindah ke Go
// persis karena pola seperti itu.
func (r *Repo) Insert(ctx context.Context, description string) (masterpenyebabkerugian.CauseOfLoss, error) {
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, fmt.Errorf("masterpenyebabkerugian/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = transaction.Rollback() }()

	id, err := issueID(ctx, transaction)
	if err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, err
	}

	if _, err := transaction.ExecContext(ctx, getQuery("cause_of_loss_insert"), id, description); err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, translateWriteError(err, "menyisipkan penyebab kerugian")
	}
	if err := transaction.Commit(); err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, fmt.Errorf("masterpenyebabkerugian/sqlstore: menyimpan penyebab kerugian baru: %w", err)
	}

	// LegacyID sengaja kosong: kueri penyisipan memang tidak mengisi OLD_M_COL_ID.
	return masterpenyebabkerugian.CauseOfLoss{ID: id, Description: description}, nil
}

// Update mengganti keterangan golongan yang sudah ada.
func (r *Repo) Update(ctx context.Context, id, description string) (masterpenyebabkerugian.CauseOfLoss, error) {
	result, err := r.db.ExecContext(ctx, getQuery("cause_of_loss_update"), description, id)
	if err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, translateWriteError(err, "mengubah penyebab kerugian")
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak ada
	// berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi. Procedure lama melakukan persis itu — ia
	// mengembalikan `'Data Sudah Diupdate dengan ID : ' || IDPega` tanpa memeriksa satu
	// baris pun tersentuh.
	touched, err := result.RowsAffected()
	if err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, fmt.Errorf("masterpenyebabkerugian/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if touched == 0 {
		return masterpenyebabkerugian.CauseOfLoss{}, masterpenyebabkerugian.ErrNotFound
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang sebenarnya, termasuk
	// OLD_M_COL_ID yang tidak ikut diubah dan tidak diketahui pemanggil.
	return r.Get(ctx, id)
}

// issueID membentuk M_COL_ID persis seperti `Database/PEGA_M_CAUSE_OF_LOSS.prc:11` dan
// `:20`: kode situs disambung nomor urut tiga digit.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR
// termasuk yang dilarang docs/Steering/09-DATABASE-STRATEGY.md §4 karena keduanya
// mengikat kueri pada dialek Oracle.
func issueID(ctx context.Context, transaction *sql.Tx) (string, error) {
	var site string
	if err := transaction.QueryRowContext(ctx, getQuery("cause_of_loss_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, ID tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu berhenti seolah tidak
			// terjadi apa-apa; di sini ia menjadi galat yang benar-benar galat.
			return "", masterpenyebabkerugian.ErrNoSite
		}
		return "", fmt.Errorf("masterpenyebabkerugian/sqlstore: membaca kode situs: %w", err)
	}

	var order int64
	if err := transaction.QueryRowContext(ctx, getQuery("cause_of_loss_next_sequence")).Scan(&order); err != nil {
		return "", fmt.Errorf("masterpenyebabkerugian/sqlstore: mengambil nomor urut: %w", err)
	}

	return strings.TrimSpace(site) + ThreeDigits(order), nil
}

// ThreeDigits meniru lpad(to_char(seq), 3, '0') pada procedure lama.
//
// # Batas yang nyata, bukan teoretis
//
// Bilangan di atas 999 dikembalikan apa adanya, sama seperti LPAD Oracle — dan ID
// ke-1000 karena itu menjadi LIMA karakter. `Database/PEGA_M_CAUSE_OF_LOSS.prc:4`
// mendeklarasikan penampungnya `varchar2(4)`, yang menunjukkan kolomnya pun selebar itu;
// bila benar, penyisipannya akan DITOLAK basis data (ORA-12899), bukan diterima dengan ID
// aneh.
//
// Berbeda dari Master Status Klaim yang batasnya sudah diverifikasi ke katalog pada
// 2026-09-17, lebar `M_COL_ID` DAN posisi urutan `M_CAUSE_SEQ` di sini **belum pernah
// dibaca siapa pun di tim ini** (`R-08`). Keduanya termasuk yang diminta ke DBA bersama
// migrasi 0005 — bukan untuk menambal perilaku ini, melainkan supaya keputusan memperlebar
// kolom diambil sebelum, bukan sesudah, penyisipan pertama yang gagal.
//
// Dipotong menjadi tiga digit? Tidak. Itu akan menghasilkan ID GANDA, yang jauh lebih
// buruk daripada penyisipan yang gagal dengan pesan jelas.
//
// Diekspor supaya perilaku ini dapat diuji.
func ThreeDigits(n int64) string {
	digits := fmt.Sprintf("%d", n)
	for len(digits) < 3 {
		digits = "0" + digits
	}
	return digits
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

func scanRow(rows rowScanner) (masterpenyebabkerugian.CauseOfLoss, error) {
	var (
		id          string
		description sql.NullString
		legacyID    sql.NullString
	)
	if err := rows.Scan(&id, &description, &legacyID); err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, err
	}
	// COL_DESC dan OLD_M_COL_ID dibaca sebagai NullString karena keduanya boleh kosong:
	// keterangan kosong memang DITERIMA modul ini, dan penomoran lama hanya melekat pada
	// baris warisan. NULL dan string kosong diperlakukan sama.
	//
	// Ketiganya dirapikan di satu tempat: bila kolomnya ternyata CHAR, nilainya kembali
	// membawa padding.
	return masterpenyebabkerugian.CauseOfLoss{
		ID:          id,
		Description: description.String,
		LegacyID:    legacyID.String,
	}.Clean(), nil
}

// PrimaryKeyName adalah nama constraint kunci utama M_CAUSE_OF_LOSS.
//
// # Nilainya DITEBAK, dan itu harus diketahui sebelum dipercaya
//
// Nama yang benar belum pernah dibaca dari `ALL_CONSTRAINTS` — DDL tabelnya tidak ada di
// export (`R-08`). Yang di bawah mengikuti pola penamaan yang TERVERIFIKASI pada tabel
// sekerabat: kunci utama M_STS_CLAIM bernama `M_STS_CLAIM_PK`, bukan `PK_M_STS_CLAIM`
// (diperiksa 2026-09-17).
//
// Akibat bila tebakan ini meleset TIDAK merusak data, hanya memperburuk pesan: bentrok
// kunci utama akan muncul sebagai 500 alih-alih 409 yang menyuruh menyimpan sekali lagi.
// Keadaan itu sendiri seharusnya mustahil, karena ID dibentuk urutan basis data.
//
// Memastikannya satu kueri, dan ia sudah masuk daftar permintaan DBA bersama migrasi 0005:
//
//	SELECT CONSTRAINT_NAME FROM ALL_CONSTRAINTS
//	 WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'M_CAUSE_OF_LOSS' AND CONSTRAINT_TYPE = 'P';
const PrimaryKeyName = "M_CAUSE_OF_LOSS_PK"

// translateWriteError mengubah pelanggaran kunci utama menjadi galat domain.
//
// Tanpa penerjemahan ini, bentrok ID akan sampai ke pengguna sebagai 500 beserta nomor
// galat Oracle. Yang dicocokkan adalah NAMA CONSTRAINT, bukan nomor galat driver — nomor
// galat berubah saat driver atau basis datanya berganti, nama objek tidak.
//
// Tidak ada cabang untuk keunikan keterangan, dan itu disengaja: modul ini tidak membuat
// indeks unik apa pun karena keterangan ganda memang DITERIMA (keputusan Work Owner
// 2026-09-20).
func translateWriteError(err error, activity string) error {
	message := strings.ToUpper(err.Error())
	if strings.Contains(message, PrimaryKeyName) {
		return masterpenyebabkerugian.ErrIDTaken
	}
	return fmt.Errorf("masterpenyebabkerugian/sqlstore: %s: %w", activity, err)
}

var _ masterpenyebabkerugian.Repo = (*Repo)(nil)

// CheckTable menguji apakah kolom yang dipakai modul ini sudah ada dan dapat dibaca akun
// aplikasi, tanpa mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: migrasi 0005 belum dijalankan DBA, versus akun aplikasi tidak
// punya hak baca atas tabel warisan.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("cause_of_loss_check_table"))
	if err != nil {
		return err
	}
	return rows.Close()
}

// CountPendingJSON menghitung baris yang dokumen JSON-nya ada tetapi kolom keterangannya
// masih kosong.
//
// Angka ini menjawab pertanyaan yang tidak dapat dijawab CheckTable: apakah langkah
// pemindahan isi pada migrasi 0005 sudah berjalan, dan apakah layar Simas Online (MENU_ID
// 21) masih menambah baris yang belum ikut dipindahkan. Nol berarti seluruh baris siap.
//
// Dipakai HANYA oleh mode periksa, tidak pernah oleh jalur yang melayani pengguna.
func (r *Repo) CountPendingJSON(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("cause_of_loss_count_pending_json")).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
