package registrasi

import "unicode/utf8"

// TechnicalPICNoteMax adalah panjang maksimum Catatan ke PIC Teknis: kolom
// POOLDATA.T_CLAIM_PNC.REMARK bertipe VARCHAR2(4000), terukur dari ALL_TAB_COLUMNS 2026-10-08.
const TechnicalPICNoteMax = 4000

// ValidateTechnicalPICNote menolak catatan yang tidak muat di kolom REMARK.
//
// Panjang dihitung dalam KARAKTER, sama dengan yang dihitung layar. Catatan berhuruf non-ASCII
// dapat melampaui 4000 byte meski kurang dari 4000 karakter; Oracle menolaknya saat itu, dan
// itu cukup jarang untuk tidak dicegah di sini.
func ValidateTechnicalPICNote(note string) error {
	if utf8.RuneCountInString(note) <= TechnicalPICNoteMax {
		return nil
	}
	return &ValidationError{Violation: []Violation{{
		Code: ViolationTechnicalPICNoteTooLong, Field: "catatan_pic_teknis",
		Message: "Catatan ke PIC Teknis is limited to 4000 characters.",
	}}}
}
