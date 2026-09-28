package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Uji di sini berjalan TANPA basis data. Yang diperiksa adalah teks SQL-nya sendiri —
// disiplin yang `09-DATABASE-STRATEGY.md` §4 tetapkan, dan yang hanya benar-benar terjaga
// bila ada yang menggagalkannya saat dilanggar.

// namaKueri adalah seluruh kueri yang dipakai adapter ini.
//
// Daftar ini menggagalkan uji bila sebuah kueri hilang dari berkas .sql — kegagalan yang
// tanpa uji baru muncul sebagai panic saat layar dibuka.
var namaKueri = []string{
	"protection_count",
	"protection_list",
	"protection_get",
	"protection_type_list",
	"claim_find",
	"protection_duplicate",
	"protection_next_sequence",
	"protection_insert",
	"protection_update",
}

// kueriGeneratorNomor adalah SATU-SATUNYA kueri yang boleh memakai konstruksi khas Oracle.
//
// `D-22` dan `D-71` menetapkan generator nomor sebagai satu-satunya sakelar dialek
// (`ADR-0005`). Menamainya di sini, bukan menuliskan literalnya di tiap uji, membuat
// pengecualian itu punya SATU tempat — dan menambah pengecualian kedua menjadi perubahan
// yang terlihat di review.
const kueriGeneratorNomor = "protection_next_sequence"

func TestSeluruhKueriTermuat(t *testing.T) {
	for _, nama := range namaKueri {
		require.NotEmpty(t, strings.TrimSpace(query(nama)), "kueri %q kosong atau tidak ada", nama)
	}
}

// TestKueriMengikutiDisiplinSQLPortabel menjaga `D-20`.
//
// Generator nomor DIKECUALIKAN seluruhnya — lihat kueriGeneratorNomor. Pengecualiannya
// dipagari uji tersendiri di bawah supaya ia tidak menyebar.
func TestKueriMengikutiDisiplinSQLPortabel(t *testing.T) {
	terlarang := []string{
		"NVL(",
		"SYSDATE",
		"ROWNUM",
		"DECODE(",
		"TO_CHAR(",
		"TO_NUMBER(",
		"TRUNC(",
		"FROM DUAL",
		"NEXTVAL",
		"SELECT *",
		"INSTR(",
		"LISTAGG(",
	}

	for _, nama := range namaKueri {
		if nama == kueriGeneratorNomor {
			continue
		}
		teks := strings.ToUpper(query(nama))
		for _, pola := range terlarang {
			require.NotContains(t, teks, pola,
				"kueri %q memakai %s yang dilarang 09-DATABASE-STRATEGY.md §4", nama, pola)
		}
	}
}

// TestGeneratorNomorMemakaiSequence menjaga pengecualiannya tetap berupa APA YANG DIHARAPKAN.
//
// Mengecualikan sebuah kueri dari disiplin portabel tanpa menyatakan isinya berarti kueri
// itu boleh berisi apa saja. Uji ini menyatakan isinya: sequence yang Work Owner buat pada
// 2026-09-24, bukan `MAX+1` yang digantikannya dan bukan konstruksi Oracle lain.
func TestGeneratorNomorMemakaiSequence(t *testing.T) {
	teks := strings.ToUpper(query(kueriGeneratorNomor))

	require.Contains(t, teks, "CLAIM_PROTECTION_SEQ.NEXTVAL",
		"generator nomor harus memakai sequence yang ditetapkan Work Owner")
	require.NotContains(t, teks, "MAX(",
		"MAX+1 sudah digantikan sequence; mengembalikannya menghidupkan lagi balapan nomor")
}

