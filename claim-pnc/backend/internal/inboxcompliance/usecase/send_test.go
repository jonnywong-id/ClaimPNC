package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
	"claim-pnc/internal/inboxcompliance/usecase"
)

// usecaseCaller adalah petugas contoh. Identitasnya tidak menentukan apa yang boleh
// dikirim — ia hanya tercatat di log, karena tabelnya tidak punya kolom pengirim.
func usecaseCaller() usecase.Caller {
	return usecase.Caller{Login: "ADMINCONTOH1"}
}

const claimKey = "ASM-FW-GCNMFW-WORK PNC-2114"

// queueStore menyusun satu klaim yang menunggu di antrean Compliance, ditambah dua baris
// yang TIDAK boleh dapat dikirim.
func queueStore() *memory.Store {
	return memory.NewStore(
		memory.Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now,
			Item: inboxcompliance.WorkItem{
				CaseID:       "PNC-2114",
				Reference:    claimKey,
				PolicyNumber: "POL-CONTOH-0001",
				InsuredName:  "PT CONTOH SATU ABADI",
			},
		},
		memory.Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now,
			Resolved:   true,
			Item: inboxcompliance.WorkItem{
				CaseID:    "PNC-SELESAI",
				Reference: "ASM-FW-GCNMFW-WORK PNC-SELESAI",
			},
		},
		memory.Row{
			Workbasket: "RCLPUCL",
			CreatedAt:  now,
			Item: inboxcompliance.WorkItem{
				CaseID:    "PNC-LAIN",
				Reference: "ASM-FW-GCNMFW-WORK PNC-LAIN",
			},
		},
	)
}

func TestSendToPostAudit(t *testing.T) {
	t.Parallel()

	store := queueStore()
	service := newService(t, store)

	sent, err := service.SendToPostAudit(
		context.Background(), portal,
		usecaseCaller(),
		inboxcompliance.PostAuditInput{Reference: claimKey, Remarks: "  to compilance  "},
	)
	require.NoError(t, err)

	// Nomornya terbit dari penyimpanan, bukan dari pemanggil, dan mulai dari 100001 —
	// rentang yang tidak mungkin dicapai Pega.
	require.Equal(t, "CPL-100001", sent.Entry.CaseID)

	// Nama tertanggung dan nomor polis DISALIN dari klaimnya, tidak diketik ulang.
	require.Equal(t, "PT CONTOH SATU ABADI", sent.Entry.InsuredName)
	require.Equal(t, "POL-CONTOH-0001", sent.Entry.PolicyNumber)

	// `NO_KLAIM` berisi kunci teknis Pega — namanya menyesatkan, dan itu ditiru apa adanya.
	require.Equal(t, claimKey, sent.Entry.ClaimNumber)

	// Catatan dipangkas spasinya; waktunya dari seam Clock, bukan dari layar.
	require.Equal(t, "to compilance", sent.Entry.Remarks)
	require.Equal(t, now, sent.Entry.SentAt)
}

// Baris yang terkirim langsung muncul di tab Post Audit, bukan setelah proses lain.
func TestSendToPostAuditLangsungTampilDiTabnya(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	_, err := service.SendToPostAudit(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.PostAuditInput{Reference: claimKey, Remarks: "diteruskan"},
	)
	require.NoError(t, err)

	listed, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{Tab: inboxcompliance.TabPostAudit},
		inboxcompliance.Pagination{},
	)
	require.NoError(t, err)

	require.Equal(t, 1, listed.Page.Total)
	require.Equal(t, "CPL-100001", listed.Page.Items[0].CaseID)
	require.Equal(t, "diteruskan", listed.Page.Items[0].ComplianceRemarks)
}

// Nomor terbit berurutan, tidak berulang.
func TestSendToPostAuditNomorBerurutan(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	var terbit []string
	for i := 0; i < 3; i++ {
		sent, err := service.SendToPostAudit(
			context.Background(), portal, usecaseCaller(),
			inboxcompliance.PostAuditInput{Reference: claimKey},
		)
		require.NoError(t, err)
		terbit = append(terbit, sent.Entry.CaseID)
	}

	require.Equal(t, []string{"CPL-100001", "CPL-100002", "CPL-100003"}, terbit)
}

// Klaim yang tidak ada di antrean DITOLAK, dan galatnya dapat dibedakan.
func TestSendToPostAuditMenolakKlaimDiLuarAntrean(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	tests := []struct {
		name      string
		reference string
	}{
		{"klaim yang sudah selesai", "ASM-FW-GCNMFW-WORK PNC-SELESAI"},
		{"klaim di antrean lain", "ASM-FW-GCNMFW-WORK PNC-LAIN"},
		{"klaim yang tidak ada", "ASM-FW-GCNMFW-WORK PNC-TIDAK-ADA"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.SendToPostAudit(
				context.Background(), portal, usecaseCaller(),
				inboxcompliance.PostAuditInput{Reference: tc.reference},
			)

			require.ErrorIs(t, err, inboxcompliance.ErrClaimNotInQueue)
		})
	}
}

func TestSendToPostAuditMenolakTanpaKlaim(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	_, err := service.SendToPostAudit(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.PostAuditInput{Remarks: "tanpa klaim"},
	)

	var validation *inboxcompliance.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxcompliance.FieldReference, validation.Violations[0].Field)
}

// Catatan BOLEH kosong — kolomnya nullable dan tidak ada bukti bahwa ia wajib.
func TestSendToPostAuditCatatanBolehKosong(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	sent, err := service.SendToPostAudit(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.PostAuditInput{Reference: claimKey, Remarks: "   "},
	)

	require.NoError(t, err)
	require.Empty(t, sent.Entry.Remarks)
}

// Catatan yang melebihi lebar kolom ditolak di sini, bukan dibiarkan Oracle yang menolak.
func TestSendToPostAuditMenolakCatatanTerlaluPanjang(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	_, err := service.SendToPostAudit(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.PostAuditInput{
			Reference: claimKey,
			Remarks:   strings.Repeat("a", inboxcompliance.RemarksMaxLength+1),
		},
	)

	var validation *inboxcompliance.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxcompliance.FieldRemarks, validation.Violations[0].Field)
}

// Portal yang tidak dikenal DITOLAK, tidak pernah dialihkan ke portal utama (`R-20`).
func TestSendToPostAuditMenolakPortalTidakDikenal(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	_, err := service.SendToPostAudit(
		context.Background(), "entitas-lain", usecaseCaller(),
		inboxcompliance.PostAuditInput{Reference: claimKey},
	)

	require.Error(t, err)
}
