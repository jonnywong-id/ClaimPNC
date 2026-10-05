package sqlstore

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/dokumenpenunjang"
)

// Repo memenuhi dokumenpenunjang.Repo.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib koneksi portal yang dituju.
//
// Portal menentukan basis data mana yang dipakai (`D-75`), dan dari basis data itulah DB
// link ke `GENERAL` ditempuh. Jadi dokumen sebuah klaim selalu dicari lewat jalur portalnya
// sendiri — bukan lewat satu jalur bersama yang akan mencampur empat badan hukum.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// NamaFolderAplikasi memenuhi dokumenpenunjang.Repo.
func (r *Repo) NamaFolderAplikasi(ctx context.Context, aplikasi string) (string, error) {
	var nama sql.NullString
	err := r.db.QueryRowContext(ctx, query("folder_aplikasi"), kunci(aplikasi)).Scan(&nama)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("%w: %s", dokumenpenunjang.ErrFolderAplikasiTidakAda, aplikasi)
	}
	if err != nil {
		return "", fmt.Errorf(
			"dokumenpenunjang/sqlstore: mencari folder penyimpanan: %w", linkError(err))
	}

	rapi := strings.TrimSpace(nama.String)
	if rapi == "" {
		// Barisnya ada tetapi kosong. Diperlakukan sama dengan tidak ada: mengirim nama
		// aplikasi kosong ke layanan penyimpanan menaruh berkas entah di mana.
		return "", fmt.Errorf("%w: %s (baris ada, NAMA_FOLDER kosong)",
			dokumenpenunjang.ErrFolderAplikasiTidakAda, aplikasi)
	}
	return rapi, nil
}

// CatatAksesUnggah memenuhi dokumenpenunjang.Repo.
//
// Tokennya dibentuk di sini, meniru `GENERAL.GET_TOKEN_STORAGE`:
//
//	standard_hash('ASMAPP' || SYSTIMESTAMP, 'MD5')
//
// MD5 dipakai karena itulah yang procedure-nya pakai dan nilainya harus sebentuk dengan
// baris yang sudah ada. Ia BUKAN pengaman — tokennya tidak pernah dikirim ke mana pun dan
// tidak pernah diperiksa; ia penanda baris. Karena itu kelemahan MD5 tidak berlaku di sini,
// dan menggantinya justru membuat barisnya berbeda bentuk dari 33 baris yang sudah ada.
func (r *Repo) CatatAksesUnggah(ctx context.Context, aplikasi, pengunggah string) (string, error) {
	token := tokenAkses(time.Now())
	_, err := r.db.ExecContext(ctx, query("catat_akses_unggah"),
		strings.TrimSpace(aplikasi),
		token,
		strings.TrimSpace(pengunggah),
	)
	if err != nil {
		return "", fmt.Errorf("dokumenpenunjang/sqlstore: mencatat izin unggah: %w", linkError(err))
	}
	return token, nil
}

// Simpan memenuhi dokumenpenunjang.Repo.
func (r *Repo) Simpan(ctx context.Context, dokumen dokumenpenunjang.Document) error {
	_, err := r.db.ExecContext(ctx, query("simpan_metadata"),
		strings.TrimSpace(dokumen.ImageID),
		nullIfEmpty(dokumen.URL),
		nullIfEmpty(dokumen.Folder),
		nullTime(dokumen.ExpiresAt),
		nullIfEmpty(dokumen.FileName),
		nullIfEmpty(namaAplikasiDari(dokumen)),
		dokumenpenunjang.JenisPenyimpanan,
		nullIfEmpty(dokumen.DocumentType),
		dokumenpenunjang.NomorKlaimUntukPenyimpanan(dokumen.ClaimNumber),
		nullTime(dokumen.UploadedAt),
	)
	if err != nil {
		return fmt.Errorf("dokumenpenunjang/sqlstore: mencatat metadata dokumen: %w", err)
	}
	return nil
}

// PerKlaim memenuhi dokumenpenunjang.Repo.
func (r *Repo) PerKlaim(
	ctx context.Context,
	nomorKlaim string,
) ([]dokumenpenunjang.Document, error) {
	rows, err := r.db.QueryContext(ctx, query("dokumen_per_klaim"),
		kunci(nomorKlaim), kunci(namaAplikasiPenyimpanan))
	if err != nil {
		return nil, fmt.Errorf("dokumenpenunjang/sqlstore: membaca dokumen klaim: %w", linkError(err))
	}
	defer rows.Close()

	daftar := make([]dokumenpenunjang.Document, 0, 8)
	for rows.Next() {
		dokumen, err := pindai(rows)
		if err != nil {
			return nil, err
		}
		daftar = append(daftar, dokumen)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"dokumenpenunjang/sqlstore: membaca baris dokumen klaim: %w", err)
	}
	return daftar, nil
}

// Ambil memenuhi dokumenpenunjang.Repo.
func (r *Repo) Ambil(
	ctx context.Context,
	imageID string,
) (dokumenpenunjang.Document, error) {
	rows, err := r.db.QueryContext(ctx, query("dokumen_menurut_imageid"),
		strings.TrimSpace(imageID))
	if err != nil {
		return dokumenpenunjang.Document{}, fmt.Errorf(
			"dokumenpenunjang/sqlstore: membaca dokumen: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return dokumenpenunjang.Document{}, fmt.Errorf(
				"dokumenpenunjang/sqlstore: membaca dokumen: %w", err)
		}
		return dokumenpenunjang.Document{}, dokumenpenunjang.ErrTidakDitemukan
	}
	return pindai(rows)
}

