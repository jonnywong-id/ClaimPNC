// Package memory memenuhi seam inboxanalystdoctor.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat dan
//     tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa penyaring dan urutannya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Keempatnya ditiru apa adanya:
//
//	isComplianceTransfer = "2"                  penanda antrean
//	pxAssignedOperatorID = pemanggil            batas kewenangan
//	pyStatusWork        != Resolved-Completed   tugas yang sudah tuntas keluar
//	urutan pxCreateDateTime DESC, pzInsKey DESC yang terbaru masuk lebih dulu
//
// Yang ditambahkan hanyalah kotak cari, dan ia ditiru persis seperti kuerinya: menyentuh
// Nomor Case DAN No Polis, tidak peka huruf besar-kecil.
package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/inboxanalystdoctor"
)

// Record adalah satu tugas contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring.
//
// Ketiga kolom penyaring itu sengaja tidak masuk `inboxanalystdoctor.AnalystDoctorTask`:
// `TransferFlag` tidak pernah sampai ke layar, dan menaruhnya di tipe domain akan membuat
// orang menduga ia bagian dari kontrak.
type Record struct {
	Task inboxanalystdoctor.AnalystDoctorTask

	// TransferFlag adalah `ClaimData.isComplianceTransfer`.
	//
	// Hanya baris bernilai `"2"` yang termasuk antrean ini. Baris bernilai `"1"` sengaja
	// disertakan di antara data contoh — itu antrean Compliance, dan satu-satunya cara
	// membuktikan penyaringnya benar-benar bekerja adalah menyediakan baris yang harus
	// TERTOLAK olehnya.
	TransferFlag string
}

// Store adalah pembaca antrean Analyst Doctor di memori.
type Store struct {
	records []Record
}

// NewStore membentuk pembaca dari baris yang diberikan.
func NewStore(records []Record) *Store {
	return &Store{records: records}
}

// List mengambil satu halaman antrean milik seorang operator.
func (s *Store) List(
	_ context.Context,
	operator string,
	f inboxanalystdoctor.Filter,
) (inboxanalystdoctor.Page, error) {
	clean := f.Normalize()

	matched := []inboxanalystdoctor.AnalystDoctorTask{}
	for _, record := range s.records {
		if !matches(record, operator, clean.Search) {
			continue
		}
		matched = append(matched, record.Task)
	}

	sortTasks(matched)

	result := inboxanalystdoctor.Page{
		Tasks: []inboxanalystdoctor.AnalystDoctorTask{},
		Total: len(matched),
	}

	if clean.Offset >= len(matched) {
		// Halaman di luar jangkauan menghasilkan senarai kosong DENGAN total yang benar.
		// Mengembalikan total nol akan membuat bilah halaman menghilang, dan pengguna yang
		// terlanjur berada di halaman sepuluh kehilangan jalan kembali.
		return result, nil
	}

	end := clean.Offset + clean.Limit
	if end > len(matched) {
		end = len(matched)
	}

	result.Tasks = matched[clean.Offset:end]
	return result, nil
}

// matches meniru ketiga penyaring Report Definition ditambah kotak cari.
func matches(record Record, operator, search string) bool {
	if record.TransferFlag != inboxanalystdoctor.TransferAnalystDoctor {
		return false
	}

	// Perbandingan operator TIDAK peka huruf besar-kecil.
	//
	// Bukan kelonggaran: `11-SECURITY.md` §3.1 mencatat nama access group muncul dalam dua
	// kapitalisasi di export, dan `PXASSIGNEDOPERATORID` menyimpan Operator ID dengan
	// keseragaman yang sama tidak terjaminnya. Menuntut kesamaan persis akan membuat antrean
	// tampak kosong bagi pengguna yang login-nya tersimpan berbeda huruf.
	if !strings.EqualFold(strings.TrimSpace(record.Task.AssignedOperator), strings.TrimSpace(operator)) {
		return false
	}

	if strings.TrimSpace(record.Task.ProcessStatus) == inboxanalystdoctor.StatusKerjaSelesai {
		return false
	}

	needle := strings.ToUpper(strings.TrimSpace(search))
	if needle == "" {
		return true
	}
	for _, field := range []string{record.Task.ClaimNumber, record.Task.PolicyNumber} {
		if strings.Contains(strings.ToUpper(field), needle) {
			return true
		}
	}
	return false
}

// sortTasks mengurutkan seperti kuerinya: pxCreateDateTime MENURUN, pzInsKey sebagai pemutus
// seri, juga menurun.
//
// Pemutus serinya bukan hiasan. Tanpa urutan yang tetap, dua baris berwaktu sama dapat
// bertukar tempat antarpermintaan — dan begitu halamannya dipotong, satu baris muncul di dua
// halaman sekaligus hilang dari halaman lain.
func sortTasks(items []inboxanalystdoctor.AnalystDoctorTask) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if left.RegisteredAt.Equal(right.RegisteredAt) {
			return left.ClaimID > right.ClaimID
		}
		return left.RegisteredAt.After(right.RegisteredAt)
	})
}
