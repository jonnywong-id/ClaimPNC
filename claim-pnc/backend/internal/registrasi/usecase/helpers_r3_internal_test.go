package usecase

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

func TestCashierRejectedErrorMessage(t *testing.T) {
	require.Equal(t, "kasir menolak: Rekening diblokir", (&CashierRejectedError{Message: "Rekening diblokir"}).Error())
}

func TestRupiahText(t *testing.T) {
	require.Equal(t, "0", rupiahText(0))
	require.Equal(t, "999", rupiahText(99_900))
	require.Equal(t, "2.500.000", rupiahText(250_000_000))
	require.Equal(t, "1.000", rupiahText(100_050))
}

// Tiga indeks berbasis 1 di luar jangkauan ditolak sebagai tindakan tidak sah.
func TestSettlementAtBounds(t *testing.T) {
	claim := registrasi.Claim{InsuredItem: []registrasi.InsuredItem{{
		Coverage: []registrasi.Coverage{{Settlement: []registrasi.SettlementLine{{AcceptedNo: "A1"}}}},
	}}}
	line, err := settlementAt(&claim, 1, 1, 1)
	require.NoError(t, err)
	require.Equal(t, "A1", line.AcceptedNo)
	for _, idx := range [][3]int{{0, 1, 1}, {2, 1, 1}, {1, 0, 1}, {1, 2, 1}, {1, 1, 0}, {1, 1, 2}} {
		_, err := settlementAt(&claim, idx[0], idx[1], idx[2])
		require.ErrorIs(t, err, registrasi.ErrInvalidAction, "%v", idx)
	}
}

func transferViolationCode(t *testing.T, err error) registrasi.ViolationCode {
	t.Helper()
	var v *registrasi.ValidationError
	require.True(t, errors.As(err, &v), "galat = %v", err)
	return v.Violation[0].Code
}

// checkTransfer: sudah ditransfer, tipe kosong, fee tanpa gross, dan propose kosong ditolak.
func TestCheckTransfer(t *testing.T) {
	require.NoError(t, checkTransfer(registrasi.SettlementLine{PaymentType: registrasi.PaymentFinal, Propose: 1}))
	require.NoError(t, checkTransfer(registrasi.SettlementLine{PaymentType: registrasi.PaymentAdjusterFee, Gross: 1}))
	require.NoError(t, checkTransfer(registrasi.SettlementLine{PaymentType: registrasi.PaymentReject}))
	require.NoError(t, checkTransfer(registrasi.SettlementLine{PaymentType: registrasi.PaymentSalvage}))

	require.Equal(t, registrasi.ViolationCommitteeTransferred,
		transferViolationCode(t, checkTransfer(registrasi.SettlementLine{PaymentType: registrasi.PaymentFinal, CommitteeCaseID: "KM.26.1"})))
	require.Equal(t, registrasi.ViolationCommitteeIncomplete, transferViolationCode(t, checkTransfer(registrasi.SettlementLine{})))
	require.Equal(t, registrasi.ViolationCommitteeIncomplete,
		transferViolationCode(t, checkTransfer(registrasi.SettlementLine{PaymentType: registrasi.PaymentAdjusterFee})))
	require.Equal(t, registrasi.ViolationCommitteeIncomplete,
		transferViolationCode(t, checkTransfer(registrasi.SettlementLine{PaymentType: registrasi.PaymentInterim})))
}

func TestDLASignatureID(t *testing.T) {
	require.Empty(t, dlaSignatureID("KOMITE LAIN"))
	require.Equal(t, "BAMBANGSG", dlaSignatureID("Bambang S"))
	require.Equal(t, "DHARMANTO", dlaSignatureID("dharmanto"))
	require.Equal(t, "ELLENSP", dlaSignatureID("Ellen"))
	// Linda cocok pada dua aturan; yang terakhir menang.
	require.Equal(t, "LINDANOVA", dlaSignatureID("Linda"))
	require.Equal(t, "INDRAGN", dlaSignatureID("Ellen dan Indra"))
}

func TestPLAEntity(t *testing.T) {
	require.Equal(t, "ASI", plaEntity("ASI"))
	require.Equal(t, "ASM", plaEntity("SMI"))
}

