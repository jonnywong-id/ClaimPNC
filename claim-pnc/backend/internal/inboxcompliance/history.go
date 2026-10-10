package inboxcompliance

import "time"

// Baris riwayat keputusan Compliance — padanan `Call InsertHistoryClaimPNC`, langkah 16-18
// `SetComplianceResult`.
//
// # Kenapa ia penting melebihi bobotnya yang tampak
//
// `D-59` menetapkan satuan izin adalah MENU dan **tidak ada pemisahan tugas**: satu orang
// dapat membuat, menyetujui, dan membayarkan satu klaim bila perannya memiliki ketiga menu
// itu. Tidak ada kontrol teknis yang mencegahnya.
//
// Akibatnya jejak audit menjadi **satu-satunya kontrol pengimbang yang tersisa**, dan baris
// inilah satu-satunya jejak keputusan Compliance sampai modul `S-5` ada.

// HistoryEntry adalah satu baris pada `POOLDATA.LIST_HISTORY_CLAIM_PNC`.
type HistoryEntry struct {
	// Reference adalah `CASEID` — `pyWorkPage.pzInsKey`, sesuai parameter `caseID` pada
	// ketiga langkah Pega.
	Reference string

	// RecordedAt adalah `CREATEDATETIME`.
	//
	// Nilainya SAMA PERSIS dengan `DIPUTUSKAN_PADA` pada baris keputusan, bukan jam basis
	// data seperti pada procedure aslinya — supaya keduanya dapat dipasangkan saat
	// menelusuri. Lihat catatan pada kueri `insert_history_claim`.
	RecordedAt time.Time

	// Note adalah `STATUSNOTE` — kalimat harfiah dari parameter `statusNote`.
	Note string

	// By adalah `USERUPDATE` — `OperatorID.pyUserIdentifier` di Pega.
	By string
}

// NewHistoryEntry menurunkan baris riwayat dari keputusan dan klaimnya.
//
// # Dua syarat, dan keduanya dari Pega — bukan tambahan
//
// Ketiga langkah riwayat ber-syarat `local.pilihan` DAN `IsPA`. Keduanya ditiru:
//
//	pilihan 0, 1, 2  menulis baris; masing-masing dengan kalimatnya sendiri
//	pilihan 3        TIDAK menulis apa pun — Pega tidak punya langkah untuk nilai itu
//	bukan PA         TIDAK menulis apa pun
//
// Syarat `IsPA` sempat saya laporkan sebagai CACAT, dengan alasan `D-59` menjadikan jejak
// audit satu-satunya kontrol sehingga klaim non-PA tanpa jejak itu berbahaya. Work Owner
// mengoreksinya: **Compliance memang proses khusus Personal Accident**, sehingga klaim
// non-PA tidak melewati jalur ini sama sekali dan tidak ada jejak yang hilang.
//
// Koreksi itu sekaligus menjelaskan tiga jalur mati yang sebelumnya janggal — tombol
// "Kirim ke PIC Teknik" ber-syarat `!IsTravel AND IsTravel`, klaim Travel tanpa tombol apa
// pun, dan langkah Ticket `SendToPICTravel`. Ketiganya sisa jalur Travel yang tidak pernah
// hidup.
//
// Nilai kedua salah berarti **tidak ada baris yang ditulis**, dan itu keadaan sah — bukan
// galat, dan bukan alasan menggagalkan penyimpanan keputusannya.
func NewHistoryEntry(decision Decision, claim WorkItem) (HistoryEntry, bool) {
	note := decision.StatusNote()
	if note == "" {
		// Lain-Lain. Pega tidak punya langkah riwayat untuk nilai itu.
		return HistoryEntry{}, false
	}

	if claim.GroupPanel != GroupPanelPersonalAccident {
		return HistoryEntry{}, false
	}

	return HistoryEntry{
		Reference:  decision.Reference,
		RecordedAt: decision.DecidedAt,
		Note:       note,
		By:         decision.DecidedBy,
	}, true
}
