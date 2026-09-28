package registrasi

import "strings"

// # PIC Teknik klaim — ClaimData.UserTeknis
//
// Di Pega, tahap yang dirutekan `PNCTeknikRouter` (Choose Surveyor, Send To Analis, Send To
// PIC Teknik) dipegang PIC Teknik KLAIMNYA. Router itu sendiri tidak ada di export (`R-04`),
// tetapi datanya tegas: dari 338 baris Choose Surveyor di POOLDATA.T_CLAIMLIST_ADMIN,
// pemegang tugasnya sama dengan USERTEKNIS_1 pada 320 baris (2026-09-28).
//
// Pega mengisi UserTeknis lebih awal, lewat `getRandomTeam_act` saat klaim dibuka. Aktivitas
// itu memanggil tiga aktivitas lain dan satu Connect-REST, dan belum direkonstruksi di sini.
// Yang dibawa hanyalah dua akibatnya yang terukur:
//
//  1. PIC Teknik yang sudah tercatat di klaim menjadi penerima tugas tahap teknis.
//  2. PIC yang dipilih router karena klaim belum punya PIC ditulis kembali ke klaim, sehingga
//     PICTEKNIK dan USERTEKNIS_1 berisi orang yang sama dengan pemegang tugasnya.

// AssignedTechnicalPIC mengembalikan PIC Teknik klaim bila tahap itu dirutekan ke PIC Teknik.
// Kosong berarti router harus memilih sendiri.
func AssignedTechnicalPIC(stage Stage, claim Claim) string {
	if stage.Router != RouterPNCTechnical {
		return ""
	}
	return strings.TrimSpace(claim.TechnicalPIC)
}

// AdoptTechnicalPIC mencatat penerima tugas tahap teknis sebagai PIC Teknik klaim, bila
// klaim belum punya PIC. PIC yang sudah ada tidak pernah ditimpa.
func AdoptTechnicalPIC(claim *Claim, stage Stage, to Assignee) {
	if stage.Router != RouterPNCTechnical || strings.TrimSpace(claim.TechnicalPIC) != "" {
		return
	}
	if operator := strings.TrimSpace(to.Operator); operator != "" {
		claim.TechnicalPIC = operator
	}
}
