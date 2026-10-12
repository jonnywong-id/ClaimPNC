package masterpanel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

// headerPanel adalah baris header lengkap, dipakai sebagian besar uji.
const headerPanel = "NAME,STS_REPAIR,STS_EDIT_QTY,STS_PREMIUM_REPAIR,STS_PECAH," +
	"STS_STICKER,STS_SISI,STS_RUSAK_PARAH,STS_AKTIF,EXCLUSION_C"

func TestPanelCSVTerbacaLengkap(t *testing.T) {
	rows, err := masterpanel.ParsePanelCSV(strings.NewReader(
		headerPanel + "\nPintu Depan,1,1,0,0,1,1,0,1,C\n"))
	require.NoError(t, err)

	require.Len(t, rows, 1)
	require.Equal(t, 2, rows[0].Line, "header terhitung baris 1")
	require.Equal(t, "Pintu Depan", rows[0].Input.Name)
	require.Equal(t, "1", rows[0].Input.RepairStatus)
	require.Equal(t, "C", rows[0].Input.ExclusionC)
}

// Header tidak peka huruf besar-kecil maupun spasi — Pega pun tidak, dan menolak berkas
// hanya karena kapitalisasi menghasilkan kegagalan yang membingungkan.
func TestHeaderTidakPekaKapitalisasiDanSpasi(t *testing.T) {
	header := " name , sts_repair ,STS_EDIT_QTY,Sts_Premium_Repair,STS_PECAH," +
		"STS_STICKER,STS_SISI,STS_RUSAK_PARAH,STS_AKTIF,EXCLUSION_C"

	rows, err := masterpanel.ParsePanelCSV(strings.NewReader(header + "\nKaca,1,1,1,1,1,1,1,1,0\n"))
	require.NoError(t, err)
	require.Equal(t, "Kaca", rows[0].Input.Name)
	require.Equal(t, "1", rows[0].Input.RepairStatus)
}

// BOM UTF-8 yang ditulis Excel di depan header harus dibuang; tanpa itu kolom pertama
// terbaca `<BOM>NAME` lalu dianggap tidak dikenal, dan berkas yang sah ditolak.
func TestBOMExcelDibuangDariHeader(t *testing.T) {
	// BOM dirangkai dari rune, BUKAN ditulis sebagai karakter di dalam source.
	// Menaruh karakternya langsung membuat kompilator Go menolak berkasnya
	// ("illegal byte order mark") — dan itu sudah dua kali terjadi di sesi ini.
	bom := string(rune(0xFEFF))

	rows, err := masterpanel.ParsePanelCSV(strings.NewReader(
		bom + headerPanel + "\nBumper,1,1,1,1,1,1,1,1,0\n"))
	require.NoError(t, err)
	require.Equal(t, "Bumper", rows[0].Input.Name)
}

func TestHeaderKurangDitolakDenganMenyebutKolomnya(t *testing.T) {
	_, err := masterpanel.ParsePanelCSV(strings.NewReader(
		"NAME,STS_REPAIR\nPintu,1\n"))

	require.ErrorIs(t, err, masterpanel.ErrCSVHeaderMissing)
	require.Contains(t, err.Error(), "STS_PECAH",
		"pengguna harus tahu kolom mana yang kurang, bukan menebak di antara sepuluh")
}

func TestBerkasTanpaBarisDataDitolak(t *testing.T) {
	_, err := masterpanel.ParsePanelCSV(strings.NewReader(headerPanel + "\n"))
	require.ErrorIs(t, err, masterpanel.ErrCSVEmpty)
}

