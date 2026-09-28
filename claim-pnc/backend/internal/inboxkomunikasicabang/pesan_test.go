package inboxkomunikasicabang_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// sender adalah pengirim contoh yang profilnya lengkap.
func sender() inboxkomunikasicabang.Caller {
	return inboxkomunikasicabang.Caller{Login: "pictekniks", Name: "PIC Teknik Surabaya"}
}

// atNoon adalah jam tetap, supaya uji tidak bergantung pada kapan ia dijalankan.
func atNoon() time.Time {
	return time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC)
}

func TestOnlyTwoDestinationsExistAndTheyComeFromTheDataTransform(t *testing.T) {
	// `CNMShowInsertKomunikasi_dt` mengisi daftarnya secara harfiah dengan dua baris:
	// "PUSAT" lalu "CABANG". Bukan dari master mana pun, dan bukan daftar yang dapat
	// bertambah — ia dua arah percakapan yang mungkin.
	require.Equal(t,
		[]inboxkomunikasicabang.Destination{
			inboxkomunikasicabang.DestinationHeadOffice,
			inboxkomunikasicabang.DestinationBranch,
		},
		inboxkomunikasicabang.Destinations())
}

func TestAMessageToTheBranchNeedsABranchChosen(t *testing.T) {
	// Pesan tanpa cabang tujuan akan tersimpan dengan COMMUNICATE_TO kosong — tidak muncul
	// di kotak masuk siapa pun, dan tidak dapat diperbaiki lewat layar mana pun.
	//
	// Kalimat galatnya dibawa APA ADANYA dari `Local.msgErr` pada activity lama: pengguna
	// layar ini sudah mengenalnya (`D-13`).
	_, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{Destination: "CABANG", Message: "halo"},
		sender(), atNoon(),
	)

	var validation *inboxkomunikasicabang.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxkomunikasicabang.FieldBranch, validation.Violations[0].Field)
	require.Equal(t, "Silakan pilih cabang terlebih dahulu",
		validation.Violations[0].Message)
}

func TestAMessageToHeadOfficeNeedsNoBranch(t *testing.T) {
	command, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{Destination: "PUSAT", Message: "halo"},
		sender(), atNoon(),
	)

	require.NoError(t, err)
	require.Equal(t, inboxkomunikasicabang.HeadOfficeCode, command.RecipientCode())
}

func TestABranchLeftOverFromASwitchedDestinationIsIgnoredNotRefused(t *testing.T) {
	// Layar mengosongkan pemilih cabang saat tujuannya berpindah ke PUSAT. Menolak
	// permintaannya akan menghukum pengguna atas isian yang sudah tidak terlihat olehnya.
	command, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{
			Destination: "PUSAT", BranchCode: "1002", Message: "halo",
		},
		sender(), atNoon(),
	)

	require.NoError(t, err)
	require.Equal(t, "", command.BranchCode)
	require.Equal(t, inboxkomunikasicabang.HeadOfficeCode, command.RecipientCode())
}

func TestTheRecipientOfABranchMessageIsTheChosenBranch(t *testing.T) {
	command, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{
			Destination: "CABANG", BranchCode: "1002", Message: "halo",
		},
		sender(), atNoon(),
	)

	require.NoError(t, err)
	require.Equal(t, "1002", command.RecipientCode())
}

func TestAnUnknownDestinationIsRefused(t *testing.T) {
	// Dropdown-nya hanya punya dua pilihan. Nilai ketiga hanya dapat datang dari layar yang
	// rusak atau dari permintaan yang disusun tangan — dan keduanya tidak boleh menghasilkan
	// percakapan yang tujuannya tidak dapat dibaca.
	_, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{Destination: "PUSAT_CABANG", Message: "halo"},
		sender(), atNoon(),
	)

	var validation *inboxkomunikasicabang.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxkomunikasicabang.FieldDestination, validation.Violations[0].Field)
}

func TestAnEmptyMessageIsRefused(t *testing.T) {
	_, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{Destination: "PUSAT", Message: "   "},
		sender(), atNoon(),
	)

	var validation *inboxkomunikasicabang.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxkomunikasicabang.FieldMessageBody, validation.Violations[0].Field)
}

func TestASenderWithoutANameIsRefused(t *testing.T) {
	// Nama pengirim tersimpan sebagai SENDERNAME. Pesan tanpa nama muncul di kotak masuk
	// penerimanya sebagai baris yang pengirimnya tidak dapat dikenali.
	_, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{Destination: "PUSAT", Message: "halo"},
		inboxkomunikasicabang.Caller{Login: "pictekniks"}, atNoon(),
	)

	require.ErrorIs(t, err, inboxkomunikasicabang.ErrCallerUnknown)
}

func TestMessageLengthIsMeasuredInCharactersNotBytes(t *testing.T) {
	// Sama dengan pada balasan, dan batasnya pun sama — keduanya mengisi kolom pada tabel
	// yang sama, sehingga dua batas yang berbeda akan menolak kalimat yang sama pada satu
	// layar dan menerimanya pada layar lain.
	message := strings.Repeat("é", 4000)

	command, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{Destination: "PUSAT", Message: message},
		sender(), atNoon(),
	)

	require.NoError(t, err)
	require.Greater(t, len(command.Message), 4000,
		"uji ini kehilangan maknanya bila pesannya ternyata satu bita per huruf")
}

func TestTheTimestampIsStoredInUTC(t *testing.T) {
	jakarta := time.FixedZone("WIB", 7*60*60)

	command, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{Destination: "PUSAT", Message: "halo"},
		sender(), time.Date(2026, 9, 24, 12, 0, 0, 0, jakarta),
	)

	require.NoError(t, err)
	require.Equal(t, time.UTC, command.CreatedAt.Location())
	require.Equal(t, 5, command.CreatedAt.Hour())
}
