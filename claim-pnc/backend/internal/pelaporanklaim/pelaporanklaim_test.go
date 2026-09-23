package pelaporanklaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/pelaporanklaim"
)

// Tahap diturunkan dari kombinasi penanda, persis seperti sistem lama menurunkannya dari
// PNCCASEID dan STATUSLOCK_1 (`RDB List/BrowseClaimRCV_Aksep-SQL.xml`).
//
// Uji ini mengunci SELURUH kombinasi, termasuk yang tampak mustahil — laporan berhasil
// klaim yang penanda transfernya kosong. Kombinasi itu memang tidak dapat terjadi lewat
// usecase, tetapi baris lama atau perbaikan data manual dapat menghasilkannya, dan
// hasilnya tidak boleh bergantung pada urutan pemeriksaan yang kebetulan.
func TestStageDerivedFromFlags(t *testing.T) {
	cases := []struct {
		name   string
		report pelaporanklaim.ClaimReport
		want   pelaporanklaim.Stage
	}{
		{
			name:   "baru dicatat",
			report: pelaporanklaim.ClaimReport{},
			want:   pelaporanklaim.StageNotTransferred,
		},
		{
			name:   "sudah ditransfer, belum jadi klaim",
			report: pelaporanklaim.ClaimReport{Transferred: true},
			want:   pelaporanklaim.StageNotRegistered,
		},
		{
			name:   "sudah jadi klaim",
			report: pelaporanklaim.ClaimReport{Transferred: true, ClaimNumber: "PNCN.26.0001"},
			want:   pelaporanklaim.StageRegistered,
		},
		{
			name: "klaimnya sudah diakseptasi",
			report: pelaporanklaim.ClaimReport{
				Transferred: true,
				ClaimNumber: "PNCN.26.0001",
				Outcome:     pelaporanklaim.OutcomeAccepted,
			},
			want: pelaporanklaim.StageAccepted,
		},
		{
			name: "klaimnya ditolak",
			report: pelaporanklaim.ClaimReport{
				Transferred: true,
				ClaimNumber: "PNCN.26.0001",
				Outcome:     pelaporanklaim.OutcomeRejected,
			},
			want: pelaporanklaim.StageRejected,
		},
		{
			// Hasil klaim diperiksa LEBIH DULU daripada nomor klaim. Bila urutannya
			// terbalik, laporan yang klaimnya sudah selesai akan tertahan selamanya di
			// "sudah diregistrasi" — karena nomor klaimnya memang tetap terisi.
			name: "hasil klaim menang atas nomor klaim",
			report: pelaporanklaim.ClaimReport{
				ClaimNumber: "PNCN.26.0001",
				Outcome:     pelaporanklaim.OutcomeAccepted,
			},
			want: pelaporanklaim.StageAccepted,
		},
		{
			// Nomor klaim berisi spasi saja bukan nomor klaim. Kolom CHAR di Oracle
			// dikembalikan berisi padding, dan tanpa perapian ini laporan yang belum
			// diregistrasi akan terbaca sudah diregistrasi.
			name:   "nomor klaim berisi spasi tidak dianggap terisi",
			report: pelaporanklaim.ClaimReport{Transferred: true, ClaimNumber: "   "},
			want:   pelaporanklaim.StageNotRegistered,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, c.report.Stage())
		})
	}
}

// Kelima tahap punya label berbahasa Indonesia, dan tidak satu pun jatuh ke nilai mentah.
//
// Layar menampilkan label ini apa adanya; tahap yang labelnya jatuh ke nilai mentah akan
// muncul sebagai "BELUM_TRANSFER" di antarmuka.
func TestEveryStageHasReadableLabel(t *testing.T) {
	for _, stage := range []pelaporanklaim.Stage{
		pelaporanklaim.StageNotTransferred,
		pelaporanklaim.StageNotRegistered,
		pelaporanklaim.StageRegistered,
		pelaporanklaim.StageAccepted,
		pelaporanklaim.StageRejected,
	} {
		require.True(t, stage.Known(), "%s harus dikenal", stage)
		require.NotEqual(t, string(stage), stage.Label(),
			"%s belum punya label; ia akan tampil sebagai nilai mentah di layar", stage)
	}

	require.False(t, pelaporanklaim.Stage("KARANGAN").Known())
}

// Hanya nama pelapor yang wajib.
//
// Ini bukan kelonggaran melainkan sifat pekerjaannya: laporan kerugian datang dengan
// kelengkapan yang berbeda-beda dan harus dapat dicatat SEKARANG lalu dilengkapi kemudian.
// Layar Pega pun tidak mewajibkan satu field pun; kelengkapan yang sesungguhnya ditegakkan
// saat registrasi.
func TestOnlyReporterNameIsRequired(t *testing.T) {
	empty := pelaporanklaim.ClaimReport{}
	violations := empty.Validate()

	require.Len(t, violations, 1, "hanya satu aturan yang boleh melanggar pada laporan kosong")
	require.Equal(t, pelaporanklaim.FieldReporterName, violations[0].Field)

	nameOnly := pelaporanklaim.ClaimReport{ReporterName: "Bagas Prasetya"}
	require.Empty(t, nameOnly.Validate(),
		"laporan yang hanya berisi nama pelapor harus dapat disimpan")
}

