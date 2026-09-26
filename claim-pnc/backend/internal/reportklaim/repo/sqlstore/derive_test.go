package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/reportklaim"
)

func wib(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, clock.ZoneWIB)
}

// Usia adalah tahun PENUH pada tanggal kejadian — bukan pembulatan ke tahun terdekat.
func TestUsiaDihitungSebagaiTahunPenuhPadaTanggalKejadian(t *testing.T) {
	lahir := wib(2000, 9, 10)

	// Sehari sebelum ulang tahun ke-26: masih 25.
	require.Equal(t, "25", usiaPadaTanggal(lahir, wib(2026, 9, 9)))
	// Tepat pada ulang tahun: sudah 26.
	require.Equal(t, "26", usiaPadaTanggal(lahir, wib(2026, 9, 10)))
	// Sehari sesudahnya: tetap 26.
	require.Equal(t, "26", usiaPadaTanggal(lahir, wib(2026, 9, 11)))
}

func TestUsiaPadaPergantianBulan(t *testing.T) {
	lahir := wib(2000, 12, 31)
	require.Equal(t, "25", usiaPadaTanggal(lahir, wib(2026, 1, 1)))
	require.Equal(t, "26", usiaPadaTanggal(lahir, wib(2026, 12, 31)))
}

// Sel kosong, bukan "0". Angka nol di kolom usia terbaca sebagai bayi, bukan sebagai
// "tidak diketahui".
func TestUsiaTanpaTanggalMenghasilkanSelKosong(t *testing.T) {
	require.Empty(t, usiaPadaTanggal(time.Time{}, wib(2026, 9, 1)))
	require.Empty(t, usiaPadaTanggal(wib(2000, 9, 1), time.Time{}))
	require.Empty(t, usiaPadaTanggal(wib(2027, 1, 1), wib(2026, 9, 1)), "kejadian mendahului kelahiran")
}

func TestTanggalTeksPadatDisusunUlang(t *testing.T) {
	require.Equal(t, "01/09/2026", tanggalDariTeksPadat("20260901"))
	require.Equal(t, "31/12/2025", tanggalDariTeksPadat("20251231"))
}

// Yang bukan delapan angka dikembalikan sebagai sel kosong — bukan dipaksa menjadi
// tanggal yang tampak sah.
func TestTanggalTeksPadatYangTidakSahMenjadiSelKosong(t *testing.T) {
	for _, tidakSah := range []string{"", "2026", "202609011", "2026AB01", "  "} {
		require.Emptyf(t, tanggalDariTeksPadat(tidakSah), "masukan %q", tidakSah)
	}
}

func TestBulanSingkatTigaHurufBesar(t *testing.T) {
	require.Equal(t, "SEP", bulanSingkat(wib(2026, 9, 1)))
	require.Equal(t, "JAN", bulanSingkat(wib(2026, 1, 31)))
	require.Empty(t, bulanSingkat(time.Time{}))
}

func TestDesimalKoma(t *testing.T) {
	require.Equal(t, "1250,75", desimalKoma("1250.75"))
	require.Empty(t, desimalKoma(""))
}

func TestBacaTanggalCSVBolakBalik(t *testing.T) {
	asal := wib(2026, 9, 8)
	require.Equal(t, asal, bacaTanggalCSV(asal.Format(tanggalCSV)))
	require.True(t, bacaTanggalCSV("").IsZero())
	require.True(t, bacaTanggalCSV("bukan tanggal").IsZero())
}

// Kolom bantu TIDAK boleh ikut ke berkas: ia bahan perhitungan, bukan isi laporan.
func TestKolomBantuDibuangSetelahDipakai(t *testing.T) {
	row := reportklaim.Row{
		"BulanCloseTanggal": "08/09/2026",
		"Resources":         "1250.75",
		"OwnRisk":           "12.5",
	}
	rejectDerive(row)

	require.Equal(t, "SEP", row["FlagASO"])
	require.Equal(t, "1250,75", row["Resources"])
	require.Equal(t, "12,5", row["OwnRisk"])

	_, masihAda := row["BulanCloseTanggal"]
	require.False(t, masihAda, "kolom bantu ikut ke berkas")
}

