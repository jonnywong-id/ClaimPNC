package registrasi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, clock.ZoneWIB)
}

// validClaim menyusun klaim yang lolos seluruh aturan, supaya tiap uji hanya perlu
// merusak satu hal.
func validClaim() registrasi.Claim {
	return registrasi.Claim{
		ID:     "klaim-uji",
		Number: "PNCN.26.0001",
		Policy: registrasi.Policy{
			Number:        "POL-1",
			Line:          registrasi.LineFire,
			CoverageStart: date(2026, time.January, 1),
			CoverageEnd:   date(2026, time.December, 31),
			Currency:      "IDR",
		},
		DateOfLoss:    date(2026, time.June, 1),
		ReportDate:    date(2026, time.June, 2),
		DateReceived:  date(2026, time.June, 3),
		Location:      "Gudang A",
		EstimateValue: registrasi.Rupiah(10_000_000),
		Currency:      "IDR",
		InsuredItem: []registrasi.InsuredItem{{
			ID:   "OBJ-1",
			Name: "Gudang",
			Coverage: []registrasi.Coverage{{
				ID:          "CVG-1",
				CauseOfLoss: "11817",
				TSI:         registrasi.Rupiah(500_000_000),
				Spreading: []registrasi.Spreading{
					{TreatyKind: "10007", Name: "OR", Share: 600_000},
					{TreatyKind: "10008", Name: "Treaty", Share: 400_000},
				},
			}},
		}},
	}
}

func parts() registrasi.Parts {
	return registrasi.Parts{Now: date(2026, time.June, 10)}
}

func violations(t *testing.T, err error) *registrasi.ValidationError {
	t.Helper()
	require.Error(t, err)
	g, ok := err.(*registrasi.ValidationError)
	require.True(t, ok, "galat harus *GalatValidasi, bukan %T", err)
	return g
}

func TestValidClaimPassesEveryRule(t *testing.T) {
	require.NoError(t, registrasi.Validate(validClaim(), parts()))
}

// TestDateOrdering menguji ketiga aturan urutan pada kasus batas: sama persis, dan
// selisih satu hari ke arah yang salah.
func TestDateOrdering(t *testing.T) {
	t.Run("tanggal sama diterima", func(t *testing.T) {
		k := validClaim()
		k.ReportDate = k.DateOfLoss
		k.DateReceived = k.DateOfLoss
		require.NoError(t, registrasi.Validate(k, parts()))
	})

	t.Run("lapor sebelum kejadian ditolak", func(t *testing.T) {
		k := validClaim()
		k.ReportDate = date(2026, time.May, 31)
		k.DateReceived = date(2026, time.May, 31)

		g := violations(t, registrasi.Validate(k, parts()))
		require.True(t, g.Has(registrasi.ViolationReportDateBeforeLoss))
	})

	t.Run("terima dokumen sebelum lapor ditolak", func(t *testing.T) {
		k := validClaim()
		k.DateReceived = date(2026, time.June, 1)
		k.ReportDate = date(2026, time.June, 2)

		g := violations(t, registrasi.Validate(k, parts()))
		require.True(t, g.Has(registrasi.ViolationReceivedBeforeReport))
	})

	t.Run("tanggal di masa depan ditolak", func(t *testing.T) {
		k := validClaim()
		k.DateOfLoss = date(2026, time.June, 11)
		k.ReportDate = date(2026, time.June, 11)
		k.DateReceived = date(2026, time.June, 11)

		g := violations(t, registrasi.Validate(k, parts()))
		require.True(t, g.Has(registrasi.ViolationLossDateInFuture))
		require.True(t, g.Has(registrasi.ViolationReportDateInFuture))
		require.True(t, g.Has(registrasi.ViolationReceivedInFuture))
	})
}

// TestPolicyPeriodBoundaries menguji keempat batas periode polis.
func TestPolicyPeriodBoundaries(t *testing.T) {
	cases := []struct {
		name     string
		lossDate time.Time
		rejected bool
	}{
		{"hari pertama periode diterima", date(2026, time.January, 1), false},
		{"sehari sebelum periode ditolak", date(2025, time.December, 31), true},
		{"hari terakhir periode diterima", date(2026, time.December, 31), false},
		{"sehari setelah periode ditolak", date(2027, time.January, 1), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k := validClaim()
			k.DateOfLoss = tc.lossDate
			k.ReportDate = tc.lossDate
			k.DateReceived = tc.lossDate

			b := registrasi.Parts{Now: date(2028, time.January, 1)}
			err := registrasi.Validate(k, b)
			if !tc.rejected {
				if err != nil {
					g := violations(t, err)
					require.False(t, g.Has(registrasi.ViolationLossOutsidePolicyPeriod))
				}
				return
			}
			g := violations(t, err)
			require.True(t, g.Has(registrasi.ViolationLossOutsidePolicyPeriod))
		})
	}
}

