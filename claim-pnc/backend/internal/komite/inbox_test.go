package komite_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/clock"
)

// Tipe Komite diturunkan dari DUA kolom, dan PAYMENTTYPE hanya bermakna saat
// TYPEKOMITE = 2.
//
// Tabel di bawah disalin apa adanya dari CASE bersarang pada
// `RDB List/ShowKomiteTerimaTolakNonMBU-SQL.xml`. Ia diuji satu per satu karena inilah
// yang dibaca pengguna sebagai "Tipe Komite" — dan di sistem lama ia tersimpan pada
// property bernama `StatusKlaim`, yang sama sekali bukan Status Klaim dalam arti `D-18`.
func TestTipeKomiteDiturunkanDariDuaKolom(t *testing.T) {
	kasus := []struct {
		typeKomite  string
		paymentType string
		harap       string
	}{
		{"1", "", "Survey Komite"},
		{"3", "", "Ex Gratia"},
		{"4", "", "Liable Klaim"},
		{"5", "", "Final"},

		{"2", "1", "Final"},
		{"2", "2", "Interim"},
		{"2", "3", "Salvage"},
		{"2", "4", "Adjuster Fee"},
		{"2", "5", "Adjustment"},
		{"2", "6", "Tolak Klaim"},
		{"2", "7", "Collection Fee"},
	}

	for _, k := range kasus {
		require.Equal(t, k.harap, komite.CommitteeKindOf(k.typeKomite, k.paymentType),
			"TYPEKOMITE=%q PAYMENTTYPE=%q", k.typeKomite, k.paymentType)
	}
}

// Nilai yang tidak dikenali jatuh ke cabang `else` yang sama dengan rule aslinya.
//
// Ditiru, bukan diperbaiki: menebak apa yang SEHARUSNYA terjadi pada tipe yang tidak
// dikenal berarti mengarang aturan yang menentukan apa yang dibaca pengguna.
func TestTipeKomiteYangTidakDikenalMengikutiCabangTerakhirRuleLama(t *testing.T) {
	require.Equal(t, "Survey Komite", komite.CommitteeKindOf("9", "1"))
	require.Equal(t, "Survey Komite", komite.CommitteeKindOf("", ""))
	require.Equal(t, "Collection Fee", komite.CommitteeKindOf("2", "bukan angka"))
}

// PAYMENTTYPE dibandingkan sebagai BILANGAN, bukan sebagai teks.
//
// Rule lama membandingkannya dengan angka telanjang (`=1`), sehingga "01" dan "1" adalah
// hal yang sama di sana. Perbandingan teks akan membuat keduanya berbeda — dan kolomnya
// datang dari sumber warisan yang penulisannya tidak seragam.
func TestPaymentTypeDibandingkanSebagaiBilangan(t *testing.T) {
	require.Equal(t, "Interim", komite.CommitteeKindOf("2", "02"))
	require.Equal(t, "Interim", komite.CommitteeKindOf("2", " 2 "))
}

// Aging dihitung sebagai selisih TANGGAL KALENDER WIB, bukan selisih jam.
//
// Bentuknya ditiru dari
// `TRUNC(SYSDATE) - TO_DATE(TO_CHAR(PXCREATEDATETIME,'dd/mm/yyyy'))`: kasus yang masuk
// kemarin sore dan dibaca pagi ini berumur 1 hari, bukan 0.
func TestAgingDihitungSebagaiSelisihTanggalKalender(t *testing.T) {
	// 20 September 2026 pukul 23:00 WIB — masih tanggal 20 di WIB, tetapi sudah 16:00
	// UTC di hari yang sama.
	masuk := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	kasus := komite.CommitteeCase{CommitteeDate: masuk}

	// Dibaca pukul 08:00 WIB keesokan harinya: selisih jamnya hanya 9, tetapi selisih
	// TANGGALNYA 1. Inilah yang membedakannya dari selisih jam, dan inilah bentuk yang
	// ditiru dari `TRUNC(SYSDATE) - TO_DATE(TO_CHAR(...,'dd/mm/yyyy'))`.
	besokPagi := time.Date(2026, 9, 21, 1, 0, 0, 0, time.UTC)
	require.Equal(t, 1, kasus.AgingDays(besokPagi))

	// Masih tanggal WIB yang sama — 23:30 WIB pada hari yang sama — tetap nol.
	require.Equal(t, 0, kasus.AgingDays(masuk.Add(30*time.Minute)))

	// Dan tepat SATU MENIT sesudah tengah malam WIB sudah menjadi satu hari, bukan nol.
	// Batas itu diperiksa eksplisit: di sinilah perhitungan berbasis selisih jam akan
	// memberi jawaban yang berbeda.
	require.Equal(t, 1, kasus.AgingDays(masuk.Add(time.Hour+time.Minute)))
}

