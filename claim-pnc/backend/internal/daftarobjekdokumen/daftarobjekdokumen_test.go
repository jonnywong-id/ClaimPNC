package daftarobjekdokumen_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftarobjekdokumen"
)

// Clean memangkas spasi di kedua ujung, dan membuang nama bisnis yang kosong.
//
// Pembuangan itu penting: grid di layar dapat meninggalkan baris yang belum diisi, dan
// menyimpannya berarti pemetaan ke bisnis bernama kosong — baris yang tidak berarti apa-apa
// dan tidak dapat dicabut lagi lewat layar.
func TestCleanMemangkasSpasiDanMembuangBarisKosong(t *testing.T) {
	clean := daftarobjekdokumen.Input{
		Description:   "  KTP Tertanggung  ",
		BusinessNames: []string{"  ANEKA  ", "   ", "", "MARINE CARGO"},
	}.Clean()

	require.Equal(t, "KTP Tertanggung", clean.Description)
	require.Equal(t, []string{"ANEKA", "MARINE CARGO"}, clean.BusinessNames)
}

// Urutan nama bisnis DIPERTAHANKAN apa adanya.
//
// Pengguna menyusun barisnya sendiri di grid, dan mengurutkannya diam-diam akan membuat
// layar menampilkan urutan yang berbeda dari yang baru saja ia simpan.
func TestCleanTidakMengurutkanUlangBisnis(t *testing.T) {
	clean := daftarobjekdokumen.Input{
		BusinessNames: []string{"TRAVEL", "ANEKA", "MARINE CARGO"},
	}.Clean()

	require.Equal(t, []string{"TRAVEL", "ANEKA", "MARINE CARGO"}, clean.BusinessNames)
}

// Keterangan KOSONG tetap sah.
//
// Bukan kelalaian: layar Pega menandai seluruh isiannya `pyRequired=false` dan tidak punya
// satu pun Validate rule untuk kelas ASM-FW-GCNMFW-Int-V_LST_DOC_OBJ. `P-5` menetapkan
// perilaku dipertahankan lebih dulu.
//
// Akibat yang sengaja diterima: objek dokumen tanpa keterangan dapat tersimpan, dan baris
// itu akan muncul sebagai pilihan kosong di layar yang merujuknya.
func TestKeteranganKosongTetapSah(t *testing.T) {
	require.NoError(t, daftarobjekdokumen.Input{}.Clean().Check())
}

// Bisnis KEMBAR tidak ditolak, mengikuti grid Pega yang tidak punya satu pun penanda
// keunikan.
func TestBisnisKembarDiterima(t *testing.T) {
	input := daftarobjekdokumen.Input{
		Description:   "Polis Asli",
		BusinessNames: []string{"ANEKA", "ANEKA"},
	}.Clean()

	require.NoError(t, input.Check())
	require.Len(t, input.BusinessNames, 2)
}

// Batas panjang adalah PENJAGA TEKNIS terhadap lebar kolom, bukan aturan bisnis — tanpa itu
// nilai yang kepanjangan ditolak Oracle dengan ORA-12899 yang sampai ke layar sebagai 500.
func TestKeteranganTerlaluPanjangDitolak(t *testing.T) {
	err := daftarobjekdokumen.Input{
		Description: strings.Repeat("A", daftarobjekdokumen.MaxDescriptionLength+1),
	}.Clean().Check()

	var validationError *daftarobjekdokumen.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, daftarobjekdokumen.FieldDescription, validationError.Violation[0].Field)
}

// Tepat pada batasnya masih diterima. Kasus "tepat di batas" adalah tempat aturan panjang
// paling sering salah satu karakter.
func TestKeteranganTepatDiBatasDiterima(t *testing.T) {
	require.NoError(t, daftarobjekdokumen.Input{
		Description: strings.Repeat("A", daftarobjekdokumen.MaxDescriptionLength),
	}.Clean().Check())
}

// Panjang dihitung dalam RUNE, bukan byte.
//
// Satu huruf beraksen memakan dua byte, dan menghitungnya sebagai byte akan membuat batas
// terasa berubah-ubah bagi pengguna — nama yang sama panjangnya di layar kadang diterima
// kadang ditolak.
func TestPanjangDihitungDalamRuneBukanByte(t *testing.T) {
	require.NoError(t, daftarobjekdokumen.Input{
		Description: strings.Repeat("é", daftarobjekdokumen.MaxDescriptionLength),
	}.Clean().Check())
}

