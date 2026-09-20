package masterpasal_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasal"
)

// Kategori diturunkan dari kodenya, dan cabang `else` MEMAAFKAN kode apa pun.
//
// Asalnya `Activity/CNMInsertPasalDataMaster-Act.xml` langkah 1:
//
//	@if(...==1,"Jaminan Polis", @if(...==2,"Pengecualian","Notifikasi"))
//
// Uji ini yang menjaga cabang `else` tetap memaafkan. Menggantinya dengan `case "3"` akan
// membuat baris lama berkode lain menampilkan sel kosong yang tampak seperti cacat layar,
// dan kesalahan itu tidak akan terlihat sampai data sungguhan dibuka.
func TestCategoryLabel(t *testing.T) {
	for name, testcase := range map[string]struct{ code, want string }{
		"jaminan polis":          {masterpasal.CategoryPolicyCoverage, "Jaminan Polis"},
		"pengecualian":           {masterpasal.CategoryException, "Pengecualian"},
		"kosong jadi notifikasi": {masterpasal.CategoryNotification, "Notifikasi"},
		"kode tak dikenal":       {"3", "Notifikasi"},
		"kode karangan":          {"XYZ", "Notifikasi"},
		"berspasi tetap cocok":   {" 1 ", "Jaminan Polis"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, testcase.want, masterpasal.CategoryLabel(testcase.code))
		})
	}
}

// Ketiga pilihan Kategori tersedia, dan "Notifikasi" memang berkode KOSONG.
//
// Kodenya kosong karena itulah cabang `else` — bukan karena kodenya belum diisi. Uji ini
// membuat pilihan itu terlihat sebagai keputusan, sehingga siapa pun yang kelak
// menggantinya dengan "3" melakukannya dengan sadar dan dengan bukti baru di tangan.
func TestCategories(t *testing.T) {
	list := masterpasal.Categories()
	require.Len(t, list, 3)

	require.Equal(t, masterpasal.CategoryPolicyCoverage, list[0].Code)
	require.Equal(t, "Jaminan Polis", list[0].Label)
	require.Equal(t, masterpasal.CategoryException, list[1].Code)
	require.Equal(t, "Pengecualian", list[1].Label)
	require.Empty(t, list[2].Code, "Notifikasi adalah cabang else; kodenya memang kosong")
	require.Equal(t, "Notifikasi", list[2].Label)

	// Menyunting hasil pemanggilan tidak boleh mengubah pemanggilan berikutnya.
	list[0].Label = "diubah"
	require.Equal(t, "Jaminan Polis", masterpasal.Categories()[0].Label)
}

// Clean memangkas seluruh isian dan MEMBUANG butir bisnis yang sama sekali kosong.
//
// Pembuangan itu satu-satunya selisih terencana atas isian Bisnis. Layar lama menerbitkan
// baris kosong seketika saat ikon tambah ditekan, dan menyimpannya berarti menyimpan butir
// yang tidak menunjuk apa pun.
func TestCleanTrimsAndDropsEmptyBusiness(t *testing.T) {
	clean := masterpasal.Input{
		Number:      "  PSL-001  ",
		Text:        "  isi pasal  ",
		Description: "  keterangan  ",
		Category:    "  1  ",
		Business: []masterpasal.Business{
			{ID: "  2001  ", Name: "  Fire / Property  "},
			{ID: "   ", Name: "   "},
			{ID: "", Name: "Diketik bebas"},
			{ID: "2002", Name: ""},
		},
	}.Clean()

	require.Equal(t, "PSL-001", clean.Number)
	require.Equal(t, "isi pasal", clean.Text)
	require.Equal(t, "keterangan", clean.Description)
	require.Equal(t, "1", clean.Category)

	require.Len(t, clean.Business, 3, "hanya butir yang KEDUA-DUANYA kosong yang dibuang")
	require.Equal(t, masterpasal.Business{ID: "2001", Name: "Fire / Property"}, clean.Business[0])
	require.Equal(t, masterpasal.Business{ID: "", Name: "Diketik bebas"}, clean.Business[1],
		"butir tanpa kode TETAP tersimpan; autocomplete lama ber-pyAllowFreeFormInput=true")
	require.Equal(t, masterpasal.Business{ID: "2002", Name: ""}, clean.Business[2])
}

