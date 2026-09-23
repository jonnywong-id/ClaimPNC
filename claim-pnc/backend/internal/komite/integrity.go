package komite

import (
	"fmt"
	"sort"

	"claim-pnc/internal/platform/money"
)

// Severity menyatakan seberapa berat sebuah temuan.
type Severity string

const (
	// SeverityDefect berarti masternya SALAH dan hasil penjenjangan tidak dapat
	// dipercaya: rentang yang tumpang tindih, berlubang, atau terbalik.
	SeverityDefect Severity = "cacat"

	// SeverityWarning berarti masternya tidak salah, tetapi ada sesuatu yang pantas
	// dilihat Work Owner sebelum dipakai memutuskan uang.
	SeverityWarning Severity = "peringatan"
)

// Jenis temuan. Dipakai layar untuk mengelompokkan dan menjelaskan, bukan untuk
// mencocokkan teks pesan. Nilainya kontrak API, sehingga tetap berbahasa Indonesia.
const (
	KindInvertedBounds = "batas_terbalik"
	KindOverlap        = "tumpang_tindih"
	KindGap            = "berlubang"
	KindDuplicateTier  = "jenjang_ganda"
	KindLadderTop      = "atap_tangga"
	KindNoTier         = "tanpa_jenjang"
)

// Finding adalah satu hal yang ditemukan pada master ambang.
type Finding struct {
	Severity Severity
	Kind     string

	BusinessLine BusinessLine

	// Band terisi hanya untuk lini yang memakainya.
	Band string

	// Message sudah berbahasa Indonesia dan siap ditampilkan apa adanya.
	Message string

	// ThresholdIDs menunjuk baris master yang terlibat, supaya temuan dapat ditelusuri
	// ke datanya alih-alih hanya dibaca sebagai keluhan.
	ThresholdIDs []string
}

// CheckIntegrity memeriksa kesehatan master ambang komite dan mengembalikan seluruh
// temuannya.
//
// # Inilah satu-satunya tempat LIMIT_TOP dipakai
//
// `D-47` menetapkan perannya dengan tegas: `LIMIT_TOP` BUKAN penyaring pemilih baris —
// memakainya untuk memilih akan mengembalikan tepat satu baris dan menghapus
// penjenjangan seluruhnya — melainkan alat memeriksa apakah tangganya tersusun rapi.
//
// # Kenapa pengelompokannya berbeda antar lini
//
// Untuk lini BERPITA (Non-MBU), tangganya dikelompokkan per pita: dua deret yang berdiri
// sendiri, dan akumulasi tidak pernah menyeberang di antaranya.
//
// Untuk lini lain, tangganya SATU deret utuh dan TYPE_KOMITE tidak boleh dipakai
// mengelompokkan. Bila dipaksa demikian, tangga PA akan terbelah menjadi dua potongan
// yang tampak berlubang parah — padahal pada PA kolom itu membedakan PA reguler dari
// PA TKI, bukan pita nilai (`D-70`).
//
// Inilah yang dimaksud catatan `D-47` bahwa validasi ini "hanya dapat dijalankan per
// lini, bukan lintas lini".
func CheckIntegrity(thresholds []Threshold, policy Policy) []Finding {
	groups := map[string][]Threshold{}
	var groupOrder []string

	for _, t := range thresholds {
		t = t.Normalized()
		if !t.IsApprovalTier() {
			continue
		}

		key := string(t.BusinessLine)
		if _, banded := policy.Bands[t.BusinessLine]; banded {
			key = string(t.BusinessLine) + "\x00" + t.CommitteeType
		}
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], t)
	}

	sort.Strings(groupOrder)

	var findings []Finding
	for _, key := range groupOrder {
		findings = append(findings, checkOneGroup(groups[key], policy)...)
	}

	if len(groupOrder) == 0 {
		findings = append(findings, Finding{
			Severity: SeverityDefect,
			Kind:     KindNoTier,
			Message: "Master ambang komite tidak memuat satu pun jenjang persetujuan yang " +
				"aktif. Tidak ada klaim yang dapat melewati komite.",
		})
	}

	return findings
}

