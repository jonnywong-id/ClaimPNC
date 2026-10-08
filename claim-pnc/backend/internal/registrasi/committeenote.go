package registrasi

import (
	"time"
	"unicode/utf8"
)

// CommitteeNote adalah isian modal "Transfer Claim ke Komite" — `Flow Action/ClaimComitee_OC`
// (section `ClaimComitee_OC`), dibuka tombol "Transfer ke Komite" grid Adjustment
// (`Section/ShowAdjustment`) dan "Transfer ke Analyst" (`Section/TrfKomiteButton`).
//
// Seluruhnya milik JAMINAN (ASM-FW-GCNMFW-Data-ObjectCoverage), dan tinggal di kolom
// POOLDATA.T_CLAIM_OBJECTCOVERAGE — kolom yang sama yang dibaca layar keputusan komite
// (`internal/komite`, `.Komite.ExtentOfLoss`, `.Komite.CircumtansesCouseOfLoss`, ...):
//
//	Kronologi Kejadian          .CircumCauseOfLoss    CURICUMOFLOSS        4000
//	Jumlah Kerugian             .ExtentOfLoss         EXTENTOFLOSS         4000
//	Polis Liability             .LegalLiability       LEGALLIABILITY       4000
//	Remaks / Catatan Analyst    .Remarks              REMARKS              4000
//	Remaks / Investigation      .RemarkInvestigation  REMARKINVESTIGATION  4000
//	Diagnose/History of Illness .Diagnose             DIAGNOSE             4000
//	Kode Diagnose               .CodeDiagnose         CODEDIAGNOSE           20
//	Desc Diagnose               .DescDiagnose         DESCDIAGNOSE          500
//	Penerima Klaim (Travel)     .TempReceiver         TEMPRECEIVER           10
//	Inisial (baca saja)         .Initial              INITIALNAME
//	Tanggal & Waktu (baca saja) .TanggalComitee       TANGGALCOMITEE
//
// Panjang kolom terukur dari ALL_TAB_COLUMNS 2026-10-08.
//
// Satu isian modal TIDAK dibawa: `.Salvage` (teks, Group Panel != 002) tidak punya kolom di
// T_CLAIM_OBJECTCOVERAGE, sehingga tidak ada tempat menyimpannya.
type CommitteeNote struct {
	Circumstances       string
	ExtentOfLoss        string
	LegalLiability      string
	Remarks             string
	RemarkInvestigation string
	Diagnose            string
	DiagnoseCode        string
	DiagnoseDesc        string
	Receiver            string

	// InitialName dan CommitteeDate ditampilkan baca saja di modal. Activity yang mengisinya
	// (PNCSaveButton2) tidak ada di export, sehingga keduanya hanya dibaca, tidak ditulis.
	InitialName   string
	CommitteeDate time.Time
}

// committeeNoteLimit adalah panjang maksimum tiap isian yang dapat diubah, dalam karakter.
var committeeNoteLimit = []struct {
	field, label string
	max          int
	value        func(CommitteeNote) string
}{
	{"kronologi_kejadian", "Kronologi Kejadian", 4000, func(n CommitteeNote) string { return n.Circumstances }},
	{"jumlah_kerugian", "Jumlah Kerugian", 4000, func(n CommitteeNote) string { return n.ExtentOfLoss }},
	{"polis_liability", "Polis Liability", 4000, func(n CommitteeNote) string { return n.LegalLiability }},
	{"remarks", "Remaks", 4000, func(n CommitteeNote) string { return n.Remarks }},
	{"remarks_investigasi", "Remaks / Investigation", 4000, func(n CommitteeNote) string { return n.RemarkInvestigation }},
	{"diagnosa", "Diagnose/History of Illness", 4000, func(n CommitteeNote) string { return n.Diagnose }},
	{"kode_diagnosa", "Kode Diagnose", 20, func(n CommitteeNote) string { return n.DiagnoseCode }},
	{"desc_diagnosa", "Desc Diagnose", 500, func(n CommitteeNote) string { return n.DiagnoseDesc }},
	{"penerima_klaim", "Penerima Klaim", 10, func(n CommitteeNote) string { return n.Receiver }},
}

// ValidateCommitteeNote menolak isian yang tidak muat di kolomnya. Seluruh pelanggaran
// dikumpulkan sekaligus.
func ValidateCommitteeNote(n CommitteeNote) error {
	var v []Violation
	for _, l := range committeeNoteLimit {
		if utf8.RuneCountInString(l.value(n)) > l.max {
			v = append(v, Violation{
				Code: ViolationCommitteeNoteTooLong, Field: l.field,
				Message: l.label + " is too long.",
			})
		}
	}
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Violation: v}
}
