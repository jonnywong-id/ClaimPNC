package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestSeluruhKueriYangDipakaiAda(t *testing.T) {
	dipakai := []string{
		"ambang_komite_daftar",
		"ambang_komite_periksa_tabel",

		"inbox_list",
		"inbox_count",
		"inbox_summary",
		"inbox_get",
		"inbox_check_table",

		"decision_list_for_cases",
		"decision_insert",
		"decision_check_table",
	}
	for _, nama := range dipakai {
		t.Run(nama, func(t *testing.T) {
			require.NotPanics(t, func() { _ = query(nama) })
			require.NotEmpty(t, strings.TrimSpace(query(nama)))
		})
	}
}

func TestKueriYangTidakAdaMenimbulkanPanik(t *testing.T) {
	require.Panics(t, func() { _ = query("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestKueriMematuhiDisiplinSQLPortabel(t *testing.T) {
	terlarang := []struct {
		pola   string
		alasan string
	}{
		{"SELECT *", "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku"},
		{"NVL(", "pakai COALESCE"},
		{"SYSDATE", "pakai CURRENT_TIMESTAMP"},
		{"DECODE(", "pakai CASE WHEN"},
		{"ROWNUM", "pakai OFFSET ... FETCH NEXT ... ROWS ONLY"},
		{"INSTR(", "pakai POSITION"},
		{"LISTAGG(", "pakai STRING_AGG"},
		{"TO_CHAR(", "pemformatan tanggal dan angka dilakukan di Go"},
		{"LPAD(", "pemformatan angka dilakukan di Go"},
		{"FROM DUAL", "tidak ada urutan maupun ekspresi tanpa tabel di modul ini"},
		// Pengacakan pada jalur Simasnet DIBAWA — Work Owner menegaskan 2026-09-18 bahwa
		// ia disengaja: ia menyebar beban di antara beberapa orang yang berwenang pada
		// tingkat yang sama, sekaligus mengecualikan orang yang mengajukan.
		//
		// Yang dilarang di sini bukan perilakunya, melainkan TEMPATNYA. Diacak di dalam
		// SQL membuat aturannya tidak dapat diuji sama sekali — hasil yang berbeda tiap
		// kali dijalankan tidak dapat dibandingkan dengan apa pun. Pengacakannya karena
		// itu pindah ke Go, di balik seam `komite.Randomizer`, sehingga pengujian memakai
		// pemilih tetap sementara produksi tetap mengacak.
		{"DBMS_RANDOM", "pengacakan ada di Go di balik seam, bukan di dalam SQL"},
	}

	for nama, teks := range queries {
		hurufBesar := strings.ToUpper(teks)
		for _, larangan := range terlarang {
			require.NotContainsf(t, hurufBesar, larangan.pola,
				"kueri %q memakai %q — %s", nama, larangan.pola, larangan.alasan)
		}
		require.NotContainsf(t, hurufBesar, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", nama)
	}
}

// kueriYangBolehMenulis adalah SATU-SATUNYA pengecualian atas aturan baca-saja.
//
// Sejak Inbox Komite ada, paket ini memuat satu pernyataan tulis: `decision_insert`. Ia
// diizinkan karena sasarannya `POOLDATA.CPNC_KOMITE_KEPUTUSAN` — tabel yang migrasi
// `0004` buat dan yang tidak disentuh sistem lain sama sekali.
//
// Daftarnya ditulis SATU PER SATU, bukan sebagai pola nama. Kueri tulis baru karena itu
// gagal di sini lebih dulu, dan pengarangnya harus menyatakan sasarannya secara sadar
// alih-alih lolos karena namanya kebetulan cocok.
var kueriYangBolehMenulis = map[string]string{
	"decision_insert": "POOLDATA.CPNC_KOMITE_KEPUTUSAN",
}

// Tabel WARISAN dibaca saja. Tidak satu pun kueri di luar daftar pengecualian di atas
// boleh mengubah isinya.
//
// Uji ini adalah penegak keputusan Work Owner 2026-09-17 dan `P-1`: POOLDATA.EMAILKOMITE
// masih ditulis Pega dan dibaca 17 kueri di sana, dan tabel Inbox Komite masih ditulis
// Pega seluruhnya. Memindahkan kepemilikannya menuntut prosedur `D-63`, dan sampai itu
// ditempuh, satu pernyataan tulis yang lolos ke sini akan membuat dua sistem menulis
// tabel yang sama — kelas kerusakan data yang hampir mustahil dilacak.
func TestTidakAdaKueriYangMenulis(t *testing.T) {
	menulis := []string{"INSERT", "UPDATE", "DELETE", "MERGE", "TRUNCATE", "DROP ", "ALTER "}

	for nama, teks := range queries {
		if _, boleh := kueriYangBolehMenulis[nama]; boleh {
			continue
		}
		hurufBesar := strings.ToUpper(teks)
		for _, pola := range menulis {
			require.NotContainsf(t, hurufBesar, pola,
				"kueri %q memakai %q — tabel warisan dibaca saja", nama, pola)
		}
	}
}

// Kueri yang boleh menulis hanya boleh menulis ke tabel MILIK APLIKASI INI.
//
// Ini penegak `P-1` yang sesungguhnya. Satu INSERT yang lolos ke tabel warisan akan
// membuat dua sistem menulis tabel yang sama dengan aturan validasi yang berbeda — kelas
// kerusakan data yang `07-MIGRATION-STRATEGY.md` sebut hampir mustahil dilacak.
func TestKueriTulisHanyaMenyentuhTabelMilikSendiri(t *testing.T) {
	for nama, sasaran := range kueriYangBolehMenulis {
		t.Run(nama, func(t *testing.T) {
			hurufBesar := strings.ToUpper(query(nama))
			require.Containsf(t, hurufBesar, sasaran,
				"kueri tulis %q tidak menyentuh %s", nama, sasaran)

			for _, warisan := range []string{
				"DATAPEGA.", "POOLDATA.EMAILKOMITE", "POOLDATA.T_CLAIM_KOMITE_LIST",
				"POOLDATA.T_CLAIM_PNC", "POOLDATA.T_CLAIM_DATA_RESULTS_AI",
				"POOLDATA.PEGA_DASHBOARDPNC",
			} {
				require.NotContainsf(t, hurufBesar, warisan,
					"kueri tulis %q menyentuh tabel warisan %s — `P-1` melarangnya", nama, warisan)
			}
		})
	}
}

// Kueri master ambang hanya menyentuh EMAILKOMITE. Menyentuh tabel lain berarti ia
// mengambil lingkup yang bukan miliknya.
//
// Dibatasi pada kueri berawalan `ambang_komite_` sejak Inbox Komite ada: inbox membaca
// tabel yang berbeda seluruhnya, dan menuntutnya menyentuh EMAILKOMITE tidak masuk akal.
func TestKueriHanyaMenyentuhTabelAmbang(t *testing.T) {
	for nama, teks := range queries {
		if !strings.HasPrefix(nama, "ambang_komite_") {
			continue
		}
		require.Containsf(t, strings.ToUpper(teks), "POOLDATA.EMAILKOMITE",
			"kueri %q tidak menyentuh tabel ambang", nama)
	}
}

// Kueri Inbox Komite tidak boleh menyentuh master ambang.
//
// Batas kepemilikannya nyata: tangga ambangnya milik `F-4`, cara membacanya milik `B-7`,
// dan inbox tidak membaca keduanya. Kueri inbox yang menyentuh EMAILKOMITE berarti
// seseorang mulai menurunkan jumlah jenjang di dalam SQL — persis tebakan yang
// `CommitteeCase.TierCount` jelaskan kenapa tidak boleh diambil.
func TestKueriInboxTidakMenyentuhMasterAmbang(t *testing.T) {
	for nama, teks := range queries {
		if !strings.HasPrefix(nama, "inbox_") {
			continue
		}
		require.NotContainsf(t, strings.ToUpper(teks), "POOLDATA.EMAILKOMITE",
			"kueri %q menyentuh master ambang", nama)
	}
}

// Penyaring pemilik WAJIB ada di setiap kueri inbox yang menghasilkan daftar.
//
// Inbox adalah daftar pekerjaan SESEORANG. Kueri daftar tanpa penyaring pemilik akan
// mengembalikan seluruh antrean komite perusahaan — beserta nilai klaim dan nama
// tertanggung — kepada siapa pun yang punya sesi.
//
// `inbox_get` dikecualikan dengan sengaja: ia mengambil SATU kasus tanpa memandang
// pemiliknya, dan pemeriksaan kepemilikan dikerjakan lapisan usecase lewat BelongsTo
// supaya "tidak ada" dan "bukan milik Anda" dapat dibedakan di log.
func TestKueriDaftarInboxSelaluMenyaringPemilik(t *testing.T) {
	daftar := []string{"inbox_list", "inbox_count", "inbox_summary"}

	for _, nama := range daftar {
		t.Run(nama, func(t *testing.T) {
			rapat := bersihkanSpasi(query(nama))
			require.Containsf(t, rapat, "w.PXASSIGNEDOPERATORID = :1",
				"kueri %q tidak menyaring pemilik", nama)

			// INNER, bukan LEFT. `InboxRegisterKomite_RD` memakai INNER JOIN ke
			// `Assign-Worklist`, dan LEFT JOIN di sini akan meloloskan case yang TIDAK
			// ditugaskan kepada siapa pun ke dalam inbox seseorang.
			require.Containsf(t, rapat, "JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK a",
				"kueri %q tidak menyambung penugasan ke case secara INNER", nama)
			require.NotContainsf(t, rapat, "LEFT JOIN DATAPEGA.PC_ASSIGN_WORKLIST",
				"kueri %q menyambung worklist secara LEFT; RD memakai INNER", nama)
		})
	}
}

// Penyaring inbox_count WAJIB sama persis dengan inbox_list.
//
// Bila keduanya berbeda, layar menampilkan jumlah halaman yang tidak pernah ada isinya —
// dan pengguna melaporkan pekerjaan yang hilang. Yang dibandingkan adalah klausa
// penyaringnya, bukan seluruh kueri: daftarnya memuat kolom dan paginasi yang memang
// tidak ada pada penghitungnya.
func TestPenyaringDaftarDanPenghitungSama(t *testing.T) {
	potong := func(teks string) string {
		atas := strings.ToUpper(teks)
		mulai := strings.LastIndex(atas, "WHERE (CASE")
		require.GreaterOrEqual(t, mulai, 0, "klausa penyaring tidak ditemukan")

		akhir := strings.Index(atas[mulai:], "ORDER BY")
		if akhir < 0 {
			return atas[mulai:]
		}
		return atas[mulai : mulai+akhir]
	}

	require.Equal(t,
		bersihkanSpasi(potong(query("inbox_count"))),
		bersihkanSpasi(potong(query("inbox_list"))),
		"penyaring inbox_list dan inbox_count berbeda")
}

// Tidak satu pun kueri daftar boleh menyentuh POOLDATA.CPNC_KOMITE_KEPUTUSAN.
//
// Work Owner menetapkan 2026-09-28 bahwa data komite dimunculkan `InboxRegisterKomite_RD`
// dan `SetDataKomitePNC_Act`, dan tabel itu tidak dipakai. Ia juga tidak pernah ada di
// Pega — ia rancangan aplikasi ini sendiri untuk MENULIS keputusan.
//
// Uji ini menjaga keputusan itu tetap berlaku. Menggabungkannya kembali ke kueri daftar
// akan mengulang kegagalan 2026-09-28 persis: migrasi `0004` belum dijalankan di
// lingkungan mana pun, sehingga satu join saja mematikan SELURUH layar dengan `ORA-00942`
// — padahal 1.542 kasus di tabel warisan baik-baik saja.
func TestKueriDaftarTidakMenyentuhTabelKeputusan(t *testing.T) {
	for nama, teks := range queries {
		if !strings.HasPrefix(nama, "inbox_") {
			continue
		}
		require.NotContainsf(t, strings.ToUpper(teks), "CPNC_KOMITE_KEPUTUSAN",
			"kueri %q menyentuh tabel keputusan; lihat catatan kepala inbox.sql", nama)
	}
}

// Kueri daftar hanya boleh menyentuh KETIGA tabel yang kedua rule sumber sebut.
//
// `InboxRegisterKomite_RD` membaca kelas Work-Komite dan `Assign-Worklist`;
// `GetKomitePAditerima` menambahkan `T_CLAIM_KOMITE_LIST` untuk kesimpulan kotak Diterima
// dan Ditolak. Tidak ada yang keempat.
//
// Tabel tambahan apa pun — penilaian AI, dashboard OR, `T_CLAIM_PNC` — berarti layar ini
// menampilkan sesuatu yang TIDAK pernah ada di layar aslinya, dan angka yang tidak dapat
// dibandingkan dengan apa pun pada uji kesetaraan (`P-5`).
func TestKueriDaftarHanyaMenyentuhTigaTabelSumber(t *testing.T) {
	terlarang := []string{
		"T_CLAIM_DATA_RESULTS_AI",
		"PEGA_DASHBOARDPNC",
		"T_CLAIM_PNC",
		"BUSINESSNEW",
	}

	for nama, teks := range queries {
		if !strings.HasPrefix(nama, "inbox_") {
			continue
		}
		hurufBesar := strings.ToUpper(teks)
		for _, tabel := range terlarang {
			require.NotContainsf(t, hurufBesar, tabel,
				"kueri %q menyentuh %s — bukan salah satu dari tiga tabel sumber", nama, tabel)
		}
	}
}

// Penyaring tahun WAJIB ada di setiap kueri daftar.
//
// `InboxRegisterKomite_RD` menyaring `pxYearNumber(.pxCreateDateTime) >= "2024"` sebagai
// penyaring `F1`. Ukurannya nyata: pada basis data ASM ia menurunkan jumlah baris yang
// lolos dari **417 menjadi 189**. Melewatkannya berarti layar menampilkan lebih dari dua
// kali lipat pekerjaan yang pernah terlihat di Pega.
//
// Yang dicari adalah penandanya, bukan angkanya: batas tahunnya dikirim sebagai parameter
// dari `komite.InboxEarliestCreatedAt`, bukan ditanam di dalam teks SQL.
func TestKueriDaftarMenyaringTahunTerawal(t *testing.T) {
	for _, nama := range []string{"inbox_list", "inbox_count", "inbox_summary"} {
		t.Run(nama, func(t *testing.T) {
			require.Containsf(t, bersihkanSpasi(query(nama)), "a.PXCREATEDATETIME >= :2",
				"kueri %q tidak menyaring tahun terawal (penyaring F1 pada RD)", nama)
		})
	}
}

// Banyaknya argumen yang disusun Go WAJIB sama dengan banyaknya penanda di kuerinya.
//
// Selisih satu penanda tidak menimbulkan galat kompilasi; ia menggeser SELURUH nilai
// sesudahnya ke kolom yang salah — operator terbaca sebagai kotak, kotak sebagai pola
// pencarian — dan hasilnya kosong tanpa satu pun pesan yang menjelaskan sebabnya.
//
// Inilah kelas kesalahan yang paling mungkin muncul setelah kembaran `_warisan` ada,
// karena keduanya dilayani satu fungsi penyusun argumen dengan penomoran yang berbeda.
func TestJumlahArgumenSesuaiJumlahPenanda(t *testing.T) {
	filter := komite.InboxFilter{Operator: "ELLENSUPRIYATI"}.Normalize()

	kasus := []struct {
		kueri   string
		argumen int
	}{
		{"inbox_count", len(filterArgs(filter))},
		{"inbox_summary", len(summaryArgs(filter))},

		// Daftar menambahkan offset dan limit di belakang argumen penghitungnya.
		{"inbox_list", len(filterArgs(filter)) + 2},
	}

	for _, k := range kasus {
		t.Run(k.kueri, func(t *testing.T) {
			require.Equal(t, k.argumen, penandaTertinggi(t, query(k.kueri)),
				"jumlah argumen tidak sama dengan jumlah penanda")
		})
	}
}

// penandaTertinggi mengembalikan nomor penanda parameter terbesar di dalam satu kueri,
// sekaligus menuntut penomorannya rapat — tanpa nomor yang terlewat.
//
// Nomor yang terlewat berarti ada argumen yang dikirim tetapi tidak pernah dipakai, dan
// itu selalu berarti salah satu dari dua hal: penyaring yang terlupa, atau nilai yang
// masuk ke kolom yang salah.
func penandaTertinggi(t *testing.T, teks string) int {
	t.Helper()

	terlihat := map[int]bool{}
	tertinggi := 0
	for _, cocok := range regexp.MustCompile(`:(\d+)`).FindAllStringSubmatch(teks, -1) {
		nomor, err := strconv.Atoi(cocok[1])
		require.NoError(t, err)
		terlihat[nomor] = true
		if nomor > tertinggi {
			tertinggi = nomor
		}
	}
	for nomor := 1; nomor <= tertinggi; nomor++ {
		require.Truef(t, terlihat[nomor], "penanda :%d tidak dipakai; penomorannya berlubang", nomor)
	}
	return tertinggi
}

// bersihkanSpasi meratakan spasi supaya perbandingan menilai ISI klausa, bukan lekukannya.
func bersihkanSpasi(teks string) string {
	return strings.Join(strings.Fields(teks), " ")
}

// Nilai dari pengguna TIDAK PERNAH masuk ke dalam teks SQL.
//
// Yang disisipkan expandCases hanyalah penanda parameter `:1, :2, …`. Uji ini menjaga
// pernyataan itu tetap benar bila fungsinya kelak disunting — celah `{ASIS:...}` warisan
// (538 kemunculan, `K-29`) tidak boleh terbuka kembali lewat pintu ini.
func TestExpandCasesMenyisipkanPenandaBukanNilai(t *testing.T) {
	pernyataan, argumen := expandCases(query("decision_list_for_cases"), []string{
		"K-1", "'; DROP TABLE POOLDATA.CPNC_KOMITE_KEPUTUSAN; --",
	})

	require.Contains(t, pernyataan, "IN (:1, :2)")
	require.NotContains(t, pernyataan, "DROP TABLE")
	require.NotContains(t, pernyataan, "K-1")
	require.Equal(t, []any{"K-1", "'; DROP TABLE POOLDATA.CPNC_KOMITE_KEPUTUSAN; --"}, argumen)
}

func TestUniqueNonEmptyMembuangYangKosongDanBerulang(t *testing.T) {
	require.Equal(t,
		[]string{"K-1", "K-2"},
		uniqueNonEmpty([]string{" K-1 ", "", "K-2", "K-1", "   "}),
	)
	require.Empty(t, uniqueNonEmpty(nil))
}

// Kolom EMAIL dan CC TIDAK boleh ikut dibaca.
//
// `D-67` menetapkan alamat pribadi pada master lama — sekurang-kurangnya enam akun Gmail
// di jalur produksi — tidak dibawa ke sistem baru, dan `D-69` mewajibkan alamat surel
// disamarkan di seluruh artefak. Tidak membacanya sejak kueri membuat alamat itu tidak
// pernah sampai ke peramban, alih-alih mengandalkan setiap lapisan sesudahnya ingat
// membuangnya.
func TestKueriTidakMembacaAlamatSurel(t *testing.T) {
	for nama, teks := range queries {
		hurufBesar := strings.ToUpper(teks)
		require.NotContainsf(t, hurufBesar, "EMAIL,",
			"kueri %q membaca kolom EMAIL", nama)
		require.NotRegexpf(t, `(?m)^\s*CC\s*,?\s*$`, hurufBesar,
			"kueri %q membaca kolom CC", nama)
	}
}

// Kolom yang dibaca harus tepat sama dengan yang dipindai kode. Selisih satu kolom
// membuat Scan gagal dengan pesan yang tidak menyebut kolom mana yang salah.
func TestJumlahKolomSesuaiDenganYangDipindai(t *testing.T) {
	const jumlahDipindai = 13 // lihat scanRow

	for _, nama := range []string{"ambang_komite_daftar", "ambang_komite_periksa_tabel"} {
		t.Run(nama, func(t *testing.T) {
			teks := query(nama)
			bagian := teks[strings.Index(strings.ToUpper(teks), "SELECT")+len("SELECT"):]
			bagian = bagian[:strings.Index(strings.ToUpper(bagian), "FROM")]
			require.Len(t, strings.Split(bagian, ","), jumlahDipindai,
				"jumlah kolom di kueri harus sama dengan yang dipindai scanRow")
		})
	}
}

// Penafsiran nilai kolom diuji tersendiri, karena tipe kolom yang sebenarnya BELUM
// diketahui — DDL tabel ini tidak pernah kita lihat (`R-08` masih terbuka).
//
// Modul ini karena itu harus tahan terhadap kedua kemungkinan: angka maupun teks.
func TestPenafsiranNilaiKolom(t *testing.T) {
	t.Run("penanda menyala hanya untuk 1 dan Y", func(t *testing.T) {
		for _, menyala := range []any{"1", " 1 ", "Y", "y", int64(1), float64(1), true} {
			require.Truef(t, toFlag(menyala), "%v seharusnya menyala", menyala)
		}
		// Kosong, NULL, dan "0" padam. Ini bukan tempat bermurah hati: STS_ADJ yang
		// salah dibaca menyala akan memasukkan baris pemberitahuan registrasi ke dalam
		// daftar penyetuju uang.
		for _, padam := range []any{nil, "", "0", " ", "N", int64(0), float64(0), false} {
			require.Falsef(t, toFlag(padam), "%v seharusnya padam", padam)
		}
	})

	t.Run("teks dirapikan dan tidak pernah bernotasi ilmiah", func(t *testing.T) {
		require.Equal(t, "", toText(nil))
		require.Equal(t, "4", toText(" 4 "))
		require.Equal(t, "MARTENPETRUSLALAMENTIK_1", toText("MARTENPETRUSLALAMENTIK_1\n"))
		require.Equal(t, "7", toText(int64(7)))
		// 'g' akan menghasilkan "1e+08" di sini; itulah sebabnya toText memakai 'f'.
		require.Equal(t, "100000000", toText(float64(100000000)))
	})

	t.Run("bilangan terbaca dari angka maupun teks", func(t *testing.T) {
		for _, masuk := range []any{int64(3), float64(3), "3", " 3 "} {
			hasil, err := toInt(masuk)
			require.NoError(t, err)
			require.Equal(t, 3, hasil)
		}

		// DEGREE kosong menjadi nol, bukan galat: baris seperti itu memang ada di
		// master dan toh bukan jenjang persetujuan.
		hasil, err := toInt(nil)
		require.NoError(t, err)
		require.Zero(t, hasil)
	})

	t.Run("nilai uang terbaca dari angka maupun teks", func(t *testing.T) {
		for _, masuk := range []any{int64(50_000_001), float64(50_000_001), "50000001"} {
			hasil, err := money.FromSQLValue(masuk)
			require.NoError(t, err)
			require.Equal(t, money.FromRupiah(50_000_001), hasil)
		}
	})
}
