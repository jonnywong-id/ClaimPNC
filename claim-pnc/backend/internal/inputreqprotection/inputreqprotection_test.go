package inputreqprotection_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
)

// Nama uji menyebutkan ATURANNYA, bukan nama fungsinya, sehingga daftar uji terbaca sebagai
// daftar aturan bisnis yang selalu mutakhir.

func TestProteksiTanpaNomorKlaimMasihDapatDisunting(t *testing.T) {
	p := inputreqprotection.Protection{ClaimNumber: ""}
	require.True(t, p.Editable())
}

func TestProteksiYangSudahTertautKlaimTerkunci(t *testing.T) {
	// `Section/InboxReqProtection_Section-Section.xml:8657` menonaktifkan tautan barisnya.
	p := inputreqprotection.Protection{ClaimNumber: "PNCN.26.0007"}
	require.False(t, p.Editable())
}

func TestNomorKlaimBerisiSpasiSajaTidakMenguncibaris(t *testing.T) {
	// Spasi kerap terbawa saat nomor disalin-tempel. Bila ia dianggap terisi, baris
	// terkunci tanpa alasan yang terlihat pengguna.
	p := inputreqprotection.Protection{ClaimNumber: "   "}
	require.True(t, p.Editable())
}

func TestStatusAkseptasiKosongBerartiBelumDiakseptasi(t *testing.T) {
	require.False(t, inputreqprotection.Protection{AcceptStatus: inputreqprotection.AcceptPending}.Accepted())
	require.True(t, inputreqprotection.Protection{AcceptStatus: inputreqprotection.AcceptApproved}.Accepted())
	require.True(t, inputreqprotection.Protection{AcceptStatus: inputreqprotection.AcceptRejected}.Accepted())
}

func TestHanyaTipeDuaYangMasukAntreanPremi(t *testing.T) {
	// `InboxOpenProtection2_RD_collection` menyaring `= "2"`; `InboxOpenProtection2_RD`
	// menyaring `!= "2"`. Tipe yang belum dikenal karena itu masuk NON PREMI, bukan lenyap.
	require.True(t, inputreqprotection.IsPremium("2"))

	for _, other := range []string{"", "1", "3", "7", "8", "99"} {
		require.False(t, inputreqprotection.IsPremium(other), "tipe %q seharusnya NON PREMI", other)
	}
}

func TestNomorProteksiBaruBerformatOPCNTahunUrut(t *testing.T) {
	require.Equal(t, "OPCN.26.0001", inputreqprotection.FormatNumber(2026, 1))
	require.Equal(t, "OPCN.26.0042", inputreqprotection.FormatNumber(2026, 42))
	require.Equal(t, "OPCN.27.0001", inputreqprotection.FormatNumber(2027, 1))
}

func TestNomorUrutDipadatkanNolAgarUrutSebagaiTeks(t *testing.T) {
	// `D-71` butir 2: tanpa pemadatan, ".10" mendahului ".9" saat diurutkan sebagai teks.
	sembilan := inputreqprotection.FormatNumber(2026, 9)
	sepuluh := inputreqprotection.FormatNumber(2026, 10)

	require.Less(t, sembilan, sepuluh,
		"nomor ke-9 harus mendahului ke-10 saat dibandingkan sebagai teks")
}

func TestNomorUrutLimaDigitTidakDipotong(t *testing.T) {
	// Memotongnya akan menerbitkan nomor ganda; kolom yang melebar jauh lebih murah.
	require.Equal(t, "OPCN.26.12345", inputreqprotection.FormatNumber(2026, 12345))
}

func TestNomorWarisanPegaTidakDiakuiSebagaiTerbitanSendiri(t *testing.T) {
	require.True(t, inputreqprotection.IssuedHere("OPCN.26.0001"))
	require.True(t, inputreqprotection.IssuedHere("  opcn.26.0001  "))

	require.False(t, inputreqprotection.IssuedHere("OPC-201"))
	require.False(t, inputreqprotection.IssuedHere(""))
}

// ── Validasi form ────────────────────────────────────────────────────────────────

func draftLengkap() inputreqprotection.Draft {
	return inputreqprotection.Draft{
		PolicyNumber:   "99.001.2026.00000001",
		ClaimNumber:    "PNCN.26.0007",
		ClaimReference: "KLAIM-CONTOH-0007",
		Type:           "1",
		Note:           "Keterangan contoh.",
	}
}

func TestIsianLengkapDiterima(t *testing.T) {
	require.NoError(t, draftLengkap().Validate())
}

func TestEmpatFieldWajibDiperiksaSeluruhnya(t *testing.T) {
	// `Activity/InsertOpenProtectionCase-Act.xml:921` menuntut keempatnya terisi.
	//
	// Yang diuji di sini bukan hanya penolakannya, melainkan bahwa SELURUH pesan datang
	// sekaligus — `11-CROSSCUTTING` §1.2 butir 1 menetapkannya sebagai kesetaraan perilaku.
	err := inputreqprotection.Draft{}.Validate()
	require.Error(t, err)

	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Len(t, v.Errors, 4, "keempat field wajib harus dilaporkan sekaligus, bukan satu per satu")

	fields := map[string]bool{}
	for _, fe := range v.Errors {
		fields[fe.Field] = true
	}
	require.True(t, fields[inputreqprotection.FieldPolicyNumber])
	require.True(t, fields[inputreqprotection.FieldClaimNumber])
	require.True(t, fields[inputreqprotection.FieldType])
	require.True(t, fields[inputreqprotection.FieldNote])
}

