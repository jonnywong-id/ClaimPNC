package mastermasking_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastermasking"
)

// valid mengembalikan baris yang seluruh isiannya benar, untuk dirusak satu per satu.
//
// Menulisnya sekali membuat setiap uji di bawah menyatakan SATU aturan saja — bila ia
// gagal, yang salah pasti aturan itu, bukan isian lain yang kebetulan ikut cacat.
func valid() mastermasking.Masking {
	return mastermasking.Masking{
		BranchID:    "100081",
		Login:       "CONTOH.ADMIN",
		Module:      "PNCSearchKlaim",
		SubModule:   "Registrasi,Dokumen,",
		SearchQuota: 10,
		ViewQuota:   10,
	}
}

// fieldOf mengembalikan daftar field yang dilanggar, supaya uji dapat menyatakan KOLOM
// mana yang ditandai — bukan sekadar "ada pelanggaran".
func fieldOf(violation []mastermasking.Violation) []string {
	field := make([]string, 0, len(violation))
	for _, v := range violation {
		field = append(field, v.Field)
	}
	return field
}

// Isian yang benar tidak menghasilkan pelanggaran apa pun.
func TestValidPassesCheck(t *testing.T) {
	require.Empty(t, mastermasking.CheckMasking(valid()))
}

// Cabang, nama pengguna, dan modul wajib diisi.
//
// Ketiganya diuji bersama karena aturannya sama dan alasannya sama: baris tanpa salah satu
// dari ketiganya adalah kewenangan yang tidak menunjuk siapa, di mana, atau berlaku pada
// apa — ia tampak memberi hak padahal tidak berlaku di mana-mana.
func TestRequiredFields(t *testing.T) {
	for name, tc := range map[string]struct {
		damage func(*mastermasking.Masking)
		field  string
	}{
		"cabang kosong": {func(m *mastermasking.Masking) { m.BranchID = "" }, mastermasking.FieldBranchID},
		"login kosong":  {func(m *mastermasking.Masking) { m.Login = "" }, mastermasking.FieldLogin},
		"modul kosong":  {func(m *mastermasking.Masking) { m.Module = "" }, mastermasking.FieldModule},
	} {
		t.Run(name, func(t *testing.T) {
			m := valid()
			tc.damage(&m)
			require.Contains(t, fieldOf(mastermasking.CheckMasking(m)), tc.field)
		})
	}
}

// Isian yang hanya berisi spasi diperlakukan sama dengan kosong.
//
// Tanpa ini, menekan spasi sekali akan melewati seluruh pemeriksaan "wajib diisi" dan
// menghasilkan baris yang di layar tampak kosong.
func TestWhitespaceOnlyCountsAsEmpty(t *testing.T) {
	m := valid()
	m.Login = "   "
	require.Contains(t, fieldOf(mastermasking.CheckMasking(m)), mastermasking.FieldLogin)
}

// Sub modul BOLEH kosong.
//
// Bukan kelalaian: satu baris portal ASM memang tidak punya sub modul, dan memaksanya
// terisi akan menolak keadaan yang sah dan sudah berjalan.
func TestSubModuleMayBeEmpty(t *testing.T) {
	m := valid()
	m.SubModule = ""
	require.Empty(t, mastermasking.CheckMasking(m))
}

// Kuota nol diterima.
//
// Nol berarti "tidak boleh mencari sama sekali", dan itu pernyataan kewenangan yang sah —
// bukan isian yang belum diisi.
func TestZeroQuotaAccepted(t *testing.T) {
	m := valid()
	m.SearchQuota = 0
	m.ViewQuota = 0
	require.Empty(t, mastermasking.CheckMasking(m))
}

// Kuota negatif ditolak, dan kuota di atas langit-langit juga.
func TestQuotaBounds(t *testing.T) {
	for name, tc := range map[string]struct {
		damage func(*mastermasking.Masking)
		field  string
	}{
		"cari negatif":   {func(m *mastermasking.Masking) { m.SearchQuota = -1 }, mastermasking.FieldSearchQuota},
		"lihat negatif":  {func(m *mastermasking.Masking) { m.ViewQuota = -1 }, mastermasking.FieldViewQuota},
		"cari kebesaran": {func(m *mastermasking.Masking) { m.SearchQuota = mastermasking.MaxQuota + 1 }, mastermasking.FieldSearchQuota},
	} {
		t.Run(name, func(t *testing.T) {
			m := valid()
			tc.damage(&m)
			require.Contains(t, fieldOf(mastermasking.CheckMasking(m)), tc.field)
		})
	}
}

// Kuota 100.000 DITERIMA.
//
// Angka ini bukan karangan: ia benar-benar ada di portal ASM. Uji ini menjaga supaya
// langit-langit tidak kelak diperketat menjadi angka yang menolak data produksi sendiri.
func TestLargeQuotaFromProductionAccepted(t *testing.T) {
	m := valid()
	m.SearchQuota = 100000
	m.ViewQuota = 100000
	require.Empty(t, mastermasking.CheckMasking(m))
}

