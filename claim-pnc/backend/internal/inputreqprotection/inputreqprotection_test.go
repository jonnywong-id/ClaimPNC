package inputreqprotection_test

import (
	"errors"
	"reflect"
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
		ClaimNumber: "PNCN.26.0007",
		Type:        "1",
		Note:        "Keterangan contoh.",
	}
}

func TestIsianLengkapDiterima(t *testing.T) {
	require.NoError(t, draftLengkap().Validate())
}

// TestFieldWajibDiperiksaSeluruhnya menjaga dua hal sekaligus.
//
// Pertama, SELURUH pesan datang bersamaan — `11-CROSSCUTTING` §1.2 butir 1 menetapkannya
// sebagai kesetaraan perilaku dengan Pega, yang menampilkan semuanya sekaligus.
//
// Kedua, yang wajib diisi PENGGUNA tinggal TIGA, bukan empat.
// `Activity/InsertOpenProtectionCase-Act.xml:921` memang menuntut empat, tetapi keempatnya
// termasuk `.PolicyNo` — yang diisi `Activity/OpenProtection-Act.xml` dari klaim, bukan
// diketik. Syarat itu terpenuhi oleh ditemukannya klaim, bukan oleh isian.
func TestFieldWajibDiperiksaSeluruhnya(t *testing.T) {
	err := inputreqprotection.Draft{}.Validate()
	require.Error(t, err)

	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Len(t, v.Errors, 3, "seluruh field wajib harus dilaporkan sekaligus, bukan satu per satu")

	fields := map[string]bool{}
	for _, fe := range v.Errors {
		fields[fe.Field] = true
	}
	require.False(t, fields[inputreqprotection.FieldPolicyNumber],
		"No Polis diturunkan dari klaim; ia tidak boleh dituntut sebagai isian")
	require.True(t, fields[inputreqprotection.FieldClaimNumber])
	require.True(t, fields[inputreqprotection.FieldType])
	require.True(t, fields[inputreqprotection.FieldNote])
}

// TestNomorKlaimTerlaluPanjangDitolakSebelumMenyentuhOracle menjaga pesan, bukan data.
//
// Tanpa pemeriksaan ini, nomor yang melebihi lebar kolom ditolak Oracle dengan `ORA-12899`:
// kalimat berbahasa Inggris yang menyebut nama kolom basis data, muncul SETELAH tombol
// simpan ditekan, dan tidak menunjuk field mana pun di layar.
func TestNomorKlaimTerlaluPanjangDitolakSebelumMenyentuhOracle(t *testing.T) {
	d := draftLengkap()
	d.ClaimNumber = strings.Repeat("X", inputreqprotection.MaxClaimNumberLength+1)

	err := d.Validate()
	require.Error(t, err)

	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Len(t, v.Errors, 1)
	require.Equal(t, inputreqprotection.FieldClaimNumber, v.Errors[0].Field)
	require.Contains(t, v.Errors[0].Message, "terlalu panjang")
}

// TestDraftTidakMenerimaClaimIDTerpisah menjaga keputusan 2026-09-24.
//
// Work Owner menegaskan ClaimNo dan ClaimID berisi nilai yang sama. Menanyakannya dua kali
// akan membuat keduanya berbeda cepat atau lambat — tanpa galat, hanya proteksi yang
// menunjuk dua klaim berbeda.
//
// Uji ini memakai refleksi karena yang dijaga adalah KETIADAAN sebuah field: menambahkannya
// kembali akan lolos setiap uji perilaku, dan baru terlihat sebagai data yang tidak cocok.
func TestDraftTidakMenerimaClaimIDTerpisah(t *testing.T) {
	_, ada := reflect.TypeOf(inputreqprotection.Draft{}).FieldByName("ClaimReference")
	require.False(t, ada,
		"ClaimID diturunkan dari ClaimNumber; ia tidak boleh menjadi isian tersendiri")
}

func TestPerubahanDOLMenuntutTanggalBaru(t *testing.T) {
	d := draftLengkap()
	d.Type = inputreqprotection.TypeChangeLossDate

	err := d.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), inputreqprotection.FieldLossDateAfter)

	// Tanggal sebelumnya TIDAK wajib — ia keadaan lama, yang boleh saja belum tercatat.
	baru := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	d.Change.LossDateAfter = &baru
	require.NoError(t, d.Validate())
}

// TestPerubahanCauseOfLossMenuntutNextCauseOfLossSaja menjaga koreksi 2026-09-24.
//
// Semula KEDUA penyebab kerugian wajib diisi pengguna. Itu keliru: "Cause Of Loss Dipilih"
// DITURUNKAN dari klaim (`Activity/OpenProtection-Act.xml`), sehingga memvalidasinya di sini
// akan menolak isian yang pengguna tidak punya cara memperbaikinya.
func TestPerubahanCauseOfLossMenuntutNextCauseOfLossSaja(t *testing.T) {
	d := draftLengkap()
	d.Type = inputreqprotection.TypeChangeCauseOfLoss

	err := d.Validate()
	require.Error(t, err)

	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Len(t, v.Errors, 1, "hanya Next Cause Of Loss yang diisi pengguna")

	d.Change.CauseOfLossAfter = "COL-CONTOH-1"
	require.NoError(t, d.Validate())
}

func TestDetailPerubahanDiabaikanUntukTipeLain(t *testing.T) {
	// Layar tidak menampilkan panelnya untuk tipe lain, sehingga isian yang kebetulan
	// terkirim tidak boleh MENOLAK penyimpanan — pengguna tidak punya cara memperbaikinya.
	d := draftLengkap()
	d.Type = "1"
	d.Change.CauseOfLossAfter = "terbawa"

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

	key := inputreqprotection.DuplicateKeyFor("99.001.2026.00000001", draftLengkap(), at, wib)

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

	key := inputreqprotection.DuplicateKeyFor("99.001.2026.00000001", d, time.Now(), time.UTC)

	require.Equal(t, "99.001.2026.00000001", key.PolicyNumber)
	require.Equal(t, "3", key.Type)
}
