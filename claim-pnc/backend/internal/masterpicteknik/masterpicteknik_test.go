package masterpicteknik_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpicteknik"
)

// valid adalah baris yang lolos seluruh aturan, dipakai sebagai titik awal setiap kasus
// supaya yang diuji benar-benar SATU aturan — bukan kombinasi yang kebetulan gagal.
func valid() masterpicteknik.Technician {
	return masterpicteknik.Technician{
		OperatorID:          "PICTEKNIK01",
		Email:               "petugas@example.invalid",
		BusinessLine:        "NONMBU",
		Group:               "A",
		Supervisor:          "PICTEKNIK02",
		ClaimCounterBelow1M: 10,
		Active:              true,
	}
}

func fields(violation []masterpicteknik.Violation) []string {
	result := make([]string, 0, len(violation))
	for _, v := range violation {
		result = append(result, v.Field)
	}
	return result
}

func TestCheckAcceptsValidTechnician(t *testing.T) {
	require.Empty(t, masterpicteknik.Check(valid()))
}

func TestCheckRejectsEachRule(t *testing.T) {
	cases := []struct {
		name  string
		build func(masterpicteknik.Technician) masterpicteknik.Technician
		field string
	}{
		{
			name:  "id operator kosong",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician { t.OperatorID = "  "; return t },
			field: masterpicteknik.FieldOperatorID,
		},
		{
			name: "id operator terlalu panjang",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician {
				t.OperatorID = strings.Repeat("A", masterpicteknik.MaxOperatorIDLength+1)
				return t
			},
			field: masterpicteknik.FieldOperatorID,
		},
		{
			name:  "email tanpa domain",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician { t.Email = "petugas@"; return t },
			field: masterpicteknik.FieldEmail,
		},
		{
			name:  "kuota negatif",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician { t.ClaimCounterBelow1M = -1; return t },
			field: masterpicteknik.FieldClaimCounterBelow1M,
		},
		{
			name: "kuota melebihi batas",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician {
				t.ClaimCounterBelow1M = masterpicteknik.MaxClaimCounter + 1
				return t
			},
			field: masterpicteknik.FieldClaimCounterBelow1M,
		},
		{
			name:  "kuota sistem lain negatif",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician { t.ClaimCounterAbove1M = -1; return t },
			field: masterpicteknik.FieldClaimCounterAbove1M,
		},
		{
			name: "kelompok di luar daftar Property",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician {
				t.Group = "Z"
				return t
			},
			field: masterpicteknik.FieldGroup,
		},
		{
			name: "bisnis di luar daftar Property",
			build: func(t masterpicteknik.Technician) masterpicteknik.Technician {
				t.BusinessLine = "MBU"
				return t
			},
			field: masterpicteknik.FieldBusinessLine,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			violation := masterpicteknik.Check(c.build(valid()))
			require.Contains(t, fields(violation), c.field)
		})
	}
}

// Kuota tepat pada batas harus DITERIMA. Kasus "tepat di batas" adalah tempat aturan
// berangka paling sering salah — `docs/Steering/14-TESTING-STRATEGY.md` §3.1 mewajibkannya.
func TestCheckAcceptsQuotaAtBoundary(t *testing.T) {
	for _, quota := range []int{0, masterpicteknik.MaxClaimCounter} {
		technician := valid()
		technician.ClaimCounterBelow1M = quota
		technician.ClaimCounterAbove1M = quota
		require.Empty(t, masterpicteknik.Check(technician))
	}
}

// Petugas tidak boleh menjadi atasan dirinya sendiri — rujukan melingkar membuat penelusuran
// rantai atasan berputar tanpa henti.
func TestCheckRejectsSelfSupervision(t *testing.T) {
	technician := valid()
	technician.Supervisor = technician.OperatorID
	require.Contains(t, fields(masterpicteknik.Check(technician)), masterpicteknik.FieldSupervisor)
}

// Perbandingannya mengabaikan besar-kecil huruf dan spasi tepi, sama dengan IDKey. Tanpa
// itu, "picteknik01" akan lolos sebagai atasan dirinya sendiri.
func TestCheckRejectsSelfSupervisionRegardlessOfCase(t *testing.T) {
	technician := valid()
	technician.Supervisor = "  picteknik01  "
	require.Contains(t, fields(masterpicteknik.Check(technician)), masterpicteknik.FieldSupervisor)
}

// Seluruh pelanggaran dikumpulkan sekaligus, bukan berhenti pada yang pertama — itu
// kesetaraan perilaku dengan sistem lama, bukan kerapian.
func TestCheckCollectsEveryViolationAtOnce(t *testing.T) {
	violation := masterpicteknik.Check(masterpicteknik.Technician{
		OperatorID:          "",
		Email:               "bukan-email",
		ClaimCounterBelow1M: -5,
	})

	found := fields(violation)
	require.Contains(t, found, masterpicteknik.FieldOperatorID)
	require.Contains(t, found, masterpicteknik.FieldEmail)
	require.Contains(t, found, masterpicteknik.FieldClaimCounterBelow1M)
	require.GreaterOrEqual(t, len(violation), 3)
}