// Seluruh pelanggaran dikumpulkan sekaligus, tidak berhenti pada yang pertama.
//
// Ini kesetaraan perilaku dengan sistem lama, yang menampilkan semua pesan bersamaan —
// mengembalikannya satu per satu akan membuat pengguna menekan Simpan berkali-kali untuk
// menemukan kesalahan berikutnya.
func TestAllViolationsCollected(t *testing.T) {
	violation := mastermasking.CheckMasking(mastermasking.Masking{SearchQuota: -1})

	field := fieldOf(violation)
	require.Contains(t, field, mastermasking.FieldBranchID)
	require.Contains(t, field, mastermasking.FieldLogin)
	require.Contains(t, field, mastermasking.FieldModule)
	require.Contains(t, field, mastermasking.FieldSearchQuota)
	require.GreaterOrEqual(t, len(violation), 4)
}

// Panjang dihitung dalam rune, bukan byte.
//
// Satu huruf beraksen memakan dua byte; menghitung byte akan membuat batas terasa
// berubah-ubah bagi pengguna tanpa sebab yang terlihat.
func TestLengthCountedInRunes(t *testing.T) {
	m := valid()
	m.Login = strings.Repeat("é", mastermasking.MaxLoginLength)
	require.Empty(t, mastermasking.CheckMasking(m), "tepat pada batas harus diterima")

	m.Login = strings.Repeat("é", mastermasking.MaxLoginLength+1)
	require.Contains(t, fieldOf(mastermasking.CheckMasking(m)), mastermasking.FieldLogin)
}

// Kode cabang lebih panjang dari kolom POOLDATA.BRANCH.ID ditolak dengan pesan yang jelas.
//
// Tanpa ini, isian seperti itu akan lolos validasi lalu ditolak sebagai "cabang tidak
// ditemukan" — pesan yang benar tetapi menyesatkan, karena sebabnya bukan cabangnya tidak
// ada melainkan kodenya mustahil.
func TestBranchIDTooLongRejected(t *testing.T) {
	m := valid()
	m.BranchID = strings.Repeat("9", mastermasking.MaxBranchIDLength+1)
	require.Contains(t, fieldOf(mastermasking.CheckMasking(m)), mastermasking.FieldBranchID)
}

// PairKey mengabaikan besar-kecil huruf dan spasi tepi.
//
// Dua baris untuk orang yang sama, satu ditulis huruf besar dan satu huruf kecil, adalah
// dua kewenangan terpisah yang keduanya berlaku — dan yang satu dapat luput saat dicabut.
func TestPairKeyIgnoresCaseAndSpace(t *testing.T) {
	require.Equal(t,
		mastermasking.PairKey("100081", "BUDI"),
		mastermasking.PairKey(" 100081 ", " budi "),
	)
	require.NotEqual(t,
		mastermasking.PairKey("100081", "BUDI"),
		mastermasking.PairKey("100082", "BUDI"),
		"cabang berbeda adalah pasangan berbeda",
	)
}

// Clean membuang spasi tepi dari seluruh isian teks.
func TestCleanTrimsEveryTextField(t *testing.T) {
	clean := mastermasking.Masking{
		BranchID:  " 100081 ",
		Login:     " BUDI ",
		Module:    " PNCSearchKlaim ",
		SubModule: " Dokumen, ",
		InputBy:   " ADMIN ",
	}.Clean()

	require.Equal(t, "100081", clean.BranchID)
	require.Equal(t, "BUDI", clean.Login)
	require.Equal(t, "PNCSearchKlaim", clean.Module)
	require.Equal(t, "Dokumen,", clean.SubModule)
	require.Equal(t, "ADMIN", clean.InputBy)
}

// Hanya keempat tipe pencarian layar lama yang dikenal; sisanya ditolak.
//
// Keempatnya diambil dari `SearchData.Type` pada `Activity/SearchDataMasking-Act.xml`:
// "1" semua · "2" cabang · "3" login · "4" status aktif.
//
// Tipe yang tidak dikenal TIDAK boleh diperlakukan sebagai "tanpa penyaring": itu akan
// menampilkan seluruh baris kepada pengguna yang mengira ia sedang mencari sesuatu yang
// sempit — dan isinya adalah daftar siapa yang boleh membuka data pribadi.
func TestSearchByKnown(t *testing.T) {
	require.True(t, mastermasking.SearchByAll.Known())
	require.True(t, mastermasking.SearchByBranch.Known())
	require.True(t, mastermasking.SearchByLogin.Known())
	require.True(t, mastermasking.SearchByStatus.Known())
	require.False(t, mastermasking.SearchBy("modul").Known())
	require.False(t, mastermasking.SearchBy("DROP TABLE").Known())
}

// Nilai status pada penyaring hanya sah bila ia salah satu dari kedua status yang ada.
//
// Perbandingannya mengabaikan besar-kecil huruf dan spasi tepi, tetapi TIDAK menerima
// nilai lain — layar lama menyusun `STS_AKTF = '…'` yang cocok persis.
func TestFilterStatusKeyword(t *testing.T) {
	for _, input := range []string{"AKTIF", "aktif", "  Aktif  "} {
		value, valid := mastermasking.Filter{Keyword: input}.StatusKeyword()
		require.True(t, valid, input)
		require.Equal(t, mastermasking.StatusActive, value)
	}

	value, valid := mastermasking.Filter{Keyword: "tidak aktif"}.StatusKeyword()
	require.True(t, valid)
	require.Equal(t, mastermasking.StatusInactive, value)

	for _, input := range []string{"", "ENTAH", "AKTIF SEKALI", "1"} {
		_, valid := mastermasking.Filter{Keyword: input}.StatusKeyword()
		require.False(t, valid, input)
	}
}
