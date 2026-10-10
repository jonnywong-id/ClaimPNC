package dashboardclaim

import (
	"context"
	"errors"
)

// ClaimDetail adalah rincian satu klaim — isi popup yang terbuka saat nomor klaim diklik.
//
// Menggantikan harness `ViewTempDetailClaim`, yang di Pega dibuka lewat `showHarness`
// bertarget `popup` dari kolom nomor klaim pada `DashboardClaimShow_Sec`.
type ClaimDetail struct {
	// Tujuh nilai pertama datang dari TABEL KERJA, bukan dari dokumen JSON.
	//
	// Dipisahkan dengan sengaja: ketujuhnya selalu ada selama klaimnya ada, sedangkan
	// dokumen JSON bisa belum terbentuk. Popup yang menggambar nomor klaim dari dokumen
	// akan tampil kosong seluruhnya pada klaim yang baru didaftarkan.
	ClaimID       string
	ClaimNumber   string
	ProcessStatus string
	ClaimStatus   string
	TechnicalPIC  string
	AdminPNC      string
	RegisteredAt  string

	// Document adalah isi klaim, hasil urai `POOLDATA.JSON_KLAIM.DATA_JSON`.
	//
	// # Kenapa peta bebas, bukan struct berisi ratusan field
	//
	// Bentuk dokumen ini BELUM PERNAH DIPERIKSA — DDL-nya tidak tersedia (`R-08`) dan isinya
	// belum pernah dilihat. Struct akan menyatakan bentuk yang belum terbukti, dan isian yang
	// namanya ternyata berbeda hilang tanpa satu pun tanda.
	//
	// Peta bebas membuat ketidakcocokan TERLIHAT: isian yang jalurnya tidak ada dapat
	// dibedakan dari isian yang ada tetapi kosong. Pola yang sama sudah dipakai
	// `internal/outstandingclaim` atas dokumen sekerabat, dengan alasan yang sama.
	//
	// Kosong BUKAN galat: gabungannya LEFT JOIN, dan klaim yang belum punya baris di
	// JSON_KLAIM tetap dapat dibuka — hanya isinya yang kosong.
	Document map[string]any
}

// ErrClaimDetailUnreadable berarti dokumen klaimnya ADA tetapi tidak dapat diurai.
//
// Dibedakan tegas dari dokumen kosong. Dokumen kosong adalah keadaan yang sah; dokumen rusak
// adalah kerusakan data, dan menyajikannya sebagai rincian kosong memberitahu pengguna
// "klaim ini memang belum diisi" — pernyataan yang salah dan tidak dapat dibantah dari layar.
var ErrClaimDetailUnreadable = errors.New("dashboardclaim: dokumen klaim tidak dapat dibaca")

// ClaimDetailReader membaca rincian satu klaim.
//
// Seam tersendiri, bukan method pada Repo: ia satu-satunya pembaca yang menyentuh
// `POOLDATA.JSON_KLAIM`, dan memisahkannya membuat batas itu terlihat di perakitan.
type ClaimDetailReader interface {
	FindClaimDetail(ctx context.Context, claimID string) (ClaimDetail, error)
}
