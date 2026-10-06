package registrasi

import (
	"context"
	"strings"
	"time"
)

// # Objek, coverage, dan spreading dari polis
//
// Pengganti `Activity/CallActivityInputRegister-Act.xml` langkah 16–20 dan 41: saat klaim
// dibuka, daftar objek diisi dari polis, dan setiap objek membawa coverage beserta
// spreading-nya. Petugas tidak mengetiknya ulang.
//
// Sumbernya satu tabel per lini, dipilih `GetListObjectFromDatabase`:
//
//	PA (002), Travel (005)  POOLDATA.T_PERSONLIST    GetListObjectTravelPA
//	Fire                    POOLDATA.T_PROPERTYLIST  GetListObjectFire
//	Marine Cargo            POOLDATA.T_CARGOLIST     GetListObjectMarine
//	Aneka                   POOLDATA.T_ANEKALIST     GetListObjectAneka
//
// Coverage dan spreading TIDAK ada di tabel itu. Sampai 2026-10-06 keduanya dibaca dari
// kolom BLOB berisi dokumen JSON pada tabel objeknya sendiri; Work Owner menggantinya
// dengan tabel relasional:
//
//	PA (002), Travel (005)  POOLDATA.T_COVERAGELIST_PERSON  + T_SPREADINGLIST
//	Fire                    POOLDATA.T_COVERAGELIST_FIRE    + T_SPREADINGLIST
//	Marine Cargo            POOLDATA.T_COVERAGELIST_CARGO   + T_SPREADINGLIST
//	Aneka                   POOLDATA.T_COVERAGELIST_ANEKA   + T_SPREADINGLIST
//
// Perpindahan itu seluruhnya urusan adapter; tipe di berkas ini tidak berubah karenanya.
// SourceCoverage dan SourceSpreading tetap menyebut apa yang polis punya, bukan dari kolom
// mana nilainya datang — itulah yang membuat penggantian sumber tidak menyentuh satu pun
// aturan pembentukan klaim di bawah.

// PolicySource menyebut tabel sumber objek sebuah polis.
type PolicySource string

const (
	SourcePerson   PolicySource = "person"   // T_PERSONLIST
	SourceProperty PolicySource = "property" // T_PROPERTYLIST
	SourceCargo    PolicySource = "cargo"    // T_CARGOLIST
	SourceAneka    PolicySource = "aneka"    // T_ANEKALIST
)

// SourceOf memilih tabel sumber seperti `GetListObjectFromDatabase`.
//
// Urutan pemeriksaannya sama dengan urutan langkah activity itu: PA/Travel, Aneka, Fire,
// lalu Marine. Rule `IsFire` berlaku untuk BusinessType "Fire" atau Group Panel 006, dan
// `IsMarineCargo` untuk BusinessType "MarineCargo" atau Group Panel 004.
//
// # Aneka adalah sisanya, bukan satu kode bisnis
//
// `When/IsAneka-When.xml` di export hanya berlaku untuk kode bisnis 10140. Itu versi
// ruleset 01-01-01 di kelas induk, dan tidak cocok dengan data: 521 dari 561 klaim Pega
// Group Panel 003 berkode bisnis lain tetap berobjek. Versi rule yang sungguh berjalan
// tidak ada di export (`R-16`), sehingga Aneka di sini adalah seluruh polis yang bukan
// PA, Travel, Fire, maupun Marine.
func SourceOf(p Policy) PolicySource {
	businessType := strings.TrimSpace(p.BusinessType)
	switch {
	case p.Line == LinePersonalAccident || p.Line == LineTravel:
		return SourcePerson
	case strings.EqualFold(businessType, "Fire") || p.Line == LineFire:
		return SourceProperty
	case strings.EqualFold(businessType, "MarineCargo") || p.Line == LineMarineCargo:
		return SourceCargo
	default:
		return SourceAneka
	}
}

// SourceItem adalah satu objek sebagaimana tercatat di tabel polis, sebelum aturan
// pembentukan klaim dijalankan.
type SourceItem struct {
	ID       string
	Name     string
	Location string
	Coverage []SourceCoverage
}

// SourceCoverage adalah satu coverage polis, satu baris T_COVERAGELIST_*.
type SourceCoverage struct {
	Code string // Coverage
	Name string // CoverageNote

	TSI         Money
	SumTSI      Money
	TSISublimit Money

	// Deleted adalah `FlagDelete = "1"`.
	Deleted bool

	Spreading []SourceSpreading
}

// SourceSpreading adalah satu baris T_SPREADINGLIST.
type SourceSpreading struct {
	TreatyType string
	// TreatyName adalah nama treaty yang ditampilkan (ORS, FAC-OUT, …).
	//
	// T_SPREADINGLIST menyimpannya di TREATYNAME, tetapi kolom itu dapat kosong. Pengisi
	// seam karena itu melengkapinya dari NOTE master REINSURANCETYPE — master yang sama
	// dengan dropdown Nama Treaty Pega.
	TreatyName string
	Share      Percent
	Deleted    bool // FlagDelete = "1"
}