// TestPolicyPeriodIndependentOfTimeOfDay adalah bukti butir 3 dari 13 perbaikan `P-5`
// tertutup.
//
// Di sistem lama, satu sisi perbandingan digeser tujuh jam dan sisi lainnya tidak — di
// dalam kondisi yang sama. Akibatnya tanggal kejadian tepat di batas periode bisa lolos
// atau ditolak tergantung jam berapa datanya tersimpan. Uji ini menjalankan aturan yang
// sama pada 24 jam berbeda dan menuntut hasil yang identik.
func TestPolicyPeriodIndependentOfTimeOfDay(t *testing.T) {
	for hour := 0; hour < 24; hour++ {
		k := validClaim()
		k.Policy.CoverageEnd = time.Date(2026, time.December, 31, hour, 0, 0, 0, clock.ZoneWIB)
		k.DateOfLoss = time.Date(2026, time.December, 31, 23-hour, 30, 0, 0, clock.ZoneWIB)
		k.ReportDate = k.DateOfLoss
		k.DateReceived = k.DateOfLoss

		b := registrasi.Parts{Now: date(2027, time.January, 5)}
		err := registrasi.Validate(k, b)
		if err != nil {
			g := violations(t, err)
			require.False(t, g.Has(registrasi.ViolationLossOutsidePolicyPeriod),
				"jam %d menghasilkan keputusan berbeda", hour)
		}
	}
}

// TestTravelDocumentReceiptLimit menguji hari ke-90 diterima dan ke-91 ditolak.
func TestTravelDocumentReceiptLimit(t *testing.T) {
	create := func(delta int) registrasi.Claim {
		k := validClaim()
		k.Policy.Line = registrasi.LineTravel
		k.DateOfLoss = date(2026, time.January, 1)
		k.ReportDate = date(2026, time.January, 1)
		k.DateReceived = clock.AddDays(k.DateOfLoss, delta)
		// Lini Travel dikecualikan dari kewajiban penyebab kerugian.
		k.InsuredItem[0].Coverage[0].CauseOfLoss = ""
		return k
	}
	b := registrasi.Parts{Now: date(2026, time.June, 1)}

	t.Run("hari ke-90 diterima", func(t *testing.T) {
		err := registrasi.Validate(create(90), b)
		if err != nil {
			g := violations(t, err)
			require.False(t, g.Has(registrasi.ViolationReceivedAfter90Days))
		}
	})

	t.Run("hari ke-91 ditolak", func(t *testing.T) {
		g := violations(t, registrasi.Validate(create(91), b))
		require.True(t, g.Has(registrasi.ViolationReceivedAfter90Days))
	})
}

// TestSevenDayReportLimit menjaga operator yang terbaca dari source tetap seperti
// adanya.
//
// Pesan di sistem lama berbunyi "tidak boleh lebih dari 7 hari", tetapi kondisinya
// menolak SEJAK hari ke-7. Selisih antara pesan dan aturan itu dibawa apa adanya
// (`P-5`), dan uji ini yang menjaganya tetap terlihat.
func TestSevenDayReportLimit(t *testing.T) {
	create := func(line registrasi.LineOfBusiness, delta int) registrasi.Claim {
		k := validClaim()
		k.Policy.Line = line
		k.DateOfLoss = date(2026, time.March, 1)
		k.ReportDate = clock.AddDays(k.DateOfLoss, delta)
		k.DateReceived = k.ReportDate
		return k
	}
	b := registrasi.Parts{Now: date(2026, time.June, 1)}

	t.Run("hari ke-6 diterima", func(t *testing.T) {
		err := registrasi.Validate(create(registrasi.LineFire, 6), b)
		if err != nil {
			g := violations(t, err)
			require.False(t, g.Has(registrasi.ViolationReportedAfter7Days))
		}
	})

	t.Run("hari ke-7 ditolak", func(t *testing.T) {
		g := violations(t, registrasi.Validate(create(registrasi.LineFire, 7), b))
		require.True(t, g.Has(registrasi.ViolationReportedAfter7Days))
	})

	t.Run("lini Personal Accident dikecualikan", func(t *testing.T) {
		k := create(registrasi.LinePersonalAccident, 30)
		err := registrasi.Validate(k, b)
		if err != nil {
			g := violations(t, err)
			require.False(t, g.Has(registrasi.ViolationReportedAfter7Days))
		}
	})
}

