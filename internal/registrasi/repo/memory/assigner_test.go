package memory_test

import (
	"context"
	"testing"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/repo/memory"
)

// RouterRCLDokter menugaskan ke ClaimData.NamaDokterRCL, yang belum dapat dipilih di aplikasi
// ini; tugasnya diparkir di ServicePNC, bukan diberikan ke PIC Teknik.
func TestRCLDoctorRouterParksInServicePNC(t *testing.T) {
	a := memory.NewAssigner(memory.SampleTeams())
	got, err := a.Assign(context.Background(), registrasi.Stage{Router: registrasi.RouterRCLDoctor}, registrasi.Claim{}, "CALLER")
	if err != nil || got.Operator != registrasi.OperatorUnassigned {
		t.Fatalf("Assign = %+v, %v; mau %s", got, err, registrasi.OperatorUnassigned)
	}
}

// PNCTeknikRouter langkah 2: UserTeknis kosong dan tidak ada petugas → ServicePNC.
func TestTechnicalRouterWithoutTeamFallsBackToServicePNC(t *testing.T) {
	a := memory.NewAssigner(map[string][]string{})
	got, err := a.Assign(context.Background(), registrasi.Stage{Router: registrasi.RouterPNCTechnical}, registrasi.Claim{}, "CALLER")
	if err != nil || got.Operator != registrasi.OperatorUnassigned {
		t.Fatalf("Assign = %+v, %v; mau %s", got, err, registrasi.OperatorUnassigned)
	}
	pic, err := a.Assign(context.Background(), registrasi.Stage{Router: registrasi.RouterPNCTechnical}, registrasi.Claim{TechnicalPIC: "PIC01"}, "CALLER")
	if err != nil || pic.Operator != "PIC01" {
		t.Fatalf("PIC Teknik klaim harus dihormati: %+v, %v", pic, err)
	}
}
