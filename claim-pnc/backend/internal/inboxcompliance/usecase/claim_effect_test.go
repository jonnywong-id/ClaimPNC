package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

// Setiap keputusan memindahkan Status Klaim ke 1151 — TERMASUK Fraud/Tolak.
//
// Langkah 1 `SetComplianceResult` tidak punya syarat sama sekali, sehingga keempat pilihan
// menulisnya. `1151` berarti Analyst menurut master `v_sts_claim`, dan itu sejalan dengan
// dua langkah terakhir activity-nya yang mengembalikan klaim ke Analyst.
func TestKeputusanSelaluMemindahkanStatusKlaim(t *testing.T) {
	t.Parallel()

	for _, pilihan := range []string{
		inboxcompliance.ChoiceFraud,
		inboxcompliance.ChoiceValid,
		inboxcompliance.ChoicePostAudit,
		inboxcompliance.ChoiceOther,
	} {
		t.Run("pilihan "+pilihan, func(t *testing.T) {
			t.Parallel()

			store := lineStore(inboxcompliance.GroupPanelPersonalAccident)
			service := newService(t, store)

			_, err := service.SubmitDecision(
				context.Background(), portal, usecaseCaller(),
				inboxcompliance.DecisionInput{
					Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: pilihan},
			)
			require.NoError(t, err)

			effect, ada := store.ClaimEffect(claimKey)
			require.True(t, ada, "klaim tidak tersentuh sama sekali")
			require.Equal(t, inboxcompliance.StatusClaimAfterCompliance, effect.StatusClaim)
		})
	}
}

// Kedua tanggal terisi HANYA pada pilihannya masing-masing.
//
// Ini yang membedakan ketiga kolom itu: Status Klaim selalu, tanggalnya tidak.
func TestTanggalKlaimTerisiHanyaPadaPilihannya(t *testing.T) {
	t.Parallel()

	kasus := []struct {
		pilihan  string
		adaValid bool
		adaKirim bool
	}{
		{inboxcompliance.ChoiceValid, true, false},
		{inboxcompliance.ChoicePostAudit, false, true},
		{inboxcompliance.ChoiceFraud, false, false},
		{inboxcompliance.ChoiceOther, false, false},
	}

	for _, c := range kasus {
		t.Run("pilihan "+c.pilihan, func(t *testing.T) {
			t.Parallel()

			store := lineStore(inboxcompliance.GroupPanelPersonalAccident)
			service := newService(t, store)

			_, err := service.SubmitDecision(
				context.Background(), portal, usecaseCaller(),
				inboxcompliance.DecisionInput{
					Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: c.pilihan},
			)
			require.NoError(t, err)

			effect, _ := store.ClaimEffect(claimKey)
			require.Equal(t, c.adaValid, effect.ValidatedAt != nil, "CPLVALID_DATE")
			require.Equal(t, c.adaKirim, effect.SentToPostAuditAt != nil, "POSTAUDIT_TF_ANALYSTDATE")
		})
	}
}

// Mengubah keputusan TIDAK menghapus tanggal yang sudah terisi.
//
// Di Pega, langkah yang tidak berjalan meninggalkan nilai lamanya — bukan mengosongkannya.
// Kuerinya meniru itu dengan `COALESCE(:n, kolom)`, dan uji ini yang menjaganya: tanpa
// COALESCE, mengubah Valid menjadi Lain-Lain akan MENGHAPUS tanggal validnya, dan
// kehilangan itu tidak terlihat sampai seseorang mencarinya berbulan-bulan kemudian.
func TestMengubahKeputusanTidakMenghapusTanggalLama(t *testing.T) {
	t.Parallel()

	store := lineStore(inboxcompliance.GroupPanelPersonalAccident)
	service := newService(t, store)

	_, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		},
	)
	require.NoError(t, err)

	sebelum, _ := store.ClaimEffect(claimKey)
	require.NotNil(t, sebelum.ValidatedAt)

	// Diubah menjadi Lain-Lain, yang tidak menyentuh kedua tanggal.
	_, err = service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceOther,
		},
	)
	require.NoError(t, err)

	sesudah, _ := store.ClaimEffect(claimKey)
	require.NotNil(t, sesudah.ValidatedAt, "tanggal valid terhapus saat keputusan diubah")
	require.Equal(t, *sebelum.ValidatedAt, *sesudah.ValidatedAt)
}
