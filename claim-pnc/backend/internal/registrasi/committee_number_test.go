package registrasi

import "testing"

func TestCommitteeCaseIDGrowsWithoutZeroPadding(t *testing.T) {
	for seq, want := range map[int64]string{1: "KMTN.26.1", 2: "KMTN.26.2", 99: "KMTN.26.99"} {
		got, err := FormatCommitteeCaseID(2026, seq)
		if err != nil || got != want {
			t.Fatalf("nomor ke-%d = %q, %v; ingin %q", seq, got, err, want)
		}
	}
}

func TestCommitteeCaseIDBeyondColumnWidthIsRefused(t *testing.T) {
	// KOMITE_ID VARCHAR2(10): KMTN.26.100 sebelas karakter. Ditolak, bukan dipotong.
	if got, err := FormatCommitteeCaseID(2026, 100); err == nil {
		t.Fatalf("nomor ke-100 = %q tanpa galat, ingin ditolak", got)
	}
}
