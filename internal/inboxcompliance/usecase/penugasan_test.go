package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
)

const picTeknik = "PICTEKNIKCONTOH"

// paStore membentuk antrean berisi satu klaim PA yang PIC Tekniknya sudah ditetapkan.
func paStore() *memory.Store {
	return memory.NewStore(memory.Row{
		Workbasket: inboxcompliance.WorkbasketCompliance,
		CreatedAt:  now,
		Item: inboxcompliance.WorkItem{
			CaseID:       "PNC-2114",
			Reference:    claimKey,
			GroupPanel:   inboxcompliance.GroupPanelPersonalAccident,
			TechnicianID: picTeknik,
		},
	})
}

// Menyimpan keputusan memindahkan klaim: tahap Compliance ditutup, Send To Analis dibuka.
//
// Inilah padanan Ticket `SendtoAnalysator`. Tanpa ini, keputusan tersimpan tetapi klaim
// tetap menunggu di Compliance — tepat keadaan yang dilaporkan Work Owner sebelum ini
// dibangun.
func TestKeputusanMemindahkanPenugasan(t *testing.T) {
	t.Parallel()

	store := paStore()
	service := newService(t, store)

	_, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		},
	)
	require.NoError(t, err)

	rows := store.Assignments(claimKey)
	require.Len(t, rows, 2, "perpindahan harus menghasilkan DUA baris, bukan satu")

	require.Equal(t, inboxcompliance.StageCompliance, rows[0].Stage)
	require.Equal(t, inboxcompliance.AssignmentWorkbasket, rows[0].Kind)
	require.Equal(t, inboxcompliance.WorkbasketCompliance, rows[0].Workbasket)
	require.Equal(t, inboxcompliance.AssignmentDone, rows[0].Status)
	require.Empty(t, rows[0].AssignedTo, "baris workbasket tidak boleh punya pemilik")

	require.Equal(t, inboxcompliance.StageSendToAnalyst, rows[1].Stage)
	require.Equal(t, inboxcompliance.AssignmentWorklist, rows[1].Kind)
	require.Equal(t, picTeknik, rows[1].AssignedTo)
	require.Equal(t, inboxcompliance.AssignmentWaiting, rows[1].Status)
	require.Empty(t, rows[1].Workbasket, "baris worklist tidak boleh punya antrean")
}

// Setelah berpindah, klaim HILANG dari antrean Compliance.
//
// Ini uji yang paling berarti di berkas ini: ia memeriksa akibat yang DILIHAT petugas,
// bukan sekadar baris yang tertulis. Tanpa penyaring `NOT EXISTS` pada kuerinya — dan
// tanpa padanannya di fake — kedua baris di atas tetap tertulis sementara klaimnya tetap
// menggantung di layar.
func TestKlaimHilangDariAntreanSetelahDiputuskan(t *testing.T) {
	t.Parallel()

	store := paStore()
	service := newService(t, store)
	ctx := context.Background()

	sebelum, err := service.List(ctx, portal, inboxcompliance.QueryInput{}, inboxcompliance.Pagination{})
	require.NoError(t, err)
	require.Len(t, sebelum.Page.Items, 1)

	_, err = service.SubmitDecision(ctx, portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		})
	require.NoError(t, err)

	sesudah, err := service.List(ctx, portal, inboxcompliance.QueryInput{}, inboxcompliance.Pagination{})
	require.NoError(t, err)
	require.Empty(t, sesudah.Page.Items, "klaim masih menggantung di antrean Compliance")
}

// Form tidak dapat dibuka lagi setelah klaimnya berpindah.
//
// Di Pega pun begitu: assignment Compliance-nya sudah tidak ada, sehingga tidak ada yang
// dapat dibuka. Galatnya ErrClaimNotInQueue — "sudah tidak di antrean", bukan "tidak
// ditemukan" — dan pembedaan itu yang membuat petugas menyegarkan daftarnya alih-alih
// mencari klaim yang sebenarnya ada.
func TestFormTidakDapatDibukaSetelahBerpindah(t *testing.T) {
	t.Parallel()

	store := paStore()
	service := newService(t, store)
	ctx := context.Background()

	_, err := service.SubmitDecision(ctx, portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		})
	require.NoError(t, err)

	_, err = service.OpenChecker(ctx, portal, claimKey)
	require.ErrorIs(t, err, inboxcompliance.ErrClaimNotInQueue)
}

// Klaim TANPA PIC Teknik tidak berpindah — dan keputusannya tetap tersimpan.
//
// Itu pilihan yang disengaja: klaim tanpa tujuan lebih baik tetap terlihat di antrean
// Compliance daripada hilang ke tempat yang tidak ada. Keadaan itu terlihat dan dapat
// diperbaiki; klaim yang lenyap tidak.
func TestTanpaPICTeknikTidakBerpindah(t *testing.T) {
	t.Parallel()

	store := lineStore(inboxcompliance.GroupPanelPersonalAccident) // tanpa TechnicianID
	service := newService(t, store)
	ctx := context.Background()

	decided, err := service.SubmitDecision(ctx, portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		})
	require.NoError(t, err, "keputusannya tetap harus tersimpan")
	require.Equal(t, inboxcompliance.ChoiceValid, decided.Decision.Choice)

	require.Empty(t, store.Assignments(claimKey))

	sesudah, err := service.List(ctx, portal, inboxcompliance.QueryInput{}, inboxcompliance.Pagination{})
	require.NoError(t, err)
	require.Len(t, sesudah.Page.Items, 1, "klaim tanpa tujuan harus tetap terlihat")
}

