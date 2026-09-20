package masterrecovery_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
)

// Nama test menyebutkan ATURANNYA, bukan nama fungsinya, supaya daftar test terbaca
// sebagai daftar aturan bisnis yang selalu mutakhir
// (`docs/Steering/14-TESTING-STRATEGY.md` §3.2).

// TestSisaMemakaiPembayaranKetikaTidakAdaPembayaranSebelumnya mengunci cabang pertama
// `HitungSisaKlaimRecovery`.
func TestSisaMemakaiPembayaranKetikaTidakAdaPembayaranSebelumnya(t *testing.T) {
	sisa := masterrecovery.Remainder(160_000, 0, 5_000)
	require.Equal(t, masterrecovery.Amount(155_000), sisa)
}

// TestSisaMengabaikanPembayaranKetikaAdaPembayaranSebelumnya mengunci cabang kedua.
//
// Perilakunya tampak keliru dibaca sekilas — pembayaran batch berjalan tidak mengurangi
// sisa — dan justru karena itu ia diuji: `P-5` menetapkan perilaku dipertahankan lebih
// dulu, dan aturan ini TIDAK ada di daftar 13 perbaikan eksplisit `D-49`. Bila kelak ada
// yang "memperbaikinya" tanpa keputusan tertulis, test inilah yang menghentikannya.
func TestSisaMengabaikanPembayaranKetikaAdaPembayaranSebelumnya(t *testing.T) {
	sisa := masterrecovery.Remainder(160_000, 5_000, 2_000)
	require.Equal(t, masterrecovery.Amount(155_000), sisa)
}

// TestSisaCocokDenganKetigaBarisProduksi menguji rumusnya terhadap data NYATA.
//
// Ketiga baris di bawah dibaca langsung dari POOLDATA.MST_RECOVERY_ASM_PENJAMINAN portal
// ASM pada 2026-09-19. Nilainya bukan data nasabah — ia angka uji yang ditinggalkan
// pengembang sistem lama, dan tidak menyebut satu pun pihak (`D-69`).
//
// Inilah bukti bahwa rumusnya dibaca benar dari activity, bukan ditafsirkan: ketiganya
// cocok tanpa perkecualian.
func TestSisaCocokDenganKetigaBarisProduksi(t *testing.T) {
	for _, baris := range []struct {
		nama       string
		klaim      masterrecovery.Amount
		sebelumnya masterrecovery.Amount
		bayar      masterrecovery.Amount
		sisa       masterrecovery.Amount
	}{
		{"batch 1 — tanpa pembayaran sebelumnya", 160_000, 0, 5_000, 155_000},
		{"batch 2 — ada pembayaran sebelumnya", 160_000, 5_000, 2_000, 155_000},
		{"batch 3 — pembayaran melebihi nilai klaim", 160_000, 7_000, 100_000, 153_000},
	} {
		t.Run(baris.nama, func(t *testing.T) {
			require.Equal(t, baris.sisa,
				masterrecovery.Remainder(baris.klaim, baris.sebelumnya, baris.bayar))
		})
	}
}

// TestSisaBolehNegatif memastikan hasil negatif TIDAK dipotong menjadi nol.
//
// Pembayaran yang melampaui nilai klaim menghasilkan sisa negatif, dan itu keadaan nyata
// yang harus terlihat — memotongnya menjadi nol akan menyembunyikan kelebihan bayar dari
// setiap orang yang membaca baris itu kelak.
func TestSisaBolehNegatif(t *testing.T) {
	require.Equal(t, masterrecovery.Amount(-40_000), masterrecovery.Remainder(10_000, 0, 50_000))
}

// TestIsianWajibDilaporkanSeluruhnyaSekaligus mengunci perilaku yang menggantikan satu
// pesan "Wajib ISI semua field" di sistem lama.
func TestIsianWajibDilaporkanSeluruhnyaSekaligus(t *testing.T) {
	pelanggaran := masterrecovery.CheckRecovery(masterrecovery.Recovery{})

	field := map[string]bool{}
	for _, v := range pelanggaran {
		field[v.Field] = true
	}

	// Keempatnya adalah isian wajib yang berupa teks. Nilai uang tidak ikut karena nol
	// DITERIMA — sistem lama pun menerimanya, dan baris produksi pertama memang bernilai
	// sebelumnya nol.
	require.True(t, field[masterrecovery.FieldPrincipalName], "nama principal wajib")
	require.True(t, field[masterrecovery.FieldYear], "tahun wajib")
	require.True(t, field[masterrecovery.FieldRemark], "keterangan wajib")
	require.True(t, field[masterrecovery.FieldCasePosition], "posisi kasus wajib")
	require.GreaterOrEqual(t, len(pelanggaran), 4, "seluruh pelanggaran dilaporkan sekaligus")
}

