package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/riwayatklaim"
)

// ProtectionRepo memenuhi seam riwayatklaim.ProtectionRepo.
//
// Ia menyentuh DUA tabel dengan kepemilikan yang berbeda, dan pembedaannya bukan detail
// teknis melainkan bentuk nyata `P-1`:
//
//	POOLDATA.MST_PROTEKSI_DATA_PNC    milik sistem lama  — HANYA DIBACA
//	POOLDATA.CPNC_PEMAKAIAN_PROTEKSI  milik aplikasi ini — ditulis (migrasi 0004)
//
// Alasan penulisannya tidak diarahkan ke tabel lama ada di kepala proteksi.sql.
type ProtectionRepo struct {
	db *sql.DB
}

// NewProtectionRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang
// dituju. Jatah proteksi seorang pengguna di satu badan hukum bukan jatahnya di badan
// hukum lain, sehingga koneksinya per entitas seperti seluruh data bisnis lain.
func NewProtectionRepo(db *sql.DB) *ProtectionRepo { return &ProtectionRepo{db: db} }

// TableName adalah nama tabel yang dibuat migrasi 0004.
//
// Ia konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam: mengganti
// namanya di migrasi tanpa mengganti konstanta ini akan membuat mode -periksa melaporkan
// tabelnya belum ada padahal sudah.
const TableName = "CPNC_PEMAKAIAN_PROTEKSI"

// Find membaca baris proteksi milik satu pengguna untuk satu modul.
//
// Ketiadaan baris dikembalikan sebagai (kosong, false, nil) — BUKAN galat. Ia keadaan yang
// sah dan sering: setiap pengguna yang belum didaftarkan di Master Proteksi Data akan
// mengalaminya, dan memperlakukannya sebagai galat akan mencatatnya sebagai kerusakan
// sistem di log padahal ia keputusan administrasi.
func (r *ProtectionRepo) Find(
	ctx context.Context,
	login, module string,
) (riwayatklaim.Protection, bool, error) {
	var (
		searchQuota sql.NullInt64
		viewQuota   sql.NullInt64
		maskPhone   sql.NullString
		maskEmail   sql.NullString
		maskIDCard  sql.NullString
		subModules  sql.NullString
	)

	row := r.db.QueryRowContext(ctx, query("protection_find"),
		strings.TrimSpace(login), strings.TrimSpace(module))

	if err := row.Scan(
		&searchQuota, &viewQuota, &maskPhone, &maskEmail, &maskIDCard, &subModules,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return riwayatklaim.Protection{}, false, nil
		}
		return riwayatklaim.Protection{}, false, fmt.Errorf("membaca master proteksi data: %w", err)
	}

	return riwayatklaim.Protection{
		Login:       strings.TrimSpace(login),
		SearchQuota: int(searchQuota.Int64),
		ViewQuota:   int(viewQuota.Int64),
		SubModules:  riwayatklaim.SplitSubModules(subModules.String),
		MaskPhone:   isYes(maskPhone),
		MaskEmail:   isYes(maskEmail),
		MaskIDCard:  isYes(maskIDCard),
	}, true, nil
}

// CountUsage menghitung pemakaian yang mengurangi jatah.
func (r *ProtectionRepo) CountUsage(ctx context.Context, login, module string) (int, error) {
	var total int
	row := r.db.QueryRowContext(ctx, query("protection_count_usage"),
		strings.TrimSpace(login), strings.TrimSpace(module))

	if err := row.Scan(&total); err != nil {
		return 0, fmt.Errorf("menghitung pemakaian jatah proteksi: %w", err)
	}
	return total, nil
}

// RecordUsage mencatat satu pemakaian.
func (r *ProtectionRepo) RecordUsage(ctx context.Context, usage riwayatklaim.Usage) error {
	consumes := 0
	if usage.ConsumesQuota {
		consumes = 1
	}

	_, err := r.db.ExecContext(ctx, query("protection_record_usage"),
		strings.TrimSpace(usage.Login),
		strings.TrimSpace(usage.Module),
		nullableText(usage.SearchTypeCode),
		nullableText(truncateValue(usage.SearchValue)),
		consumes,
		usage.At.UTC(),
	)
	if err != nil {
		return fmt.Errorf("mencatat pemakaian jatah proteksi: %w", err)
	}
	return nil
}

// TableReady menyatakan tabel pemakaian sudah dibuat DBA.
//
// Dipakai mode -periksa, supaya kesiapan migrasi 0004 terbaca sebelum ada pengguna yang
// menemukannya sebagai galat di tengah pekerjaan.
func (r *ProtectionRepo) TableReady(ctx context.Context) (bool, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, query("protection_check_table")).Scan(&total); err != nil {
		return false, fmt.Errorf("memeriksa tabel %s: %w", TableName, err)
	}
	return total > 0, nil
}

// maxSearchValue membatasi panjang nilai pencarian yang dicatat.
//
// Kolomnya VARCHAR2(400) di migrasi 0004, dan nilai yang lebih panjang akan ditolak
// Oracle dengan ORA-12899. Memotongnya di sini membuat pencarian tetap berhasil dan
// jejaknya tetap tercatat — kehilangan ekor sebuah kata kunci jauh lebih ringan daripada
// pencarian yang gagal karena jejaknya tidak muat.
const maxSearchValue = 400

func truncateValue(value string) string {
	if len(value) <= maxSearchValue {
		return value
	}
	return value[:maxSearchValue]
}

// nullableText mengubah teks kosong menjadi NULL.
//
// Kolom yang kosong dan kolom yang NULL berbeda artinya di sini: NULL berarti "tidak
// berlaku pada baris ini" — pembukaan layar tidak punya tipe pencarian — sedangkan teks
// kosong akan terbaca sebagai pencarian dengan kata kunci kosong.
func nullableText(value string) any {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil
	}
	return clean
}

// isYes membaca penanda masking milik sistem lama.
//
// Nilainya "Ya" — begitulah activity lama membandingkannya
// (`GETDATAMASKINGMST.pxResults(1).Country=="Ya"`). Perbandingannya dibuat tidak peka
// besar-kecil huruf dan tahan spasi tepi karena kolomnya VARCHAR2 tanpa penyeragaman:
// satu spasi di ujung akan membuat penanda yang menyala terbaca padam.
func isYes(value sql.NullString) bool {
	return strings.EqualFold(strings.TrimSpace(value.String), "Ya")
}
