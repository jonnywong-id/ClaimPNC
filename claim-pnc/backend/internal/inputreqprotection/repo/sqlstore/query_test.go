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
		// '1' pada STATUS_ACTIVE dan '%' pada LIKE.
		// Dua literal tetap yang memang bukan nilai pengguna, ditambah nama KELAS PEGA pada
		// claim_find — yang bukan data melainkan penanda tipe baris, sama sifatnya dengan
		// nama tabel. Ia tidak pernah datang dari pemanggil.
		for _, literal := range []string{"'1'", "'%'", "'ASM-FW-GCNMFW-Work-PNC'"} {
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

// TestPencarianKlaimMemakaiLEFTJOIN menjaga klaim yang datanya tidak lengkap tetap TERBACA.
//
// Hanya 1.393 dari 2.634 klaim punya baris di `T_CLAIM_PNC`, dan 1.281 punya objek.
// `INNER JOIN` akan membuat separuh klaim tampak TIDAK ADA — dan pengguna menerima "klaim
// tidak ditemukan" untuk klaim yang jelas-jelas ada.
func TestPencarianKlaimMemakaiLEFTJOIN(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))
	require.Contains(t, teks, "LEFT JOIN POOLDATA.T_CLAIM_PNC")
	require.NotContains(t, teks, "INNER JOIN")
}

// TestPencarianKlaimTidakMembacaKolomYangNolTerisi menjaga koreksi 2026-09-24.
//
// Versi pertama kueri ini membaca `DATEOFLOSS` dan `CAUSEOFLOSS` dari tabel kerja Pega.
// Hitungan atas 2.634 klaim membantahnya: KEDUA kolom itu nol terisi di sana. Kalau dipakai,
// form menampilkan "Current Date Of Loss" kosong pada SETIAP klaim — tanpa satu pun galat
// yang memberi tahu sebabnya.
func TestPencarianKlaimTidakMembacaKolomYangNolTerisi(t *testing.T) {
	teks := strings.ToUpper(query("claim_find"))
	require.NotContains(t, teks, "W.DATEOFLOSS",
		"DATEOFLOSS nol terisi di tabel kerja Pega; sumbernya T_CLAIM_PNC")
	require.NotContains(t, teks, "W.CAUSEOFLOSS",
		"CAUSEOFLOSS nol terisi di tabel kerja Pega; sumbernya T_CLAIM_OBJECTCOVERAGE")
	require.Contains(t, teks, "C.DATEOFLOSS")
	require.Contains(t, teks, "T_CLAIM_OBJECTCOVERAGE")
}