// TestNilaiNolDiterima memastikan isian bernilai nol tidak tertolak sebagai "kosong".
//
// Ini kesetaraan perilaku yang halus: prasyarat Pega menguji `== ""`, dan "0" bukan "".
func TestNilaiNolDiterima(t *testing.T) {
	pelanggaran := masterrecovery.CheckRecovery(masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "pengembalian sebagian",
		CasePosition:  "dalam proses",
		ClaimAmount:   0,
		Payment:       0,
	})
	require.Empty(t, pelanggaran)
}

// TestNilaiNegatifDitolak mengunci satu-satunya pengetatan yang disengaja pada validasi.
func TestNilaiNegatifDitolak(t *testing.T) {
	pelanggaran := masterrecovery.CheckRecovery(masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "keterangan",
		CasePosition:  "posisi",
		ClaimAmount:   -1,
	})
	require.Len(t, pelanggaran, 1)
	require.Equal(t, masterrecovery.FieldClaimAmount, pelanggaran[0].Field)
}

// TestIsianTerlaluPanjangDitolakSebelumMenyentuhOracle memastikan batas kolom dijaga di
// sini, bukan dibiarkan menjadi ORA-12899 yang tidak menyebut isian mana.
func TestIsianTerlaluPanjangDitolakSebelumMenyentuhOracle(t *testing.T) {
	pelanggaran := masterrecovery.CheckRecovery(masterrecovery.Recovery{
		PrincipalName: strings.Repeat("A", masterrecovery.MaxPrincipalNameLength+1),
		Year:          "2026",
		Remark:        "keterangan",
		CasePosition:  "posisi",
	})
	require.Len(t, pelanggaran, 1)
	require.Equal(t, masterrecovery.FieldPrincipalName, pelanggaran[0].Field)
}

// TestTahunHarusEmpatAngka mengunci aturan bentuk yang menggantikan dropdown `GetListYear`
// yang hilang dari export.
func TestTahunHarusEmpatAngka(t *testing.T) {
	for _, kasus := range []struct {
		tahun    string
		diterima bool
	}{
		{"2026", true},
		{"2018", true},
		{"", false},
		{"26", false},
		{"20266", false},
		{"dua ribu", false},
		{"1999", false}, // di bawah batas bawah
	} {
		t.Run("tahun "+kasus.tahun, func(t *testing.T) {
			pelanggaran := masterrecovery.CheckYear(kasus.tahun)
			if kasus.diterima {
				require.Empty(t, pelanggaran)
				return
			}
			require.NotEmpty(t, pelanggaran)
			require.Equal(t, masterrecovery.FieldYear, pelanggaran[0].Field)
		})
	}
}

// TestNilaiUangMenerimaPemisahRibuanDanMenolakPecahan mengunci ParseAmount.
func TestNilaiUangMenerimaPemisahRibuanDanMenolakPecahan(t *testing.T) {
	for _, kasus := range []struct {
		teks  string
		nilai masterrecovery.Amount
		galat error
	}{
		{"1000000", 1_000_000, nil},
		{"1.000.000", 1_000_000, nil},
		{"1,000,000", 1_000_000, nil},
		{"  250000  ", 250_000, nil},
		{"-5000", -5_000, nil},
		{"", 0, masterrecovery.ErrAmountEmpty},
		{"seribu", 0, masterrecovery.ErrAmountInvalid},
	} {
		t.Run("nilai "+kasus.teks, func(t *testing.T) {
			nilai, err := masterrecovery.ParseAmount(kasus.teks)
			if kasus.galat != nil {
				require.ErrorIs(t, err, kasus.galat)
				return
			}
			require.NoError(t, err)
			require.Equal(t, kasus.nilai, nilai)
		})
	}
}

