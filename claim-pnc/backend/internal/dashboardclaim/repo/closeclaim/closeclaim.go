// Package closeclaim memenuhi seam dashboardclaim.ClosedClaimReader dengan memakai ulang
// modul inboxcloseclaim.
//
// # Kenapa adapter, bukan kueri sendiri
//
// Populasi tile CLOSE CLAIM sama persis dengan layar Inbox Close Claim:
//
//	PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
//	PYSTATUSWORK IN ('Resolved-Completed', 'Resolved-Rejected')
//	+ penyaring lini bisnis yang identik
//
// Menyalin kuerinya ke modul ini akan membuat dua salinan aturan yang sama. Keduanya akan
// menyimpang pada perubahan berikutnya — bukan karena seseorang lalai, melainkan karena
// tidak ada apa pun yang memberi tahu bahwa salinan kedua ada.
//
// # Kenapa adapter, bukan impor langsung dari usecase
//
// Modul domain tidak saling impor. `dashboardclaim` mendeklarasikan seam-nya sendiri, dan
// paket INILAH satu-satunya tempat kedua modul bertemu. Akibatnya `dashboardclaim` dapat
// diuji tanpa `inboxcloseclaim` sama sekali, dan `inboxcloseclaim` tidak tahu layar ini ada.
//
// # Kenapa lewat Repo, bukan lewat Service
//
// `inboxcloseclaimusecase.Service.List` ikut membaca permintaan ReOpen yang tertunda atas
// setiap baris. Dashboard tidak menampilkannya, sehingga memanggilnya berarti satu kueri
// tambahan per pembukaan layar untuk hasil yang dibuang — dan kueri itu membaca tabel yang
// migrasinya belum dijalankan DBA di lingkungan mana pun.
package closeclaim

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/inboxcloseclaim"
)

// Reader membaca klaim tutup lewat penyimpanan milik modul inboxcloseclaim.
type Reader struct {
	claims inboxcloseclaim.RepoSelector
}

// NewReader membentuk adapter.
//
// Menolak selector kosong saat START, bukan saat layar dibuka: perakitan yang kurang harus
// gagal keras di awal (`12-CROSSCUTTING` §3.1).
func NewReader(claims inboxcloseclaim.RepoSelector) (*Reader, error) {
	if claims == nil {
		return nil, errors.New("dashboardclaim/closeclaim: pemilih repo klaim tutup wajib diisi")
	}
	return &Reader{claims: claims}, nil
}

// Count menghitung klaim tutup yang cocok dengan penyaring.
//
// Ia memanggil List dengan satu baris lalu membaca totalnya, karena seam Repo modul itu
// tidak menyediakan penghitung tersendiri. Kueri hitungnya tetap berjalan penuh — yang
// dibatasi hanya jumlah baris yang ikut terbaca, bukan populasi yang dihitung.
func (r *Reader) Count(ctx context.Context, portalAlias string, f dashboardclaim.Filter) (int, error) {
	claims, err := r.claims(portalAlias)
	if err != nil {
		return 0, err
	}

	filter := translate(f)
	filter.Limit = 1
	filter.Offset = 0

	page, err := claims.List(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("dashboardclaim/closeclaim: menghitung klaim tutup: %w", err)
	}
	return page.Total, nil
}

// List membaca satu halaman klaim tutup.
func (r *Reader) List(ctx context.Context, portalAlias string, f dashboardclaim.Filter) (dashboardclaim.ClaimPage, error) {
	claims, err := r.claims(portalAlias)
	if err != nil {
		return dashboardclaim.ClaimPage{}, err
	}

	page, err := claims.List(ctx, translate(f))
	if err != nil {
		return dashboardclaim.ClaimPage{}, fmt.Errorf("dashboardclaim/closeclaim: membaca daftar klaim tutup: %w", err)
	}

	rows := make([]dashboardclaim.ClaimRow, 0, len(page.Claims))
	for _, claim := range page.Claims {
		rows = append(rows, adapt(claim))
	}
	return dashboardclaim.ClaimPage{Rows: rows, Total: page.Total}, nil
}

// translate memetakan penyaring dashboard menjadi penyaring modul inboxcloseclaim.
//
// Hanya tiga hal yang berpindah — lini bisnis, kotak cari, dan paginasi — karena hanya
// ketiganya yang dimiliki layar ini. Penyaring Status Transfer dan Status Bayar milik layar
// Inbox Close Claim sengaja DIBIARKAN KOSONG, dan kosong di sana berarti tanpa saringan.
//
// Nilai lini bisnisnya sengaja tidak dikonversi lewat tabel pemetaan: kelimanya bernilai
// teks yang sama persis di kedua modul, dan pemetaan yang hanya menyalin justru
// menyembunyikan saat keduanya kelak berbeda. Uji menjaga kesamaannya.
func translate(f dashboardclaim.Filter) inboxcloseclaim.Filter {
	f = f.Normalize()
	return inboxcloseclaim.Filter{
		Search:   f.Search,
		Business: inboxcloseclaim.BusinessLine(f.Business),
		Limit:    f.Limit,
		Offset:   f.Offset,
	}
}

// adapt memetakan satu klaim tutup menjadi baris telusur dashboard.
//
// Kolom yang tidak digambar layar ini — tanggal tutup, tanggal selesai, status transfer
// kasir — tidak ikut dibawa. Membawanya berarti menambah kolom pada kontrak yang tidak
// dipakai siapa pun, dan kontrak lebih sulit dikecilkan daripada dibesarkan.
func adapt(claim inboxcloseclaim.ClosedClaim) dashboardclaim.ClaimRow {
	return dashboardclaim.ClaimRow{
		ClaimID:         claim.ClaimID,
		ClaimNumber:     claim.ClaimNumber,
		PolicyNumber:    claim.PolicyNumber,
		InsuredName:     claim.InsuredName,
		BusinessName:    claim.BusinessName,
		BusinessSource:  claim.BusinessSource,
		BranchName:      claim.BranchName,
		TechnicalPIC:    claim.TechnicalPIC,
		AdminPNC:        claim.AdminPNC,
		ClaimStatusCode: claim.ClaimStatusCode,
		ProcessStatus:   claim.ProcessStatus,
		LossDate:        claim.LossDate,
		RegisteredAt:    claim.RegisteredAt,
	}
}
