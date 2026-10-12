package komite_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/repo/memory"
)

// saring mengambil temuan berjenis tertentu, supaya pernyataan di bawah menunjuk hal
// yang tepat alih-alih menghitung panjang senarai.
func saring(temuan []komite.Finding, jenis string) []komite.Finding {
	var hasil []komite.Finding
	for _, t := range temuan {
		if t.Kind == jenis {
			hasil = append(hasil, t)
		}
	}
	return hasil
}

func punyaLini(temuan []komite.Finding, lini komite.BusinessLine) bool {
	for _, t := range temuan {
		if t.BusinessLine == lini {
			return true
		}
	}
	return false
}

// Master yang berlaku hari ini TIDAK punya satu pun cacat — tidak ada rentang yang
// tumpang tindih, berlubang, maupun terbalik.
//
// Pengujian ini bernilai justru karena ia lulus: bila kelak seseorang menyunting master
// dan merusak tangganya, inilah yang gagal lebih dulu, sebelum ada klaim yang disetujui
// oleh jumlah orang yang keliru.
func TestMasterYangBerlakuTidakPunyaCacat(t *testing.T) {
	temuan := komite.CheckIntegrity(memory.SampleThresholds(), komite.DefaultPolicy())

	for _, s := range temuan {
		require.NotEqual(t, komite.SeverityDefect, s.Severity,
			"master yang berlaku seharusnya bersih dari cacat, tetapi ditemukan: %s — %s",
			s.BusinessLine, s.Message)
	}
}

// Atap tangga PA dan Travel dilaporkan, dan inilah pertanyaan terbuka pada
// `TKT-B07-001` yang kini terjawab sebagai laporan otomatis alih-alih catatan lepas.
//
// Perhatikan pernyataannya: tangganya BERHENTI MEMBEDAKAN di Rp 200.000.000 — bukan
// "tidak punya penyetuju sama sekali" seperti yang tertulis di tiket. Di bawah aturan
// kumulatif, klaim di atas nilai itu justru mendapat SELURUH penyetuju.
func TestAtapTanggaPADanTravelDilaporkan(t *testing.T) {
	temuan := komite.CheckIntegrity(memory.SampleThresholds(), komite.DefaultPolicy())
	atap := saring(temuan, komite.KindLadderTop)

	require.True(t, punyaLini(atap, "PA"), "atap tangga PA harus dilaporkan")
	require.True(t, punyaLini(atap, "TRAVEL"), "atap tangga Travel harus dilaporkan")

	for _, s := range atap {
		require.Equal(t, komite.SeverityWarning, s.Severity,
			"atap tangga adalah peringatan, bukan cacat — klaimnya tetap punya penyetuju")
	}

	// Dan pengujian di jenjang_test membuktikan pernyataan itu benar: PA Rp 5 miliar
	// tetap mendapat 4 penyetuju.
	hasil := tentukan(t, rp(5_000_000_000), "PA")
	require.Equal(t, 4, hasil.TierCount())
}

// Pita bawah Non-MBU berhenti TEPAT di batas pita, dan itu memang benar: di atas nilai
// itu klaim berpindah ke pita berikutnya. Melaporkannya akan menjadi peringatan palsu
// yang muncul setiap kali layar dibuka.
func TestAtapPitaBawahTidakDilaporkanKarenaMemangBenar(t *testing.T) {
	temuan := komite.CheckIntegrity(memory.SampleThresholds(), komite.DefaultPolicy())

	for _, s := range saring(temuan, komite.KindLadderTop) {
		if s.BusinessLine == komite.BusinessLineNonMBU && s.Band == komite.BandLower {
			t.Fatalf("atap pita bawah Non-MBU tidak boleh dilaporkan: %s", s.Message)
		}
	}
}

// Dua baris Non-MBU pita 1 ber-DEGREE sama dilaporkan sebagai peringatan.
//
// Ia bukan cacat — `D-52` mencatat DEGREE hanya menentukan urutan dan boleh berulang —
// tetapi ia satu-satunya sumber perbedaan urutan terhadap Pega, dan karena itu pantas
// terlihat sebelum uji kesetaraan dijalankan.
func TestJenjangGandaDilaporkanSebagaiPeringatan(t *testing.T) {
	temuan := komite.CheckIntegrity(memory.SampleThresholds(), komite.DefaultPolicy())
	ganda := saring(temuan, komite.KindDuplicateTier)

	require.NotEmpty(t, ganda, "dua baris Non-MBU ber-DEGREE 1 harus dilaporkan")
	require.True(t, punyaLini(ganda, komite.BusinessLineNonMBU))

	for _, s := range ganda {
		require.Equal(t, komite.SeverityWarning, s.Severity)
		require.Len(t, s.ThresholdIDs, 2, "kedua baris yang seri harus disebutkan")
	}
}

// Pengelompokan PA HARUS per lini, bukan per TYPE_KOMITE.
//
// Bila dipaksa per TYPE_KOMITE, tangga PA terbelah menjadi dua potongan — {0–10jt,
// 100jt–200jt} dan {10jt–50jt, 50jt–100jt} — yang tampak berlubang parah. Padahal pada
// PA kolom itu membedakan PA reguler dari PA TKI, bukan pita nilai (`D-70`).
func TestPengelompokanPAPerLiniBukanPerTypeKomite(t *testing.T) {
	temuan := komite.CheckIntegrity(memory.SampleThresholds(), komite.DefaultPolicy())

	for _, s := range saring(temuan, komite.KindGap) {
		require.NotEqual(t, komite.BusinessLine("PA"), s.BusinessLine,
			"tangga PA dilaporkan berlubang — tanda pengelompokannya memakai TYPE_KOMITE")
	}
	for _, s := range saring(temuan, komite.KindOverlap) {
		require.NotEqual(t, komite.BusinessLine("PA"), s.BusinessLine)
	}
}

