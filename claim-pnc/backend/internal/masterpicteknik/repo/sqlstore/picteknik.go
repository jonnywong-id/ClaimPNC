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
func RepoBaru(db *sql.DB) *Repo { return &Repo{db: db} }

// Daftar membaca seluruh PIC teknik.
func (r *Repo) Daftar(ctx context.Context) ([]masterpicteknik.PICTeknik, error) {
	baris, err := r.db.QueryContext(ctx, ambilKueri("pic_teknik_daftar"))
	if err != nil {
		return nil, fmt.Errorf("masterpicteknik/sqlstore: membaca daftar PIC: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var hasil []masterpicteknik.PICTeknik
	for baris.Next() {
		p, err := pindaiSatuBaris(baris)
		if err != nil {
			return nil, fmt.Errorf("masterpicteknik/sqlstore: membaca baris PIC: %w", err)
		}
		hasil = append(hasil, p)
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("masterpicteknik/sqlstore: menelusuri daftar PIC: %w", err)
	}
	return hasil, nil
}

// Ambil membaca satu PIC teknik.
func (r *Repo) Ambil(ctx context.Context, idOperator string) (masterpicteknik.PICTeknik, error) {
	baris := r.db.QueryRowContext(ctx, ambilKueri("pic_teknik_ambil"), idOperator)

	p, err := pindaiSatuBaris(baris)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrTidakDitemukan
	case err != nil:
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/sqlstore: membaca PIC %q: %w", idOperator, err)
	}
	return p, nil
}

// Sisip menyimpan PIC teknik baru.
//
// Keberadaan baris diperiksa lebih dulu supaya bentrok dijawab ErrSudahAda, bukan galat
// kunci ganda dari driver yang tidak dapat dibedakan pemanggil. Ia tetap bukan jaminan:
// dua permintaan yang tiba bersamaan dapat sama-sama lolos, dan yang menahannya adalah
// primary key di basis data — yang galatnya juga diterjemahkan di sini.
func (r *Repo) Sisip(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	if _, err := r.Ambil(ctx, p.IDOperator); err == nil {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrSudahAda
	} else if !errors.Is(err, masterpicteknik.ErrTidakDitemukan) {
		return masterpicteknik.PICTeknik{}, err
	}

	_, err := r.db.ExecContext(ctx, ambilKueri("pic_teknik_sisip"),
		p.IDOperator,
		p.Nama,
		p.Email,
		p.LiniBisnis,
		p.Grup,
		p.Atasan,
		p.Kuota,
		p.KuotaLuar,
		sandiAktif(p.Aktif),
	)
	if err != nil {
		if kunciGanda(err) {
			return masterpicteknik.PICTeknik{}, masterpicteknik.ErrSudahAda
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/sqlstore: menyisipkan PIC: %w", err)
	}
	return r.Ambil(ctx, p.IDOperator)
}

// Perbarui mengubah PIC teknik yang sudah ada.
func (r *Repo) Perbarui(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	hasil, err := r.db.ExecContext(ctx, ambilKueri("pic_teknik_perbarui"),
		p.Nama,
		p.Email,
		p.LiniBisnis,
		p.Grup,
		p.Atasan,
		p.Kuota,
		p.KuotaLuar,
		sandiAktif(p.Aktif),
		p.IDOperator,
	)
	if err != nil {
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/sqlstore: memperbarui PIC: %w", err)
	}

	// Sebagian driver tidak melaporkan jumlah baris yang tersentuh. Ketiadaan angka
	// BUKAN bukti tidak ada yang berubah, sehingga hanya angka nol yang sungguhan
	// dianggap "tidak ditemukan".
	if jumlah, err := hasil.RowsAffected(); err == nil && jumlah == 0 {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrTidakDitemukan
	}
	return r.Ambil(ctx, p.IDOperator)
}

// PeriksaTabel memastikan seluruh kolom dapat dibaca akun aplikasi, tanpa mengambil satu
// baris pun. Dipakai mode periksa pada binary.
func (r *Repo) PeriksaTabel(ctx context.Context) error {
	baris, err := r.db.QueryContext(ctx, ambilKueri("pic_teknik_periksa_tabel"))
	if err != nil {
		return fmt.Errorf("masterpicteknik/sqlstore: memeriksa tabel MST_USER_TEKNIK: %w", err)
	}
	defer func() { _ = baris.Close() }()
	return baris.Err()
}

// DirektoriRepo memenuhi seam DirektoriOperator dengan DATAPEGA.PR_OPERATORS.
//
// Tabel itu milik Pega dan hanya DIBACA (ADR-0004). Ia dipisahkan dari Repo karena
// sumbernya memang tabel lain dengan pemilik lain — menyatukannya akan menyamarkan
// bahwa satu di antaranya kita tulis dan satu lagi tidak.
type DirektoriRepo struct {
	db *sql.DB
}

// DirektoriRepoBaru membentuk pembaca direktori operator.
func DirektoriRepoBaru(db *sql.DB) *DirektoriRepo { return &DirektoriRepo{db: db} }

// NamaOperator mencari nama petugas di direktori operator.
func (d *DirektoriRepo) NamaOperator(ctx context.Context, idOperator string) (string, error) {
	var nama sql.NullString

	err := d.db.QueryRowContext(ctx, ambilKueri("operator_nama"), idOperator).Scan(&nama)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", masterpicteknik.ErrOperatorTidakDikenal
	case err != nil:
		// Kegagalan kueri terhadap tabel milik pihak lain diperlakukan sebagai
		// direktori tidak terhubung, bukan sebagai "tidak terdaftar": keduanya menuntut
		// tindak lanjut yang berbeda, dan menyamakannya akan menyuruh pengguna
		// memperbaiki ID yang sebenarnya sudah benar.
		return "", fmt.Errorf("%w: %v", masterpicteknik.ErrDirektoriTidakTerhubung, err)
	}

	bersih := strings.TrimSpace(nama.String)
	if bersih == "" {
		// Baris ada tetapi namanya kosong. Sistem lama pun menolaknya — prasyaratnya
		// `TempDcol.MCL_NAME==""`, bukan "baris tidak ditemukan".
		return "", masterpicteknik.ErrOperatorTidakDikenal
	}
	return bersih, nil
}

// pemindai menyatukan *sql.Row dan *sql.Rows sehingga satu fungsi pemindaian melayani
// keduanya. Tanpa ini, sepuluh kolom harus ditulis dua kali dan kedua salinannya harus
// diingat untuk diubah bersama-sama.
type pemindai interface {
	Scan(tujuan ...any) error
}

func pindaiSatuBaris(p pemindai) (masterpicteknik.PICTeknik, error) {
	var (
		id, nama, email, lini, grup, atasan, grupPanel, aktif sql.NullString
		kuota, kuotaLuar                                      sql.NullInt64
	)

	if err := p.Scan(&id, &nama, &email, &lini, &grup, &atasan, &kuota, &kuotaLuar, &grupPanel, &aktif); err != nil {
		return masterpicteknik.PICTeknik{}, err
	}

	return masterpicteknik.PICTeknik{
		IDOperator: teks(id),
		Nama:       teks(nama),
		Email:      teks(email),
		LiniBisnis: teks(lini),
		Grup:       teks(grup),
		Atasan:     teks(atasan),
		Kuota:      int(kuota.Int64),
		KuotaLuar:  int(kuotaLuar.Int64),
		GrupPanel:  teks(grupPanel),
		Aktif:      bacaAktif(teks(aktif)),
	}, nil
}

// sandiAktif memetakan status aktif ke isi kolom STS_AKTIF.
//
// "1" dan "0" — BUKAN "Ya"/"Tidak" seperti kolom bernama sama pada POOLDATA.LST_ACCOUNT.
// Dua tabel berbeda memakai sandi berbeda, dan itu justru alasan pemetaannya ditulis
// sebagai fungsi bernama di sini alih-alih diketik ulang di setiap pemanggilan.
func sandiAktif(aktif bool) string {
	if aktif {
		return masterpicteknik.SandiAktif
	}
	return "0"
}

// bacaAktif menafsirkan isi kolom STS_AKTIF.
//
// Menerima beberapa ejaan yang mungkin sudah telanjur ada di tabel. Menolak mengenali
// baris lama hanya karena ejaannya berbeda akan menampilkan petugas aktif sebagai
// nonaktif — dan petugas nonaktif tidak menerima penugasan, sehingga kesalahan itu
// langsung berakibat pada pembagian kerja.
func bacaAktif(nilai string) bool {
	switch strings.ToUpper(strings.TrimSpace(nilai)) {
	case "1", "Y", "YA", "AKTIF", "A":
		return true
	default:
		return false
	}
}

func teks(n sql.NullString) string {
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
func kunciGanda(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ORA-00001")
}

var (
	_ masterpicteknik.Repo              = (*Repo)(nil)
	_ masterpicteknik.DirektoriOperator = (*DirektoriRepo)(nil)
)
