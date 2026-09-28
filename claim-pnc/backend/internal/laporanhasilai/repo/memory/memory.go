// Package memory memenuhi seam laporanhasilai.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat
//     dan tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa penyaringnya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Keempat syarat kueri ditiru satu per satu — rentang tanggal setengah terbuka,
// `TANGGALKOMITE` tidak NULL, adanya baris komite ber-`STATUSCASE` Resolved-Completed, dan
// urutan `KOMITE_ID, KOMITEKE, OBJECTID, COVERAGEID` yang menentukan baris mana masuk
// halaman pertama.
//
// Termasuk yang paling mudah terlewat: penyimpanan ini pun MEMBIARKAN lima kolom kosong,
// sama seperti kuerinya. Mengisinya di sini akan membuat layar terlihat benar saat
// dikembangkan dan kosong saat dijalankan — kelas cacat yang paling mahal ditemukan.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/laporanhasilai"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring atau
// mengurutkan.
//
// Keempat kolom tambahan tidak masuk laporanhasilai.Row dengan sengaja: tak satu pun
// sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga ia bagian dari
// kontrak.
type Row struct {
	// CommitteeID — `T_CLAIM_KOMITE_LIST.KOMITE_ID`, kunci pengurutan pertama.
	CommitteeID string

	// CommitteeStep — `KOMITEKE`, jenjang komite. Kunci pengurutan kedua, dan penentu
	// apakah nomor klaim tergambar (lihat `claimNumberOf` pada sqlstore).
	CommitteeStep string

	// ObjectID dan CoverageID — pemutus seri urutan, dan penyusun kunci baris.
	ObjectID   string
	CoverageID string

	// ClaimNumber — `NO_KLAIM` APA ADANYA, sebelum dikosongkan oleh aturan jenjang.
	ClaimNumber string

	// ApproveCode — `STATUSAPPROVE` mentah.
	ApproveCode string

	// CommitteeDate — `TANGGALKOMITE`. Nol berarti NULL, dan baris seperti itu HILANG
	// dari hasil. Baris semacam itu sengaja ada di data contoh: tanpanya, penyaring yang
	// lupa tidak akan pernah ketahuan.
	CommitteeDate time.Time

	// AIResult — `RESULTAI`. Kosong berarti AI belum menilai, dan barisnya TETAP muncul.
	AIResult string

	// AIDate — `TGLAI`.
	AIDate time.Time

	// CaseResolved menandai ADA baris komite ber-KOMITE_ID sama yang
	// `STATUSCASE = 'Resolved-Completed'`.
	//
	// Ia boolean, bukan salinan kolom STATUSCASE, karena syarat aslinya memang berupa
	// keberadaan — bukan nilai pada baris ini sendiri. Lihat catatan `EXISTS` pada berkas
	// .sql.
	CaseResolved bool
}

// Repo menyimpan baris contoh di memori.
type Repo struct {
	mutex sync.RWMutex
	rows  []Row
}

// NewRepo membentuk penyimpanan dari baris contoh.
func NewRepo(rows ...Row) *Repo {
	stored := make([]Row, 0, len(rows))
	stored = append(stored, rows...)
	return &Repo{rows: stored}
}

// List mengembalikan satu halaman beserta cacah seluruh baris yang cocok.
func (r *Repo) List(
	_ context.Context,
	filter laporanhasilai.Filter,
	page laporanhasilai.Pagination,
) (laporanhasilai.Page, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	matched := r.matching(filter)
	result := laporanhasilai.Page{Total: len(matched)}

	clean := page.Normalize()
	offset := clean.Offset()
	if offset >= len(matched) {
		return result, nil
	}

	end := offset + clean.Size
	if end > len(matched) {
		end = len(matched)
	}

	for _, one := range matched[offset:end] {
		result.Rows = append(result.Rows, toDomain(one))
	}
	return result, nil
}