// Nama pelapor berisi spasi saja sama dengan kosong.
//
// Tanpa perapian ini, satu spasi cukup untuk melewati satu-satunya aturan wajib yang ada.
func TestWhitespaceReporterNameRejected(t *testing.T) {
	report := pelaporanklaim.ClaimReport{ReporterName: "   \t  "}

	violations := report.Validate()
	require.Len(t, violations, 1)
	require.Equal(t, pelaporanklaim.FieldReporterName, violations[0].Field)
}

// SELURUH pelanggaran dikumpulkan, bukan berhenti pada yang pertama.
//
// Ini kesetaraan perilaku, bukan selera: sistem lama menampilkan semua pesan sekaligus.
// Pada form berisi 20 kolom, mengembalikannya satu per satu berarti pengguna menekan
// Simpan berkali-kali untuk menemukan kesalahan berikutnya.
func TestAllViolationsCollectedAtOnce(t *testing.T) {
	report := pelaporanklaim.ClaimReport{
		ReporterName:   "",
		SenderEmail:    "bukan-surel",
		InsuredEmail:   "juga bukan@",
		EstimatedValue: "seratus juta",
		DocumentCount:  -1,
	}

	violations := report.Validate()
	require.Len(t, violations, 5, "kelima pelanggaran harus dilaporkan bersamaan")

	fields := map[string]bool{}
	for _, v := range violations {
		fields[v.Field] = true
		require.NotEmpty(t, v.Message, "setiap pelanggaran wajib punya pesan yang dapat dibaca")
	}
	require.True(t, fields[pelaporanklaim.FieldReporterName])
	require.True(t, fields[pelaporanklaim.FieldSenderEmail])
	require.True(t, fields[pelaporanklaim.FieldInsuredEmail])
	require.True(t, fields[pelaporanklaim.FieldEstimatedValue])
	require.True(t, fields[pelaporanklaim.FieldDocumentCount])
}

// Panjang dihitung dalam RUNE, bukan byte.
//
// Satu huruf beraksen memakan dua byte. Bila dihitung per byte, batas 100 karakter akan
// terasa berubah-ubah bagi pengguna — nama beraksen ditolak pada 50 huruf, nama biasa
// diterima sampai 100.
func TestLengthCountedInRunesNotBytes(t *testing.T) {
	// 100 huruf beraksen = 200 byte, tetapi tepat 100 rune.
	accented := ""
	for i := 0; i < 100; i++ {
		accented += "é"
	}
	require.Len(t, []byte(accented), 200, "prasyarat uji: masukannya memang 200 byte")

	report := pelaporanklaim.ClaimReport{ReporterName: accented}
	require.Empty(t, report.Validate(), "100 rune harus diterima walau 200 byte")

	report.ReporterName = accented + "é"
	require.Len(t, report.Validate(), 1, "101 rune harus ditolak")
}

// Nilai uang hanya diterima dalam bentuk yang dapat dikirim apa adanya ke basis data.
//
// Pemisah ribuan sengaja DITOLAK, bukan dibuang diam-diam: "1.500" berarti seribu lima
// ratus bagi sebagian orang dan satu koma lima bagi yang lain. Menebaknya berarti salah
// pada separuh kasus, dan salahnya seratus kali lipat.
func TestLooksLikeMoney(t *testing.T) {
	accepted := []string{"0", "1", "1000000", "450000000", "12.5", "12.50", "0.01",
		"1234567890123456"}
	for _, value := range accepted {
		require.True(t, pelaporanklaim.LooksLikeMoney(value), "%q seharusnya diterima", value)
	}

	rejected := []string{
		"",                  // kosong
		"  ",                // spasi
		"1.500.000",         // pemisah ribuan
		"1,5",               // koma sebagai desimal
		"-100",              // negatif
		"+100",              // bertanda
		"12.345",            // tiga angka desimal
		"12.",               // titik menggantung
		".5",                // tanpa bagian bulat
		"1e9",               // notasi ilmiah
		"Rp 100",            // bersimbol
		"12345678901234567", // 17 digit, melewati NUMBER(18,2)
	}
	for _, value := range rejected {
		require.False(t, pelaporanklaim.LooksLikeMoney(value), "%q seharusnya ditolak", value)
	}
}