// TestPencocokanPrincipalMengabaikanBesarKecilHuruf mengunci peniruan `upper(...)` pada
// ADD_NEWMASTERVIRTUALACCOUNT.
//
// Bila pencocokan di sini lebih ketat daripada di procedure, aplikasi akan menerbitkan VA
// kedua untuk principal yang menurut basis data sudah punya.
func TestPencocokanPrincipalMengabaikanBesarKecilHuruf(t *testing.T) {
	require.Equal(t,
		masterrecovery.PrincipalKey("ABC-001", "PT Contoh Nusantara"),
		masterrecovery.PrincipalKey(" abc-001 ", " pt contoh nusantara "),
	)
}

// TestPrincipalBerbedaTidakTertukar memastikan penyambungan kunci tidak menyatukan dua
// principal yang berbeda hanya karena potongan namanya bersambung.
func TestPrincipalBerbedaTidakTertukar(t *testing.T) {
	require.NotEqual(t,
		masterrecovery.PrincipalKey("AB", "CDEF"),
		masterrecovery.PrincipalKey("ABC", "DEF"),
	)
}

// TestBerkasKlaimMembacaTitikKomaMaupunKoma memastikan berkas ekspor Excel berbahasa
// Indonesia — yang memakai titik koma — tidak ditolak.
func TestBerkasKlaimMembacaTitikKomaMaupunKoma(t *testing.T) {
	for _, kasus := range []struct {
		nama string
		isi  string
	}{
		{"titik koma", "No Polis;Nilai Klaim\nPOL-1;1000\nPOL-2;2000\n"},
		{"koma", "No Polis,Nilai Klaim\nPOL-1,1000\nPOL-2,2000\n"},
	} {
		t.Run(kasus.nama, func(t *testing.T) {
			baris, ditolak, err := masterrecovery.ParseClaimLine(strings.NewReader(kasus.isi))
			require.NoError(t, err)
			require.Empty(t, ditolak)
			require.Len(t, baris, 2)
			require.Equal(t, "POL-1", baris[0].PolicyNo)
			require.Equal(t, masterrecovery.Amount(1_000), baris[0].ClaimAmount)
			require.Equal(t, masterrecovery.Amount(3_000), masterrecovery.TotalClaimAmount(baris))
		})
	}
}

// TestBerkasKlaimTanpaJudulTetapTerbaca memastikan baris judul tidak diwajibkan.
func TestBerkasKlaimTanpaJudulTetapTerbaca(t *testing.T) {
	baris, ditolak, err := masterrecovery.ParseClaimLine(strings.NewReader("POL-1;1000\nPOL-2;2000\n"))
	require.NoError(t, err)
	require.Empty(t, ditolak)
	require.Len(t, baris, 2)
}

// TestBerkasKlaimMembuangBOMExcel mengunci pembuangan tiga bita tak terlihat di awal
// berkas.
//
// Tanpa pembuangan itu, nomor polis pada baris pertama membawa BOM di depannya dan tidak
// akan pernah cocok dengan polis mana pun — kegagalan yang tidak terlihat di layar mana
// pun.
func TestBerkasKlaimMembuangBOMExcel(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	baris, _, err := masterrecovery.ParseClaimLine(strings.NewReader(bom + "POL-1;1000\n"))
	require.NoError(t, err)
	require.Len(t, baris, 1)
	require.Equal(t, "POL-1", baris[0].PolicyNo)
}

// TestBarisKlaimCacatDitolakTerpisahDariYangSah memastikan berkas dengan satu baris cacat
// tetap berguna, dan petugas tahu persis baris mana yang tidak ikut.
func TestBarisKlaimCacatDitolakTerpisahDariYangSah(t *testing.T) {
	isi := "No Polis;Nilai Klaim\nPOL-1;1000\n;5000\nPOL-3;bukanangka\nPOL-4;2000\n"

	baris, ditolak, err := masterrecovery.ParseClaimLine(strings.NewReader(isi))
	require.NoError(t, err)
	require.Len(t, baris, 2, "hanya baris yang sah yang dikembalikan")
	require.Len(t, ditolak, 2, "kedua baris cacat dilaporkan")

	// Nomor barisnya disebut supaya petugas dapat menemukannya di berkas aslinya.
	require.Contains(t, ditolak[0].Message, "Baris 3")
	require.Contains(t, ditolak[1].Message, "Baris 4")
}