// Baris kosong di ujung berkas lazim — setiap berkas yang diakhiri enter punya satu.
// Melaporkannya sebagai kegagalan membuat berkas yang sah terlihat bermasalah.
func TestBarisKosongDiUjungDilewati(t *testing.T) {
	rows, err := masterpanel.ParsePanelCSV(strings.NewReader(
		headerPanel + "\nPintu,1,1,1,1,1,1,1,1,0\n\n\n"))
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

// Kolom terakhir yang hilang karena nilainya kosong TIDAK menolak barisnya: berkas nyata
// kerap seperti itu, dan yang kurang diisi kosong.
func TestKolomTerakhirHilangDiisiKosong(t *testing.T) {
	rows, err := masterpanel.ParsePanelCSV(strings.NewReader(
		headerPanel + "\nPintu,1,1,1,1,1,1,1,1\n"))
	require.NoError(t, err)
	require.Equal(t, "", rows[0].Input.ExclusionC)
}

func TestBarisMelebihiBatasDitolak(t *testing.T) {
	var berkas strings.Builder
	berkas.WriteString(headerPanel + "\n")
	for i := 0; i <= masterpanel.MaxCSVRows; i++ {
		berkas.WriteString("Panel,1,1,1,1,1,1,1,1,0\n")
	}

	_, err := masterpanel.ParsePanelCSV(strings.NewReader(berkas.String()))
	require.ErrorIs(t, err, masterpanel.ErrCSVTooManyRows)
}

// Berkas bertitik-koma — lahir dari Excel berlokal Indonesia — terbaca sebagai SATU kolom,
// sehingga kegagalannya muncul sebagai header tidak lengkap. Itu pesan yang benar: yang
// salah memang susunan berkasnya.
func TestBerkasBertitikKomaDitolakSebagaiHeaderKurang(t *testing.T) {
	_, err := masterpanel.ParsePanelCSV(strings.NewReader(
		strings.ReplaceAll(headerPanel, ",", ";") + "\nPintu;1;1;1;1;1;1;1;1;0\n"))
	require.ErrorIs(t, err, masterpanel.ErrCSVHeaderMissing)
}

// --- lokasi: kolomnya dibaca dari Activity/PNCUploadLokasiSisiPanel_Act ---
//
// Berkasnya memuat NAME (nama panel), pyLabel (nama lokasi), dan STS_SISI (sisi sebagai
// KATA). Nama `pyLabel` memang ganjil untuk berkas yang disusun manusia — ia properti
// bawaan Pega — tetapi itulah yang dibaca activity-nya.

func TestLokasiCSVMemakaiKolomPega(t *testing.T) {
	rows, err := masterpanel.ParseLocationCSV(strings.NewReader(
		"NAME,pyLabel,STS_SISI\nPintu Depan,KIRI DEPAN,KIRI\n"))
	require.NoError(t, err)

	require.Len(t, rows, 1)
	require.Equal(t, "Pintu Depan", rows[0].PanelName)
	require.Equal(t, "KIRI DEPAN", rows[0].Location.Name)
	require.Equal(t, masterpanel.SideLeft, rows[0].Location.Side)
}

// Sisi datang sebagai KATA, bukan sandi:
// `@If(STS_SISI=="KIRI","1",(@if(STS_SISI=="KANAN","2","-")))`.
func TestSisiLokasiDiterjemahkanDariKata(t *testing.T) {
	for kata, sisi := range map[string]masterpanel.Side{
		"KIRI":  masterpanel.SideLeft,
		"KANAN": masterpanel.SideRight,
		"kiri":  masterpanel.SideLeft,
		"DEPAN": masterpanel.SideNone,
		"":      masterpanel.SideNone,
	} {
		rows, err := masterpanel.ParseLocationCSV(strings.NewReader(
			"NAME,pyLabel,STS_SISI\nPintu,LOKASI," + kata + "\n"))
		require.NoError(t, err)
		require.Equalf(t, sisi, rows[0].Location.Side, "kata %q", kata)
	}
}

// `LOKASI` dan `LOKASI_PANEL` diterima sebagai alias pyLabel — berkas yang disusun tangan
// wajar memakai kata yang terbaca manusia.
func TestAliasKolomLokasiDiterima(t *testing.T) {
	for _, header := range []string{"NAME,LOKASI,STS_SISI", "NAME,LOKASI_PANEL,STS_SISI"} {
		rows, err := masterpanel.ParseLocationCSV(strings.NewReader(
			header + "\nPintu,KIRI DEPAN,KIRI\n"))
		require.NoErrorf(t, err, header)
		require.Equal(t, "KIRI DEPAN", rows[0].Location.Name)
	}
}

// NAME dan ID_PANEL saling menggantikan, sehingga keduanya TIDAK boleh wajib di header.
func TestHeaderLokasiTidakMenuntutKeduaPenunjukPanel(t *testing.T) {
	_, errNama := masterpanel.ParseLocationCSV(strings.NewReader(
		"NAME,pyLabel,STS_SISI\nPintu,KIRI,KIRI\n"))
	_, errID := masterpanel.ParseLocationCSV(strings.NewReader(
		"ID_PANEL,pyLabel,STS_SISI\n01000001,KIRI,KIRI\n"))

	require.NoError(t, errNama)
	require.NoError(t, errID)
}

// --- penerjemahan kata pada berkas MASTER ---

// Delapan kolom berisi KATA, bukan sandi. Meneruskannya apa adanya akan menyimpan teks
// "TIDAK" ke kolom yang seharusnya berisi "0" — tanpa satu pun galat.
func TestKataTidakMenjadiNolSelainnyaMenjadiSatu(t *testing.T) {
	hasil := masterpanel.ApplyCSVWords(masterpanel.Input{
		Name: "kap mesin", ShatterStatus: "TIDAK", StickerStatus: "YA",
		SideStatus: " tidak ", SevereDamageStatus: "", ExclusionC: "TIDAK",
		EditQuantityStatus: "YA", PremiumRepairStatus: "TIDAK",
	})

	require.Equal(t, "0", hasil.ShatterStatus)
	require.Equal(t, "1", hasil.StickerStatus)
	require.Equal(t, "0", hasil.SideStatus, "spasi dan kapitalisasi tidak boleh mengubah hasil")
	require.Equal(t, "1", hasil.SevereDamageStatus, "sel KOSONG pun menjadi 1 — itu perilaku @IF Pega")
	require.Equal(t, "0", hasil.ExclusionC)
	require.Equal(t, "1", hasil.EditQuantityStatus)
	require.Equal(t, "0", hasil.PremiumRepairStatus)
}

// `@IF(x="GANTI","1",(@IF(x="JASA","2","3")))` — tiga keadaan, bukan dua.
func TestStatusRepairPunyaTigaKeadaan(t *testing.T) {
	for kata, sandi := range map[string]string{
		"GANTI": "1", "JASA": "2", "ganti": "1", "PERBAIKAN": "3", "": "3",
	} {
		hasil := masterpanel.ApplyCSVWords(masterpanel.Input{Name: "X", RepairStatus: kata})
		require.Equalf(t, sandi, hasil.RepairStatus, "kata %q", kata)
	}
}

func TestNamaPanelDihurufbesarkan(t *testing.T) {
	hasil := masterpanel.ApplyCSVWords(masterpanel.Input{Name: "  kap mesin  "})
	require.Equal(t, "KAP MESIN", hasil.Name)
}

// STS_AKTIF TIDAK dibaca dari berkas: Pega menyetelnya "1" tanpa satu pun @IF.
// Akibatnya satu baris CSV tidak dapat menonaktifkan panel.
func TestStatusAktifDipaksaSatuDanTidakDibacaDariBerkas(t *testing.T) {
	hasil := masterpanel.ApplyCSVWords(masterpanel.Input{Name: "X", ActiveStatus: "TIDAK"})
	require.Equal(t, masterpanel.ActiveStatusOnUpload, hasil.ActiveStatus)
	require.Equal(t, "1", hasil.ActiveStatus)
}

// Kolom STS_AKTIF yang ADA di berkas pun diabaikan penguraiannya.
func TestKolomStatusAktifDiBerkasDiabaikan(t *testing.T) {
	rows, err := masterpanel.ParsePanelCSV(strings.NewReader(
		headerPanel + "\nPintu,GANTI,YA,YA,TIDAK,YA,TIDAK,YA,TIDAK,YA\n"))
	require.NoError(t, err)
	require.Equal(t, "", rows[0].Input.ActiveStatus, "kolomnya tidak dipetakan ke isian mana pun")
}
