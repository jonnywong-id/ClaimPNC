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

// RepoBaru membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func RepoBaru(db *sql.DB) *Repo { return &Repo{db: db} }

// Daftar membaca seluruh status klaim.
func (r *Repo) Daftar(ctx context.Context) ([]masterstatus.StatusKlaim, error) {
	baris, err := r.db.QueryContext(ctx, ambilKueri("status_klaim_daftar"))
	if err != nil {
		return nil, fmt.Errorf("masterstatus/sqlstore: membaca daftar status: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var hasil []masterstatus.StatusKlaim
	for baris.Next() {
		status, err := pindaiSatuBaris(baris)
		if err != nil {
			return nil, fmt.Errorf("masterstatus/sqlstore: membaca baris status: %w", err)
		}
		hasil = append(hasil, status)
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("masterstatus/sqlstore: menelusuri daftar status: %w", err)
	}
	return hasil, nil
}

// Ambil membaca satu status klaim.
func (r *Repo) Ambil(ctx context.Context, kode string) (masterstatus.StatusKlaim, error) {
	baris := r.db.QueryRowContext(ctx, ambilKueri("status_klaim_ambil"), kode)

	status, err := pindaiSatuBaris(baris)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterstatus.StatusKlaim{}, masterstatus.ErrTidakDitemukan
	case err != nil:
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/sqlstore: membaca status: %w", err)
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
func (r *Repo) Sisip(ctx context.Context, label string) (masterstatus.StatusKlaim, error) {
	transaksi, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = transaksi.Rollback() }()

	kode, err := terbitkanKode(ctx, transaksi)
	if err != nil {
		return masterstatus.StatusKlaim{}, err
	}

	if _, err := transaksi.ExecContext(ctx, ambilKueri("status_klaim_sisip"), kode, label); err != nil {
		return masterstatus.StatusKlaim{}, terjemahkanGalatTulis(err, "menyisipkan status")
	}
	if err := transaksi.Commit(); err != nil {
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/sqlstore: menyimpan status baru: %w", err)
	}

	return masterstatus.StatusKlaim{Kode: kode, Label: label}, nil
}

// Perbarui mengganti label status yang sudah ada.
func (r *Repo) Perbarui(ctx context.Context, kode, label string) (masterstatus.StatusKlaim, error) {
	hasil, err := r.db.ExecContext(ctx, ambilKueri("status_klaim_perbarui"), label, kode)
	if err != nil {
		return masterstatus.StatusKlaim{}, terjemahkanGalatTulis(err, "mengubah status")
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap kode yang tidak
	// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan"
	// atas perubahan yang tidak pernah terjadi.
	tersentuh, err := hasil.RowsAffected()
	if err != nil {
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if tersentuh == 0 {
		return masterstatus.StatusKlaim{}, masterstatus.ErrTidakDitemukan
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang sebenarnya, termasuk
	// OLD_LSC_ID yang tidak ikut diubah dan tidak diketahui pemanggil.
	return r.Ambil(ctx, kode)
}

// terbitkanKode membentuk LSC_ID persis seperti PEGA_M_STS_CLAIM.prc baris 11 dan 19:
// kode situs disambung nomor urut tiga digit.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR
// termasuk yang dilarang docs/Steering/09-DATABASE-STRATEGY.md §4 karena keduanya
// mengikat kueri pada dialek Oracle.
func terbitkanKode(ctx context.Context, transaksi *sql.Tx) (string, error) {
	var situs string
	if err := transaksi.QueryRowContext(ctx, ambilKueri("status_klaim_situs")).Scan(&situs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, kode tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu berhenti seolah tidak
			// terjadi apa-apa; di sini ia menjadi galat yang benar-benar galat.
			return "", errors.New("masterstatus/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
		}
		return "", fmt.Errorf("masterstatus/sqlstore: membaca kode situs: %w", err)
	}

	var urutan int64
	if err := transaksi.QueryRowContext(ctx, ambilKueri("status_klaim_urutan_berikutnya")).Scan(&urutan); err != nil {
		return "", fmt.Errorf("masterstatus/sqlstore: mengambil nomor urut: %w", err)
	}

	return strings.TrimSpace(situs) + TigaDigit(urutan), nil
}

// TigaDigit meniru lpad(to_char(seq), 3, '0') pada procedure lama.
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
func TigaDigit(n int64) string {
	angka := fmt.Sprintf("%d", n)
	for len(angka) < 3 {
		angka = "0" + angka
	}
	return angka
}

// pemindai menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type pemindai interface{ Scan(tujuan ...any) error }

func pindaiSatuBaris(baris pemindai) (masterstatus.StatusKlaim, error) {
	var (
		kode     string
		label    sql.NullString
		kodeLama sql.NullString
	)
	if err := baris.Scan(&kode, &label, &kodeLama); err != nil {
		return masterstatus.StatusKlaim{}, err
	}
	// Ketiganya dirapikan di satu tempat: OLD_LSC_ID tersimpan sebagai CHAR berisi
	// padding, dan label warisan dapat membawa spasi tepi.
	return masterstatus.StatusKlaim{
		Kode:     kode,
		Label:    label.String,
		KodeLama: kodeLama.String,
	}.Bersih(), nil
}

// Dua nama objek basis data yang dibaca saat menerjemahkan galat.
//
//   - NamaIndeksLabel dibuat migrasi 0002; ia yang menegakkan keunikan label.
//   - NamaKunciUtama SUDAH ADA di basis data. Namanya diverifikasi langsung dari
//     ALL_CONSTRAINTS pada 2026-09-17: `M_STS_CLAIM_PK`, bukan `PK_M_STS_CLAIM` seperti
//     yang saya tulis mula-mula dengan menebak dari pola penamaan migrasi 0001.
//
// Keduanya konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam.
const (
	NamaIndeksLabel = "UX_M_STS_CLAIM_LABEL"
	NamaKunciUtama  = "M_STS_CLAIM_PK"
)

// terjemahkanGalatTulis mengubah pelanggaran indeks unik menjadi galat domain.
//
// Tanpa penerjemahan ini, bentrok label akan sampai ke pengguna sebagai 500 beserta
// nomor galat Oracle. Yang dicocokkan adalah NAMA INDEKS yang kita buat sendiri di
// migrasi 0002, bukan nomor galat driver — nama itu milik kita dan tidak berubah saat
// driver atau basis datanya berganti.
func terjemahkanGalatTulis(err error, kegiatan string) error {
	pesan := strings.ToUpper(err.Error())
	switch {
	case strings.Contains(pesan, NamaIndeksLabel):
		return masterstatus.ErrLabelSudahAda
	case strings.Contains(pesan, NamaKunciUtama):
		// Kunci utama bentrok berarti nomor urut mengeluarkan kode yang sudah dipakai.
		// Seharusnya mustahil; bila terjadi ia harus terlihat, bukan menimpa baris lain.
		return masterstatus.ErrKodeSudahAda
	default:
		return fmt.Errorf("masterstatus/sqlstore: %s: %w", kegiatan, err)
	}
}

var _ masterstatus.Repo = (*Repo)(nil)

// PeriksaTabel menguji apakah kedua kolom yang ditambahkan migrasi 0002 sudah ada dan
// dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: migrasi belum dijalankan DBA, versus akun aplikasi tidak
// punya hak baca atas tabel warisan.
func (r *Repo) PeriksaTabel(ctx context.Context) error {
	baris, err := r.db.QueryContext(ctx, ambilKueri("status_klaim_periksa_tabel"))
	if err != nil {
		return err
	}
	return baris.Close()
}
