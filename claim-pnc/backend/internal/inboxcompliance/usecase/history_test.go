package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

// Keterangan baris riwayat dikunci HARFIAH pada parameter `statusNote` Pega.
//
// Ia dikunci karena baris riwayat adalah JEJAK AUDIT, dan `D-59` menjadikannya
// satu-satunya kontrol pengimbang. Keterangan yang bergeser sedikit pun membuat jejak itu
// bercerita hal yang tidak persis terjadi — dan tidak ada yang akan menyadarinya, karena
// tidak ada layar yang membandingkannya dengan Pega.
//
// Ketiga kalimat ini SEMPAT dikarang pada versi sebelumnya ("Compliance FRAUD", dan
// seterusnya), ditulis sebelum parameter langkahnya terbaca. Uji ini yang mencegahnya
// terulang.
func TestKeteranganRiwayatHarfiahSepertiPega(t *testing.T) {
	t.Parallel()

	kasus := map[string]string{
		inboxcompliance.ChoiceFraud:     "Send by Compliance to Analyst and CPL Status is FRAUD",
		inboxcompliance.ChoiceValid:     "Send by Compliance to Analyst and CPL Status is Valid",
		inboxcompliance.ChoicePostAudit: "Send by Compliance to Analyst and CPL Status is Post Audit",
	}

	for pilihan, keterangan := range kasus {
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

			rows := store.History(claimKey)
			require.Len(t, rows, 1)
			require.Equal(t, keterangan, rows[0].Note)
			require.Equal(t, claimKey, rows[0].Reference)
			require.NotEmpty(t, rows[0].By, "jejak tanpa pelaku bukan jejak")
		})
	}
}

// Lain-Lain TIDAK menulis baris riwayat sama sekali.
//
// Pega tidak punya langkah riwayat ber-syarat `pilihan == "3"`. Saya sempat melaporkannya
// sebagai cacat; Work Owner memutuskan ditiru apa adanya (2026-10-07).
//
// Uji ini ada supaya keputusan itu terlihat sebagai PILIHAN, bukan kelalaian — orang
// berikutnya yang menemukan Lain-Lain tanpa jejak akan menemukan uji ini lebih dulu.
func TestLainLainTidakMenulisRiwayat(t *testing.T) {
	t.Parallel()

	store := lineStore(inboxcompliance.GroupPanelPersonalAccident)
	service := newService(t, store)

	_, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceOther,
		},
	)
	require.NoError(t, err)
	require.Empty(t, store.History(claimKey))
}

// Klaim BUKAN PA tidak menulis riwayat.
//
// Ketiga langkah Pega ber-syarat `IsPA`. Work Owner menegaskan Compliance memang proses
// khusus Personal Accident, sehingga klaim non-PA tidak melewati jalur ini — bukan jejak
// yang hilang.
func TestKlaimBukanPATidakMenulisRiwayat(t *testing.T) {
	t.Parallel()

	// 006 = Fire/Property. Bukan PA, dan bukan Travel — sehingga tombol Simpan tetap ada
	// dan keputusannya tetap tersimpan; yang tidak ada hanyalah baris riwayatnya.
	store := lineStore("006")
	service := newService(t, store)

	decided, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		},
	)
	require.NoError(t, err, "keputusannya tetap harus tersimpan")
	require.Equal(t, inboxcompliance.ChoiceValid, decided.Decision.Choice)

	require.Empty(t, store.History(claimKey))
}

// Riwayat bersifat APPEND-ONLY: keputusan yang diubah menambah baris, tidak menimpa.
//
// Justru perubahan keputusan itulah yang paling ingin terlihat di jejak audit. Baris yang
// tertimpa akan menghapus bukti bahwa keputusan pernah berbeda.
func TestRiwayatMenambahBarisSaatKeputusanDiubah(t *testing.T) {
	t.Parallel()

	store := lineStore(inboxcompliance.GroupPanelPersonalAccident)
	service := newService(t, store)

	for _, pilihan := range []string{
		inboxcompliance.ChoiceValid,
		inboxcompliance.ChoiceFraud,
	} {
		_, err := service.SubmitDecision(
			context.Background(), portal, usecaseCaller(),
			inboxcompliance.DecisionInput{
				Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: pilihan},
		)
		require.NoError(t, err)
	}

	rows := store.History(claimKey)
	require.Len(t, rows, 2, "baris lama tertimpa; jejak perubahan keputusan hilang")
	require.Contains(t, rows[0].Note, "Valid")
	require.Contains(t, rows[1].Note, "FRAUD")
}

// Waktu baris riwayat SAMA PERSIS dengan waktu keputusannya.
//
// Procedure Pega memakai jam basis data, sehingga keduanya dapat berselisih. Di sini
// keduanya dari seam Clock yang sama (`F-5`), supaya baris riwayat dan baris keputusan
// dapat dipasangkan saat menelusuri.
func TestWaktuRiwayatSamaDenganWaktuKeputusan(t *testing.T) {
	t.Parallel()

	store := lineStore(inboxcompliance.GroupPanelPersonalAccident)
	service := newService(t, store)

	decided, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		},
	)
	require.NoError(t, err)

	rows := store.History(claimKey)
	require.Len(t, rows, 1)
	require.Equal(t, decided.Decision.DecidedAt, rows[0].RecordedAt)
}
