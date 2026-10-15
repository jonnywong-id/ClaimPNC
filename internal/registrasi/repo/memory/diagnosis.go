package memory

import (
	"context"
	"strings"

	"claim-pnc/internal/registrasi"
)

// DiagnosisDirectory adalah master diagnosa contoh (kode ICD umum).
type DiagnosisDirectory struct{}

var sampleDiagnosis = []registrasi.DiagnosisOption{
	{Code: "A00", Description: "CHOLERA"},
	{Code: "A01.0", Description: "TYPHOID FEVER"},
	{Code: "S52.5", Description: "FRACTURE OF LOWER END OF RADIUS"},
	{Code: "S92.0", Description: "FRACTURE OF CALCANEUS"},
	{Code: "T14.0", Description: "SUPERFICIAL INJURY OF UNSPECIFIED BODY REGION"},
}

// SearchDiagnosis mencari menurut kode persis atau sebagian deskripsi.
func (DiagnosisDirectory) SearchDiagnosis(_ context.Context, text string) ([]registrasi.DiagnosisOption, error) {
	text = strings.ToUpper(strings.TrimSpace(text))
	if text == "" {
		return nil, nil
	}
	var out []registrasi.DiagnosisOption
	for _, d := range sampleDiagnosis {
		if strings.ToUpper(d.Code) == text || strings.Contains(d.Description, text) {
			out = append(out, d)
		}
	}
	return out, nil
}

var _ registrasi.DiagnosisDirectory = DiagnosisDirectory{}
