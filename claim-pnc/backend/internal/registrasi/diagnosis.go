package registrasi

import "context"

// DiagnosisOption adalah satu kode diagnosa (ICD) — baris `sm.m_diagnosis@asmd`, yang dipilih
// tombol Pilih pada modal "Transfer Claim ke Komite" (PA) menjadi `.CodeDiagnose` dan
// `.DescDiagnose`.
type DiagnosisOption struct {
	Code        string // DIAGNOSIS_KODE — `.CityID` pada grid Pega
	Description string // DIAGNOSIS_DESC — `.City` pada grid Pega
}

// DiagnosisDirectory mencari kode diagnosa — `CariKodeDiagnosKlaimPA` →
// `RDB List/GetKodeDiagnosaKlaimPa`.
type DiagnosisDirectory interface {
	// SearchDiagnosis mengembalikan diagnosa yang kodenya sama persis dengan teks (tanpa
	// membedakan huruf besar-kecil), atau deskripsinya memuat teks itu.
	SearchDiagnosis(ctx context.Context, text string) ([]DiagnosisOption, error)
}
