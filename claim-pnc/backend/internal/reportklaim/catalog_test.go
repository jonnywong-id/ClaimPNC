package reportklaim_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
)

// Layar lama memuat 28 panel. Angka ini dijaga karena ia satu-satunya pemeriksaan yang
// menangkap panel yang hilang saat berkas katalog dipecah per kelompok — panel yang lupa
// disebut di salah satu berkas tidak akan membuat apa pun gagal dikompilasi.
func TestKatalogMemuatKeduapuluhDelapanPanel(t *testing.T) {
	require.Len(t, reportklaim.Catalog(), 28)
}

func TestKodeLaporanTidakBolehKembar(t *testing.T) {
	seen := map[reportklaim.Code]string{}
	for _, r := range reportklaim.Catalog() {
		prev, ada := seen[r.Code]
		require.Falsef(t, ada, "kode %q dipakai dua kali: %q dan %q", r.Code, prev, r.Title)
		seen[r.Code] = r.Title
	}
}

// Urutan harness adalah satu-satunya urutan yang dapat dicocokkan dengan layar lama.
// Bila sebuah panel ada di katalog tetapi tidak di pegaOrder, ia hilang dari urutan itu
// tanpa satu pun tanda.
func TestUrutanPegaMemuatSeluruhPanel(t *testing.T) {
	require.Len(t, reportklaim.CatalogInPegaOrder(), len(reportklaim.Catalog()))
}

func TestUrutanPegaDiawaliTATDanDiakhiriDataKomite(t *testing.T) {
	order := reportklaim.CatalogInPegaOrder()
	require.Equal(t, "REPORT TAT", order[0].Title)
	require.Equal(t, "REPORT DATA KOMITE", order[len(order)-1].Title)
}

// Setiap panel WAJIB punya sekurang-kurangnya satu tombol, dan judul tombolnya tidak
// boleh kosong — kartu tanpa tombol adalah kartu yang tidak dapat dipakai.
func TestSetiapPanelPunyaTombol(t *testing.T) {
	for _, r := range reportklaim.Catalog() {
		require.NotEmptyf(t, r.Actions, "%s tanpa tombol", r.Title)
		for _, a := range r.Actions {
			require.NotEmptyf(t, strings.TrimSpace(a.Label), "%s: tombol tanpa label", r.Title)
		}
	}
}

// Hanya satu panel yang bertombol dua di harness. Bila kelak ada yang kedua, uji ini
// gagal dan memaksa penambahnya menjelaskan panel mana — bukan menambahnya diam-diam.
func TestHanyaPanelDataKomiteYangBertombolDua(t *testing.T) {
	var jamak []string
	for _, r := range reportklaim.Catalog() {
		if len(r.Actions) > 1 {
			jamak = append(jamak, r.Title)
		}
	}
	require.Equal(t, []string{"REPORT DATA KOMITE"}, jamak)
}

// Aksi berkode kosong sah HANYA pada panel bertombol tunggal; panel bertombol dua wajib
// memberi kode pada setiap tombolnya, kalau tidak keduanya tidak dapat dibedakan.
func TestPanelBertombolJamakMemberiKodePadaSetiapTombol(t *testing.T) {
	for _, r := range reportklaim.Catalog() {
		if len(r.Actions) == 1 {
			continue
		}
		for _, a := range r.Actions {
			require.NotEmptyf(t, a.Code, "%s: tombol %q tanpa kode", r.Title, a.Label)
		}
	}
}

func TestAksiTanpaKodeMengembalikanSatuSatunyaTombol(t *testing.T) {
	r, ok := reportklaim.Find(reportklaim.CodePLA)
	require.True(t, ok)

	a, ok := r.Action("")
	require.True(t, ok)
	require.Equal(t, "Export Data PLA", a.Label)
}

// Memilihkan Approve atau Rejected untuk pengguna berarti mengunduh laporan yang
// berbeda isi dari yang diminta.
func TestPanelDataKomiteMenolakAksiTanpaKode(t *testing.T) {
	r, ok := reportklaim.Find(reportklaim.CodeKomite)
	require.True(t, ok)

	_, ok = r.Action("")
	require.False(t, ok)

	approve, ok := r.Action(reportklaim.ActionKomiteApprove)
	require.True(t, ok)
	require.Equal(t, "1", approve.FixedParam["statusapprove"])

	rejected, ok := r.Action(reportklaim.ActionKomiteRejected)
	require.True(t, ok)
	require.Equal(t, "2", rejected.FixedParam["statusapprove"])
}