// TestPengecualianDialekTidakMenyebar memagari sakelar dialek pada satu kueri saja.
//
// `ADR-0005` menetapkan generator nomor sebagai satu-satunya tempatnya. Tanpa uji ini,
// kueri berikutnya yang butuh jalan pintas Oracle akan mengambilnya tanpa ada yang
// menghentikan — dan janji SQL portabel `D-20` runtuh sedikit demi sedikit.
func TestPengecualianDialekTidakMenyebar(t *testing.T) {
	for _, nama := range namaKueri {
		if nama == kueriGeneratorNomor {
			continue
		}
		teks := strings.ToUpper(query(nama))
		require.NotContains(t, teks, "NEXTVAL",
			"hanya generator nomor yang boleh menyentuh sequence (ADR-0005)")
		require.NotContains(t, teks, "FROM DUAL",
			"hanya generator nomor yang boleh memakai FROM DUAL (ADR-0005)")
	}
}

// TestNamaTipeDiambilDenganLEFTJOIN menjaga kode tipe yang tak dikenal tetap TERLIHAT.
//
// Master `M_CLAIM_PROTECTION_TYPE` tidak menjamin memuat setiap kode yang pernah tersimpan:
// produksi memakai tipe '9' pada 25 baris sementara export Pega tidak mengenalnya sama
// sekali. INNER JOIN akan MENGHILANGKAN baris seperti itu dari inbox — tanpa galat, tanpa
// gejala, dan justru pada baris yang paling perlu diperiksa manusia.
func TestNamaTipeDiambilDenganLEFTJOIN(t *testing.T) {
	for _, nama := range []string{"protection_list", "protection_get"} {
		teks := strings.ToUpper(query(nama))
		require.Contains(t, teks, "LEFT JOIN POOLDATA.M_CLAIM_PROTECTION_TYPE",
			"kueri %q harus mengambil nama tipe dengan LEFT JOIN", nama)
		require.NotContains(t, teks, "INNER JOIN",
			"kueri %q tidak boleh memakai INNER JOIN ke master tipe", nama)
	}
}

// TestKueriHitungTidakIkutJoinMaster menjaga hitungan tetap murni.
//
// Yang dihitung barisnya, dan nama tipe tidak mengubah jumlahnya. Join yang tidak dipakai
// hanya menambah kerja basis data pada kueri yang berjalan di setiap pemuatan halaman.
func TestKueriHitungTidakIkutJoinMaster(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("protection_count")),
		"M_CLAIM_PROTECTION_TYPE")
}

// TestSeluruhKueriBacaMenyaringSoftDelete menjaga `D-66`.
//
// Satu kueri yang lupa menyaringnya akan menampilkan baris yang seharusnya hilang — kelas
// cacat baru yang tidak ada di sistem lama (`09-DATABASE-STRATEGY.md` §8.1). Kegagalannya
// tidak punya gejala: layar tampil normal, hanya isinya lebih banyak dari seharusnya.
func TestSeluruhKueriBacaMenyaringSoftDelete(t *testing.T) {
	for _, nama := range []string{
		"protection_count", "protection_list", "protection_get",
		"protection_duplicate", "protection_update",
	} {
		require.Contains(t, strings.ToUpper(query(nama)), "STATUS_ACTIVE",
			"kueri %q tidak menyaring penanda hapus (D-66)", nama)
	}
}

// TestKueriDaftarMenyaringBelumDiakseptasi menjaga definisi layar ini.
//
// `Report Definition/InboxReqOpenProtection_RD-RD.xml` menyaring `.AcceptStatus IS NULL`,
// dan itu BUKAN pilihan pengguna melainkan definisi layarnya. Menghapusnya akan menampilkan
// permintaan yang sudah selesai di layar yang bukan tempatnya.
func TestKueriDaftarMenyaringBelumDiakseptasi(t *testing.T) {
	for _, nama := range []string{"protection_count", "protection_list"} {
		require.Contains(t, strings.ToUpper(query(nama)), "APPROVAL_STATUS IS NULL",
			"kueri %q harus menyaring permintaan yang belum diakseptasi", nama)
	}
}