func TestRentangTumpangTindihTerdeteksi(t *testing.T) {
	ambang := []komite.Threshold{
		{
			ID: "1", Name: "A", OperatorID: "A", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		},
		{
			// Mulai SEBELUM baris sebelumnya berakhir.
			ID: "2", Name: "B", OperatorID: "B", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(40_000_000), UpperBound: rp(100_000_000), Tier: 2,
			Active: true, ForAdjustment: true,
		},
	}

	temuan := saring(komite.CheckIntegrity(ambang, komite.DefaultPolicy()),
		komite.KindOverlap)

	require.Len(t, temuan, 1)
	require.Equal(t, komite.SeverityDefect, temuan[0].Severity)
	require.ElementsMatch(t, []string{"1", "2"}, temuan[0].ThresholdIDs)
}

func TestRentangBerlubangTerdeteksi(t *testing.T) {
	ambang := []komite.Threshold{
		{
			ID: "1", Name: "A", OperatorID: "A", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		},
		{
			// Melompat, meninggalkan Rp 50.000.001 sampai Rp 59.999.999 tanpa jenjang
			// yang secara khusus mewakilinya.
			ID: "2", Name: "B", OperatorID: "B", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(60_000_000), UpperBound: rp(100_000_000), Tier: 2,
			Active: true, ForAdjustment: true,
		},
	}

	temuan := saring(komite.CheckIntegrity(ambang, komite.DefaultPolicy()),
		komite.KindGap)

	require.Len(t, temuan, 1)
	require.Equal(t, komite.SeverityDefect, temuan[0].Severity)
}

// Selisih TEPAT satu rupiah adalah sambungan yang benar, bukan lubang. Inilah pola yang
// dipakai seluruh tangga di master: 50.000.000 diikuti 50.000.001.
func TestSelisihSatuRupiahBukanLubang(t *testing.T) {
	ambang := []komite.Threshold{
		{
			ID: "1", Name: "A", OperatorID: "A", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		},
		{
			ID: "2", Name: "B", OperatorID: "B", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(50_000_001), UpperBound: rp(100_000_000), Tier: 2,
			Active: true, ForAdjustment: true,
		},
	}

	temuan := komite.CheckIntegrity(ambang, komite.DefaultPolicy())
	require.Empty(t, saring(temuan, komite.KindGap))
	require.Empty(t, saring(temuan, komite.KindOverlap))
}

func TestBatasTerbalikTerdeteksi(t *testing.T) {
	ambang := []komite.Threshold{{
		ID: "1", Name: "A", OperatorID: "A", BusinessLine: "PA", CommitteeType: "1",
		LowerBound: rp(100_000_000), UpperBound: rp(50_000_000), Tier: 1,
		Active: true, ForAdjustment: true,
	}}

	temuan := saring(komite.CheckIntegrity(ambang, komite.DefaultPolicy()),
		komite.KindInvertedBounds)

	require.Len(t, temuan, 1)
	require.Equal(t, komite.SeverityDefect, temuan[0].Severity)
}

// Batas atas nol berarti "tidak berbatas atas", bukan batas terbalik. Bonding memakainya
// demikian, dan kueri Bonding di sistem lama memang tidak menyaring LIMIT sama sekali.
func TestBatasAtasNolBukanCacat(t *testing.T) {
	ambang := []komite.Threshold{{
		ID: "74", Name: "RIZALGREATLIN", OperatorID: "RIZALGREATLIN",
		BusinessLine: "BONDING", CommitteeType: "1",
		LowerBound: rp(0), UpperBound: rp(0), Tier: 1,
		Active: true, ForAdjustment: true,
	}}

	temuan := komite.CheckIntegrity(ambang, komite.DefaultPolicy())
	for _, s := range temuan {
		require.NotEqual(t, komite.SeverityDefect, s.Severity, s.Message)
	}
	require.Empty(t, saring(temuan, komite.KindLadderTop),
		"tangga tanpa batas atas tidak punya atap untuk dilaporkan")
}

// Master yang tidak punya satu pun jenjang aktif adalah cacat, bukan keadaan kosong yang
// wajar: tidak ada klaim yang dapat melewati komite sama sekali.
func TestMasterTanpaJenjangAktifAdalahCacat(t *testing.T) {
	ambang := []komite.Threshold{{
		ID: "1", Name: "A", OperatorID: "A", BusinessLine: "PA", CommitteeType: "1",
		LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
		Active: false, ForAdjustment: true,
	}}

	temuan := saring(komite.CheckIntegrity(ambang, komite.DefaultPolicy()),
		komite.KindNoTier)

	require.Len(t, temuan, 1)
	require.Equal(t, komite.SeverityDefect, temuan[0].Severity)
}

// Pemeriksaan tidak boleh mengubah senarai milik pemanggil. Ia mengurutkan di dalamnya,
// dan tanpa salinan, urutan master yang dipakai layar akan ikut berubah diam-diam.
func TestPeriksaIntegritasTidakMengubahMasukan(t *testing.T) {
	asli := memory.SampleThresholds()
	salinan := append([]komite.Threshold(nil), asli...)

	komite.CheckIntegrity(asli, komite.DefaultPolicy())

	require.Equal(t, salinan, asli, "senarai masukan ikut terurut — salinan tidak dibuat")
}
