package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/dashboardclaim"
)

// AssignmentWriter memindahkan PIC Teknik klaim pada basis data satu portal.
//
// Dipisah dari `Repo` dengan alasan yang sama seperti seam-nya di domain: ia satu-satunya
// bagian aplikasi ini yang MENULIS, bukan hanya membaca.
type AssignmentWriter struct {
	db *sql.DB
}

// NewAssignmentWriter membungkus satu koneksi.
func NewAssignmentWriter(db *sql.DB) *AssignmentWriter { return &AssignmentWriter{db: db} }

// MovePIC memindahkan PIC Teknik satu klaim beserta pencacah bebannya.
//
// Keempat tulisannya berada dalam SATU transaksi. Bila salah satunya gagal, tidak ada yang
// berubah — bukan karena kerapian, melainkan karena keadaan setengah jalan di sini tidak
// dapat dibedakan dari keadaan yang sah: PIC yang sudah berpindah sementara pencacahnya
// belum tampak persis seperti pemindahan yang berhasil.
func (w *AssignmentWriter) MovePIC(
	ctx context.Context,
	move dashboardclaim.PICMove,
) (dashboardclaim.PICMoveResult, error) {
	claimID := strings.TrimSpace(move.ClaimID)
	toOperator := strings.TrimSpace(move.ToOperator)

	if claimID == "" || toOperator == "" {
		return dashboardclaim.PICMoveResult{}, fmt.Errorf(
			"sqlstore: pemindahan PIC menuntut klaim dan petugas tujuan")
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return dashboardclaim.PICMoveResult{}, fmt.Errorf("sqlstore: membuka transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat. Sesudah Commit berhasil ia tidak berbuat apa-apa,
	// dan pada setiap jalur keluar lain ia yang memulihkan.
	defer func() { _ = tx.Rollback() }()

	// PIC lama dibaca DI DALAM transaksi, dengan kunci baris. Membacanya di luar membuat dua
	// pemindahan bersamaan sama-sama melihat PIC lama yang sama, lalu keduanya menurunkan
	// pencacah orang yang sama dua kali.
	var current sql.NullString
	err = tx.QueryRowContext(ctx, query("pic_sekarang"), claimID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return dashboardclaim.PICMoveResult{}, dashboardclaim.ErrClaimNotFound
	}
	if err != nil {
		// `FOR UPDATE` menuntut hak MENULIS, bukan hanya membaca — jadi kekurangan GRANT
		// sudah tertangkap di sini, sebelum satu kolom pun berubah.
		if needsDBA(err) {
			return dashboardclaim.PICMoveResult{}, fmt.Errorf(
				"%w: %v", dashboardclaim.ErrAssignmentUnavailable, err)
		}
		return dashboardclaim.PICMoveResult{}, fmt.Errorf("sqlstore: membaca PIC berjalan: %w", err)
	}

	from := strings.TrimSpace(text(current))

	// Memindahkan ke petugas yang sama bukan galat, tetapi juga tidak boleh menggeser
	// pencacah: beban orang itu tidak bertambah hanya karena tombolnya ditekan dua kali.
	if from == toOperator {
		if err := tx.Commit(); err != nil {
			return dashboardclaim.PICMoveResult{}, fmt.Errorf("sqlstore: menutup transaksi: %w", err)
		}
		return dashboardclaim.PICMoveResult{FromOperator: from}, nil
	}

	if _, err := tx.ExecContext(ctx, query("pindah_pic"), toOperator, claimID); err != nil {
		// Haknya diminta per kolom dan terpisah — lihat pindahpic.sql. Selama GRANT itu
		// belum diberikan, setiap penekanan Assign gagal DI SINI.
		if needsDBA(err) {
			return dashboardclaim.PICMoveResult{}, fmt.Errorf(
				"%w: %v", dashboardclaim.ErrAssignmentUnavailable, err)
		}
		return dashboardclaim.PICMoveResult{}, fmt.Errorf("sqlstore: memindahkan PIC: %w", err)
	}

	if _, err := tx.ExecContext(ctx, query("pencacah_naik"), toOperator); err != nil {
		if needsDBA(err) {
			return dashboardclaim.PICMoveResult{}, fmt.Errorf(
				"%w: %v", dashboardclaim.ErrAssignmentUnavailable, err)
		}
		return dashboardclaim.PICMoveResult{}, fmt.Errorf("sqlstore: menaikkan beban PIC baru: %w", err)
	}

	// Beban PIC LAMA tidak diturunkan, dan itu bukan kelalaian.
	//
	// Kueri Pega-nya (`UpdateTotalJobChild_sql`) menurunkan `TOTAL_JOB` saja — bukan
	// `COUNTER_QUOTA`. `TOTAL_JOB` tidak ada pada tabel yang dapat kita tulis, sehingga tidak
	// ada satu kolom pun yang tersisa untuk diturunkan. Rinciannya di `pindahpic.sql`.
	//
	// Menurunkan `COUNTER_QUOTA` sebagai gantinya DITOLAK: ia pencacah yang di sistem lama
	// memang tidak pernah berkurang, dan ia yang menentukan pemilihan petugas otomatis
	// (`R-04`). Menurunkannya mengubah siapa yang terpilih — perubahan perilaku yang
	// menyamar sebagai perbaikan.

	// Tabel ringkasan dashboard dikunci nomor klaim, bukan PZINSKEY. Klaim tanpa nomor —
	// tidak seharusnya ada, tetapi data warisan menyimpannya — dilewati alih-alih
	// menggagalkan seluruh pemindahan.
	if number := strings.TrimSpace(move.ClaimNumber); number != "" {
		if _, err := tx.ExecContext(ctx, query("dashboard_pic"), toOperator, number); err != nil {
			return dashboardclaim.PICMoveResult{}, fmt.Errorf(
				"sqlstore: memperbarui PIC pada ringkasan dashboard: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return dashboardclaim.PICMoveResult{}, fmt.Errorf("sqlstore: menutup transaksi: %w", err)
	}

	return dashboardclaim.PICMoveResult{FromOperator: from}, nil
}

// MoveAllForOperator memindahkan seluruh klaim berjalan milik satu petugas.
//
// Satu pernyataan, bukan perulangan per klaim. Jumlah klaim seorang petugas tidak diketahui
// di muka, dan memutarnya satu per satu membuat kegagalan di tengah meninggalkan sebagian
// berpindah dan sebagian tidak — keadaan yang tidak dapat dibedakan dari pemindahan yang
// berhasil.
//
// # Pencacah beban TIDAK digeser di sini
//
// Pada jalur per baris, `COUNTER_QUOTA` ikut bergerak karena Pega pun menggesernya
// (`AddTJobCQuota_SQL`). Jalur massal di Pega menempuh activity yang BERBEDA —
// `GCNMTransferDataKlaim_act` — dan activity itu tidak memanggil kedua kueri pencacah.
//
// Dibiarkan sama: menggesernya di sini akan menambahkan perilaku yang di layar lama tidak
// ada, dan pencacah yang bergerak lebih banyak daripada semestinya memengaruhi pemilihan
// petugas otomatis (`R-04`).
func (w *AssignmentWriter) MoveAllForOperator(ctx context.Context, from, to string) (int, error) {
	fromOperator := strings.TrimSpace(from)
	toOperator := strings.TrimSpace(to)

	if fromOperator == "" || toOperator == "" {
		return 0, fmt.Errorf("sqlstore: pemindahan massal menuntut petugas asal dan tujuan")
	}
	if fromOperator == toOperator {
		// Bukan galat, tetapi juga tidak ada yang berpindah. Menjalankan UPDATE-nya akan
		// melaporkan baris "terpengaruh" yang sebenarnya tidak berubah apa pun.
		return 0, nil
	}

	result, err := w.db.ExecContext(ctx, query("pindah_pic_massal"), toOperator, fromOperator)
	if err != nil {
		if needsDBA(err) {
			return 0, fmt.Errorf("%w: %v", dashboardclaim.ErrAssignmentUnavailable, err)
		}
		return 0, fmt.Errorf("sqlstore: memindahkan seluruh klaim petugas: %w", err)
	}

	// Jumlah baris dipakai LAYAR untuk menyatakan berapa klaim yang berpindah. Driver yang
	// tidak mendukungnya mengembalikan galat; itu bukan kegagalan pemindahan, jadi
	// pemindahannya tetap dinyatakan berhasil dengan jumlah tidak diketahui.
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return int(affected), nil
}

// CheckPICMovable membuktikan kolom PIC Teknik benar-benar dapat ditulis.
//
// Dipanggil `claimpnc -periksa`. Tanpa baris ini, satu-satunya cara mengetahuinya adalah
// menekan tombol Transfer dan menerima `503` — dan yang menemukannya pengguna, bukan operator.
//
// Pernyataannya TIDAK mengubah satu baris pun (`WHERE 1 = 0`), tetapi Oracle tetap memeriksa
// nama kolom DAN hak akses saat mem-parse-nya. Lihat `pindahpic.sql` untuk alasan ia berupa
// UPDATE dan bukan SELECT.
//
// # KOREKSI (2026-10-08): kegagalan di sini jarang soal GRANT
//
// Keterangan sebelumnya menyatakan kegagalan berarti "GRANT belum diberikan". Itu keliru untuk
// keadaan yang sebenarnya: `POOLDATA_ASM_PENGGUNA=POOLDATA` — aplikasi menyambung **sebagai
// pemilik skema**, dan pemilik skema sudah memiliki seluruh hak atas objeknya sendiri.
// `GRANT … TO POOLDATA` bahkan ditolak Oracle dengan ORA-01749.
//
// Penyebab yang jauh lebih mungkin adalah **nama kolom**, dan itu sudah pernah terjadi:
// `pencacah_naik` sempat menulis `TOTAL_JOB`, kolom yang ternyata hanya ada pada view
// `V_MST_USER_TEKNIS`. Pesan di bawah karena itu menyebut kolom lebih dulu, baru GRANT.
func (w *AssignmentWriter) CheckPICMovable(ctx context.Context) error {
	if _, err := w.db.ExecContext(ctx, query("pindah_pic_check")); err != nil {
		return fmt.Errorf(
			"kolom POOLDATA.T_CLAIMLIST_ADMIN(USERTEKNIS_1) tidak dapat ditulis.\n"+
				"Periksa NAMA KOLOM lebih dulu: bila aplikasi menyambung sebagai pemilik skema "+
				"POOLDATA — dan pada ASM memang demikian — sebabnya bukan hak akses.\n"+
				"Hanya bila ia menyambung sebagai akun NON-pemilik, hak ini perlu diminta ke DBA "+
				"(`D-63`), di basis data setiap entitas:\n"+
				"    GRANT UPDATE (USERTEKNIS_1) ON POOLDATA.T_CLAIMLIST_ADMIN TO <akun>;\n"+
				"galat: %w",
			err)
	}
	return nil
}

// CheckWorkloadWritable membuktikan pencacah beban PIC dapat ditulis.
//
// Dipanggil `claimpnc -periksa`. Ia terpisah dari CheckPICMovable karena yang gagal berbeda
// dan akibatnya berbeda: tanpa hak pada tabel kerja Pega, pemindahan tidak terjadi sama
// sekali; tanpa kolom pencacah, pemindahannya berhasil lalu dibatalkan di tengah transaksi.
//
// Ia menyebut `COUNTER_QUOTA` — kolom yang sama dengan `pencacah_naik`. Probe yang hanya
// membuktikan tabelnya ada akan lulus terhadap kolom yang tidak ada, dan itulah kegagalan
// yang membuat probe ini ditulis: `TOTAL_JOB` ternyata tidak ada pada tabel ini.
func (w *AssignmentWriter) CheckWorkloadWritable(ctx context.Context) error {
	if _, err := w.db.ExecContext(ctx, query("pencacah_check")); err != nil {
		return fmt.Errorf(
			"pencacah beban POOLDATA.MST_USER_TEKNIK(COUNTER_QUOTA) tidak dapat ditulis akun "+
				"aplikasi; tanpa itu setiap pemindahan PIC Teknik batal di tengah transaksi: %w",
			err)
	}
	return nil
}

// CheckDashboardWritable membuktikan tabel ringkasan dashboard dapat ditulis.
//
// DITAMBAHKAN 2026-10-08, menutup lubang yang ditemukan saat menjawab pertanyaan Work Owner
// "kenapa masih memakai DATAPEGA.PC_ASM_FW_GCNMFW_WORK": jalur Transfer ternyata menulis TIGA
// tabel, bukan satu, dan hanya dua yang punya probe.
//
// Tanpa baris ini `-periksa` melaporkan seluruhnya hijau sementara Transfer tetap gagal — persis
// kegagalan yang membuat kedua probe lain ditulis.
func (w *AssignmentWriter) CheckDashboardWritable(ctx context.Context) error {
	if _, err := w.db.ExecContext(ctx, query("dashboard_pic_check")); err != nil {
		return fmt.Errorf(
			"tabel ringkasan POOLDATA.PEGA_DASHBOARDPNC(PIC) tidak dapat ditulis akun aplikasi; "+
				"tanpa itu setiap pemindahan PIC Teknik batal di tengah transaksi: %w",
			err)
	}
	return nil
}

// MoveAllMatching memindahkan SELURUH klaim yang cocok dengan penyaring layar.
//
// Inilah yang dijalankan "Select All" ketika pengguna memilih seluruh hasil, bukan 25 baris
// pada halaman yang terbuka. Satu pernyataan, bukan perulangan: pada data hari ini pilihannya
// 1.639 klaim, dan 1.639 permintaan terpisah membuat kegagalan di tengah meninggalkan
// sebagian berpindah — keadaan yang tidak dapat dibedakan dari pemindahan yang berhasil.
//
// Penyaringnya dibaca dari layar dan diikat dengan urutan yang SAMA dengan kueri daftar,
// lewat `outstandingFilterArgs`. Itu bukan kerapian: argumen yang bergeser satu posisi akan
// memindahkan himpunan klaim yang berbeda dari yang dilihat pengguna, tanpa galat apa pun.
func (w *AssignmentWriter) MoveAllMatching(
	ctx context.Context,
	filter dashboardclaim.Filter,
	to string,
) (int, error) {
	toOperator := strings.TrimSpace(to)
	if toOperator == "" {
		return 0, fmt.Errorf("sqlstore: pemindahan menuntut petugas tujuan")
	}

	// Operator tujuan menempati :1; penyaringnya menyusul mulai :2.
	args := append([]any{toOperator}, outstandingFilterArgs(filter.Normalize())...)

	result, err := w.db.ExecContext(ctx, query("pindah_pic_saring"), args...)
	if err != nil {
		if needsDBA(err) {
			return 0, fmt.Errorf("%w: %v", dashboardclaim.ErrAssignmentUnavailable, err)
		}
		return 0, fmt.Errorf("sqlstore: memindahkan klaim yang cocok penyaring: %w", err)
	}

	moved, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak melaporkan jumlah baris bukan kegagalan pemindahan — barisnya
		// sudah berpindah. Yang hilang hanya angkanya, dan layar menyatakannya apa adanya.
		return 0, nil
	}
	return int(moved), nil
}
