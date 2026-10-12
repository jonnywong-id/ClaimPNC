package registrasi_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Link SendApproveCommittee: YA membawa login, TIDAK membawa nama; parameter sama dengan Service REST Pega.
func TestBuildSurveyCommitteeLinks(t *testing.T) {
	l := registrasi.BuildSurveyCommitteeLinks("https://contoh.example/", registrasi.SurveyLinkParams{
		CaseID: "PNCN.26.1", ObjectID: "3", SurveyorType: "2", Login: "ADJ1", Name: "PT ADJ", CommitteeID: "KMTN.26.4", Count: 1, Key: "k1",
	})
	accept, err := url.Parse(l.Accept)
	require.NoError(t, err)
	require.Equal(t, registrasi.SurveyAcceptPath, accept.Path)
	q := accept.Query()
	require.Equal(t, "1", q.Get("aproved"))
	require.Equal(t, "ADJ1", q.Get("surveyor"))
	require.Equal(t, "KMTN.26.4", q.Get("komiteid"))
	require.Equal(t, "1", q.Get("komitecount"))
	require.Equal(t, "k1", q.Get("key"))

	confirm, err := url.Parse(l.Confirm)
	require.NoError(t, err)
	require.Equal(t, registrasi.SurveyConfirmPath, confirm.Path)
	require.Equal(t, "0", confirm.Query().Get("aproved"))
	require.Equal(t, "PT ADJ", confirm.Query().Get("surveyor"))
}

// Email anggota ke-2 hanya berisi PILIH ADJUSTER (`tempCvg.ObjectID == '2'`).
func TestRenderSurveyCommitteeMail(t *testing.T) {
	claim := registrasi.Claim{Number: "PNCN.26.1", Policy: registrasi.Policy{InsuredName: "PT CONTOH", Number: "123"},
		InsuredItem: []registrasi.InsuredItem{{ID: "3", Name: "GUDANG", Coverage: []registrasi.Coverage{{Name: "FIRE", TSI: registrasi.Rupiah(1_000_000)}}}}}
	o := registrasi.SurveyObject{ObjectID: "3", Name: "GUDANG", SurveyorType: "2", SurveyorName: "PT ADJ"}
	links := registrasi.SurveyCommitteeLinks{Accept: "https://contoh.example/a", Confirm: "https://contoh.example/c"}

	first, err := registrasi.RenderSurveyCommitteeMail(registrasi.SurveyCommitteeMail{Claim: claim, Object: o, Count: 1, Links: links, At: time.Now()})
	require.NoError(t, err)
	require.Contains(t, first, "Apakah anda menyetujui pengajuan Adjuster ini?")
	require.Contains(t, first, ">YA<")
	require.Contains(t, first, "CLAIM COMMITEE")
	require.Contains(t, first, "1.000.000")

	second, err := registrasi.RenderSurveyCommitteeMail(registrasi.SurveyCommitteeMail{Claim: claim, Object: o, Count: 2, Links: links, At: time.Now()})
	require.NoError(t, err)
	require.Contains(t, second, "Pilih Adjuster untuk case ini ?")
	require.False(t, strings.Contains(second, ">YA<"))
	require.Equal(t, "AKSEPTASI KOMITE SURVEYOR PNCN.26.1", registrasi.SurveyCommitteeSubject("PNCN.26.1"))
}