// Summarize mencacah seluruh baris yang cocok.
//
// Pembandingnya PERSIS, tanpa pemangkasan — sama seperti kueri ringkasannya. Bila yang
// satu memangkas dan yang lain tidak, uji yang lulus di sini tidak menyatakan apa pun
// tentang yang berjalan di Oracle.
func (r *Repo) Summarize(
	_ context.Context,
	filter laporanhasilai.Filter,
) (laporanhasilai.Summary, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	matched := r.matching(filter)

	var (
		aiAccepted        int
		aiRejected        int
		committeeAccepted int
		committeeRejected int
	)

	for _, one := range matched {
		switch one.AIResult {
		case laporanhasilai.AIAccepted:
			aiAccepted++
		case laporanhasilai.AIRejected:
			aiRejected++
		}

		switch one.ApproveCode {
		case laporanhasilai.CommitteeApprovedCode:
			committeeAccepted++
		case laporanhasilai.CommitteeRejectedCode:
			committeeRejected++
		}
	}

	total := len(matched)

	return laporanhasilai.Summary{
		Committee: tallyOf(
			laporanhasilai.SubjectCommittee, committeeAccepted, committeeRejected, total),
		AI: tallyOf(laporanhasilai.SubjectAI, aiAccepted, aiRejected, total),
	}, nil
}

// matching menyaring lalu mengurutkan, meniru WHERE dan ORDER BY kuerinya.
//
// Pemanggil wajib sudah memegang kunci baca.
func (r *Repo) matching(filter laporanhasilai.Filter) []Row {
	clean := filter.Clean()
	from := clean.From
	to := clean.ToExclusive()

	matched := make([]Row, 0, len(r.rows))
	for _, one := range r.rows {
		if one.CommitteeDate.IsZero() || !one.CaseResolved {
			continue
		}
		// Rentang setengah terbuka, sama persis dengan kuerinya: batas bawah inklusif,
		// batas atas eksklusif.
		if one.CommitteeDate.Before(from) || !one.CommitteeDate.Before(to) {
			continue
		}
		matched = append(matched, one)
	}

	sort.SliceStable(matched, func(i, j int) bool {
		return orderKey(matched[i]) < orderKey(matched[j])
	})
	return matched
}

// orderKey menyusun kunci urut dari keempat kolom, pada urutan yang sama dengan ORDER BY.
//
// Perbandingannya TEKS, bukan angka — sama seperti basis data memperlakukan kolom yang
// tipenya belum diketahui (`R-08`). Akibatnya jenjang "10" berada sebelum "2", dan itu
// memang yang terjadi di Oracle bila kolomnya bertipe teks. Meniru urutan yang "lebih
// masuk akal" di sini akan menyembunyikan perbedaan yang nyata.
func orderKey(one Row) string {
	return strings.Join([]string{
		one.CommitteeID,
		one.CommitteeStep,
		one.ObjectID,
		one.CoverageID,
	}, "\x00")
}

// toDomain mengubah baris contoh menjadi baris domain.
//
// Kelima field yang kuerinya tidak pilih TIDAK diisi di sini. Lihat doc paket.
func toDomain(one Row) laporanhasilai.Row {
	return laporanhasilai.Row{
		ID: strings.Join([]string{
			one.CommitteeID, one.CommitteeStep, one.ObjectID, one.CoverageID,
		}, "|"),
		ClaimNumber:         claimNumberOf(one),
		CommitteeStatus:     laporanhasilai.CommitteeLabel(one.ApproveCode),
		CommitteeStatusCode: one.ApproveCode,
		CommitteeDate:       one.CommitteeDate,
		AIStatus:            one.AIResult,
		AIDate:              one.AIDate,
	}
}

// claimNumberOf meniru `CASE WHEN B.KOMITEKE = '1' THEN B.NO_KLAIM ELSE '' END`.
func claimNumberOf(one Row) string {
	if strings.TrimSpace(one.CommitteeStep) != "1" {
		return ""
	}
	return one.ClaimNumber
}

// tallyOf menyusun satu baris ringkasan; Menunggu dihitung sebagai sisa.
func tallyOf(subject string, accepted, rejected, rowCount int) laporanhasilai.Tally {
	pending := rowCount - accepted - rejected
	if pending < 0 {
		pending = 0
	}
	return laporanhasilai.Tally{
		Subject:  subject,
		Accepted: accepted,
		Rejected: rejected,
		Pending:  pending,
	}
}

var _ laporanhasilai.Repo = (*Repo)(nil)