// TestKueriGetTidakMenyaringStatusAkseptasi menjaga kebalikannya.
//
// Form harus tetap dapat dibuka untuk permintaan yang baru saja diakseptasi orang lain,
// supaya pesannya dapat menjelaskan apa yang terjadi — bukan sekadar "tidak ditemukan".
func TestKueriGetTidakMenyaringStatusAkseptasi(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("protection_get")), "APPROVAL_STATUS IS NULL")
}

// TestPenyisipanTidakMenyentuhKolomAkseptasi menjaga `P-1`.
//
// Ketiga kolom itu milik modul inboxacceptopenprotection. Menuliskannya dari sini berarti
// dua modul menulis kolom yang sama, dan akibatnya bukan galat melainkan keputusan
// akseptasi yang tertimpa.
func TestPenyisipanTidakMenyentuhKolomAkseptasi(t *testing.T) {
	for _, nama := range []string{"protection_insert", "protection_update"} {
		teks := strings.ToUpper(query(nama))
		for _, kolom := range []string{"APPROVAL_STATUS", "RESOLVED_BY", "RESOLVED_DATETIME"} {
			if nama == "protection_update" && kolom == "APPROVAL_STATUS" {
				// Pada UPDATE ia muncul di WHERE sebagai penjaga, bukan di SET.
				require.NotContains(t, strings.SplitN(teks, "WHERE", 2)[0], kolom,
					"kueri %q tidak boleh MENULIS %s", nama, kolom)
				continue
			}
			require.NotContains(t, teks, kolom, "kueri %q tidak boleh menyentuh %s", nama, kolom)
		}
	}
}

// TestPenyuntinganMenolakBarisYangSudahTertautKlaim menjaga aturan penguncian.
//
// `Section/InboxReqProtection_Section-Section.xml:8657` menonaktifkan tautan baris ketika
// nomor klaimnya terisi. Syaratnya ada DI DALAM WHERE, bukan hanya diperiksa di Go: yang
// menahan dua permintaan bersamaan adalah basis data.
func TestPenyuntinganMenolakBarisYangSudahTertautKlaim(t *testing.T) {
	teks := strings.ToUpper(query("protection_update"))
	require.Contains(t, teks, "CLAIM_NO IS NULL")
	require.Contains(t, teks, "APPROVAL_STATUS IS NULL")
}

// TestKueriMemakaiParameterBinding menjaga penutup celah `{ASIS:…}` warisan.
//
// Tidak ada satu pun nilai yang dirangkai ke dalam teks SQL; seluruhnya lewat penanda
// bind Oracle (`:1`, `:2`, …).
func TestKueriMemakaiParameterBinding(t *testing.T) {
	for _, nama := range namaKueri {
		teks := query(nama)
		// Tanda kutip tunggal hanya boleh muncul pada literal yang memang tetap —
		// '1' pada STATUS_ACTIVE dan '%' pada LIKE. Keduanya bukan nilai pengguna dan tidak
		// pernah datang dari pemanggil.
		//
		// Nama kelas Pega sempat ada di daftar ini, sebagai penyaring `PXOBJCLASS` pada
		// claim_find. Ia hilang ketika tabel kerja Pega berhenti dibaca (2026-09-26); yang
		// tersisa dari nama itu adalah satu konstanta Go, `prefixKunciKlaimPega`.
		for _, literal := range []string{"'1'", "'%'"} {
			teks = strings.ReplaceAll(teks, literal, "")
		}
		require.NotContains(t, teks, "'",
			"kueri %q memuat literal teks selain yang diizinkan; nilai harus lewat bind", nama)
	}
}

// TestPencarianKlaimHanyaMembaca menjaga `P-1`.
//
// Tabel klaim dimiliki Pega selama masa paralel. `P-1` melarang dua sistem MENULIS satu
// tabel; membaca tidak dilarang. Uji ini menggagalkan penambahan pernyataan tulis ke sana —
// pelanggaran yang tidak menghasilkan galat, hanya data yang diam-diam tidak konsisten.
func TestPencarianKlaimHanyaMembaca(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))
	for _, kata := range []string{"INSERT", "UPDATE", "DELETE", "MERGE"} {
		require.NotContains(t, teks, kata, "claim_find hanya boleh MEMBACA tabel klaim (P-1)")
	}
}