func TestNomorKlaimYangBelumDicariDitolakDenganPesanTersendiri(t *testing.T) {
	// `Activity/ValidationInputProtection-Act.xml`: mengetik nomor klaim tidak cukup —
	// klaimnya harus ditemukan. Pesannya disalin apa adanya dari sistem lama.
	d := draftLengkap()
	d.ClaimReference = ""

	err := d.Validate()
	require.Error(t, err)

	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Len(t, v.Errors, 1)
	require.Equal(t, inputreqprotection.FieldClaimNumber, v.Errors[0].Field)
	require.Contains(t, v.Errors[0].Message, "Cari Ulang No Klaim")
}

func TestPerubahanDOLMenuntutTanggalBaru(t *testing.T) {
	d := draftLengkap()
	d.Type = inputreqprotection.TypeChangeLossDate

	err := d.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), inputreqprotection.FieldLossDateAfter)

	// Tanggal sebelumnya TIDAK wajib — ia keadaan lama, yang boleh saja belum tercatat.
	baru := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	d.ChangeDetail.LossDateAfter = &baru
	require.NoError(t, d.Validate())
}

func TestPerubahanCauseOfLossMenuntutKeduaPenyebab(t *testing.T) {
	d := draftLengkap()
	d.Type = inputreqprotection.TypeChangeCauseOfLoss

	err := d.Validate()
	require.Error(t, err)

	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Len(t, v.Errors, 2)

	d.ChangeDetail.CauseOfLossID = "12002"
	d.ChangeDetail.CauseOfLossMasterID = "COL-CONTOH-1"
	require.NoError(t, d.Validate())
}

func TestDetailPerubahanDiabaikanUntukTipeLain(t *testing.T) {
	// Layar tidak menampilkan panelnya untuk tipe lain, sehingga isian yang kebetulan
	// terkirim tidak boleh MENOLAK penyimpanan — pengguna tidak punya cara memperbaikinya.
	d := draftLengkap()
	d.Type = "1"
	d.ChangeDetail.CauseOfLossID = "terbawa"

	require.NoError(t, d.Validate())
}

// TestTitikPadaNomorKlaimTidakDibuang menjaga SELISIH TERENCANA terhadap sistem lama.
//
// `Activity/ValidationInputProtection-Act.xml:288` menjalankan
// `@replaceAll(.CaseID, ".", "")` tanpa syarat. Itu tidak berbahaya bagi nomor `PNC-1865`,
// tetapi terhadap `PNCN.YY.xxxx` (`D-71`) ia menghasilkan `PNCN260001` — tautan ke klaim
// putus TANPA GALAT.
//
// Uji ini akan gagal bila seseorang kelak "menyelaraskan" perilaku itu dengan Pega.
func TestTitikPadaNomorKlaimTidakDibuang(t *testing.T) {
	d := draftLengkap()
	d.ClaimNumber = "  PNCN.26.0007  "

	normalized := d.Normalize()

	require.Equal(t, "PNCN.26.0007", normalized.ClaimNumber,
		"titik pada nomor klaim format D-71 tidak boleh dibuang; lihat komentar Draft.Normalize")
	require.Equal(t, 2, strings.Count(normalized.ClaimNumber, "."))
}

// ── Proteksi ganda ───────────────────────────────────────────────────────────────

func TestHariDitentukanDiZonaWIBBukanUTC(t *testing.T) {
	// Sistem lama memakai `@CurrentDate("dd MMM yyyy","WIB")`. Waktu disimpan UTC, sehingga
	// tanpa konversi seluruh permintaan antara 00.00 dan 07.00 WIB akan dibandingkan
	// terhadap hari sebelumnya.
	wib := time.FixedZone("WIB", 7*60*60)

	// 2026-09-22 22.30 UTC = 2026-09-23 05.30 WIB.
	at := time.Date(2026, time.September, 22, 22, 30, 0, 0, time.UTC)

	key := inputreqprotection.DuplicateKeyFor(draftLengkap(), at, wib)

	year, month, day := key.Day.Date()
	require.Equal(t, 2026, year)
	require.Equal(t, time.September, month)
	require.Equal(t, 23, day, "pukul 05.30 WIB adalah hari berikutnya, bukan hari sebelumnya")
}

func TestKunciGandaMemakaiPolisDanTipeSaja(t *testing.T) {
	// Pesannya sendiri menyebut kuncinya: "no polis dan tipe proteksi yang sama di hari
	// ini". Nomor klaim TIDAK ikut — proteksi ganda dilarang meski klaimnya berbeda.
	d := draftLengkap()
	d.Type = "3"

	key := inputreqprotection.DuplicateKeyFor(d, time.Now(), time.UTC)

	require.Equal(t, "99.001.2026.00000001", key.PolicyNumber)
	require.Equal(t, "3", key.Type)
}