// Satu dokumen dikirim apa adanya; lebih dari satu dibungkus ZIP berisi semuanya.
func TestPackDLAAndPLA(t *testing.T) {
	one := []plaFile{{name: "A.pdf", content: []byte("%PDF-a")}}
	r, err := packDLA(one)
	require.NoError(t, err)
	require.Equal(t, DLAResult{FileName: "A.pdf", ContentType: "application/pdf", Content: []byte("%PDF-a")}, r)

	two := append(one, plaFile{name: "B.pdf", content: []byte("%PDF-b")})
	r, err = packDLA(two)
	require.NoError(t, err)
	require.Equal(t, "DLA.zip", r.FileName)
	require.Equal(t, "application/zip", r.ContentType)
	requireZip(t, r.Content, map[string]string{"A.pdf": "%PDF-a", "B.pdf": "%PDF-b"})

	p, err := packPLA(two, 2)
	require.NoError(t, err)
	require.Equal(t, "PLA.zip", p.FileName)
	require.Equal(t, 2, p.Issued)
	requireZip(t, p.Content, map[string]string{"A.pdf": "%PDF-a", "B.pdf": "%PDF-b"})
	p, err = packPLA(one, 0)
	require.NoError(t, err)
	require.Equal(t, "application/pdf", p.ContentType)
}

func requireZip(t *testing.T, content []byte, want map[string]string) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	got := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		_ = rc.Close()
		got[f.Name] = string(b)
	}
	require.Equal(t, want, got)
}

func TestDLAWarnings(t *testing.T) {
	require.Nil(t, dlaWarnings([]registrasi.DLA{{Number: "D1", RecipientCode: "77"}}))
	require.Equal(t, []string{"- Error Reas DLA  Kosong", "- Error Reas DLA D2 Kosong"},
		dlaWarnings([]registrasi.DLA{{RecipientCode: "1"}, {Number: "D2"}}))
}

// Basis awal Fac Out: TSI jaminan untuk Group Panel 006 atau CaseID ASM, selain itu SumOfTSI.
func TestDLAInitialBase(t *testing.T) {
	sc := dlaScope{
		claim:    registrasi.Claim{Policy: registrasi.Policy{Line: registrasi.LineMiscellaneous}},
		coverage: registrasi.Coverage{TSI: 1_000_00},
	}
	require.Equal(t, big.NewRat(1000, 1), dlaInitialBase(sc, registrasi.DLAPolicy{}))
	sum := big.NewRat(5000, 1)
	require.Equal(t, sum, dlaInitialBase(sc, registrasi.DLAPolicy{SumOfTSI: sum}))
	require.Equal(t, big.NewRat(1000, 1), dlaInitialBase(sc, registrasi.DLAPolicy{CaseID: "ASM-1", SumOfTSI: sum}))
	sc.claim.Policy.Line = "006"
	require.Equal(t, big.NewRat(1000, 1), dlaInitialBase(sc, registrasi.DLAPolicy{SumOfTSI: sum}))
}

// totalConvert: baris terpilih ditambah setiap baris terakseptasi; salvage mengurangi.
// Baris terpilih yang terakseptasi terhitung dua kali — perilaku Pega yang dibawa (`P-5`).
func TestDLATotalConvert(t *testing.T) {
	one := registrasi.ExchangeRateOne
	selected := registrasi.SettlementLine{Value: 1_000_00, Rate: one, AcceptanceStatus: "1", AcceptedNo: "A1"}
	sc := dlaScope{
		line: selected,
		coverage: registrasi.Coverage{Settlement: []registrasi.SettlementLine{
			selected,
			{Value: 200_00, Rate: one, AcceptanceStatus: "1", PaymentType: registrasi.PaymentSalvage, AcceptedNo: "A2"},
			{Value: 50_00, Rate: one, AcceptanceStatus: "1", AcceptanceLODStatus: "0"},
			{Value: 70_00, Rate: one, AcceptanceStatus: "0"},
			{Value: 30_00, Rate: one, AcceptanceStatus: " 1 ", AcceptedNo: "A3"},
		}},
	}
	require.Equal(t, big.NewRat(1000+1000-200+30, 1), dlaTotalConvert(sc))
}

// Flow dan CanWork adalah pintu baca bagi transport.
func TestServiceFlowAndCanWork(t *testing.T) {
	s := &Service{flow: registrasi.RegisterFlow()}
	require.Equal(t, registrasi.StageViewPolicy, s.Flow().Start)
	task := registrasi.Task{Stage: registrasi.StageViewPolicy, Owner: "NIK1"}
	require.True(t, s.CanWork(task, Caller{Identity: "NIK1"}))
	require.False(t, s.CanWork(task, Caller{Identity: "LAIN"}))
}