// # Kenapa susunan kolom diuji sekaku ini
//
// Karena ~700 kolomnya DIEKSTRAK dari export, bukan diketik satu per satu. Kesalahan
// pada pengekstraknya tidak akan terlihat sebagai galat kompilasi — ia akan terlihat
// sebagai berkas CSV yang kolomnya bergeser, dan pergeseran satu kolom pada laporan nilai
// uang adalah kesalahan yang terbaca masuk akal.
func TestSetiapVarianPunyaKolomDanKolomnyaLengkap(t *testing.T) {
	kasus := []reportklaim.Filter{
		{BusinessLine: reportklaim.BusinessLineAll},
		{BusinessLine: reportklaim.BusinessLinePA},
		{BusinessLine: reportklaim.BusinessLineTravel},
		{BusinessLine: reportklaim.BusinessLineNonMBU},
		{BusinessLine: reportklaim.BusinessLineBonding},
		{Detail: true},
	}
	for _, r := range reportklaim.Catalog() {
		for _, f := range kasus {
			cols := r.Columns(f)
			require.NotEmptyf(t, cols, "%s: tanpa kolom untuk penyaring %+v", r.Title, f)
			for i, c := range cols {
				require.NotEmptyf(t, c.Field, "%s: kolom ke-%d tanpa nama properti", r.Title, i+1)
			}
		}
	}
}

// Setiap laporan WAJIB punya susunan penutup — susunan yang berlaku bila tidak ada syarat
// yang cocok. Tanpa itu, satu kombinasi penyaring yang tidak terpikirkan menghasilkan
// berkas TANPA satu kolom pun, dan berkas kosong tidak terbedakan dari "tidak ada data".
func TestSetiapLaporanPunyaSusunanPenutup(t *testing.T) {
	// Penyaring yang sengaja dibuat tidak masuk akal: lini bisnis yang tidak ada di
	// daftar mana pun, dan kotak centang kosong.
	aneh := reportklaim.Filter{BusinessLine: reportklaim.BusinessLine("tidak-ada")}
	for _, r := range reportklaim.Catalog() {
		require.NotEmptyf(t, r.Columns(aneh), "%s: tidak punya susunan penutup", r.Title)
	}
}

// Satu-satunya kolom tanpa judul di seluruh katalog ada di laporan Data AI Klaim, dan ia
// memang begitu di export — 14 properti dengan 13 judul. Uji ini menjaga agar tidak ada
// kolom tanpa judul yang MASUK diam-diam di tempat lain.
func TestHanyaLaporanAIKlaimYangPunyaKolomTanpaJudul(t *testing.T) {
	var tanpaJudul []string
	for _, r := range reportklaim.Catalog() {
		for _, c := range r.Columns(reportklaim.Filter{}) {
			if strings.TrimSpace(c.Header) == "" {
				tanpaJudul = append(tanpaJudul, r.Title+"/"+c.Field)
			}
		}
	}
	require.Equal(t, []string{"REPORT DATA AI KLAIM/District"}, tanpaJudul)
}

// Susunan kolom laporan TAT berbeda menurut lini bisnisnya, dan angkanya dikunci karena
// ketiganya terbaca langsung dari export.
func TestSusunanKolomTATMengikutiLiniBisnis(t *testing.T) {
	r, ok := reportklaim.Find(reportklaim.CodeTAT)
	require.True(t, ok)

	require.Len(t, r.Columns(reportklaim.Filter{BusinessLine: reportklaim.BusinessLinePA}), 52)
	require.Len(t, r.Columns(reportklaim.Filter{BusinessLine: reportklaim.BusinessLineTravel}), 52)
	require.Len(t, r.Columns(reportklaim.Filter{BusinessLine: reportklaim.BusinessLineBonding}), 34)
	require.Len(t, r.Columns(reportklaim.Filter{BusinessLine: reportklaim.BusinessLineNonMBU}), 39)
	require.Len(t, r.Columns(reportklaim.Filter{BusinessLine: reportklaim.BusinessLineAll}), 39)
}

