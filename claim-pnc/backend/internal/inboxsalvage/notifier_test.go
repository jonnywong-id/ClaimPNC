package inboxsalvage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Subjek surel disalin dari `SetStsSalvagePNC_act:10212` apa adanya:
//
//	"Pengajuan Salvage an " + pyWorkPage.Policy.QQName
func TestSubmissionNoticeSubjectFollowsTheOldRule(t *testing.T) {
	notice := inboxsalvage.SubmissionNotice{InsuredName: "PT Contoh Sejahtera"}
	require.Equal(t, "Pengajuan Salvage an PT Contoh Sejahtera", notice.Subject())
}

// Nama tertanggung yang kosong TIDAK menyisakan ekor menggantung.
//
// Ia dapat kosong: nama itu dibaca ulang dari klaimnya setelah penyimpanan, dan pembacaan
// itu boleh gagal tanpa menggagalkan surelnya. Yang tidak boleh terjadi adalah subjek
// berbunyi "Pengajuan Salvage an " — kalimat yang terbaca seperti surel yang terpotong.
func TestSubmissionNoticeSubjectDropsTheTailWhenTheInsuredIsUnknown(t *testing.T) {
	for name, insured := range map[string]string{
		"kosong":       "",
		"spasi belaka": "   ",
	} {
		t.Run(name, func(t *testing.T) {
			notice := inboxsalvage.SubmissionNotice{InsuredName: insured}
			require.Equal(t, "Pengajuan Salvage", notice.Subject())
		})
	}
}