// TestPencarianKlaimBersumberT_CLAIM_PNC menjaga ketetapan Work Owner 2026-09-26.
//
//	"cari noklaim di input req nya ke t_claim_pnc"
//	"jangan gunakan t_claimlist_admin sama sekali, gunakan t_claim_pnc saja"
//
// Dua tabel yang TIDAK boleh kembali, masing-masing dengan sebabnya sendiri:
//
//   - `T_CLAIMLIST_ADMIN` — tabel BACA untuk dashboard. Membacanya di sini membuat layar ini
//     bergantung pada proses pengisi yang jadwalnya di luar modul ini.
//   - `PC_ASM_FW_GCNMFW_WORK` — tabel kerja Pega. Ia sempat menjadi tabel utama kueri ini;
//     mengembalikannya membatalkan ketetapan di atas tanpa ada yang menyadarinya, sebab
//     hasilnya justru terlihat LEBIH lengkap.
func TestPencarianKlaimBersumberT_CLAIM_PNC(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))

	require.Contains(t, teks, "FROM POOLDATA.T_CLAIM_PNC",
		"sumber klaim harus T_CLAIM_PNC")
	require.NotContains(t, teks, "T_CLAIMLIST_ADMIN",
		"T_CLAIMLIST_ADMIN tabel dashboard — tidak dibaca modul ini")
	require.NotContains(t, teks, "PC_ASM_FW_GCNMFW_WORK",
		"tabel kerja Pega tidak lagi dibaca; sumbernya T_CLAIM_PNC saja")
}

// TestPencarianKlaimMencocokkanDuaBentukCLAIMID menjaga klaim WARISAN tetap ditemukan.
//
// `CLAIMID` punya dua bentuk, dan hanya dua — terukur atas 2.176 baris pada 2026-09-26:
//
//	ASM-FW-GCNMFW-WORK PNC-1865   2.167 baris
//	PNCN.26.0007                      9 baris
//
// Mencocokkan satu bentuk saja tidak menghasilkan galat; ia hanya membuat satu golongan
// klaim menjawab "tidak ditemukan". Dua bind menyatakan keduanya dicoba.
//
// `CLAIMNO` sengaja TIDAK dipakai: ia kosong pada 479 baris, berulang pada satu pasang, dan
// berbeda isi dari nomor turunan CLAIMID pada 11 baris.
func TestPencarianKlaimMencocokkanDuaBentukCLAIMID(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))

	require.Contains(t, teks, "UPPER(TRIM(C.CLAIMID)) IN (:1, :2)",
		"kedua bentuk CLAIMID harus dicoba, lewat dua bind terpisah")
	require.NotContains(t, teks, "C.CLAIMNO",
		"CLAIMNO tidak dapat diandalkan sebagai kunci; lihat komentar kueri")
}

// TestPencarianKlaimTidakMemotongDiSQL menjaga dua batas sekaligus.
//
// `INSTR(` ada di daftar pola terlarang `09-DATABASE-STRATEGY.md` §4, dan penggantinya yang
// disebut di sana — `POSITION(x IN y)` — **tidak didukung Oracle**; diuji langsung
// 2026-09-26 dan menghasilkan ORA-00907. Jadi memotong prefix di dalam SQL akan melanggar
// `D-20` atau gagal berjalan.
//
// Pemotongannya karena itu ada di Go (`nomorKlaimDari`). Uji ini menahan keduanya kembali.
func TestPencarianKlaimTidakMemotongDiSQL(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))
	require.NotContains(t, teks, "POSITION(",
		"POSITION(x IN y) tidak didukung Oracle — ORA-00907")
	require.NotContains(t, teks, "SUBSTR(",
		"pemotongan awalan kunci Pega dikerjakan di Go, bukan di SQL")
}