// PerbaruiTautan memenuhi dokumenpenunjang.Repo.
func (r *Repo) PerbaruiTautan(ctx context.Context, imageID string, hasil dokumenpenunjang.HasilUnggah) error {
	_, err := r.db.ExecContext(ctx, query("perbarui_tautan"),
		strings.TrimSpace(hasil.URL),
		nullTime(hasil.ExpiresAt),
		nullIfEmpty(hasil.Folder),
		strings.TrimSpace(imageID),
	)
	if err != nil {
		return fmt.Errorf("dokumenpenunjang/sqlstore: menyimpan alamat baru: %w", linkError(err))
	}
	return nil
}

// namaAplikasiPenyimpanan adalah nilai `APPNAME` yang modul ini tulis dan baca.
//
// # Kenapa ia konstanta di sini, bukan hasil pencarian seperti saat mengunggah
//
// Saat MENGUNGGAH, nama aplikasi dicari di master supaya berkasnya mendarat di folder yang
// benar. Saat MEMBACA, yang dibutuhkan adalah penyaring: dokumen mana yang milik aplikasi
// ini. Keduanya kebetulan bernilai sama hari ini, dan menyamakannya dalam kode akan membuat
// perubahan master diam-diam menyembunyikan seluruh dokumen lama.
//
// Nilainya diambil dari data produksi: 2.050 baris ber-`APPNAME = 'klaimpnc'`.
const namaAplikasiPenyimpanan = "klaimpnc"

// namaAplikasiDari menentukan APPNAME yang ditulis.
//
// Folder yang dipakai saat unggah sudah tersimpan pada dokumen; APPNAME mengikutinya supaya
// yang ditulis dan yang dibaca tetap satu nilai.
func namaAplikasiDari(dokumenpenunjang.Document) string { return namaAplikasiPenyimpanan }

// pindai membaca satu baris menjadi Document.
//
// Satu fungsi melayani KEDUA kueri, dan itu sebabnya daftar kolom keduanya dibuat sama
// persis. Dua pemindai untuk bentuk yang sama adalah dua tempat yang dapat berbeda.
func pindai(rows *sql.Rows) (dokumenpenunjang.Document, error) {
	var (
		imageID           string
		nama, url, folder sql.NullString
		exp               sql.NullTime
		tipe, nomorKlaim  sql.NullString
		diunggah          sql.NullTime
	)
	if err := rows.Scan(&imageID, &nama, &url, &folder, &exp,
		&tipe, &nomorKlaim, &diunggah); err != nil {
		return dokumenpenunjang.Document{}, fmt.Errorf(
			"dokumenpenunjang/sqlstore: memindai dokumen: %w", err)
	}

	dokumen := dokumenpenunjang.Document{
		ImageID:      strings.TrimSpace(imageID),
		FileName:     teks(nama),
		URL:          teks(url),
		Folder:       teks(folder),
		DocumentType: teks(tipe),
		ClaimNumber:  teks(nomorKlaim),
	}
	if exp.Valid {
		waktu := exp.Time
		dokumen.ExpiresAt = &waktu
	}
	if diunggah.Valid {
		waktu := diunggah.Time
		dokumen.UploadedAt = &waktu
	}
	return dokumen, nil
}

// tokenAkses meniru `standard_hash('ASMAPP' || SYSTIMESTAMP, 'MD5')`.
//
// Ketelitian nanodetik dipakai supaya dua unggahan pada detik yang sama tetap menghasilkan
// token berbeda — sama seperti `SYSTIMESTAMP` yang beresolusi mikrodetik. Token yang kembar
// tidak merusak apa pun hari ini, tetapi ia menghapus kemampuan membedakan dua baris jejak.
func tokenAkses(saat time.Time) string {
	sidik := md5.Sum([]byte("ASMAPP" + saat.Format("2006-01-02 15:04:05.000000000")))
	return strings.ToUpper(hex.EncodeToString(sidik[:]))
}

// kunci menormalkan nilai pembanding menjadi huruf besar tanpa spasi tepi.
// oracleNetworkCodes adalah galat Oracle Net: yang gagal sambungannya, bukan kuerinya.
// Lewat DB link, itulah tanda database ASMD di ujung link tidak dapat dijangkau.
var oracleNetworkCodes = []string{
	"ORA-12541", "ORA-12543", "ORA-12545", "ORA-12170", "ORA-12514", "ORA-12537",
	"ORA-12560", "ORA-12154", "ORA-02019", "ORA-03113", "ORA-03135",
}

// linkError menandai galat sambungan DB link sebagai ErrLinkTakTerjangkau, tanpa
// membuang galat aslinya (yang tetap masuk log).
func linkError(err error) error {
	message := err.Error()
	for _, code := range oracleNetworkCodes {
		if strings.Contains(message, code) {
			return fmt.Errorf("%w: %w", dokumenpenunjang.ErrLinkTakTerjangkau, err)
		}
	}
	return err
}

func kunci(v string) string { return strings.ToUpper(strings.TrimSpace(v)) }

// teks mengubah kolom yang boleh NULL menjadi string rapi.
func teks(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}

// nullIfEmpty mengirim NULL alih-alih teks kosong.
//
// Oracle memperlakukan teks kosong SEBAGAI NULL pada VARCHAR2, sehingga keduanya berakhir
// sama di sana. Yang dijaga adalah niatnya tetap terbaca dari kode, dan adapter ini tetap
// benar bila kelak dijalankan terhadap PostgreSQL — yang MEMBEDAKAN keduanya (`D-20`).
func nullIfEmpty(v string) any {
	if rapi := strings.TrimSpace(v); rapi != "" {
		return rapi
	}
	return nil
}

// nullTime mengirim NULL untuk waktu yang tidak ada.
func nullTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}