// Nama bisnis yang terlalu panjang ditolak, dan pesannya menyebut nama yang bersangkutan —
// pada grid berisi banyak baris, "ada yang kepanjangan" tanpa menyebut yang mana tidak dapat
// ditindaklanjuti.
func TestNamaBisnisTerlaluPanjangDitolak(t *testing.T) {
	panjang := strings.Repeat("B", daftarobjekdokumen.MaxBusinessNameLength+1)

	err := daftarobjekdokumen.Input{
		BusinessNames: []string{"ANEKA", panjang},
	}.Clean().Check()

	var validationError *daftarobjekdokumen.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Equal(t, daftarobjekdokumen.FieldBusiness, validationError.Violation[0].Field)
	require.Contains(t, validationError.Violation[0].Message, panjang)
}

// SELURUH pelanggaran dikumpulkan sekaligus, bukan yang pertama saja (`P-5`).
//
// Pengguna yang salah pada dua hal harus melihat keduanya dalam satu kali simpan;
// mengembalikan satu per satu berarti ia menekan Simpan berkali-kali hanya untuk menemukan
// kesalahan berikutnya.
func TestSeluruhPelanggaranDikumpulkanSekaligus(t *testing.T) {
	err := daftarobjekdokumen.Input{
		Description:   strings.Repeat("A", daftarobjekdokumen.MaxDescriptionLength+1),
		BusinessNames: []string{strings.Repeat("B", daftarobjekdokumen.MaxBusinessNameLength+1)},
	}.Clean().Check()

	var validationError *daftarobjekdokumen.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 2)
}

// Nama isian yang dilaporkan SAMA dengan nama field JSON, sehingga layar dapat menyorot
// isian yang salah tanpa memetakan apa pun.
//
// Uji ini yang menjaga keduanya tidak berpisah diam-diam: mengganti nama field di `dto.go`
// tanpa mengganti konstanta di sini akan membuat pesan galat menunjuk isian yang tidak ada
// di layar, dan sorotannya hilang tanpa satu pun galat.
func TestNamaIsianSamaDenganFieldJSON(t *testing.T) {
	require.Equal(t, "objek_dokumen", daftarobjekdokumen.FieldDescription)
	require.Equal(t, "bisnis", daftarobjekdokumen.FieldBusiness)
}

// Pencocokan nama bisnis mengabaikan besar-kecil huruf dan spasi tepi.
//
// Ia perlu karena namanya boleh diketik bebas, sehingga "Aneka" dan "ANEKA" pasti terjadi —
// dan tanpa penormalan, yang pertama tidak akan menemukan ID bisnis yang kedua.
func TestNormalizeBusinessName(t *testing.T) {
	require.Equal(t, "ANEKA", daftarobjekdokumen.NormalizeBusinessName("  aneka "))
	require.Equal(t, "FIRE / PROPERTY", daftarobjekdokumen.NormalizeBusinessName("Fire / Property"))
}

// Pesan ErrValidation menyebut seluruh isinya, supaya jejak log dapat dibaca tanpa membuka
// badan responsnya.
func TestValidationErrorMenyebutSeluruhPelanggaran(t *testing.T) {
	err := &daftarobjekdokumen.ValidationError{
		Violation: []daftarobjekdokumen.Violation{
			{Field: "objek_dokumen", Message: "terlalu panjang"},
			{Field: "bisnis", Message: "terlalu panjang"},
		},
	}

	require.Contains(t, err.Error(), "objek_dokumen: terlalu panjang")
	require.Contains(t, err.Error(), "bisnis: terlalu panjang")
}

// ErrNotFound adalah sentinel yang dapat dibandingkan errors.Is — transport bergantung
// padanya untuk menjawab 404, dan menggantinya dengan galat baru akan membuat jawabannya
// diam-diam berubah menjadi 500.
func TestErrNotFoundAdalahSentinel(t *testing.T) {
	require.True(t, errors.Is(daftarobjekdokumen.ErrNotFound, daftarobjekdokumen.ErrNotFound))
	require.Contains(t, daftarobjekdokumen.ErrNotFound.Error(), "daftarobjekdokumen:")
}

// Batas panjang di sini harus SAMA dengan yang dipasang form di frontend
// (`DocumentObjectForm.tsx`). Uji ini tidak dapat membaca berkas TypeScript-nya; yang
// dikerjakannya adalah membuat angkanya terlihat dan memaksa perubahan di sini menjadi
// perubahan yang disadari.
func TestLengthLimitsAreMirroredInTheFrontend(t *testing.T) {
	require.Equal(t, 100, daftarobjekdokumen.MaxDescriptionLength,
		"ubah juga MAX_DESCRIPTION_LENGTH di frontend/src/modules/daftar-objek-dokumen/DocumentObjectForm.tsx")
	require.Equal(t, 100, daftarobjekdokumen.MaxBusinessNameLength,
		"ubah juga MAX_BUSINESS_NAME_LENGTH di frontend/src/modules/daftar-objek-dokumen/DocumentObjectForm.tsx")
}
