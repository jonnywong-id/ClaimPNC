// Package memory memenuhi seam inboxadmin.Repo dengan penyimpanan di memori.
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
// di Oracle. Penyaring lini bisnis, kata kunci, kepemilikan, dan mode kurir ditiru
// sedekat-dekatnya dengan predikat SQL-nya — termasuk kenyataan bahwa NONMBU bukan
// "selain PA dan Travel" melainkan empat Group Panel dikurangi kelompok Bonding.
package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/inboxadmin"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring.
//
// Keempat kolom tambahan itu tidak masuk inboxadmin.WorkItem dengan sengaja: ia tidak
// pernah sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga ia
// bagian dari kontrak.
type Row struct {
	Item inboxadmin.WorkItem

	// Tab adalah kode tab tempat baris ini muncul.
	Tab string

	// GroupPanel — `GROUPPANEL_1`, dipakai penyaring lini bisnis.
	GroupPanel string

	// BusinessGroupID — `BUSINESS.BUSINESSGROUPID`, dipakai penyaring lini bisnis.
	BusinessGroupID string

	// Owner adalah login yang memiliki baris ini, dipakai tab yang ScopedToCaller.
	Owner string

	// Courier — `KURIR`, membedakan Unregistered RCV dari RCV Online.
	Courier string
}

// Store adalah penyimpanan antrean di memori.
//
// Ia tidak dilindungi mutex karena tidak pernah berubah setelah dibentuk: modul ini hanya
// membaca, dan tidak ada satu pun operasi yang menulis.
type Store struct {
	rows []Row
}

// NewStore membentuk penyimpanan berisi baris yang diberikan.
func NewStore(rows ...Row) *Store {
	return &Store{rows: rows}
}

// NewSampleStore membentuk penyimpanan berisi contoh bawaan.
func NewSampleStore() *Store {
	return NewStore(SampleRows()...)
}

// List mengembalikan seluruh baris satu tab yang lolos penyaring, belum dipaginasi.
func (s *Store) List(_ context.Context, q inboxadmin.Query) ([]inboxadmin.WorkItem, error) {
	result := []inboxadmin.WorkItem{}

	for _, candidate := range s.rows {
		if candidate.Tab != q.Tab.Code {
			continue
		}
		if q.Tab.ScopedToCaller && !strings.EqualFold(candidate.Owner, q.Caller.Login) {
			continue
		}
		if !matchesCourier(candidate, q.Tab.Code) {
			continue
		}
		if !matchesBusiness(candidate, q.Business) {
			continue
		}
		if !matchesKeyword(candidate, q.Keyword) {
			continue
		}
		result = append(result, candidate.Item)
	}

	// Urutan ditetapkan supaya paginasi di atasnya stabil. Nilai yang dipakai adalah
	// tanggal terbaru lebih dulu, mengikuti `ORDER BY PXCREATEDATETIME DESC` kuerinya;
	// baris tanpa tanggal diurutkan menurut Case ID supaya tetap deterministik.
	sort.SliceStable(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.InputDate != nil && right.InputDate != nil && !left.InputDate.Equal(*right.InputDate) {
			return left.InputDate.After(*right.InputDate)
		}
		return left.CaseID < right.CaseID
	})

	return result, nil
}

// matchesCourier meniru saringan kurir yang membedakan kedua tab Unregistered RCV.
func matchesCourier(candidate Row, tab string) bool {
	switch tab {
	case inboxadmin.TabUnregisteredRCV:
		return candidate.Courier != "Auto Service"
	case inboxadmin.TabRCVOnline:
		return candidate.Courier == "Auto Service"
	default:
		return true
	}
}

// bondingGroups adalah keempat BUSINESSGROUPID kelompok Bonding.
//
// Nilainya diambil apa adanya dari `Activity/SetTempClaimRegistandNotRegist-Act.xml`
// langkah 12 dan 13. Ia dipakai DUA arah: Bonding memilih yang ada di daftar ini, Non-MBU
// membuang yang ada di daftar ini.
var bondingGroups = map[string]bool{
	"10008": true, "10010": true, "10015": true, "10023": true,
}

// nonMBUPanels adalah keempat Group Panel yang dianggap Non-MBU.
var nonMBUPanels = map[string]bool{
	"003": true, "004": true, "006": true, "009": true,
}

// matchesBusiness meniru penyaring lini bisnis.
func matchesBusiness(candidate Row, line inboxadmin.BusinessLine) bool {
	switch line {
	case inboxadmin.BusinessAll:
		return true
	case inboxadmin.BusinessNonMBU:
		return nonMBUPanels[candidate.GroupPanel] && !bondingGroups[candidate.BusinessGroupID]
	case inboxadmin.BusinessBonding:
		return bondingGroups[candidate.BusinessGroupID]
	case inboxadmin.BusinessPA:
		return candidate.GroupPanel == "002"
	case inboxadmin.BusinessTravel:
		return candidate.GroupPanel == "005"
	default:
		return false
	}
}

// matchesKeyword meniru kotak cari, yang hanya menyentuh DUA kolom.
//
// Itu memang seluruh jangkauannya di sistem lama: `a.pyid` dan `a.policyno`. Mencari nama
// tertanggung di kotak ini tidak pernah membuahkan hasil, dan layar menyatakannya lewat
// teks petunjuk di bawah kotaknya.
func matchesKeyword(candidate Row, keyword string) bool {
	if keyword == "" {
		return true
	}
	needle := strings.ToUpper(keyword)
	return strings.Contains(strings.ToUpper(candidate.Item.CaseID), needle) ||
		strings.Contains(strings.ToUpper(candidate.Item.PolicyNumber), needle)
}