// TestPencarianKlaimMembacaKolomDariTabelYangBENAR menjaga koreksi 2026-09-24.
//
// Versi pertama kueri ini membaca `DATEOFLOSS` dan `CAUSEOFLOSS` dari tabel kerja Pega.
// Hitungan atas 2.634 klaim membantahnya: KEDUA kolom itu nol terisi di sana. Kalau dipakai,
// form menampilkan "Current Date Of Loss" kosong pada SETIAP klaim — tanpa satu pun galat
// yang memberi tahu sebabnya.
//
// Sejak tabel kerja Pega berhenti dibaca, salah-sumber itu tidak mungkin lagi terjadi. Yang
// masih perlu dijaga adalah kedua kolomnya memang DIAMBIL, dan dari tabel yang benar —
// `CAUSEOFLOSS` tidak ada di `T_CLAIM_PNC`, ia milik `T_CLAIM_OBJECTCOVERAGE`.
func TestPencarianKlaimMembacaKolomDariTabelYangBENAR(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))
	require.Contains(t, teks, "C.DATEOFLOSS",
		"DOL diambil dari T_CLAIM_PNC")
	require.Contains(t, teks, "T_CLAIM_OBJECTCOVERAGE",
		"CAUSEOFLOSS hanya ada di T_CLAIM_OBJECTCOVERAGE")
	require.Contains(t, teks, "T_CLAIM_OBJECTLIST",
		"OBJECTNAME hanya ada di T_CLAIM_OBJECTLIST")
}

// TestClaimFindMengambilCLAIMID menjaga sumber `ID_CLAIM` bagi klaim Pega.
//
// `ID_CLAIM` sempat terisi nomor klaim untuk SETIAP baris, termasuk klaim Pega yang
// seharusnya menyimpan IDPEGA. Sumber nilai yang benar adalah `CLAIMID` UTUH — kolom yang
// sama dengan kunci `UPDATE T_CLAIM_PNC` saat proteksinya disetujui.
//
// Menghapusnya kelak tidak akan menghasilkan galat: `ID_CLAIM` hanya akan diam-diam berisi
// nilai yang salah bagi klaim warisan, dan penerapan perubahannya gagal belakangan.
func TestClaimFindMengambilCLAIMID(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))

	pilihan := strings.SplitN(teks, "FROM POOLDATA.T_CLAIM_PNC", 2)[0]
	require.Contains(t, pilihan, "C.CLAIMID,",
		"claim_find harus MEMILIH CLAIMID utuh, bukan hanya memakainya di WHERE")
}

// TestNomorKlaimDiturunkanDariCLAIMID menjaga pemotongan awalan tetap benar untuk KEDUA
// bentuk, dan tetap tahan perbedaan huruf besar-kecil.
//
// Data produksi menulis `ASM-FW-GCNMFW-WORK`, sementara nama kelas yang sama muncul sebagai
// `ASM-FW-GCNMFW-Work-PNC` di tempat lain. Pencocokan yang peka huruf akan membuat satu ejaan
// lolos dan ejaan lain tidak — tanpa gejala, karena keduanya tetap menghasilkan teks.
func TestNomorKlaimDiturunkanDariCLAIMID(t *testing.T) {
	for _, uji := range []struct {
		nama    string
		claimID string
		mau     string
	}{
		{"klaim warisan Pega", "ASM-FW-GCNMFW-WORK PNC-1865", "PNC-1865"},
		{"ejaan huruf campur", "ASM-FW-GCNMFW-Work PNC-1865", "PNC-1865"},
		{"klaim sistem baru tanpa awalan", "PNCN.26.0007", "PNCN.26.0007"},
		{"berspasi di tepi", "  ASM-FW-GCNMFW-WORK PNC-1865  ", "PNC-1865"},
		{"nomor warisan tanpa pola PNC-", "ASM-FW-GCNMFW-WORK KLAIM-LAMA", "KLAIM-LAMA"},
		{"kosong", "   ", ""},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			require.Equal(t, uji.mau, nomorKlaimDari(uji.claimID))
		})
	}
}