func TestCauseOfLossRequiredExceptTravel(t *testing.T) {
	t.Run("lini selain Travel menuntut penyebab kerugian", func(t *testing.T) {
		k := validClaim()
		k.InsuredItem[0].Coverage[0].CauseOfLoss = ""

		g := violations(t, registrasi.Validate(k, parts()))
		require.True(t, g.Has(registrasi.ViolationCauseOfLossEmpty))
	})

	t.Run("lini Travel dikecualikan", func(t *testing.T) {
		k := validClaim()
		k.Policy.Line = registrasi.LineTravel
		k.InsuredItem[0].Coverage[0].CauseOfLoss = ""

		err := registrasi.Validate(k, parts())
		if err != nil {
			g := violations(t, err)
			require.False(t, g.Has(registrasi.ViolationCauseOfLossEmpty))
		}
	})
}

func TestSLIKNumberRequiredForCreditGuarantee(t *testing.T) {
	k := validClaim()
	k.Policy.CreditGuarantee = true

	g := violations(t, registrasi.Validate(k, parts()))
	require.True(t, g.Has(registrasi.ViolationSLIKNumberEmpty))

	k.SLIKNumber = "SLIK-001"
	require.NoError(t, registrasi.Validate(k, parts()))
}

func TestItemWithoutCoverageRejected(t *testing.T) {
	k := validClaim()
	k.InsuredItem = append(k.InsuredItem, registrasi.InsuredItem{ID: "OBJ-2", Name: "Mesin"})

	g := violations(t, registrasi.Validate(k, parts()))
	require.True(t, g.Has(registrasi.ViolationItemWithoutCoverage))
}

func TestEstimateMayNotExceedTSI(t *testing.T) {
	k := validClaim()
	k.EstimateValue = registrasi.Rupiah(500_000_001)

	g := violations(t, registrasi.Validate(k, parts()))
	require.True(t, g.Has(registrasi.ViolationEstimateExceedsTSI))

	k.EstimateValue = registrasi.Rupiah(500_000_000)
	require.NoError(t, registrasi.Validate(k, parts()))
}

// TestSpreadingTotal menguji toleransi `ADR-0016`, termasuk dua nilai yang di sistem
// lama LOLOS karena pencocokannya memakai substring.
func TestSpreadingTotal(t *testing.T) {
	create := func(share ...registrasi.Percent) registrasi.Claim {
		k := validClaim()
		row := make([]registrasi.Spreading, 0, len(share))
		for _, s := range share {
			row = append(row, registrasi.Spreading{TreatyKind: "10008", Name: "T", Share: s})
		}
		k.InsuredItem[0].Coverage[0].Spreading = row
		return k
	}

	cases := []struct {
		name     string
		share    []registrasi.Percent
		rejected bool
	}{
		{"tepat 100%", []registrasi.Percent{1_000_000}, false},
		{"batas bawah 99,9999%", []registrasi.Percent{999_999}, false},
		{"batas atas 100,0001%", []registrasi.Percent{1_000_001}, false},
		{"99,9998% ditolak", []registrasi.Percent{999_998}, true},
		{"100,0002% ditolak", []registrasi.Percent{1_000_002}, true},
		// Kedua nilai berikut LOLOS di sistem lama: `@contains(total,"100.0")` benar
		// untuk 1100.0, dan `@contains(total,"99.99")` benar untuk 199.99.
		{"199,99% ditolak", []registrasi.Percent{1_999_900}, true},
		{"1100% ditolak", []registrasi.Percent{11_000_000}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := registrasi.Validate(create(tc.share...), parts())
			if !tc.rejected {
				require.NoError(t, err)
				return
			}
			g := violations(t, err)
			require.True(t, g.Has(registrasi.ViolationSpreadingTotalNot100))
		})
	}
}

