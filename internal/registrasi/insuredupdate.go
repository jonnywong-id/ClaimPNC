package registrasi

import (
	"strings"
	"unicode/utf8"
)

// InsuredUpdate adalah isian DATA TERTANGGUNG KLAIM Input Register PA: No KTP dan "Pengkinian
// Data" (No. HP, Email). Disimpan ke POOLDATA.T_CLAIM_PNC — PENGKINIAN_NO_KTP, PENGKINIAN_NO_HP,
// PENGKINIAN_EMAIL (Work Owner 2026-10-09). Panjang kolom terukur dari ALL_TAB_COLUMNS
// 2026-10-09: 100, 100, dan 200.
type InsuredUpdate struct {
	IDCard string
	Phone  string
	Email  string
}

// Trimmed membuang spasi di awal dan akhir ketiga isian.
func (u InsuredUpdate) Trimmed() InsuredUpdate {
	return InsuredUpdate{
		IDCard: strings.TrimSpace(u.IDCard), Phone: strings.TrimSpace(u.Phone), Email: strings.TrimSpace(u.Email),
	}
}

// ReportTypeMax adalah panjang kolom T_CLAIM_PNC.REPORTTYPE (VARCHAR2(10)), terukur dari
// ALL_TAB_COLUMNS 2026-10-09.
const ReportTypeMax = 10

// ValidateReportType menolak Jenis Laporan yang tidak muat di kolom REPORTTYPE.
func ValidateReportType(code string) error {
	if utf8.RuneCountInString(code) <= ReportTypeMax {
		return nil
	}
	return &ValidationError{Violation: []Violation{{
		Code: ViolationReportTypeTooLong, Field: "jenis_laporan", Message: "Jenis Laporan is too long.",
	}}}
}

// ValidateInsuredUpdate menolak isian yang tidak muat di kolomnya; seluruh pelanggaran
// dikumpulkan sekaligus.
func ValidateInsuredUpdate(u InsuredUpdate) error {
	var v []Violation
	for _, f := range []struct {
		field, label, value string
		max                 int
	}{
		{"pengkinian_no_ktp", "No KTP", u.IDCard, 100},
		{"pengkinian_no_hp", "No. HP", u.Phone, 100},
		{"pengkinian_email", "Email", u.Email, 200},
	} {
		if utf8.RuneCountInString(f.value) > f.max {
			v = append(v, Violation{Code: ViolationInsuredUpdateTooLong, Field: f.field, Message: f.label + " is too long."})
		}
	}
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Violation: v}
}
