package masterbengkel_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
)

// validInput adalah isian yang lolos seluruh pemeriksaan.
//
// Uji di bawah mengubah SATU isian darinya, sehingga yang membuat sebuah kasus gagal
// selalu terbaca dari kasusnya sendiri.
func validInput() masterbengkel.Input {
	return masterbengkel.Input{
		Name:            "Bengkel Contoh Utama",
		Address:         "Jalan Contoh Nomor 1",
		Phone:           "021-0000001",
		Email:           "kontak@contoh.example",
		BranchID:        "001",
		BranchName:      "Cabang Contoh",
		CityID:          "3171",
		CityName:        "Jakarta Pusat",
		PartnerStatus:   "1",
		WorkshopStatus:  "1",
		Login:           "bengkelcontoh",
		BankID:          "002",
		BankName:        "Bank Contoh",
		AccountNumber:   "1000000001",
		ValueAddedTax:   "11",
		ServiceDiscount: "10",
	}
}

func TestValidInputPasses(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// Ketiga isian wajib ditolak saat kosong.
//
// Daftarnya SENGAJA pendek. Sistem lama tidak punya satu pun prasyarat "wajib diisi"
// pada layar ini; yang diwajibkan di sini hanyalah yang tanpanya baris itu tidak dapat
// dipakai siapa pun — dan masing-masing dapat dibenarkan dari export. Uji ini yang
// menjaga daftar itu tidak diam-diam bertambah, karena setiap tambahan berarti menolak
// penambahan yang hari ini diterima.
func TestRequiredFields(t *testing.T) {
	for _, c := range []struct {
		field string
		blank func(*masterbengkel.Input)
	}{
		{"nama_bengkel", func(i *masterbengkel.Input) { i.Name = "" }},
		{"status_rekanan", func(i *masterbengkel.Input) { i.PartnerStatus = "" }},
		{"login_aplikasi", func(i *masterbengkel.Input) { i.Login = "" }},
	} {
		t.Run(c.field, func(t *testing.T) {
			input := validInput()
			c.blank(&input)

			require.Equal(t, []string{c.field}, violatedFields(input.Clean().Check()))
		})
	}
}

// Bengkel NON-REKANAN boleh tidak punya login aplikasi.
//
// Ini bukan kelonggaran melainkan peniruan: `Activity/ValidationLoginBengkel_act`
// melompati seluruh urusan login pada prasyarat `Local.STS_REKANAN=='0'`. Mewajibkannya
// akan menolak baris yang hari ini sah.
func TestNonPartnerMayHaveNoLogin(t *testing.T) {
	input := validInput()
	input.PartnerStatus = masterbengkel.PartnerStatusNonPartner
	input.Login = ""

	require.NoError(t, input.Clean().Check())
}

// Bengkel REKANAN tanpa login tetap ditolak.
//
// Kebalikan dari kasus di atas, dan sama pentingnya: bengkel rekanan tanpa login akan
// tersimpan diam-diam tanpa pernah dapat masuk, dan tidak ada satu pun layar di sistem
// lama yang menjelaskan sebabnya.
func TestPartnerWithoutLoginRejected(t *testing.T) {
	input := validInput()
	input.PartnerStatus = "1"
	input.Login = ""

	require.Equal(t, []string{"login_aplikasi"}, violatedFields(input.Clean().Check()))
}

// Seluruh isian selain ketiga yang wajib boleh kosong.
//
// Sistem lama menerimanya, dan menolaknya di sini berarti selisih perilaku yang tidak
// diminta siapa pun.
func TestEverythingElseMayBeBlank(t *testing.T) {
	input := masterbengkel.Input{
		Name:          "Bengkel Paling Sederhana",
		PartnerStatus: masterbengkel.PartnerStatusNonPartner,
	}
	require.NoError(t, input.Clean().Check())
}

// Kelima persentase ditolak bila bukan angka 0–100.
//
// SELISIH YANG DIRENCANAKAN terhadap Pega, yang menerima teks apa pun — termasuk "abc" —
// dan membiarkan akibatnya muncul jauh di hilir pada perhitungan yang memakainya.
func TestPercentFields(t *testing.T) {
	percentField := map[string]func(*masterbengkel.Input, string){
		"ppn":              func(i *masterbengkel.Input, v string) { i.ValueAddedTax = v },
		"diskon_jasa":      func(i *masterbengkel.Input, v string) { i.ServiceDiscount = v },
		"diskon_sparepart": func(i *masterbengkel.Input, v string) { i.PartDiscount = v },
		"persen_material":  func(i *masterbengkel.Input, v string) { i.MaterialPercent = v },
		"pct_selisih_pl":   func(i *masterbengkel.Input, v string) { i.PriceListGapPercent = v },
	}

	for field, set := range percentField {
		t.Run(field+"/ditolak", func(t *testing.T) {
			for _, value := range []string{"abc", "12abc", "-1", "101", "NaN", "Inf"} {
				input := validInput()
				set(&input, value)
				require.Containsf(t, violatedFields(input.Clean().Check()), field,
					"%q seharusnya ditolak", value)
			}
		})

		t.Run(field+"/diterima", func(t *testing.T) {
			// Koma DAN titik keduanya diterima: petugas Indonesia mengetik "12,5"
			// sementara nilai yang tersimpan di basis data memakai "12.5".
			for _, value := range []string{"", "0", "11", "100", "12,5", "12.5"} {
				input := validInput()
				set(&input, value)
				require.NotContainsf(t, violatedFields(input.Clean().Check()), field,
					"%q seharusnya diterima", value)
			}
		})
	}
}

// Nilai persentase TIDAK PERNAH diubah bentuknya saat disimpan.
//
// Yang tersimpan tetap teks apa adanya (D-51): pembulatan dan penyeragaman pemisah
// desimal tidak boleh terjadi di jalur ini.
func TestPercentValueKeptVerbatim(t *testing.T) {
	input := validInput()
	input.ServiceDiscount = "12,50"

	clean := input.Clean()
	require.NoError(t, clean.Check())
	require.Equal(t, "12,50", clean.ServiceDiscount)
}

// SELURUH pelanggaran dikembalikan sekaligus, bukan yang pertama saja (P-5).
func TestAllViolationsReturnedAtOnce(t *testing.T) {
	input := masterbengkel.Input{ValueAddedTax: "abc"}

	field := violatedFields(input.Clean().Check())
	require.Contains(t, field, "nama_bengkel")
	require.Contains(t, field, "status_rekanan")
	require.Contains(t, field, "ppn")
	require.GreaterOrEqual(t, len(field), 3)
}

// Spasi di kedua ujung dipangkas sebelum diperiksa DAN sebelum disimpan.
func TestCleanTrimsEveryField(t *testing.T) {
	input := masterbengkel.Input{
		Name:          "  Bengkel Contoh  ",
		PartnerStatus: " 1 ",
		Login:         "  bengkelcontoh  ",
		TaxNumber:     "  00.000.000.0-000.001  ",
	}

	clean := input.Clean()
	require.Equal(t, "Bengkel Contoh", clean.Name)
	require.Equal(t, "1", clean.PartnerStatus)
	require.Equal(t, "bengkelcontoh", clean.Login)
	require.Equal(t, "00.000.000.0-000.001", clean.TaxNumber)
}

// Isian yang hanya berisi spasi diperlakukan sebagai kosong.
func TestBlankSpaceCountsAsEmpty(t *testing.T) {
	input := validInput()
	input.Name = "   "

	require.Contains(t, violatedFields(input.Clean().Check()), "nama_bengkel")
}

// Ketiga status persetujuan dikenal; selebihnya tidak.
func TestApprovalStatusKnown(t *testing.T) {
	for _, s := range []masterbengkel.ApprovalStatus{
		masterbengkel.StatusPending,
		masterbengkel.StatusApproved,
		masterbengkel.StatusRejected,
	} {
		require.True(t, s.Known())
		require.NotEmpty(t, s.Label())
	}

	for _, s := range []masterbengkel.ApprovalStatus{"", "3", "approve", "0 "} {
		require.Falsef(t, s.Known(), "status %q seharusnya tidak dikenal", s)
		require.Empty(t, s.Label())
	}
}

// Label status mengikuti caption tab Pega apa adanya (D-13).
func TestApprovalStatusLabelFollowsPegaTabs(t *testing.T) {
	require.Equal(t, "Approve", masterbengkel.StatusApproved.Label())
	require.Equal(t, "Waiting Approval", masterbengkel.StatusPending.Label())
	require.Equal(t, "Reject", masterbengkel.StatusRejected.Label())
}

// ComposeID meniru `Database/PEGA_M_BENGKEL_HE.prc:19` persis.
func TestComposeID(t *testing.T) {
	for _, c := range []struct {
		site     string
		sequence int64
		expected string
	}{
		{"01", 1, "010000000001"},
		{"01", 8125, "010000008125"},
		{"02", 1234567890, "021234567890"},
		{"  01  ", 7, "010000000007"},
	} {
		require.Equal(t, c.expected, masterbengkel.ComposeID(c.site, c.sequence, 10))
	}
}

// Nomor urut yang LEBIH PANJANG dari lebarnya tidak dipotong.
//
// LPAD Oracle memotongnya dari kanan, sehingga urutan yang melampaui sepuluh digit akan
// menghasilkan kunci yang bertabrakan dengan urutan lain — diam-diam. Di sini kuncinya
// dibiarkan tumbuh: ia menjadi lebih panjang, dan itu TERLIHAT.
func TestComposeIDDoesNotTruncate(t *testing.T) {
	id := masterbengkel.ComposeID("01", 12345678901, 10)
	require.Equal(t, "0112345678901", id)
	require.Len(t, id, 13)
}

// Pesan galat domain menyebut modulnya, supaya log dapat ditelusuri asalnya.
func TestErrorsNameTheModule(t *testing.T) {
	for _, err := range []error{
		masterbengkel.ErrNotFound,
		masterbengkel.ErrNameTaken,
		masterbengkel.ErrLoginTaken,
		masterbengkel.ErrUnknownStatus,
	} {
		require.True(t, strings.HasPrefix(err.Error(), "masterbengkel:"))
	}
}

// OneViolation menghasilkan galat yang BERBENTUK SAMA dengan pelanggaran isian lain.
//
// Tanpa itu, satu pemeriksaan akan menjawab 500 sementara pemeriksaan lain menjawab 422
// pada isiannya — dan pengguna tidak punya cara mengetahui isian mana yang salah.
func TestOneViolationIsAValidationError(t *testing.T) {
	err := masterbengkel.OneViolation("nama_bengkel", "Nama tersebut telah digunakan.")

	var validationError *masterbengkel.ValidationError
	require.True(t, errors.As(err, &validationError))
	require.Equal(t, []string{"nama_bengkel"}, violatedFields(err))
}

// violatedFields mengambil nama isian yang dilanggar dari sebuah galat validasi.
func violatedFields(err error) []string {
	var validationError *masterbengkel.ValidationError
	if !errors.As(err, &validationError) {
		return nil
	}

	field := make([]string, 0, len(validationError.Violation))
	for _, p := range validationError.Violation {
		field = append(field, p.Field)
	}
	return field
}
