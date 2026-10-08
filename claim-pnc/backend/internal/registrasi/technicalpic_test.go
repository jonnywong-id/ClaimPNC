package registrasi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// TestPlanTechnicalPICFollowsGetRandomTeam menjaga urutan `getRandomTeam_act` dan turunannya.
func TestPlanTechnicalPICFollowsGetRandomTeam(t *testing.T) {
	fire := registrasi.Policy{Line: registrasi.LineFire}
	with := func(edit func(*registrasi.Claim)) registrasi.Claim {
		c := registrasi.Claim{Policy: fire, CreatedBy: "ADMIN1"}
		edit(&c)
		return c
	}
	under := registrasi.Rupiah(500_000_000)

	cases := []struct {
		name  string
		claim registrasi.Claim
		est   registrasi.Money
		want  registrasi.TechnicalPICPlan
	}{
		{"SIMASNET = admin", with(func(c *registrasi.Claim) { c.Portal = "ASI" }), under,
			registrasi.TechnicalPICPlan{Operator: "ADMIN1"}},
		{"JONI_1 = admin", with(func(c *registrasi.Claim) { c.CreatedBy = "JONI_1" }), under,
			registrasi.TechnicalPICPlan{Operator: "JONI_1"}},
		{"cabang 100639", with(func(c *registrasi.Claim) { c.Policy.BranchCode = "100639" }), under,
			registrasi.TechnicalPICPlan{Operator: "DHARMANTORAHARDJO"}},
		{"PA ber-BusinessType PA", with(func(c *registrasi.Claim) {
			c.Policy.Line = registrasi.LinePersonalAccident
			c.Policy.BusinessType = "PA"
			c.TKI = true
		}), under, registrasi.TechnicalPICPlan{Pool: registrasi.PoolProcedure, TKI: true}},
		{"PA selain BusinessType PA tanpa PIC", with(func(c *registrasi.Claim) {
			c.Policy.Line = registrasi.LinePersonalAccident
			c.Policy.BusinessType = "PAYDI_PA"
		}), under, registrasi.TechnicalPICPlan{}},
		{"Travel ber-BusinessType Travel tanpa PIC (step 25)", with(func(c *registrasi.Claim) {
			c.Policy.Line = registrasi.LineTravel
			c.Policy.BusinessType = "Travel"
		}), under, registrasi.TechnicalPICPlan{}},
		{"Asuransi Kredit = admin", with(func(c *registrasi.Claim) { c.Policy.BusinessCode = "10053" }), under,
			registrasi.TechnicalPICPlan{Operator: "ADMIN1"}},
		{"Bonding = admin", with(func(c *registrasi.Claim) { c.Policy.BusinessType = "BondingKBG" }), under,
			registrasi.TechnicalPICPlan{Operator: "ADMIN1"}},
		{"NONMBU leader < 1M", with(func(c *registrasi.Claim) { c.Policy.Coinsurance.Role = "LEADER" }), under,
			registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU}},
		{"NONMBU > 1M", with(func(*registrasi.Claim) {}), registrasi.Rupiah(1_000_000_001),
			registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU, Large: true}},
		{"tepat 1M tanpa kandidat", with(func(*registrasi.Claim) {}), registrasi.Rupiah(1_000_000_000),
			registrasi.TechnicalPICPlan{}},
		{"Fac In = tim C tanpa kandidat tetap", with(func(c *registrasi.Claim) {
			c.Policy.TypeOfCoins = "F"
			c.Policy.SourceOfBusiness = "10001551"
		}), under, registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU, TeamC: true}},
		{"member = tim C", with(func(c *registrasi.Claim) { c.Policy.Coinsurance.Role = "MEMBER" }), under,
			registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU, TeamC: true}},
		{"member sumbis IBS", with(func(c *registrasi.Claim) {
			c.Policy.Coinsurance.Role = "MEMBER"
			c.Policy.SourceOfBusiness = "10000952"
		}), under, registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU, TeamC: true, Preferred: []string{"YOSECHRISTOFER"}}},
		{"member kode bisnis 10029", with(func(c *registrasi.Claim) {
			c.Policy.Coinsurance.Role = "MEMBER"
			c.Policy.BusinessCode = "10029"
		}), under, registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU, TeamC: true, Preferred: []string{"TONY"}}},
		{"member Asuransi TOTAL", with(func(c *registrasi.Claim) {
			c.Policy.Coinsurance.Role = "MEMBER"
			c.Policy.SourceOfBusiness = "10053930"
		}), under, registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU, TeamC: true, Preferred: []string{"YOSECHRISTOFER", "TONY"}}},
		{"member KBRU", with(func(c *registrasi.Claim) {
			c.Policy.Coinsurance.Role = "MEMBER"
			c.Policy.SourceOfBusinessName = "PT KBRU"
		}), under, registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU, TeamC: true, Preferred: []string{"BERITAARIELIEZERTARIGAN@GMAIL.COM", "TONY"}}},
		{"leader sumbis IBS tanpa kandidat tetap", with(func(c *registrasi.Claim) {
			c.Policy.Coinsurance.Role = "LEADER"
			c.Policy.SourceOfBusiness = "10000952"
		}), under, registrasi.TechnicalPICPlan{Pool: registrasi.PoolNonMBU}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, registrasi.PlanTechnicalPIC(c.claim, c.est))
		})
	}
}

// TestFaceSheetTechnicalPICOverrides: `DownloadClaimFaceSheet_act` step 8 dan 12.
func TestFaceSheetTechnicalPICOverrides(t *testing.T) {
	phk := registrasi.Claim{InsuredItem: []registrasi.InsuredItem{{Coverage: []registrasi.Coverage{{ID: "10023"}}}}}
	require.Equal(t, "BAMBANGSETIADJIGUNAWAN", registrasi.FaceSheetTechnicalPIC(phk))
	require.Equal(t, "JONI_1", registrasi.FaceSheetTechnicalPIC(registrasi.Claim{CreatedBy: "JONI_1"}))
	require.Empty(t, registrasi.FaceSheetTechnicalPIC(registrasi.Claim{}))

	require.False(t, registrasi.HasTechnicalPIC(registrasi.Claim{TechnicalPIC: " - "}))
	require.Equal(t, "DIBADYASANTI", registrasi.PATechnicalPIC(true))
}

// TestAttendanceRules: getRandomTeam_act step 15.6–15.8.
func TestAttendanceRules(t *testing.T) {
	require.True(t, registrasi.Attendance{Day: "SABTU"}.Closed())
	require.True(t, registrasi.Attendance{Day: "MINGGU"}.Closed())
	require.True(t, registrasi.Attendance{Holiday: "1"}.Closed())
	require.False(t, registrasi.Attendance{Day: "SENIN", Holiday: "0"}.Closed())
	require.True(t, registrasi.Attendance{RuleTimeIn: "000000"}.Absent())
	require.False(t, registrasi.Attendance{RuleTimeIn: "080000"}.Absent())

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	claim := registrasi.Claim{DateOfLoss: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Policy: registrasi.Policy{Number: "P1", CoverageStart: start, CoverageEnd: end}}
	require.True(t, registrasi.AttendanceApplies(claim))
	claim.DateOfLoss = end.AddDate(0, 0, 1)
	require.False(t, registrasi.AttendanceApplies(claim))
	claim.DateOfLoss, claim.Policy.Number = start, ""
	require.False(t, registrasi.AttendanceApplies(claim))
}
