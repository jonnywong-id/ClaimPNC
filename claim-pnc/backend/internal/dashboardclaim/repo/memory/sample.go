package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleOutstanding adalah klaim berjalan contoh.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Group Panel 009 — varian Aneka yang TIDAK termasuk Non-MBU di sistem lama.
func SampleOutstanding() []ClaimRecord {
	return sampledata.Must[[]ClaimRecord](sampleJSON, "SampleOutstanding")
}

// SampleClosed adalah klaim tutup contoh.
func SampleClosed() []ClaimRecord { return sampledata.Must[[]ClaimRecord](sampleJSON, "SampleClosed") }

// SampleSurveys adalah survei contoh untuk kedua tile survei.
func SampleSurveys() []SurveyRecord {
	return sampledata.Must[[]SurveyRecord](sampleJSON, "SampleSurveys")
}
