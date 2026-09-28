package inputreqprotection_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
)

// Uji di berkas ini menjaga aturan `ID_CLAIM` yang ditegaskan Work Owner 2026-09-26:
//
//	klaim dari Pega     ID_CLAIM = IDPEGA (PZINSKEY)
//	klaim sistem baru   ID_CLAIM = nomor klaim
//
// Keliru di sini TIDAK menghasilkan galat apa pun — hanya proteksi yang menunjuk klaim
// dengan kunci yang tidak dikenali sistem tujuannya.

func TestKlaimPegaMemakaiIDPegaSebagaiClaimReference(t *testing.T) {
	ref := inputreqprotection.ClaimReferenceOf(inputreqprotection.Claim{
		Number: "PNC-1865",
		PegaID: "ASM-FW-GCNMFW-WORK PNC-1865",
	})

	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-1865", ref)
	require.NotEqual(t, "PNC-1865", ref,
		"klaim Pega TIDAK boleh menyimpan nomor klaimnya sendiri sebagai ID_CLAIM")
}

func TestKlaimSistemBaruMemakaiNomorKlaimSebagaiClaimReference(t *testing.T) {
	// Tanpa PegaID: klaim ini tidak ada di tabel kerja Pega.
	ref := inputreqprotection.ClaimReferenceOf(inputreqprotection.Claim{
		Number: "PNCN.26.0007",
	})

	require.Equal(t, "PNCN.26.0007", ref)
}

func TestCabangDitentukanKeberadaanIDPegaBukanBentukNomor(t *testing.T) {
	// Dua baris warisan di produksi TIDAK berpola `PNC-` sama sekali
	// (`kolom-open-protection.md` §1). Menebak cabangnya dari awalan nomor akan gagal pada
	// keduanya, dan gagalnya diam.
	tanpaPola := inputreqprotection.ClaimReferenceOf(inputreqprotection.Claim{
		Number: "KLAIM-WARISAN-TANPA-POLA",
		PegaID: "ASM-FW-GCNMFW-WORK KLAIM-WARISAN-TANPA-POLA",
	})
	require.Equal(t, "ASM-FW-GCNMFW-WORK KLAIM-WARISAN-TANPA-POLA", tanpaPola)

	// Sebaliknya: nomor berawalan `PNC-` TANPA IDPEGA tetap memakai nomornya sendiri.
	// Awalan bukan penentu.
	berawalanTanpaPega := inputreqprotection.ClaimReferenceOf(inputreqprotection.Claim{
		Number: "PNC-9999",
	})
	require.Equal(t, "PNC-9999", berawalanTanpaPega)
}

func TestSpasiDiUjungTidakIkutTersimpan(t *testing.T) {
	// Kolomnya diisi lewat penyalinan data warisan, dan spasi di ujung akan membuat
	// perbandingan ke sistem tujuan gagal tanpa gejala.
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-1865",
		inputreqprotection.ClaimReferenceOf(inputreqprotection.Claim{
			Number: "  PNC-1865  ",
			PegaID: "  ASM-FW-GCNMFW-WORK PNC-1865  ",
		}))

	require.Equal(t, "PNCN.26.0007",
		inputreqprotection.ClaimReferenceOf(inputreqprotection.Claim{
			Number: "  PNCN.26.0007  ",
		}))
}

func TestPegaIDBerisiSpasiSajaDiperlakukanSebagaiKosong(t *testing.T) {
	// Kolom `VARCHAR2` yang "kosong" pada data warisan tidak selalu NULL. Memperlakukan
	// spasi sebagai terisi akan menyimpan ID_CLAIM berisi spasi — yang tidak menunjuk klaim
	// mana pun, dan tidak tampak keliru saat dibaca sekilas.
	require.Equal(t, "PNCN.26.0007",
		inputreqprotection.ClaimReferenceOf(inputreqprotection.Claim{
			Number: "PNCN.26.0007",
			PegaID: "   ",
		}))
}
