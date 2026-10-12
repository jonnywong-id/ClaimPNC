package inboxrcl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	now      = time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC)
	loss     = time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	release  = time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	printedA = time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC)
)

func TestTombolHanyaBerlakuPadaModeLayarnya(t *testing.T) {
	for _, d := range []Decision{DecisionApprove, DecisionDisagree} {
		_, err := Plan(PUCLState{Mode: ModeMSIG}, d, "", now)
		require.ErrorIsf(t, err, ErrDecisionNotAllowed, "%s pada mode MSIG", d)
	}
	for _, d := range []Decision{DecisionSubmit, DecisionBackMSIG} {
		_, err := Plan(PUCLState{Mode: ModeRCL}, d, "", now)
		require.ErrorIsf(t, err, ErrDecisionNotAllowed, "%s pada mode RCL", d)
	}
	_, err := Plan(PUCLState{Mode: ModePUCL}, DecisionApprove, "", now)
	require.ErrorIs(t, err, ErrDecisionNotAllowed)
}

func TestParseDecisionHanyaKeempatNilaiTombol(t *testing.T) {
	for _, raw := range []string{"SETUJU", "MSIG", "TidakSetuju", "BackMSIG"} {
		_, ok := ParseDecision(raw)
		require.Truef(t, ok, raw)
	}
	for _, raw := range []string{"", "Setuju", "setuju", "tolak"} {
		_, ok := ParseDecision(raw)
		require.Falsef(t, ok, raw)
	}
}

// Langkah 8, 10, 12, 18 — Setuju pada mode RCL.
func TestSetujuMenjadikanKlaimAktifRCLDanMasukAntreanRCLPUCL(t *testing.T) {
	out, err := Plan(PUCLState{
		Mode: ModeRCL, StatusCase: "1", PUCLApprove: "1", LetterPrintedAt: printedA,
		DateOfLoss: loss, DoctorReason: "lama",
	}, DecisionApprove, "diabaikan", now)
	require.NoError(t, err)

	require.Equal(t, StatusClaimRCL, out.StatusClaim, "langkah 10 menimpa 1155 langkah 8")
	require.Equal(t, StatusKlaimActive, out.StatusKlaim)
	require.Equal(t, StatusCaseOpen, out.StatusCase)
	require.Equal(t, "", out.PUCLApprove, "langkah 10 mengosongkan PUCLApprove")
	require.True(t, out.LetterPrintedAt.IsZero(), "tanggal cetak dikosongkan — surat dicetak petugas RCL/PUCL")
	require.Equal(t, now, out.SentToPUCLAt)
	require.Equal(t, now, out.AnalystSentAt)
	require.Equal(t, now, out.ClaimAgeFrom, "langkah 10 menimpa LamaKlaim langkah 2/3")
	require.Equal(t, "lama", out.DoctorReason, "Alasan Dokter tidak disentuh tombol Setuju")
	require.Equal(t, []string{HistorySentToDoctor}, out.History)
	require.Equal(t, TicketSendToPUCL, out.Ticket)
	require.Equal(t, StageRCLPUCL, out.NextStage)
	require.Equal(t, QueueWorkbasket, out.NextQueue)
}

// Langkah 8, 12, 18, 19 — Submit pada mode MSIG. Langkah 10 tidak berlaku.
func TestSubmitMSIGMenjadiPosisiRCLDokterDanMasukAntreanRCLPUCL(t *testing.T) {
	out, err := Plan(PUCLState{Mode: ModeMSIG, StatusKlaim: "1151", DischargedAt: release, DateOfLoss: loss},
		DecisionSubmit, "", now)
	require.NoError(t, err)

	require.Equal(t, StatusClaimRCLDoctor, out.StatusClaim)
	require.Equal(t, "1151", out.StatusKlaim, "mode MSIG tidak menyentuh StatusKlaim")
	require.Equal(t, StatusCaseOpen, out.StatusCase, "langkah 5 — kosong menjadi 0")
	require.Equal(t, release, out.ClaimAgeFrom, "langkah 2 — tanggal selesai rawat inap")
	require.Equal(t, []string{HistorySentToDoctor, HistorySentToMSIG}, out.History)
	require.Equal(t, StageRCLPUCL, out.NextStage)
}

// Langkah 1 (RCLSendKomiteReject_act), 16, 18, 19, 21 — Tidak Setuju pada mode RCL.
func TestTidakSetujuKembaliKeAnalystDenganAlasanDokter(t *testing.T) {
	out, err := Plan(PUCLState{Mode: ModeRCL, StatusCase: "0", DateOfLoss: loss, LetterPrintedAt: printedA},
		DecisionDisagree, "  Diagnosa dijamin.  ", now)
	require.NoError(t, err)

	require.Equal(t, StatusClaimAnalyst, out.StatusClaim)
	require.Equal(t, StatusKlaimActive, out.StatusKlaim, "RCLSendKomiteReject_act langkah 1")
	require.Equal(t, "", out.StatusCase, "langkah 16")
	require.True(t, out.AnalystSentAt.IsZero(), "langkah 16 mengosongkan TanggalAnalystSendRCL")

	// Baris ini DIBALIK pada 2026-10-06, dan bentuk lamanya patut dicatat: ia berbunyi
	// `require.Equal(t, now, out.SentToPUCLAt, "langkah 8 tetap berlaku")` — membaca
	// langkah 8 sendiri-sendiri, tanpa menanyakan kolom mana yang dibaca penyaring C.
	// Dengan begitu ia MENGUNCI cacat yang dilaporkan: klaim tetap di Inbox RCL sesudah
	// Tidak Setuju. Uji yang hijau tidak berarti perilakunya benar.
	require.True(t, out.SentToPUCLAt.IsZero(),
		"TGL_KIRIM_PUCL adalah penyaring C Inbox RCL — langkah 16 harus mengosongkannya")
	require.Equal(t, loss, out.ClaimAgeFrom, "langkah 3 — tanggal kejadian")
	require.Equal(t, printedA, out.LetterPrintedAt, "tanggal cetak dipertahankan")
	require.Equal(t, "Diagnosa dijamin.", out.DoctorReason)
	require.Equal(t, []string{HistorySentToDoctor, HistorySentToMSIG}, out.History)
	require.Equal(t, TicketSendToAnalyst, out.Ticket, "langkah 21 menimpa RCLDokter langkah 20")
	require.Equal(t, StageSendToAnalyst, out.NextStage)
	require.Equal(t, QueueWorklist, out.NextQueue)
}

