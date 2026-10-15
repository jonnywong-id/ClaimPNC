package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// BranchResolver memenuhi seam inboxkomunikasicabang.BranchResolver dengan SQL.
//
// Ia terpisah dari Repo dengan sengaja: Repo membaca percakapan di basis data satu entitas,
// sedangkan penerjemah ini membaca HRD dan master pengguna asuransi lewat DB Link. Keduanya
// dapat gagal sendiri-sendiri, dan menyatukannya membuat kegagalan yang satu tampak seperti
// kegagalan yang lain.
//
// Di layar ini pembedaan itu menentukan APA YANG DILIHAT petugas: sumber cabang yang mati
// menutup layarnya, sementara petugas yang sekadar tidak terdaftar tetap dilayani sebagai
// kantor pusat (`P-5`).
type BranchResolver struct {
	db *sql.DB
}

// branchTimeout membatasi lama penerjemahan cabang.
//
// # Kenapa ada batas waktu di sini, dan tidak di kueri lain modul ini
//
// Kueri lain membaca basis data entitas sendiri. Yang ini menembus DB LINK ke basis data
// lain — dan `10-API-STRATEGY.md` §8.2 menetapkan batas waktu WAJIB pada setiap pemanggilan
// keluar. DB Link adalah pemanggilan keluar yang menyamar sebagai kueri biasa: ia melewati
// jaringan, dan sambungan yang mati dapat menggantung sampai batas waktu TCP alih-alih
// menjawab galat.
//
// # Kenapa menggantung lebih buruk daripada galat
//
// Permintaan yang menggantung tidak terbedakan dari layar yang tidak bekerja. Tidak ada
// pesan apa pun, dan pengguna menyerah sebelum jawabannya tiba.
//
// Lima detik: cukup longgar untuk DB Link yang sehat pada jaringan kantor, cukup pendek
// untuk membuat kegagalannya terasa sebagai kegagalan, bukan sebagai kelambatan. Angkanya
// sama dengan modul Inbox Laporan Klaim, dan kesamaannya disengaja — keduanya menembus
// sambungan yang sama.
const branchTimeout = 5 * time.Second

// NewBranchResolver membentuk penerjemah; db wajib sudah terhubung.
func NewBranchResolver(db *sql.DB) *BranchResolver {
	return &BranchResolver{db: db}
}

// Resolve menerjemahkan login petugas menjadi kode cabang klaimnya.
//
// Tiga keluaran yang dibedakan, dan pemanggil memperlakukannya berbeda:
//
//	("1001", true,  nil)  cabangnya ditemukan
//	("",     false, nil)  petugasnya tidak terdaftar di HRD — BUKAN galat
//	("",     false, err)  sumbernya tidak dapat dibaca — DB Link mati, hak akses kurang
//
// Baris kedua terjadi untuk petugas non-karyawan: broker dan surveyor independen masuk lewat
// `POOLDATA.M_LOGIN_PNC` dan memang tidak pernah ada di HRD. Di layar ini mereka TETAP
// dilayani — sebagai kantor pusat, mengikuti precondition `KodeCabang == ""` pada
// `PNCCountKomunikasiCabang_Act` (`P-5`, keputusan Work Owner 2026-09-24).
func (r *BranchResolver) Resolve(ctx context.Context, login string) (string, bool, error) {
	clean := strings.TrimSpace(login)
	if clean == "" {
		return "", false, nil
	}

	ctx, cancel := context.WithTimeout(ctx, branchTimeout)
	defer cancel()

	var code sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("branch_of_login"), clean).Scan(&code)

	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// Dibedakan dari galat basis data biasa karena perbaikannya berbeda: yang ini
		// menunjuk sambungan ke HRD, bukan kueri maupun hak akses.
		return "", false, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: penerjemahan cabang %q tidak dijawab dalam %s; "+
				"DB Link ke HRD kemungkinan tidak hidup", clean, branchTimeout)
	}
	if err != nil {
		return "", false, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menerjemahkan cabang %q: %w", clean, err)
	}

	// Kolom CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa
	// pun. Kode yang tidak dipangkas tidak akan pernah cocok dengan penyaringnya sendiri.
	value := strings.TrimSpace(code.String)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

// CheckTable memastikan ketiga objek yang dibutuhkan dapat dibaca akun aplikasi.
//
// Ia dijalankan dengan login karangan yang pasti tidak ada, sehingga tidak mengembalikan
// satu baris pun — yang diuji adalah KETERBACAAN objeknya, bukan isinya. Dua di antaranya
// berada di basis data lain lewat DB Link, dan kegagalannya adalah kelas kegagalan
// tersendiri: bukan "migrasi belum jalan", melainkan "sambungan ke HRD tidak hidup".
func (r *BranchResolver) CheckTable(ctx context.Context) error {
	var code sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("branch_of_login"), "__periksa__").Scan(&code)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: BRANCH, LST_USER_ASURANSI, atau V_HRD_MST "+
				"tidak dapat dibaca: %w", err)
	}
	return nil
}

var _ inboxkomunikasicabang.BranchResolver = (*BranchResolver)(nil)
