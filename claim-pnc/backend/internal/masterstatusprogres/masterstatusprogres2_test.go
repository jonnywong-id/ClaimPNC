package masterstatusprogres_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatusprogres"
)

// Clean memangkas kedua ujung setiap isian.
//
// Tanpa ini, nama yang diketik dengan spasi di ujung lolos pemeriksaan "wajib diisi"
// padahal isinya kosong — dan tersimpan sebagai baris master yang tampak kosong di layar.
func TestInput2Clean(t *testing.T) {
	cleaned := masterstatusprogres.Input2{
		Name:     "  SURAT PERMINTAAN DOKUMEN DIKIRIM  ",
		ParentID: "  01  ",
	}.Clean()

	require.Equal(t, "SURAT PERMINTAAN DOKUMEN DIKIRIM", cleaned.Name)
	require.Equal(t, "01", cleaned.ParentID)
}

// Check mengembalikan SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera: layar lama menampilkan semua pesan bersamaan,
// dan mengembalikan satu per satu akan membuat pengguna menekan Simpan berkali-kali untuk
// menemukan kesalahan yang sebenarnya sudah diketahui sejak awal.
func TestInput2Check(t *testing.T) {
	testCase := []struct {
		name      string
		input     masterstatusprogres.Input2
		violation []string
	}{
		{
			name:  "isian lengkap diterima",
			input: masterstatusprogres.Input2{Name: "DOKUMEN SUSULAN DITERIMA", ParentID: "01"},
		},
		{
			name:      "nama kosong ditolak",
			input:     masterstatusprogres.Input2{Name: "", ParentID: "01"},
			violation: []string{"nama"},
		},
		{
			name:      "induk tidak dipilih ditolak",
			input:     masterstatusprogres.Input2{Name: "DOKUMEN SUSULAN DITERIMA", ParentID: ""},
			violation: []string{"id_induk"},
		},
		{
			name: "nama melebihi panjang kolom ditolak",
			input: masterstatusprogres.Input2{
				Name:     strings.Repeat("A", masterstatusprogres.MaxNameLength2+1),
				ParentID: "01",
			},
			violation: []string{"nama"},
		},
		{
			name:  "nama tepat sepanjang batas diterima",
			input: masterstatusprogres.Input2{Name: strings.Repeat("A", masterstatusprogres.MaxNameLength2), ParentID: "01"},
		},
		{
			// Kasus yang membuktikan pelanggaran DIKUMPULKAN, bukan dihentikan pada yang
			// pertama.
			name:      "kedua isian kosong menghasilkan dua pelanggaran",
			input:     masterstatusprogres.Input2{},
			violation: []string{"nama", "id_induk"},
		},
	}

	for _, singleCase := range testCase {
		t.Run(singleCase.name, func(t *testing.T) {
			err := singleCase.input.Check()

			if len(singleCase.violation) == 0 {
				require.NoError(t, err)
				return
			}

			var validationError *masterstatusprogres.ValidationError
			require.ErrorAs(t, err, &validationError)

			field := make([]string, 0, len(validationError.Violation))
			for _, v := range validationError.Violation {
				field = append(field, v.Field)
				require.NotEmpty(t, v.Message, "setiap pelanggaran wajib punya pesan untuk pengguna")
			}
			require.ElementsMatch(t, singleCase.violation, field)
		})
	}
}

// Nama isian pada pelanggaran WAJIB sama persis dengan field JSON yang dikirim layar.
//
// Kalau berbeda, pesan galat menggantung di atas form alih-alih menempel di isian yang
// salah — dan itu gagal secara diam-diam: responsnya tetap 422 dan tetap tampak benar.
func TestInput2ViolationFieldMatchesJSONName(t *testing.T) {
	err := masterstatusprogres.Input2{}.Check()

	var validationError *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationError)

	field := make([]string, 0, len(validationError.Violation))
	for _, v := range validationError.Violation {
		field = append(field, v.Field)
	}
	// Keduanya dideklarasikan di http/dto2.go pada SaveRequest2.
	require.ElementsMatch(t, []string{"nama", "id_induk"}, field)
}

// FormatID2 memakai angka POLOS, tanpa awalan "0".
//
// Ini perbedaan nyata terhadap tingkat 1, terverifikasi dua kali — dari rule
// (Activity/InsertMstStatusProgress2_act memakai nilainya apa adanya, sementara tingkat 1
// merangkai "0"+) dan dari data yang beredar (GetDataProgressClaim-SQL.xml menyaring
// STATUS_PROGRESS2 dengan '2', '24', '60', bukan '02').
//
// Menyamakannya dengan tingkat 1 akan menerbitkan ID yang tidak dapat dicocokkan dengan
// baris GCNM_PROGRESS_CLAIM yang sudah ada — dan tidak ada galat apa pun yang muncul.
func TestFormatID2HasNoPrefix(t *testing.T) {
	require.Equal(t, "1", masterstatusprogres.FormatID2(1))
	require.Equal(t, "9", masterstatusprogres.FormatID2(9))
	require.Equal(t, "10", masterstatusprogres.FormatID2(10))
	require.Equal(t, "60", masterstatusprogres.FormatID2(60))

	for _, sequence := range []int{1, 9, 10, 60, 100} {
		require.NotEqual(t, "0", string(masterstatusprogres.FormatID2(sequence)[0]),
			"tingkat 2 tidak pernah berawalan nol")
	}
}

// ErrParentNotFound dibedakan dari ErrNotFound.
//
// Keduanya berarti "tidak ada", tetapi yang satu menunjuk isian yang dapat diperbaiki
// pengguna (422) dan yang lain menunjuk baris yang dimintanya (404). Menyatukannya akan
// membuat layar mengatakan barisnya sendiri hilang padahal yang hilang adalah induknya.
func TestParentNotFoundIsDistinct(t *testing.T) {
	require.False(t, errors.Is(masterstatusprogres.ErrParentNotFound, masterstatusprogres.ErrNotFound))
	require.False(t, errors.Is(masterstatusprogres.ErrNotFound, masterstatusprogres.ErrParentNotFound))
}
