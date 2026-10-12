package inboxservicecenter

import "strings"

// ApprovalFilter menyatakan baris ber-`STS_APPROVAL` mana yang masuk sebuah tab.
//
// # Kenapa ia tipe tersendiri, bukan sekadar satu kode
//
// Karena penyaringnya tidak seragam. `Activity/DataServiceCenter-Act.xml` menyusunnya
// dalam empat langkah, dan hanya yang pertama yang berlaku umum:
//
//	langkah 8   selalu jalan          and sts_approval = '<stsapprove>'
//	langkah 12  bila stsapprove "2"   and sts_approval in ('2','3')    ← menimpa
//	langkah 15  bila stsapprove ""    and sts_approval is null         ← menimpa
//	langkah 14  bila stsapprove "komite"  sts_approval='0' AND komiteapprove = <saya>
//
// Jadi ada tiga bentuk yang berbeda — sama dengan satu nilai, termasuk salah satu dari dua
// nilai, dan IS NULL. Memaksanya menjadi satu kode berarti membuang perbedaan itu.
//
// Langkah 14 TIDAK dibawa: `komite` bukan salah satu dari keempat tab layar ini. Keempat
// section hanya pernah mengirim "", "0", "1", dan "2"; `komite` datang dari layar komite,
// yang modulnya sendiri.
type ApprovalFilter struct {
	// Codes adalah kode `STS_APPROVAL` yang diterima. Kosong berarti tidak ada kode yang
	// diterima — lihat MatchNull.
	Codes []string

	// MatchNull berarti baris yang `STS_APPROVAL`-nya NULL yang justru diterima.
	//
	// Ia terpisah dari Codes dan tidak dapat digabung dengannya, karena di SQL `= NULL`
	// tidak pernah benar. Hanya tab Registrasi SC yang memakainya.
	MatchNull bool
}

// approvalFilterFor menyerahkan penyaring status persetujuan milik sebuah tab.
func approvalFilterFor(tab Tab) ApprovalFilter {
	switch tab.Code {
	case TabRegistration:
		// `and sts_approval is null` — langkah 15.
		return ApprovalFilter{MatchNull: true}

	case TabRejected:
		// `and sts_approval in ('2','3')` — langkah 12.
		//
		// TLO dan REJECT dikumpulkan di satu tab, dan itu memang yang dikehendaki: judul
		// tabnya "Rejected", tetapi isinya kedua keputusan yang sama-sama berakhir tanpa
		// perbaikan.
		return ApprovalFilter{Codes: []string{ApprovalTotalLoss, ApprovalRejected}}

	default:
		// `and sts_approval = '<stsapprove>'` — langkah 8, bentuk umumnya.
		return ApprovalFilter{Codes: []string{tab.PegaParam}}
	}
}

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string

	// Keyword adalah isi kotak "Cari".
	Keyword string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolomnya.
	Tab Tab

	// Approval adalah penyaring status persetujuan yang berlaku bagi tab itu.
	Approval ApprovalFilter

	// Keyword adalah kata kunci pencarian yang BENAR-BENAR dipakai, sudah dipangkas.
	Keyword string

	// Caller adalah identitas pemanggil. Keempat tab menyaring `PIC` menurut nilai ini.
	Caller Caller
}

// Paginated menyatakan hasil permintaan ini dipotong per halaman.
//
// # Kenapa pencarian mematikan paginasi
//
// Karena begitulah sistem lama bekerja, dan itu tertulis pada prakondisi dua langkah yang
// berpasangan di `Activity/DataServiceCenter-Act.xml`:
//
//	langkah 22  "Jika Tidak Ada Pencarian // Set Row"
//	            WHEN TempSearch.BranchID=="" -> true=2 (lanjut)
//	            TempSQL.AlasanTerlambat = "WHERE rn >= <awal> AND rn <= <akhir>"
//
//	langkah 23  "Jika Ada Pencarian // Tidak Set Row"
//	            WHEN TempSearch.BranchID=="" -> true=3 (lewati), false=2 (lanjut)
//	            TempSQL.AlasanTerlambat = ""
//
// Klausa `rn` itulah satu-satunya paginasi kuerinya. Begitu ia dikosongkan, seluruh baris
// yang cocok terbawa sekaligus.
//
// Perilakunya ditiru apa adanya (`P-5`) dan dinyatakan terbuka lewat Limitations, bukan
// diam-diam diperbaiki — memaginasi hasil pencarian akan mengubah jumlah baris yang terlihat
// pengguna, dan itu selisih yang akan dilaporkan uji kesetaraan gerbang 1.
func (q Query) Paginated() bool {
	return q.Keyword == ""
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTab
	}

	tab, known := FindTab(code)
	if !known {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Tab tidak dikenal.",
		}})
	}

	return Query{
		Tab:      tab,
		Approval: approvalFilterFor(tab),
		Keyword:  strings.TrimSpace(input.Keyword),
		Caller:   cleanCaller,
	}, nil
}