// Kotak centang memilih SUSUNAN KOLOM, bukan baris — lihat FilterDetail.
func TestKotakCentangMemilihSusunanKolomBukanBaris(t *testing.T) {
	for _, tc := range []struct {
		code    reportklaim.Code
		ringkas int
		rinci   int
	}{
		{reportklaim.CodeAkseptasi, 26, 42},
		{reportklaim.CodeOSKomite, 27, 43},
		{reportklaim.CodeOSBelumKomite, 21, 36},
	} {
		r, ok := reportklaim.Find(tc.code)
		require.True(t, ok)
		require.Lenf(t, r.Columns(reportklaim.Filter{Detail: false}), tc.ringkas, "%s ringkas", r.Title)
		require.Lenf(t, r.Columns(reportklaim.Filter{Detail: true}), tc.rinci, "%s rinci", r.Title)
	}
}

// Dua laporan menghasilkan berkas 80-an kolom HANYA untuk Non-MBU; selain itu 11 kolom.
// Yang memastikannya adalah nama kuerinya sendiri, yang berakhiran NONMBU.
func TestSusunanNonMBUJauhLebihLebar(t *testing.T) {
	for _, tc := range []struct {
		code   reportklaim.Code
		nonMBU int
		lain   int
	}{
		{reportklaim.CodeCloseKlaim, 81, 11},
		{reportklaim.CodeTemporaryCloseKlaim, 81, 11},
		{reportklaim.CodeKomite, 80, 11},
	} {
		r, ok := reportklaim.Find(tc.code)
		require.True(t, ok)
		require.Lenf(t, r.Columns(reportklaim.Filter{BusinessLine: reportklaim.BusinessLineNonMBU}), tc.nonMBU, "%s non-mbu", r.Title)
		require.Lenf(t, r.Columns(reportklaim.Filter{BusinessLine: reportklaim.BusinessLinePA}), tc.lain, "%s selain non-mbu", r.Title)
	}
}

// Tiga laporan terhalang, dan ketiganya WAJIB menyebutkan sebab dan penghalangnya. Panel
// terhalang tanpa keterangan hanya memberi tahu pengguna bahwa sesuatu tidak berfungsi.
//
// Ketiganya terhalang oleh sebab yang BERBEDA, dan daftar ini dikunci supaya penambahan
// keempat harus dijelaskan, bukan masuk diam-diam:
//
//	compliance  isinya properti klipboard Pega, bukan kolom basis data   R-16
//	adjuster    kedua Report Definition-nya tidak ada di export          R-16
//	mitra       penyaring BARISNYA menempuh DB Link @ASMD                R-03
func TestLaporanTerhalangMenyebutkanSebabnya(t *testing.T) {
	var terhalang []reportklaim.Code
	for _, r := range reportklaim.Catalog() {
		if r.Availability.Ready {
			continue
		}
		terhalang = append(terhalang, r.Code)
		require.NotEmptyf(t, r.Availability.Reason, "%s: terhalang tanpa sebab", r.Title)
		require.NotEmptyf(t, r.Availability.Blocker, "%s: terhalang tanpa penghalang", r.Title)
	}
	require.ElementsMatch(t, []reportklaim.Code{
		reportklaim.CodeCompliance,
		reportklaim.CodeAdjuster,
		reportklaim.CodeMitra,
	}, terhalang)
}

func TestLookupMembedakanKodeSalahDariLaporanTerhalang(t *testing.T) {
	_, err := reportklaim.Lookup("tidak-ada")
	require.ErrorIs(t, err, reportklaim.ErrUnknownReport)

	_, err = reportklaim.Lookup(reportklaim.CodeAdjuster)
	require.ErrorIs(t, err, reportklaim.ErrReportNotReady)

	r, err := reportklaim.Lookup(reportklaim.CodeTAT)
	require.NoError(t, err)
	require.Equal(t, "REPORT TAT", r.Title)
}

// Setiap laporan yang DAPAT dijalankan wajib menyebut activity asalnya. Tanpa itu,
// pertanyaan "kolom ini dari mana" hanya dapat dijawab dengan menelusuri ulang export.
func TestSetiapLaporanMenyebutActivityAsalnya(t *testing.T) {
	for _, r := range reportklaim.Catalog() {
		require.NotEmptyf(t, r.Source.Activity, "%s tanpa activity asal", r.Title)
	}
}

// Satu-satunya laporan tanpa rule Connect-SQL adalah Adjuster, dan justru itu sebabnya ia
// terhalang: ia memakai Report Definition, dan keduanya tidak ada di export.
func TestHanyaLaporanAdjusterYangTanpaRuleSQL(t *testing.T) {
	var tanpaSQL []reportklaim.Code
	for _, r := range reportklaim.Catalog() {
		if len(r.Source.SQLRule) == 0 {
			tanpaSQL = append(tanpaSQL, r.Code)
		}
	}
	require.Equal(t, []reportklaim.Code{reportklaim.CodeAdjuster}, tanpaSQL)
}