// Pemeriksaan surel sengaja longgar.
//
// Satu-satunya cara membuktikan sebuah alamat sah adalah mengirim surel ke sana;
// pemeriksaan yang ketat hanya menolak alamat sah yang bentuknya tidak biasa. Yang
// ditangkap hanya salah ketik yang jelas.
func TestLooksLikeEmailCatchesObviousTypos(t *testing.T) {
	accepted := []string{
		"a@b.co",
		"nama.panjang+tag@sub.domain.example",
		"UPPER@CASE.EXAMPLE",
	}
	for _, address := range accepted {
		require.True(t, pelaporanklaim.LooksLikeEmail(address), "%q seharusnya diterima", address)
	}

	rejected := []string{"", "tanpa-at", "@tanpa-lokal.example", "tanpa-domain@",
		"tanpa.titik@domain", "ada spasi@domain.example", "akhiran@domain."}
	for _, address := range rejected {
		require.False(t, pelaporanklaim.LooksLikeEmail(address), "%q seharusnya ditolak", address)
	}
}

// Filter dirapikan ke nilai yang aman dipakai kueri.
//
// Batas yang tidak masuk akal — nol, negatif, atau raksasa — tidak boleh sampai ke basis
// data. Batas raksasa adalah cara termurah membuat satu permintaan menarik seluruh tabel.
func TestFilterNormalizedToSafeValues(t *testing.T) {
	defaults := pelaporanklaim.Filter{}.Normalize()
	require.Equal(t, pelaporanklaim.DefaultLimit, defaults.Limit)
	require.Zero(t, defaults.Offset)

	negative := pelaporanklaim.Filter{Limit: -5, Offset: -3}.Normalize()
	require.Equal(t, pelaporanklaim.DefaultLimit, negative.Limit)
	require.Zero(t, negative.Offset, "lewati negatif akan membuat OFFSET ditolak basis data")

	huge := pelaporanklaim.Filter{Limit: 100000}.Normalize()
	require.Equal(t, pelaporanklaim.MaxLimit, huge.Limit)

	padded := pelaporanklaim.Filter{Search: "  PNCN  ", BranchCode: " 100081 "}.Normalize()
	require.Equal(t, "PNCN", padded.Search)
	require.Equal(t, "100081", padded.BranchCode)
}

// Laporan yang sudah diregistrasi tidak dapat ditransfer lagi, dan laporan yang sudah
// ditransfer tidak dapat ditransfer ulang.
func TestCanTransferOnlyBeforeEitherHappened(t *testing.T) {
	fresh := pelaporanklaim.ClaimReport{}
	require.True(t, fresh.CanTransfer())

	moved := pelaporanklaim.ClaimReport{Transferred: true}
	require.False(t, moved.CanTransfer())

	claimed := pelaporanklaim.ClaimReport{ClaimNumber: "PNCN.26.0001"}
	require.False(t, claimed.CanTransfer(),
		"laporan yang sudah jadi klaim tidak boleh ditransfer, walau penanda transfernya kosong")
}

// Clean membuang spasi tepi dari SELURUH field teks.
//
// Nomor polis yang diketik dengan spasi di ujung tidak akan pernah cocok saat laporan ini
// kelak dicari untuk ditautkan ke klaim.
func TestCleanTrimsEveryTextField(t *testing.T) {
	messy := pelaporanklaim.ClaimReport{
		Number:         "  LPK.26.0001  ",
		ReporterName:   "  Bagas Prasetya ",
		PolicyNumber:   "\tCONTOH-PL-000117\n",
		InsuredName:    " PT Harapan Sentosa ",
		ClaimNumber:    "  PNCN.26.0148  ",
		CreatedBy:      " petugas.penerimaan ",
		EstimatedValue: " 450000000 ",
	}

	clean := messy.Clean()
	require.Equal(t, "LPK.26.0001", clean.Number)
	require.Equal(t, "Bagas Prasetya", clean.ReporterName)
	require.Equal(t, "CONTOH-PL-000117", clean.PolicyNumber)
	require.Equal(t, "PT Harapan Sentosa", clean.InsuredName)
	require.Equal(t, "PNCN.26.0148", clean.ClaimNumber)
	require.Equal(t, "petugas.penerimaan", clean.CreatedBy)
	require.Equal(t, "450000000", clean.EstimatedValue)
}

// Waktu tidak ikut disentuh Clean.
//
// Uji ini ada supaya penambahan field waktu berikutnya tidak diam-diam dirapikan menjadi
// nol oleh Clean yang ditulis ulang tanpa memperhatikannya.
func TestCleanLeavesTimesAlone(t *testing.T) {
	recordedAt := time.Date(2026, time.September, 18, 7, 30, 0, 0, time.UTC)
	lossDate := time.Date(2026, time.September, 9, 0, 0, 0, 0, time.UTC)

	report := pelaporanklaim.ClaimReport{
		ReporterName: " Bagas ",
		CreatedAt:    recordedAt,
		LossDate:     &lossDate,
	}.Clean()

	require.Equal(t, recordedAt, report.CreatedAt)
	require.NotNil(t, report.LossDate)
	require.Equal(t, lossDate, *report.LossDate)
}