// Satu-satunya isian wajib adalah No Pasal, dan itu memang seluruh isi layar lama.
func TestCheckOnlyRequiresNumber(t *testing.T) {
	t.Run("No Pasal kosong ditolak", func(t *testing.T) {
		err := masterpasal.Input{}.Clean().Check()

		var validationError *masterpasal.ValidationError
		require.True(t, errors.As(err, &validationError))
		require.Len(t, validationError.Violation, 1)
		require.Equal(t, "no_pasal", validationError.Violation[0].Field)
		require.Contains(t, validationError.Violation[0].Message, "No Pasal")
	})

	t.Run("hanya spasi tetap ditolak", func(t *testing.T) {
		require.Error(t, masterpasal.Input{Number: "   "}.Clean().Check())
	})

	// Ketiganya sengaja TIDAK diwajibkan — keputusan Work Owner 2026-09-19, "jalankan as
	// is". Uji ini yang menjaga keputusan itu tetap terlihat: menambahkan salah satunya
	// sebagai isian wajib akan menggagalkannya, sehingga perubahan itu tidak dapat
	// terjadi tanpa disadari.
	t.Run("sisanya boleh kosong seluruhnya", func(t *testing.T) {
		require.NoError(t, masterpasal.Input{Number: "PSL-001"}.Clean().Check())
	})
}

// No Pasal TIDAK diperiksa keunikannya. Kuncinya IDDATA, dan Pega pun tidak memeriksanya.
//
// Uji ini tidak dapat membuktikan ketiadaan aturan secara langsung; yang dibuktikannya
// adalah dua isian bernomor sama sama-sama lolos pemeriksaan. Bila kelak keunikan
// diberlakukan, uji ini gagal dan keputusannya dibahas ulang — bukan berubah diam-diam.
func TestCheckAllowsDuplicateNumber(t *testing.T) {
	first := masterpasal.Input{Number: "PSL-001", Text: "pertama"}.Clean()
	second := masterpasal.Input{Number: "PSL-001", Text: "kedua"}.Clean()

	require.NoError(t, first.Check())
	require.NoError(t, second.Check())
}

// Nomor IDDATA diturunkan di Go, dan kedua cacat procedure lama tidak ikut terbawa.
func TestNextSequence(t *testing.T) {
	t.Run("tabel kosong mulai dari 1", func(t *testing.T) {
		// Procedure lama memakai `max()` TANPA `NVL`, sehingga pada tabel kosong ia
		// menyisipkan IDDATA kosong — baris pertama lahir tanpa kunci.
		require.Equal(t, "1", masterpasal.NextSequence(nil))
	})

	t.Run("melanjutkan yang terbesar", func(t *testing.T) {
		require.Equal(t, "13", masterpasal.NextSequence([]string{"1", "12", "7"}))
	})

	t.Run("ID bukan angka dilewati, bukan menggagalkan", func(t *testing.T) {
		// `TO_NUMBER` pada procedure lama gagal dengan ORA-01722 di sini, dan kegagalan
		// itu menghentikan SELURUH penambahan.
		require.Equal(t, "6", masterpasal.NextSequence([]string{"5", "PSL-A", ""}))
	})

	t.Run("ID bukan angka tetap dihitung terpakai", func(t *testing.T) {
		// "1" sudah dipakai baris berbentuk lain, sehingga calon berikutnya harus
		// melewatinya alih-alih menabraknya.
		require.Equal(t, "2", masterpasal.NextSequence([]string{"1"}))
	})

	t.Run("spasi tepi tidak membuat ID tampak berbeda", func(t *testing.T) {
		require.Equal(t, "3", masterpasal.NextSequence([]string{" 1 ", "2 "}))
	})
}

// FormatID menghasilkan angka polos — tanpa awalan dan tanpa nol di depan.
//
// `to_char(JUM_PASAL)` pada procedure lama tidak memakai format mask, sehingga bentuk
// "01" atau "001" justru akan menyimpang darinya.
func TestFormatID(t *testing.T) {
	require.Equal(t, "1", masterpasal.FormatID(1))
	require.Equal(t, "42", masterpasal.FormatID(42))
	require.Equal(t, "100", masterpasal.FormatID(100))
}