func TestRemovedSpreadingNotCounted(t *testing.T) {
	k := validClaim()
	k.InsuredItem[0].Coverage[0].Spreading = append(k.InsuredItem[0].Coverage[0].Spreading,
		registrasi.Spreading{TreatyKind: "10008", Name: "Dibatalkan", Share: 500_000, Removed: true})

	require.NoError(t, registrasi.Validate(k, parts()))
}

func TestFacOutRequiresFacOfferItem(t *testing.T) {
	k := validClaim()
	k.InsuredItem[0].Coverage[0].Spreading[1].TreatyKind = registrasi.TreatyFacOut

	g := violations(t, registrasi.Validate(k, parts()))
	require.True(t, g.Has(registrasi.ViolationFacOfferIncomplete))

	k.InsuredItem[0].Coverage[0].Spreading[1].FacOfferItem = "Gudang A"
	require.NoError(t, registrasi.Validate(k, parts()))
}

func TestDuplicateClaimNamesExistingNumber(t *testing.T) {
	b := parts()
	b.Duplicates = []registrasi.DuplicateClaim{{Number: "PNCN.26.0007", InsuredItem: "OBJ-1"}}

	g := violations(t, registrasi.Validate(validClaim(), b))
	require.True(t, g.Has(registrasi.ViolationDuplicateClaim))

	p, ok := g.First()
	require.True(t, ok)
	require.Contains(t, p.Message, "PNCN.26.0007")
}

// TestDuplicateKeys menguji perbedaan kunci antar-lini yang terbaca dari langkah 32
// dan 33.
func TestDuplicateKeys(t *testing.T) {
	t.Run("lini selain PA menyertakan lokasi", func(t *testing.T) {
		key := registrasi.DuplicateKeys(validClaim())
		require.Len(t, key, 1)
		require.Equal(t, "Gudang A", key[0].Location)
		require.Empty(t, key[0].CauseOfLoss)
	})

	t.Run("lini PA memakai dua kunci", func(t *testing.T) {
		k := validClaim()
		k.Policy.Line = registrasi.LinePersonalAccident
		k.InsuredItem[0].Coverage[0].CauseOfLoss = registrasi.CauseOfLossPA

		key := registrasi.DuplicateKeys(k)
		require.Len(t, key, 2)

		// Kunci pertama TANPA lokasi — langkah 32.2 dilewati saat IsPA.
		require.Empty(t, key[0].Location)
		require.Empty(t, key[0].CauseOfLoss)

		// Kunci kedua DENGAN lokasi dan penyebab kerugian 12002 — langkah 33.
		require.Equal(t, "Gudang A", key[1].Location)
		require.Equal(t, registrasi.CauseOfLossPA, key[1].CauseOfLoss)
	})

	t.Run("lini PA tanpa penyebab 12002 hanya memakai satu kunci", func(t *testing.T) {
		k := validClaim()
		k.Policy.Line = registrasi.LinePersonalAccident

		require.Len(t, registrasi.DuplicateKeys(k), 1)
	})
}

// TestViolationOrderFollowsPega menjaga janji kesetaraan: pelanggaran
// pertama selalu sama dengan satu-satunya pesan yang ditampilkan Pega.
func TestViolationOrderFollowsPega(t *testing.T) {
	k := validClaim()
	k.Policy.CreditGuarantee = true         // langkah 4.2 dan 5 — paling awal
	k.ReportDate = date(2026, time.May, 20) // langkah 22 — jauh setelahnya
	k.DateReceived = date(2026, time.May, 20)

	g := violations(t, registrasi.Validate(k, parts()))
	p, ok := g.First()
	require.True(t, ok)
	require.Equal(t, registrasi.ViolationSLIKNumberEmpty, p.Code)
}

func TestExchangeConversionKeepsFullPrecision(t *testing.T) {
	// USD 1.000,00 pada kurs 16.250,5000 menjadi Rp 16.250.500,00 — tanpa pembulatan
	// di tengah jalan.
	value := registrasi.Rupiah(1_000)
	result := value.Convert(registrasi.ExchangeRate(162_505_000))
	require.Equal(t, registrasi.Rupiah(16_250_500), result)
}

func TestFormatRupiah(t *testing.T) {
	require.Equal(t, "Rp 1.000.000,00", registrasi.FormatRupiah(registrasi.Rupiah(1_000_000)))
	require.Equal(t, "Rp 0,00", registrasi.FormatRupiah(0))
	require.Equal(t, "Rp 1.234,56", registrasi.FormatRupiah(registrasi.Money(123_456)))
}