// lineStoreWithPIC membentuk antrean berisi satu klaim lini tertentu, ber-PIC Teknik.
func lineStoreWithPIC(groupPanel string) *memory.Store {
	return memory.NewStore(memory.Row{
		Workbasket: inboxcompliance.WorkbasketCompliance,
		CreatedAt:  now,
		Item: inboxcompliance.WorkItem{
			CaseID:       "PNC-2114",
			Reference:    claimKey,
			GroupPanel:   groupPanel,
			TechnicianID: picTeknik,
		},
	})
}

// Tujuan perpindahan ditentukan LINI BISNIS, bukan tombol yang ditekan.
//
// `Activity/SetComplianceResult-Act.xml` memanggil SetTicket dua kali:
//
//	langkah 22  SendtoAnalysator  bila `isPA_PNC`   -> Assignment5 "Send To Analis"
//	langkah 23  SendToPICTravel   bila `IsTravel`   -> Assignment8 "Send To PIC Teknik"
//
// Lini selain keduanya TIDAK menyalakan tiket apa pun, sehingga klaimnya tetap menunggu
// di antrean Compliance.
//
// # Kenapa uji ini ada
//
// Versi pertama memindahkan SETIAP klaim ber-PIC ke "Send To Analis" tanpa memeriksa lini
// — dan tidak satu pun uji menangkapnya, karena seluruh uji memakai klaim PA. Baris Travel
// dan "006" di bawah itulah yang membuat cacatnya terlihat.
func TestTujuanPerpindahanMengikutiLiniBisnis(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		groupPanel string
		wantStage  string // kosong berarti TIDAK berpindah
	}{
		{"PA ke Send To Analis", inboxcompliance.GroupPanelPersonalAccident,
			inboxcompliance.StageSendToAnalyst},
		{"Travel ke Send To PIC Teknik", inboxcompliance.GroupPanelTravel,
			inboxcompliance.StageSendToTechnician},
		{"Fire tidak berpindah", "006", ""},
		{"lini kosong tidak berpindah", "", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			store := lineStoreWithPIC(c.groupPanel)
			service := newService(t, store)
			ctx := context.Background()

			_, err := service.SubmitDecision(ctx, portal, usecaseCaller(),
				inboxcompliance.DecisionInput{
					Action:    inboxcompliance.ActionSend,
					Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
				})
			require.NoError(t, err, "keputusan harus tersimpan apa pun lininya")

			rows := store.Assignments(claimKey)

			if c.wantStage == "" {
				require.Empty(t, rows, "lini ini tidak punya tiket, jadi tidak boleh berpindah")

				sesudah, err := service.List(ctx, portal,
					inboxcompliance.QueryInput{}, inboxcompliance.Pagination{})
				require.NoError(t, err)
				require.Len(t, sesudah.Page.Items, 1,
					"klaim yang tidak berpindah harus tetap terlihat di antrean")
				return
			}

			require.Len(t, rows, 2)
			require.Equal(t, inboxcompliance.StageCompliance, rows[0].Stage)
			require.Equal(t, c.wantStage, rows[1].Stage)
			require.Equal(t, picTeknik, rows[1].AssignedTo)
		})
	}
}

// "Simpan Data" MENYIMPAN tanpa memindahkan klaim.
//
// Di Pega, sel 78 menjalankan `save` + `InsertHistoryClaimPNC_compilance` saja — TANPA
// `SetComplianceResult`. Karena hanya activity itu yang memasang Ticket, menekan Simpan
// meninggalkan klaimnya di antrean Compliance.
//
// # Kenapa uji ini yang paling penting di berkas ini
//
// Versi sebelumnya menjalankan seluruh akibat pada setiap simpan, sehingga menekan
// "Simpan Data" memindahkan klaim keluar dari antrean — dan petugas TIDAK DAPAT
// membatalkannya, karena form itu tidak dapat dibuka lagi setelah klaimnya berpindah.
func TestSimpanTidakMemindahkanKlaim(t *testing.T) {
	t.Parallel()

	store := paStore()
	service := newService(t, store)
	ctx := context.Background()

	decided, err := service.SubmitDecision(ctx, portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Reference: claimKey,
			Choice:    inboxcompliance.ChoiceValid,
			Action:    inboxcompliance.ActionSave,
		})
	require.NoError(t, err)
	require.Equal(t, inboxcompliance.ChoiceValid, decided.Decision.Choice,
		"keputusannya tetap harus tersimpan")

	require.Empty(t, store.Assignments(claimKey), "Simpan tidak boleh memindahkan apa pun")
	require.Empty(t, store.History(claimKey),
		"baris riwayat pilihan milik SetComplianceResult, bukan Simpan")

	// Dan klaimnya masih dapat dibuka lagi — justru itu gunanya Simpan.
	sesudah, err := service.List(ctx, portal,
		inboxcompliance.QueryInput{}, inboxcompliance.Pagination{})
	require.NoError(t, err)
	require.Len(t, sesudah.Page.Items, 1, "klaim harus tetap menunggu di antrean Compliance")

	opened, err := service.OpenChecker(ctx, portal, claimKey)
	require.NoError(t, err, "form harus masih dapat dibuka setelah Simpan")
	require.NotNil(t, opened.Case.Decision, "pilihan yang sudah disimpan harus tersemai")
}

// Aksi KOSONG diperlakukan sebagai Simpan, bukan Kirim.
//
// Klien lama yang belum mengirim `aksi` karena itu tidak diam-diam memindahkan klaim —
// bentuk bawaan yang paling tidak berakibat.
func TestAksiKosongDiperlakukanSebagaiSimpan(t *testing.T) {
	t.Parallel()

	store := paStore()
	service := newService(t, store)

	_, err := service.SubmitDecision(context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Reference: claimKey, Choice: inboxcompliance.ChoiceValid,
		})
	require.NoError(t, err)
	require.Empty(t, store.Assignments(claimKey))
}