// Periode Produksi Klaim PA mengisi KELIMA kolom "Tahun" dengan nilai yang sama —
// satu per kelompok risiko, persis bentuk berkas lama.
func TestPeriodeProduksiPAMengisiKelimaKolom(t *testing.T) {
	row := reportklaim.Row{"TahunPeriode": "2026", "BulanPeriode": "9"}
	derivePeriodeProduksiPA(row)

	for _, field := range periodeProduksiPA {
		require.Equalf(t, "2026-09", row[field], "kolom %s", field)
	}
	_, masihAda := row["TahunPeriode"]
	require.False(t, masihAda)
}

// Kalender libur yang tidak tersedia membuat keempat kolom "Lama proses" menjadi SEL
// KOSONG — bukan angka yang dihitung tanpa hari libur, yang akan lebih besar dari yang
// sebenarnya tanpa satu pun tanda di berkas.
func TestKolomLamaProsesKosongSaatKalenderTidakTersedia(t *testing.T) {
	repo := NewRepo(nil, nil) // tanpa koneksi kedua
	derive, err := tatDerive(t.Context(), repo, reportklaim.Filter{
		From: wib(2026, 9, 1),
		To:   wib(2026, 9, 30),
	})
	require.NoError(t, err)

	row := reportklaim.Row{
		"City":                     "01/09/2026",
		"NewNoKTP":                 "08/09/2026",
		"NewEmail":                 "10/09/2026",
		"CountryID":                "03/09/2026",
		"TanggalBayar":             "11/09/2026",
		"TanggalTerimaDokumenTeks": "20260901",
		"TanggalLaporanTeks":       "20260902",
		"DOB":                      "10/09/2000",
		"DaftarObjek":              "01/09/2026",
	}
	derive(row)

	for _, kolom := range tatWorkingDayColumn {
		require.Emptyf(t, row[kolom.field], "kolom %q (%s)", kolom.field, kolom.judul)
	}

	// Yang TIDAK bergantung pada kalender tetap terisi.
	require.Equal(t, "01/09/2026", row["CommentKomiteClosecase"])
	require.Equal(t, "02/09/2026", row["CASEDB"])
	require.Equal(t, "25", row["USIA"])

	for _, bantu := range tatHelperColumn {
		_, masihAda := row[bantu]
		require.Falsef(t, masihAda, "kolom bantu %q ikut ke berkas", bantu)
	}
}

// Keempat kolom "Lama proses" memakai PASANGAN TANGGAL yang berbeda-beda. Uji ini
// mengunci pasangannya: tertukar satu pasang berarti kolom "Lama aksep - trf kasir"
// menampilkan lama registrasi, dan angkanya tetap terlihat masuk akal.
func TestPasanganTanggalKolomLamaProsesTidakTertukar(t *testing.T) {
	pasangan := map[string][2]string{}
	for _, k := range tatWorkingDayColumn {
		pasangan[k.field] = [2]string{k.dari, k.sampai}
	}

	require.Equal(t, [2]string{"City", "NewNoKTP"}, pasangan["CloseClaimNote"],
		"Lama proses regis-tf ke teknik: registrasi -> transfer PIC")
	require.Equal(t, [2]string{"City", "NewEmail"}, pasangan["CityID"],
		"Lama regis - tanggal trf kasir: registrasi -> transfer kasir")
	require.Equal(t, [2]string{"CountryID", "NewEmail"}, pasangan["OccupationCode"],
		"Lama aksep - trf kasir: akseptasi -> transfer kasir")
	require.Equal(t, [2]string{"CountryID", "TanggalBayar"}, pasangan["Conveyance"],
		"Lama akseptasi - tanggal bayar: akseptasi -> bayar")
}

// Hanya SATU dari empat kolom yang menampilkan -1 apa adanya; tiga lainnya memetakannya
// menjadi "0" mengikuti `@If(Param.CountBusiness=="-1","0",…)` di export.
func TestHanyaSatuKolomLamaProsesYangTidakDinolkan(t *testing.T) {
	var tanpaNol []string
	for _, k := range tatWorkingDayColumn {
		if !k.nolkanBilaTakDiketahui {
			tanpaNol = append(tanpaNol, k.field)
		}
	}
	require.Equal(t, []string{"CityID"}, tanpaNol)
}