// checkOneGroup memeriksa satu tangga — satu lini, atau satu pita di dalam lini.
func checkOneGroup(rows []Threshold, policy Policy) []Finding {
	if len(rows) == 0 {
		return nil
	}

	line := rows[0].BusinessLine
	rule, banded := policy.Bands[line]
	band := ""
	if banded {
		band = rows[0].CommitteeType
	}

	// Disalin lebih dulu: sort mengubah senarai di tempat, dan mengurutkan milik
	// pemanggil akan mengubah data yang bukan milik fungsi ini.
	ladder := append([]Threshold(nil), rows...)
	sort.SliceStable(ladder, func(i, j int) bool {
		if ladder[i].LowerBound != ladder[j].LowerBound {
			return ladder[i].LowerBound < ladder[j].LowerBound
		}
		return ladder[i].ID < ladder[j].ID
	})

	build := func(severity Severity, kind, message string, ids ...string) Finding {
		return Finding{
			Severity:     severity,
			Kind:         kind,
			BusinessLine: line,
			Band:         band,
			Message:      message,
			ThresholdIDs: ids,
		}
	}

	var findings []Finding

	// 1 — batas terbalik. Diperiksa lebih dulu karena baris yang terbalik membuat
	// pemeriksaan kesinambungan di bawahnya menghasilkan pesan yang menyesatkan.
	for _, t := range ladder {
		if t.UpperBound < t.LowerBound && t.UpperBound != money.Zero {
			findings = append(findings, build(SeverityDefect, KindInvertedBounds,
				fmt.Sprintf("Batas atas (%s) lebih kecil daripada batas bawah (%s).",
					t.UpperBound, t.LowerBound),
				t.ID))
		}
	}

	// 2 — kesinambungan antar anak tangga.
	//
	// Anak tangga berikutnya seharusnya mulai TEPAT satu rupiah di atas batas atas anak
	// tangga sebelumnya. Satuannya rupiah, bukan sen, karena seluruh isi master ditulis
	// dalam rupiah bulat dan tangganya memang melangkah demikian —
	// 50.000.000 diikuti 50.000.001.
	oneRupiah := money.FromRupiah(1)
	for i := 1; i < len(ladder); i++ {
		previous, next := ladder[i-1], ladder[i]

		// Baris ber-batas atas nol diperlakukan sebagai "tidak berbatas atas" dan
		// dilewati: pada master yang berlaku, BONDING memakai 0/0 sebagai penanda
		// bahwa tangganya memang tidak bertingkat.
		if previous.UpperBound == money.Zero {
			continue
		}

		switch {
		case next.LowerBound <= previous.UpperBound:
			findings = append(findings, build(SeverityDefect, KindOverlap,
				fmt.Sprintf("Rentang %s–%s dan %s–%s saling menindih. "+
					"Nilai di antaranya masuk dua jenjang sekaligus.",
					previous.LowerBound, previous.UpperBound,
					next.LowerBound, next.UpperBound),
				previous.ID, next.ID))

		case next.LowerBound > previous.UpperBound+oneRupiah:
			findings = append(findings, build(SeverityDefect, KindGap,
				fmt.Sprintf("Ada lubang antara %s dan %s. "+
					"Tidak ada jenjang yang secara khusus mewakili nilai di antaranya.",
					previous.UpperBound, next.LowerBound),
				previous.ID, next.ID))
		}
	}

	// 3 — jenjang ganda.
	//
	// Bukan cacat: `D-52` mencatat DEGREE hanya menentukan urutan dan boleh berulang.
	// Tetapi kueri lama mengurutkan dengan `ORDER BY DEGREE` saja, sehingga saat seri
	// urutannya ditentukan basis data dan dapat berubah antar eksekusi. Itu pantas
	// dilihat, karena ia satu-satunya sumber perbedaan urutan terhadap Pega.
	byTier := map[int][]string{}
	var tierOrder []int
	for _, t := range ladder {
		if _, exists := byTier[t.Tier]; !exists {
			tierOrder = append(tierOrder, t.Tier)
		}
		byTier[t.Tier] = append(byTier[t.Tier], t.ID)
	}
	sort.Ints(tierOrder)
	for _, tier := range tierOrder {
		if len(byTier[tier]) > 1 {
			findings = append(findings, build(SeverityWarning, KindDuplicateTier,
				fmt.Sprintf("Ada %d baris ber-jenjang %d. Urutan menyetujui di antara "+
					"keduanya tidak ditentukan master, sehingga sistem lama dapat "+
					"mengurutkannya berbeda-beda.", len(byTier[tier]), tier),
				byTier[tier]...))
		}
	}

	// 4 — atap tangga.
	//
	// Pernyataannya harus tepat, dan di sinilah tiket `TKT-B07-001` terlalu jauh: ia
	// menulis klaim di atas atap "tidak punya penyetuju sama sekali". Di bawah aturan
	// kumulatif itu TIDAK BENAR — klaim sebesar apa pun tetap memenuhi seluruh batas
	// bawah, sehingga justru mendapat SELURUH penyetuju pada tangga itu.
	//
	// Yang sebenarnya terjadi: tangganya berhenti membedakan. `D-52` sudah menyatakan
	// hal yang sama — "klaim PA atau Travel di atas nilai itu tetap mendapat 4 dan 3
	// jenjang karena model kumulatif, tetapi tidak ada baris yang secara eksplisit
	// mencakupnya".
	top := money.Zero
	var topIDs []string
	unbounded := false
	for _, t := range ladder {
		if t.UpperBound == money.Zero {
			unbounded = true
			continue
		}
		if t.UpperBound > top {
			top = t.UpperBound
			topIDs = []string{t.ID}
		}
	}

	// Pada pita bawah, atap yang sama dengan batas pita justru BENAR — di atas nilai itu
	// klaim memang berpindah ke pita berikutnya. Melaporkannya akan menjadi peringatan
	// palsu yang muncul setiap kali layar dibuka.
	expectedTop := banded && band == rule.Lower && top == rule.Boundary

	if !unbounded && !expectedTop && len(topIDs) > 0 {
		findings = append(findings, build(SeverityWarning, KindLadderTop,
			fmt.Sprintf("Tangga berhenti di %s. Klaim di atas nilai itu tetap "+
				"mendapat seluruh %d penyetuju karena aturannya kumulatif, tetapi "+
				"tidak ada jenjang yang secara khusus mewakilinya.",
				top, len(ladder)),
			topIDs...))
	}

	return findings
}
