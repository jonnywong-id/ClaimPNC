package inboxpladla_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
)

func replier() inboxpladla.Caller {
	return inboxpladla.Caller{Login: "REASCONTOH", Name: "Mitra Contoh"}
}

const sampleClaimKey = "ASM-FW-GCNMFW-WORK PNC-2001"

// Balasan yang sah menghasilkan perintah lengkap.
func TestAValidReplyBecomesACommand(t *testing.T) {
	now := time.Date(2026, time.February, 8, 9, 0, 0, 0, time.UTC)

	command, err := inboxpladla.NewReplyCommand(
		sampleClaimKey,
		inboxpladla.ReplyInput{ConversationID: " KOM-01 ", Message: "  Setuju.  "},
		replier(), now,
	)

	require.NoError(t, err)
	require.Equal(t, "KOM-01", command.ConversationID, "spasi di ujung dipangkas")
	require.Equal(t, "Setuju.", command.Message)
	require.Equal(t, "Mitra Contoh", command.ReplierName())
	require.Equal(t, now.UTC(), command.RepliedAt)
}

// Balasan KOSONG ditolak.
func TestAnEmptyReplyIsRefused(t *testing.T) {
	_, err := inboxpladla.NewReplyCommand(
		sampleClaimKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01", Message: "   "},
		replier(), time.Now(),
	)

	var validation *inboxpladla.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxpladla.FieldReply, validation.Violations[0].Field)
}

// SELURUH pelanggaran dikumpulkan, bukan yang pertama saja (`P-5`).
func TestEveryViolationIsReportedAtOnce(t *testing.T) {
	_, err := inboxpladla.NewReplyCommand(
		sampleClaimKey,
		inboxpladla.ReplyInput{ConversationID: "", Message: ""},
		replier(), time.Now(),
	)

	var validation *inboxpladla.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2,
		"nomor percakapan kosong DAN balasan kosong harus disampaikan bersamaan")
}

// Panjang balasan dihitung dalam RUNE, bukan bita.
//
// Menghitung bita akan menolak kalimat yang lebih pendek daripada batasnya hanya karena
// hurufnya bukan ASCII — dan pembaca layar ini menulis dalam bahasa Indonesia maupun
// Inggris.
func TestTheReplyLengthIsCountedInRunesNotBytes(t *testing.T) {
	// 4.000 huruf beraksen = 8.000 bita, tetapi tepat pada batasnya.
	_, err := inboxpladla.NewReplyCommand(
		sampleClaimKey,
		inboxpladla.ReplyInput{
			ConversationID: "KOM-01",
			Message:        strings.Repeat("é", 4000),
		},
		replier(), time.Now(),
	)
	require.NoError(t, err)

	_, err = inboxpladla.NewReplyCommand(
		sampleClaimKey,
		inboxpladla.ReplyInput{
			ConversationID: "KOM-01",
			Message:        strings.Repeat("é", 4001),
		},
		replier(), time.Now(),
	)
	require.Error(t, err)
}

// Nama pembalas yang KOSONG jatuh ke login-nya, bukan ke teks kosong.
//
// `REPLYFROMNAME` hanya keterangan yang digambar, dan kolom kosong membuat petugas internal
// membaca balasan tanpa tahu dari siapa. Login selalu ada dan selalu dapat ditelusuri.
func TestAnEmptyReplierNameFallsBackToTheLogin(t *testing.T) {
	command, err := inboxpladla.NewReplyCommand(
		sampleClaimKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01", Message: "Setuju."},
		inboxpladla.Caller{Login: "REASCONTOH"}, time.Now(),
	)

	require.NoError(t, err)
	require.Equal(t, "REASCONTOH", command.ReplierName())
}

// Pemanggil tanpa login DITOLAK — ia penyaring, bukan sekadar jejak.
func TestAReplyWithoutALoginIsRefused(t *testing.T) {
	_, err := inboxpladla.NewReplyCommand(
		sampleClaimKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01", Message: "Setuju."},
		inboxpladla.Caller{Name: "Tanpa Login"}, time.Now(),
	)

	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)
}

// Jenis pemberitahuan dibaca tanpa memedulikan huruf besar-kecil.
func TestTheAdviceKindIsReadCaseInsensitively(t *testing.T) {
	for _, raw := range []string{"PLA", "pla", " Pla "} {
		kind, known := inboxpladla.ParseAdviceKind(raw)
		require.True(t, known, "%q", raw)
		require.Equal(t, inboxpladla.AdviceKindPLA, kind)
	}

	_, known := inboxpladla.ParseAdviceKind("PREDLA")
	require.False(t, known, "hanya PLA dan DLA yang dikenali")
}

// Grid DLA punya satu kolom LEBIH — nomor akseptasi.
func TestOnlyTheDLAGridHasAnAcceptanceNumberColumn(t *testing.T) {
	pla := inboxpladla.AdviceColumns(inboxpladla.AdviceKindPLA)
	dla := inboxpladla.AdviceColumns(inboxpladla.AdviceKindDLA)

	require.Len(t, pla, 4)
	require.Len(t, dla, 5)

	require.Equal(t, "No PLA", pla[0].Title)
	require.Equal(t, "No DLA", dla[0].Title)
	require.Equal(t, "No Akseptasi", dla[3].Title)
}

// Kedua tombol yang belum dibangun menjawab alasannya MASING-MASING.
//
// Satu kalimat untuk keduanya akan membuat pengguna yang menekan "Download ALL DLA"
// membaca penjelasan tentang PLA, lalu menyimpulkan tombol yang ia tekan bukan tombol yang
// ia kira.
func TestEachUnbuiltButtonAnswersItsOwnReason(t *testing.T) {
	pla := inboxpladla.NewNotAvailable(string(inboxpladla.ActionDownloadAllPLA))
	dla := inboxpladla.NewNotAvailable(string(inboxpladla.ActionDownloadAllDLA))

	require.Contains(t, pla.Reason(), "Download ALL PLA")
	require.Contains(t, dla.Reason(), "Download ALL DLA")
	require.NotEqual(t, pla.Reason(), dla.Reason())

	// Keduanya menyebut apa yang DAPAT dilakukan hari ini, bukan sekadar apa yang belum
	// ada. Pembacanya pihak luar yang tidak dapat kita latih.
	for _, err := range []*inboxpladla.NotAvailableError{pla, dla} {
		require.Contains(t, err.Reason(), "satu per satu")
	}

	// Tindakan yang tidak dikenal tetap menjadi penolakan, bukan galat lain.
	unknown := inboxpladla.NewNotAvailable("entah")
	require.ErrorIs(t, unknown, inboxpladla.ErrWriteNotAvailable)
	require.NotEmpty(t, unknown.Reason())
}