func TestNamaBerkasMenyertakanRentangTanggalHanyaBilaLaporannyaMemakaiTanggal(t *testing.T) {
	dari := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	sampai := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	pakaiTanggal, _ := reportklaim.Find(reportklaim.CodePLA)
	require.Equal(t,
		"Laporan Data PLA 20260901-20260930.csv",
		pakaiTanggal.FileName(reportklaim.Filter{From: dari, To: sampai}),
	)

	// Laporan yang kuerinya tidak menerima tanggal tidak boleh menyandang rentang di
	// nama berkasnya — rentang yang tidak berlaku pada isinya adalah keterangan yang
	// menyesatkan.
	tanpaTanggal, _ := reportklaim.Find(reportklaim.CodeRegistSimasOnline)
	require.Equal(t,
		"Laporan Regist Simas On Line.csv",
		tanpaTanggal.FileName(reportklaim.Filter{From: dari, To: sampai}),
	)
}

func TestValidasiMengumpulkanSeluruhPelanggaranSekaligus(t *testing.T) {
	r, ok := reportklaim.Find(reportklaim.CodeTAT)
	require.True(t, ok)

	v := r.Validate(reportklaim.Filter{})
	require.Len(t, v, 1)
	require.Equal(t, "dari", v[0].Field)
}

func TestValidasiMenolakSampaiYangMendahuluiDari(t *testing.T) {
	r, ok := reportklaim.Find(reportklaim.CodeTAT)
	require.True(t, ok)

	v := r.Validate(reportklaim.Filter{
		From: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	})
	require.Len(t, v, 1)
	require.Equal(t, "sampai", v[0].Field)
}

// Panel Klaim Per Bisnis TIDAK dapat dijalankan tanpa memilih bisnis — kuerinya tidak
// punya cabang "bila kosong", dan hasilnya akan kosong tanpa satu pun tanda.
func TestPanelKlaimPerBisnisMewajibkanPilihanBisnis(t *testing.T) {
	r, ok := reportklaim.Find(reportklaim.CodeKlaimPerBisnis)
	require.True(t, ok)

	v := r.Validate(reportklaim.Filter{})
	require.Len(t, v, 1)
	require.Equal(t, "bisnis", v[0].Field)

	require.Empty(t, r.Validate(reportklaim.Filter{BusinessCode: "10076"}))
}

// Laporan yang kuerinya tidak menerima tanggal tidak boleh MEWAJIBKAN tanggal.
func TestLaporanTanpaPenyaringTanggalTidakMewajibkanTanggal(t *testing.T) {
	r, ok := reportklaim.Find(reportklaim.CodeKomunikasiKlaim)
	require.True(t, ok)
	require.Empty(t, r.Validate(reportklaim.Filter{}))
}

func TestPilihanLiniBisnisMemuatKelimaNilaiExport(t *testing.T) {
	opts := reportklaim.BusinessLineOptions()
	require.Len(t, opts, 5)
	require.Equal(t, "----- Pilih -----", opts[0].Name)

	var nilai []string
	for _, o := range opts {
		nilai = append(nilai, string(o.Value))
	}
	require.Equal(t, []string{"", "002", "005", "346", "003"}, nilai)
}

func TestPilihanLiniBisnisAsingDitolak(t *testing.T) {
	_, ok := reportklaim.ParseBusinessLine("004")
	require.False(t, ok)

	line, ok := reportklaim.ParseBusinessLine("346")
	require.True(t, ok)
	require.Equal(t, reportklaim.BusinessLineNonMBU, line)
}

// Nama berkas masuk ke header Content-Disposition; satu tanda kutip atau baris baru di
// sana membuat header itu dapat disisipi.
func TestNamaBerkasTidakMemuatKarakterBerbahaya(t *testing.T) {
	for _, r := range reportklaim.Catalog() {
		name := r.FileName(reportklaim.Filter{})
		require.NotContainsf(t, name, `"`, "%s", r.Title)
		require.NotContainsf(t, name, "\n", "%s", r.Title)
		require.NotContainsf(t, name, "/", "%s", r.Title)
		require.Truef(t, strings.HasSuffix(name, ".csv"), "%s: %q", r.Title, name)
	}
}