// Tanggal komite di masa depan dilaporkan 0, bukan angka minus.
//
// Ia berarti data yang keliru, dan umur negatif terbaca seperti hitungan mundur menuju
// tenggat — arti yang sama sekali berbeda dari yang sebenarnya terjadi.
func TestAgingTidakPernahNegatif(t *testing.T) {
	besok := clock.AddDays(waktuKeputusan, 2)
	kasus := komite.CommitteeCase{CommitteeDate: besok}

	require.Equal(t, 0, kasus.AgingDays(waktuKeputusan))
}

func TestAgingTanpaTanggalKomiteBernilaiNol(t *testing.T) {
	require.Equal(t, 0, komite.CommitteeCase{}.AgingDays(waktuKeputusan))
}

// Kepemilikan dibandingkan lewat OperatorKey, sehingga besar-kecil huruf dan spasi tepi
// tidak menentukan siapa yang berwenang menyetujui uang.
//
// Ini bukan kelonggaran kosmetik: `OPERATOR_ID` pada `Database/emailkomite.csv` baris
// ID 4 berakhir dengan BARIS BARU, dan tiga nama access group Pega tercatat muncul dalam
// dua kapitalisasi berbeda (`docs/Steering/11-SECURITY.md` §3.1).
func TestKepemilikanTidakBergantungPenulisan(t *testing.T) {
	kasus := komite.CommitteeCase{AssignedOperator: " ellensupriyati\n"}

	require.True(t, kasus.BelongsTo("ELLENSUPRIYATI"))
	require.True(t, kasus.BelongsTo(" ellenSupriyati "))
	require.False(t, kasus.BelongsTo("INDRAGUNAWAN"))
}

// Operator kosong TIDAK pernah dianggap memiliki apa pun.
//
// Sesi yang gagal membawa identitas lalu diam-diam diartikan "cocok dengan apa saja"
// akan membuka seluruh antrean komite bagi siapa pun yang punya token.
func TestOperatorKosongTidakMemilikiApaPun(t *testing.T) {
	require.False(t, komite.CommitteeCase{AssignedOperator: "ELLENSUPRIYATI"}.BelongsTo(""))
	require.False(t, komite.CommitteeCase{AssignedOperator: ""}.BelongsTo(""))
	require.False(t, komite.CommitteeCase{AssignedOperator: ""}.BelongsTo("   "))
}

// Batas halaman ditegakkan, dan permintaan yang lebih besar DIPANGKAS ke batas.
//
// Sistem lama memasang `pyMaxRecords=500` pada 54 dari 56 laporan, dan itu bukan paginasi
// melainkan PEMOTONGAN — hasil ke-501 hilang tanpa satu pun tanda (`T-12`). Di sini yang
// dibatasi adalah ukuran halaman, sementara Total tetap melaporkan seluruh yang cocok.
func TestBatasHalamanDitegakkan(t *testing.T) {
	require.Equal(t, komite.DefaultPageSize,
		komite.InboxFilter{}.Normalize().Limit)

	require.Equal(t, komite.MaxPageSize,
		komite.InboxFilter{Limit: komite.MaxPageSize + 500}.Normalize().Limit)

	require.Equal(t, 10, komite.InboxFilter{Limit: 10}.Normalize().Limit)
	require.Zero(t, komite.InboxFilter{Offset: -5}.Normalize().Offset)
}

