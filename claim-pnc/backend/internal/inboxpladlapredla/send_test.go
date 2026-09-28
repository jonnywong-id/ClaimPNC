package inboxpladlapredla_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
)

// Dokumen tanpa alamat reasuradur TIDAK dapat dikirim.
//
// Pega memeriksanya pula ("Email Reinsurer Kosong"). Tanpa pemeriksaan ini, kegagalannya
// muncul sebagai galat SMTP — yang terbaca sebagai gangguan jaringan, bukan sebagai data
// yang belum lengkap.
func TestAnAdviceWithoutAReinsurerEmailCannotBeSent(t *testing.T) {
	advice := inboxpladlapredla.SendableAdvice{Email: "   "}

	require.ErrorIs(t, advice.CanBeSent(), inboxpladlapredla.ErrReinsurerEmailEmpty)
}

// Dokumen yang SUDAH terkirim tidak dikirim ulang.
func TestAnAlreadySentAdviceCannotBeSentAgain(t *testing.T) {
	advice := inboxpladlapredla.SendableAdvice{
		Email: "reasuradur@contoh.example",
		Sent:  "1",
	}

	require.ErrorIs(t, advice.CanBeSent(), inboxpladlapredla.ErrAdviceAlreadySent)
}

// Kolom alamat yang memuat BEBERAPA alamat dipecah menjadi daftar penerima.
//
// Isian yang diketik manusia kerap memuat beberapa alamat dipisahkan koma atau titik
// koma. Mengirim ke teks gabungan itu apa adanya akan ditolak relay SMTP, dan
// penolakannya terbaca sebagai gangguan jaringan.
func TestSeveralAddressesInOneFieldBecomeSeveralRecipients(t *testing.T) {
	advice := inboxpladlapredla.SendableAdvice{
		Email: "satu@contoh.example; dua@contoh.example , tiga@contoh.example",
	}

	require.Equal(t, []string{
		"satu@contoh.example", "dua@contoh.example", "tiga@contoh.example",
	}, advice.Recipients())
}

// Alamat yang sama hanya dikirimi SEKALI, dan perbedaan huruf besar-kecil tidak dihitung.
//
// Tanpa ini, satu reasuradur menerima dua surat yang sama karena kolomnya memuat
// alamatnya dua kali dengan kapitalisasi berbeda.
func TestDuplicateRecipientsAreSentToOnce(t *testing.T) {
	advice := inboxpladlapredla.SendableAdvice{
		Email: "reas@contoh.example, REAS@contoh.example",
	}

	require.Len(t, advice.Recipients(), 1)
}

// Isian yang bukan alamat dibuang, bukan dikirimi.
func TestTextThatIsNotAnAddressIsNotARecipient(t *testing.T) {
	advice := inboxpladlapredla.SendableAdvice{Email: "belum ada, -"}

	require.Empty(t, advice.Recipients())
}

// Bahasa surat mengikuti NEGARA reasuradur, dan kosong berarti Inggris.
//
// Pega memakai `@contains(param.country,"INDONESIA")`. Aturannya ditiru, termasuk
// kelonggarannya. Negara yang kosong menghasilkan bahasa Inggris — pilihan yang lebih
// aman: surat berbahasa Inggris kepada reasuradur Indonesia tetap terbaca, sedangkan
// sebaliknya belum tentu.
func TestLetterLanguageFollowsTheReinsurerCountry(t *testing.T) {
	kasus := []struct {
		negara    string
		indonesia bool
	}{
		{"INDONESIA", true},
		{"indonesia", true},
		{"REPUBLIC OF INDONESIA", true},
		{"SINGAPORE", false},
		{"", false},
	}

	for _, satu := range kasus {
		advice := inboxpladlapredla.SendableAdvice{Country: satu.negara}
		require.Equal(t, satu.indonesia, advice.InIndonesian(),
			"negara %q salah dinilai", satu.negara)
	}
}

// Subjek surat menyebut jenis dokumennya, dan bentuknya mengikuti Pega.
func TestTheSubjectNamesTheAdviceKind(t *testing.T) {
	claim := inboxpladlapredla.ClaimSummary{
		ClaimNo:  "PNC-1001",
		PolicyNo: "00.000.2026.00001",
		Insured:  "Tertanggung Contoh",
		Business: "Marine Cargo",
		LossDate: "2026-01-23",
	}

	pla, ok := inboxpladlapredla.FindTab("pla")
	require.True(t, ok)
	dla, ok := inboxpladlapredla.FindTab("dla")
	require.True(t, ok)

	require.Contains(t, inboxpladlapredla.Subject(pla, claim),
		"PLA Supporting Document Claim Marine Cargo")
	require.Contains(t, inboxpladlapredla.Subject(dla, claim),
		"DLA Supporting Document Claim Marine Cargo")

	// Keterangan klaimnya ikut, persis seperti bentuk Pega.
	require.Contains(t, inboxpladlapredla.Subject(pla, claim), "PNC-1001")
	require.Contains(t, inboxpladlapredla.Subject(pla, claim),
		"a/n. Tertanggung Contoh")
	require.Contains(t, inboxpladlapredla.Subject(pla, claim),
		"Policy : 00.000.2026.00001")
	require.Contains(t, inboxpladlapredla.Subject(pla, claim), "Dol : 2026-01-23")
}

// Kategori lampiran mengikuti jenis dokumennya.
//
// `UpdateDetailPLA2-Act.xml:251646` memakai `"PLA"`. Kategori yang salah menghasilkan
// surat tanpa lampiran — dan itu tidak menghasilkan galat apa pun.
func TestTheAttachmentCategoryFollowsTheAdviceKind(t *testing.T) {
	pla, ok := inboxpladlapredla.FindTab("pla")
	require.True(t, ok)
	dla, ok := inboxpladlapredla.FindTab("dla")
	require.True(t, ok)
	pre, ok := inboxpladlapredla.FindTab("pre-dla")
	require.True(t, ok)

	require.Equal(t, "PLA", inboxpladlapredla.AttachmentCategory(pla))
	require.Equal(t, "DLA", inboxpladlapredla.AttachmentCategory(dla))

	// Tab Pre DLA tidak mengirim surat; kategorinya tidak pernah dipakai, dan nilainya
	// jatuh ke PLA. Diuji supaya perubahannya terlihat bila kelak ia dipakai.
	require.Equal(t, "PLA", inboxpladlapredla.AttachmentCategory(pre))
}