// GEJALA YANG DILAPORKAN: sesudah Tidak Setuju, klaim harus KELUAR dari Inbox RCL —
// termasuk ketika pemilik berikutnya adalah orang yang sama.
//
// # Kenapa "orang yang sama" bukan kasus mengada-ada
//
// Penyaring Inbox RCL adalah `A AND B AND C AND D`. Sesudah Tidak Setuju, B dan D tidak
// berubah sama sekali, sehingga yang dapat mengeluarkan klaim hanya A (pemilik berpindah)
// atau C (tanggal dikosongkan). Bila analis dan PIC Teknik adalah akun yang sama — lumrah
// di lingkungan uji, dan mungkin di cabang kecil — A TETAP cocok, dan C menjadi
// satu-satunya yang tersisa.
//
// Uji ini karena itu tidak memeriksa pemiliknya sama sekali. Ia memeriksa bahwa keputusan
// itu sendiri sudah cukup untuk mengeluarkan klaim dari antrean dokter.
func TestTidakSetujuMengeluarkanKlaimDariInboxRCLApaPunPemilikBerikutnya(t *testing.T) {
	for _, keputusan := range []Decision{DecisionDisagree, DecisionBackMSIG} {
		mode := ModeRCL
		if keputusan == DecisionBackMSIG {
			mode = ModeMSIG
		}

		out, err := Plan(PUCLState{Mode: mode, StatusCase: "0", DateOfLoss: loss}, keputusan, "", now)
		require.NoError(t, err)

		// Penyaring C `InboxRCLDokter_RD`: `TGL_KIRIM_PUCL IS NOT NULL`. Kosong berarti
		// barisnya gugur, siapa pun pemiliknya.
		require.Truef(t, out.SentToPUCLAt.IsZero(),
			"%s: penyaring C harus gugur — tanpa ini klaim tetap di Inbox RCL", keputusan)

		// Dan ia memang kembali ke analis, bukan sekadar menghilang.
		require.Equal(t, StageSendToAnalyst, out.NextStage)
		require.Equal(t, TicketSendToAnalyst, out.Ticket)
		require.Equal(t, StatusClaimAnalyst, out.StatusClaim)
	}
}

// Langkah 16, 18, 21 — Back pada mode MSIG. Riwayat MSIG tidak ditulis.
func TestBackMSIGKembaliKeAnalystTanpaRiwayatMSIG(t *testing.T) {
	out, err := Plan(PUCLState{Mode: ModeMSIG, StatusKlaim: "AKTIF"}, DecisionBackMSIG, "", now)
	require.NoError(t, err)

	require.Equal(t, StatusClaimAnalyst, out.StatusClaim)
	require.Equal(t, "AKTIF", out.StatusKlaim)
	require.Equal(t, "", out.DoctorReason, "Alasan Dokter tidak wajib diisi")
	require.True(t, out.ClaimAgeFrom.IsZero(), "tanpa tanggal kejadian LamaKlaim kosong")
	require.Equal(t, []string{HistorySentToDoctor}, out.History)
	require.Equal(t, StageSendToAnalyst, out.NextStage)
}

// Langkah 6 dan 7 — kirim ulang setelah pernah disetujui PUCL.
func TestKirimUlangSetelahDisetujuiPUCL(t *testing.T) {
	// Langkah 6: StatusCase "1" — hanya PUCLApprove yang berubah.
	out, err := Plan(PUCLState{Mode: ModeMSIG, StatusCase: "1", PUCLApprove: "1", LetterPrintedAt: printedA},
		DecisionSubmit, "", now)
	require.NoError(t, err)
	require.Equal(t, PUCLApproveNo, out.PUCLApprove)
	require.Equal(t, "1", out.StatusCase)
	require.Equal(t, printedA, out.LetterPrintedAt)

	// Langkah 7: StatusCase bukan "1" — PUCLApprove, StatusCase, dan tanggal cetak berubah.
	out, err = Plan(PUCLState{Mode: ModeMSIG, StatusCase: "0", PUCLApprove: "1", LetterPrintedAt: printedA},
		DecisionSubmit, "", now)
	require.NoError(t, err)
	require.Equal(t, PUCLApproveNo, out.PUCLApprove)
	require.Equal(t, StatusCaseOpen, out.StatusCase)
	require.True(t, out.LetterPrintedAt.IsZero())
}

func TestKunciRiwayat(t *testing.T) {
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-2067", HistoryKey(" PNC-2067 "))
	require.Equal(t, "PNCN.26.31", HistoryKey("PNCN.26.31"))
}