func TestNewValidationErrorIsTrulyNilWhenClean(t *testing.T) {
	// Nil bertipe error yang BENAR-BENAR nil, bukan pointer nil terbungkus interface —
	// kalau tidak, `if err != nil` di pemanggil akan selalu benar.
	require.Nil(t, masterpicteknik.NewValidationError(nil))
	require.NoError(t, masterpicteknik.NewValidationError([]masterpicteknik.Violation{}))
	require.Error(t, masterpicteknik.NewValidationError([]masterpicteknik.Violation{{Field: "x", Message: "y"}}))
}

// Daftar nilainya DISALIN dari Property rule, bukan dikarang dari data yang kebetulan
// terlihat. Uji ini yang menahannya berubah diam-diam.
//
//	Property/TEAM_GROUP_property.xml     A · B · C
//	Property/TYPE_BUSINESS_property.xml  NONMBU · TRAVEL · PA · BONDING
func TestValueListsMatchPropertyRules(t *testing.T) {
	require.Equal(t, []string{"A", "B", "C"}, masterpicteknik.GroupCodes)
	require.Equal(t, []string{"NONMBU", "TRAVEL", "PA", "BONDING"}, masterpicteknik.BusinessCodes)
}

func TestCheckAcceptsEveryValueFromPropertyRules(t *testing.T) {
	for _, group := range masterpicteknik.GroupCodes {
		technician := valid()
		technician.Group = group
		require.Emptyf(t, masterpicteknik.Check(technician), "kelompok %q seharusnya sah", group)
	}
	for _, business := range masterpicteknik.BusinessCodes {
		technician := valid()
		technician.BusinessLine = business
		require.Emptyf(t, masterpicteknik.Check(technician), "bisnis %q seharusnya sah", business)
	}
}

// Kosong diterima: layar Pega menyediakan pilihan `--Pilih--`, dan kolomnya NULLABLE.
func TestCheckAcceptsEmptyGroupAndBusiness(t *testing.T) {
	technician := valid()
	technician.Group = ""
	technician.BusinessLine = ""
	require.Empty(t, masterpicteknik.Check(technician))
}

// Besar-kecil huruf diabaikan supaya baris lama yang tersimpan huruf kecil tetap dapat
// disunting petugasnya sendiri.
func TestCheckAcceptsListValuesRegardlessOfCase(t *testing.T) {
	technician := valid()
	technician.Group = "a"
	technician.BusinessLine = "nonmbu"
	require.Empty(t, masterpicteknik.Check(technician))
}

func TestIDKeyNormalises(t *testing.T) {
	require.Equal(t, "BUDI", masterpicteknik.IDKey("  budi  "))
	require.Equal(t, masterpicteknik.IDKey("BUDI"), masterpicteknik.IDKey("budi"))
}

func TestCleanTrimsEveryTextField(t *testing.T) {
	cleaned := masterpicteknik.Technician{
		OperatorID:   "  A  ",
		Name:         "  Nama  ",
		Email:        "  a@b.co  ",
		BusinessLine: "  NONMBU  ",
		Group:        "  G  ",
		Supervisor:   "  B  ",
		PanelGroup:   "  P  ",
	}.Clean()

	require.Equal(t, "A", cleaned.OperatorID)
	require.Equal(t, "Nama", cleaned.Name)
	require.Equal(t, "a@b.co", cleaned.Email)
	require.Equal(t, "NONMBU", cleaned.BusinessLine)
	require.Equal(t, "G", cleaned.Group)
	require.Equal(t, "B", cleaned.Supervisor)
	require.Equal(t, "P", cleaned.PanelGroup)
}

func TestEmailPlausible(t *testing.T) {
	accepted := []string{"a@b.co", "nama.lengkap@sub.domain.id", "  a@b.co  "}
	for _, address := range accepted {
		require.Truef(t, masterpicteknik.EmailPlausible(address), "seharusnya diterima: %q", address)
	}

	rejected := []string{"", "tanpa-at", "@b.co", "a@", "a@b", "a@@b.co", "a@b."}
	for _, address := range rejected {
		require.Falsef(t, masterpicteknik.EmailPlausible(address), "seharusnya ditolak: %q", address)
	}
}

// Sandi aktif adalah "1", BUKAN "Ya" seperti STS_AKTIF pada tabel master lain. Dua tabel
// memakai sandi berbeda untuk kolom bernama sama, dan uji ini yang menahan salah satunya
// diam-diam berubah mengikuti yang lain.
func TestActiveCodeIsOne(t *testing.T) {
	require.Equal(t, "1", masterpicteknik.ActiveCode)
	require.Equal(t, "0", masterpicteknik.InactiveCode)
}

// Surel kosong DITERIMA.
//
// Section Pega tidak memuat satu pun `pyRequired=true`, dan kolomnya NULLABLE.
// Mewajibkannya — yang sempat dilakukan di sini — membuat setiap baris lama yang surelnya
// kosong tidak dapat disunting sama sekali, termasuk untuk dinonaktifkan.
func TestCheckAcceptsEmptyEmail(t *testing.T) {
	technician := valid()
	technician.Email = ""

	require.Empty(t, masterpicteknik.Check(technician))
}