// PolicyItemSource adalah seam ke tabel objek polis.
//
// Pengisinya WAJIB membaca versi polis yang sama dengan snapshot klaim (`Policy.ProdKe`),
// sama seperti setiap kueri `GetListObject*` yang menyaring `prodke`.
type PolicyItemSource interface {
	Items(ctx context.Context, policy Policy) ([]SourceItem, error)
}

// BuildInsuredItems membentuk objek klaim dari objek polis.
//
// # Aturan yang dibawa
//
//   - Coverage bertanda hapus tidak ikut.
//   - TSI coverage (`ShowCoverage`): TSISublimit bila lebih dari nol, selain itu TSI, dan
//     bila TSI pun kosong, SumTSI.
//   - Spreading (`SetspreadingtoCoverage` langkah 7–8): baris bertanda hapus dibuang, baris
//     berjenis treaty sama dijumlahkan share-nya, dan baris ber-share nol atau kurang
//     dibuang.
//   - Polis kargo Open Policy (`TypeOfPolicy = '2'`) yang coverage-nya tanpa spreading
//     memakai spreading coverage pertama objek pertama (`SetspreadingtoCoverage` langkah 5).
//
// # Coverage hanya bila tanggal kejadian di dalam periode polis
//
// Setiap `GetListObject*` menyalin coverage dengan prasyarat
// `@CompareDates(DateOfLoss, StartDateTime) && @CompareDates(EndDateTime, DateOfLoss)`.
// Objek tetap dibuat; hanya coverage-nya yang tidak. Klaim yang tanggal kejadiannya
// belum diketahui mendapat coverage-nya — pemeriksaan periode tetap dijalankan gerbang
// validasi Input Register.
func BuildInsuredItems(policy Policy, lossDate time.Time, source []SourceItem) []InsuredItem {
	withCoverage := lossDate.IsZero() || withinPeriod(policy, lossDate)

	fallback := firstSpreading(source)
	openCargo := SourceOf(policy) == SourceCargo && strings.TrimSpace(policy.Kind) == "2"

	result := make([]InsuredItem, 0, len(source))
	for _, s := range source {
		// Objek tanpa ID dibuang (`GetListObjectFromDatabase` langkah 5).
		if strings.TrimSpace(s.ID) == "" {
			continue
		}
		item := InsuredItem{ID: strings.TrimSpace(s.ID), Name: strings.TrimSpace(s.Name), Location: strings.TrimSpace(s.Location)}
		if withCoverage {
			for _, c := range s.Coverage {
				if c.Deleted || strings.TrimSpace(c.Code) == "" {
					continue
				}
				spreading := mergeSpreading(c.Spreading)
				if len(spreading) == 0 && openCargo {
					spreading = mergeSpreading(fallback)
				}
				item.Coverage = append(item.Coverage, Coverage{
					ID:        strings.TrimSpace(c.Code),
					Name:      strings.TrimSpace(c.Name),
					TSI:       coverageTSI(c),
					Spreading: spreading,
				})
			}
		}
		result = append(result, item)
	}
	return result
}

func withinPeriod(policy Policy, lossDate time.Time) bool {
	if !policy.CoverageStart.IsZero() && lossDate.Before(policy.CoverageStart) {
		return false
	}
	if !policy.CoverageEnd.IsZero() && lossDate.After(policy.CoverageEnd) {
		return false
	}
	return true
}

func coverageTSI(c SourceCoverage) Money {
	switch {
	case c.TSISublimit > 0:
		return c.TSISublimit
	case c.TSI > 0:
		return c.TSI
	default:
		return c.SumTSI
	}
}

// firstSpreading adalah spreading coverage pertama objek pertama.
func firstSpreading(source []SourceItem) []SourceSpreading {
	for _, s := range source {
		for _, c := range s.Coverage {
			return c.Spreading
		}
	}
	return nil
}

// mergeSpreading menjumlahkan baris berjenis treaty sama, dengan urutan kemunculan
// pertama dipertahankan.
func mergeSpreading(rows []SourceSpreading) []Spreading {
	var result []Spreading
	index := map[string]int{}
	for _, r := range rows {
		kind := strings.TrimSpace(r.TreatyType)
		if r.Deleted || kind == "" {
			continue
		}
		if i, ok := index[kind]; ok {
			result[i].Share += r.Share
			continue
		}
		index[kind] = len(result)
		result = append(result, Spreading{TreatyKind: kind, Name: strings.TrimSpace(r.TreatyName), Share: r.Share})
	}
	kept := result[:0]
	for _, s := range result {
		if s.Share > 0 {
			kept = append(kept, s)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}