// TestBerkasKlaimKosongDibedakanDariBerkasCacat memastikan layar dapat mengatakan hal yang
// benar — keduanya menuntut tindakan berbeda dari petugas.
func TestBerkasKlaimKosongDibedakanDariBerkasCacat(t *testing.T) {
	_, _, err := masterrecovery.ParseClaimLine(strings.NewReader("\n\n   \n"))
	require.ErrorIs(t, err, masterrecovery.ErrClaimLineEmpty)
}

// TestFormatUnggahanDapatDibacaPembacanyaSendiri mengunci lingkaran tertutup antara berkas
// contoh yang DIUNDUH dan pembaca yang MEMBACANYA.
//
// Keduanya berasal dari satu berkas sumber, dan test ini yang memastikan keduanya tidak
// pernah berbeda pendapat — kegagalan yang, bila terjadi, hanya ketahuan dari keluhan
// petugas.
func TestFormatUnggahanDapatDibacaPembacanyaSendiri(t *testing.T) {
	baris, ditolak, err := masterrecovery.ParseClaimLine(strings.NewReader(masterrecovery.ClaimLineTemplate()))
	require.NoError(t, err)
	require.Empty(t, ditolak)
	require.Len(t, baris, 2, "kedua baris contoh terbaca sebagai data, bukan sebagai judul")
}

// TestBuktiBayarKosongDitolak memastikan berkas tanpa isi tidak tersimpan sebagai lampiran
// yang tampak ada tetapi tidak dapat dibuka siapa pun.
func TestBuktiBayarKosongDitolak(t *testing.T) {
	pelanggaran := masterrecovery.CheckDocument(masterrecovery.Document{Name: "bukti.pdf"})
	require.NotEmpty(t, pelanggaran)
	require.Equal(t, masterrecovery.FieldDocument, pelanggaran[0].Field)
}

// TestPanelVAMenuntutKetigaIsiannya mengunci pengetatan terhadap layar lama, yang hanya
// menandai Email sebagai wajib.
//
// Client ID dan Nama Principal keduanya dipakai sebagai KUNCI pencarian VA yang sudah ada;
// salah satu yang kosong membuat pencocokan itu mencocokkan hal yang salah.
func TestPanelVAMenuntutKetigaIsiannya(t *testing.T) {
	pelanggaran := masterrecovery.CheckVirtualAccountRequest(masterrecovery.VirtualAccountRequest{})

	field := map[string]bool{}
	for _, v := range pelanggaran {
		field[v.Field] = true
	}
	require.True(t, field[masterrecovery.FieldClientID])
	require.True(t, field[masterrecovery.FieldPrincipalName])
	require.True(t, field[masterrecovery.FieldEmail])
}

// TestSurelSalahKetikDitolak memastikan pemeriksaan dangkal tetap menangkap salah ketik
// yang nyata, tanpa menolak alamat sah yang tidak umum.
func TestSurelSalahKetikDitolak(t *testing.T) {
	pelanggaran := masterrecovery.CheckVirtualAccountRequest(masterrecovery.VirtualAccountRequest{
		ClientID:      "ABC-001",
		PrincipalName: "PT CONTOH",
		Email:         "petugas.sinarmas.co.id",
	})
	require.Len(t, pelanggaran, 1)
	require.Equal(t, masterrecovery.FieldEmail, pelanggaran[0].Field)
}

// TestPerapianMembuangSpasiTepi memastikan nama principal yang sama tidak tersimpan dalam
// dua bentuk yang kemudian gagal dicocokkan saat VA-nya dicari kembali.
func TestPerapianMembuangSpasiTepi(t *testing.T) {
	bersih := masterrecovery.Recovery{
		PrincipalName: "  PT CONTOH  ",
		ClientID:      " ABC-001 ",
		ClaimLine:     []masterrecovery.ClaimLine{{PolicyNo: " POL-1 "}},
	}.Clean()

	require.Equal(t, "PT CONTOH", bersih.PrincipalName)
	require.Equal(t, "ABC-001", bersih.ClientID)
	require.Equal(t, "POL-1", bersih.ClaimLine[0].PolicyNo)
}