// Kotak yang tidak dikenali jatuh ke Outstanding, bukan ke "semua".
//
// Outstanding adalah satu-satunya kotak yang berisi PEKERJAAN, dan itulah yang paling
// masuk akal ditampilkan saat permintaannya tidak jelas.
func TestKotakYangTidakDikenalJatuhKeOutstanding(t *testing.T) {
	require.Equal(t, komite.InboxOutstanding, komite.InboxFilter{}.Normalize().Kind)
	require.Equal(t, komite.InboxOutstanding, komite.InboxFilter{Kind: "entahlah"}.Normalize().Kind)
	require.Equal(t, komite.InboxRejected, komite.InboxFilter{Kind: komite.InboxRejected}.Normalize().Kind)
}

// Operator dinormalkan, TETAPI yang kosong dibiarkan kosong.
//
// Repo-lah yang menerjemahkan kosong menjadi NOL BARIS. Kegagalan yang aman pada sebuah
// inbox adalah menampilkan terlalu sedikit, bukan terlalu banyak.
func TestOperatorDinormalkanDanYangKosongDibiarkanKosong(t *testing.T) {
	require.Equal(t, "ELLENSUPRIYATI",
		komite.InboxFilter{Operator: " ellensupriyati "}.Normalize().Operator)
	require.Empty(t, komite.InboxFilter{Operator: "   "}.Normalize().Operator)
}

// Rentang tanggal terbalik DITOLAK, bukan ditukar diam-diam.
//
// Menukarnya akan menampilkan hasil yang benar untuk pertanyaan yang tidak diajukan
// pengguna, dan ia tidak akan pernah tahu bahwa isiannya keliru.
func TestRentangTanggalTerbalikDitolak(t *testing.T) {
	penyaring := komite.InboxFilter{
		DateFrom: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
		DateTo:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}.Normalize()

	require.True(t, penyaring.DateRangeInverted())

	err := penyaring.Validate()
	var validasi *komite.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violations, 1)
	require.Equal(t, komite.FieldDateTo, validasi.Violations[0].Field)
}

// Rentang sehari — dari dan sampai pada tanggal yang sama — SAH.
//
// Inklusif di kedua ujung, sehingga "cari tanggal 20 saja" berhasil. Batas eksklusif di
// ujung atas adalah kesalahan paling umum pada penyaring tanggal, dan akibatnya adalah
// pekerjaan hari itu tampak tidak ada.
func TestRentangSehariDianggapSah(t *testing.T) {
	sehari := time.Date(2026, 9, 20, 9, 30, 0, 0, time.UTC)
	penyaring := komite.InboxFilter{DateFrom: sehari, DateTo: sehari}.Normalize()

	require.False(t, penyaring.DateRangeInverted())
	require.NoError(t, penyaring.Validate())
	require.Equal(t, penyaring.DateFrom, penyaring.DateTo, "keduanya dipotong ke tanggal WIB yang sama")
}

func TestRingkasanMenjawabJumlahPerKotak(t *testing.T) {
	ringkasan := komite.InboxSummary{Outstanding: 7, Accepted: 3, Rejected: 1}

	require.Equal(t, 7, ringkasan.Count(komite.InboxOutstanding))
	require.Equal(t, 3, ringkasan.Count(komite.InboxAccepted))
	require.Equal(t, 1, ringkasan.Count(komite.InboxRejected))
	require.Zero(t, ringkasan.Count("entahlah"))
}

// Urutan tab mengikuti alur kerja, bukan abjad.
//
// Anggota komite membuka layar ini untuk MENGERJAKAN sesuatu; yang menunggu harus lebih
// dulu.
func TestUrutanKotakMengikutiAlurKerja(t *testing.T) {
	require.Equal(t,
		[]komite.InboxKind{komite.InboxOutstanding, komite.InboxAccepted, komite.InboxRejected},
		komite.InboxKinds(),
	)
}
