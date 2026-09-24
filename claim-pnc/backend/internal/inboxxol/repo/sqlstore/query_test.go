package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxxol"
)

// Kesembilan kueri yang WAJIB ada.
//
// Daftar ini ditulis lengkap, bukan diturunkan dari isi berkas .sql. Menurunkannya akan
// membuat uji ini lulus ketika sebuah kueri TERHAPUS — persis kejadian yang paling ingin
// ditangkap, karena kehilangannya baru terlihat saat layar dibuka di produksi.
var requiredQueries = []string{
	"master_list",
	"master_pending_committee",
	"master_business_list",
	"claim_summary",
	"breakdown_business",
	"breakdown_treaty_inward",
	"advice_list_pla",
	"advice_list_dla",
	"approval_advice_queue",
	"cause_of_loss_list",
}

func TestSeluruhKueriYangDibutuhkanAda(t *testing.T) {
	for _, name := range requiredQueries {
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, strings.TrimSpace(query(name)),
				"kueri %q kosong atau tidak ada di berkas .sql", name)
		})
	}
}

func TestTidakAdaKueriTakTerpakai(t *testing.T) {
	wanted := map[string]bool{}
	for _, name := range requiredQueries {
		wanted[name] = true
	}
	for name := range queries {
		require.True(t, wanted[name],
			"kueri %q ada di berkas .sql tetapi tidak terdaftar di requiredQueries — "+
				"kueri yang tidak dipanggil siapa pun adalah SQL yang tidak pernah di-review ulang", name)
	}
}

// TestTidakAdaPernyataanYangMenulis menegakkan batas yang ditetapkan Work Owner
// 2026-09-20: modul ini membaca saja.
//
// Ia bukan sekadar kerapian. Keempat tabel yang ditulis sistem lama —
// XOL_TABLE_ALL_KLAIM, T_PLA_XOL, T_DLA_XOL, dan MST_XOL_PNC — tetap dimiliki Pega
// selama masa paralel (`P-1`). Satu INSERT yang lolos ke sini berarti dua sistem menulis
// satu tabel dengan aturan berbeda, dan akibatnya bukan galat melainkan data yang saling
// menimpa tanpa jejak.
func TestTidakAdaPernyataanYangMenulis(t *testing.T) {
	forbidden := []string{"INSERT", "UPDATE", "DELETE", "MERGE", "TRUNCATE", "COMMIT"}
	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, word := range forbidden {
			require.NotRegexp(t, regexp.MustCompile(`\b`+word+`\b`), upper,
				"kueri %q memuat %s — modul ini tidak menulis apa pun (P-1)", name, word)
		}
		require.True(t, strings.HasPrefix(strings.ToUpper(strings.TrimSpace(text)), "SELECT"),
			"kueri %q tidak dimulai dengan SELECT", name)
	}
}

// TestTidakAdaPolaSQLTerlarang menegakkan disiplin SQL portabel `D-20`.
//
// `TO_CHAR` DIKECUALIKAN di modul ini, dan pengecualiannya disengaja. Larangan
// `09-DATABASE-STRATEGY.md` §4 menyasar TO_CHAR untuk PEMFORMATAN TAMPILAN; di sini ia
// dipakai sebagai PREDIKAT terhadap kolom yang memang menyimpan teks bertanggal
// (`XOL_TABLE_ALL_KLAIM.DOL` dan `T_CLAIM_INWARD_XOL.DATEOFLOSS` keduanya VARCHAR
// berformat dd/mm/yyyy). Menghapusnya berarti mengubah baris mana yang cocok — bukan
// mengubah tampilannya. Keduanya tersedia di Oracle maupun PostgreSQL.
func TestTidakAdaPolaSQLTerlarang(t *testing.T) {
	forbidden := map[string]string{
		`\bNVL\s*\(`:      "pakai COALESCE",
		`\bSYSDATE\b`:     "pakai CURRENT_TIMESTAMP",
		`\bDECODE\s*\(`:   "pakai CASE WHEN",
		`\bROWNUM\b`:      "pakai OFFSET … FETCH NEXT … ROWS ONLY",
		`\bINSTR\s*\(`:    "pakai POSITION",
		`\bLISTAGG\s*\(`:  "rangkai di Go, bukan di SQL",
		`\bFROM\s+DUAL\b`: "hilangkan klausa FROM",
		`\bSELECT\s+\*`:   "sebutkan nama kolom",
		`\(\+\)`:          "pakai LEFT JOIN",
	}
	for name, text := range queries {
		upper := strings.ToUpper(text)
		for pattern, fix := range forbidden {
			require.NotRegexp(t, regexp.MustCompile(pattern), upper,
				"kueri %q memakai pola terlarang %s — %s", name, pattern, fix)
		}
	}
}

// TestTidakAdaPemanggilanFunctionBasisData menegakkan `D-02`.
//
// Kedua function yang dipakai jalur XOL lama sudah ditulis ulang di tempatnya:
// GET_GROUPBUSINESS_XOL menjadi master_business_list, dan GETCURRENCYSTANDARD menjadi
// subkueri pada breakdown_treaty_inward. Keduanya tidak boleh kembali.
func TestTidakAdaPemanggilanFunctionBasisData(t *testing.T) {
	forbidden := []string{
		"GET_GROUPBUSINESS_XOL",
		"GETCURRENCYSTANDARD",
		"GETSELISIHJAM",
		"GET_WORKING_HOURS",
	}
	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, fn := range forbidden {
			require.NotContains(t, upper, fn,
				"kueri %q memanggil function basis data %s — D-02 melarangnya", name, fn)
		}
	}
}

// TestTidakAdaDBLink menegakkan `D-25`: akses lintas basis data diganti API, bukan
// dibawa apa adanya.
func TestTidakAdaDBLink(t *testing.T) {
	for name, text := range queries {
		require.NotRegexp(t, regexp.MustCompile(`@[A-Za-z_][A-Za-z0-9_]*`), text,
			"kueri %q memakai DB Link — D-25 menggantinya dengan API", name)
	}
}

// TestKeduaKueriPemberitahuanKembarPersis menjaga advice_list_pla dan advice_list_dla
// tetap sejalan.
//
// Keduanya sengaja ditulis dua kali alih-alih dirangkai dengan nama tabel berparameter —
// nama tabel tidak dapat menempuh parameter binding. Harga dari pilihan itu adalah
// kembaran yang dapat berpisah diam-diam, dan uji inilah yang membayarnya: yang berbeda
// HANYA nama tabelnya.
func TestKeduaKueriPemberitahuanKembarPersis(t *testing.T) {
	pla := query("advice_list_pla")
	dla := query("advice_list_dla")

	normalized := strings.Replace(pla, "POOLDATA.T_PLA_XOL", "POOLDATA.T_DLA_XOL", 1)
	require.Equal(t, normalized, dla,
		"advice_list_pla dan advice_list_dla berbeda pada hal selain nama tabel")

	require.Contains(t, pla, "POOLDATA.T_PLA_XOL")
	require.Contains(t, dla, "POOLDATA.T_DLA_XOL")
	require.NotContains(t, pla, "T_DLA_XOL")
	require.NotContains(t, dla, "T_PLA_XOL")
}

// TestKeduaKueriMasterMemakaiKolomYangSama menjaga master_list dan
// master_pending_committee tetap dapat dipindai scanMaster yang sama.
//
// Keduanya dipindai fungsi yang SATU, dan urutan kolom yang bergeser di salah satunya
// tidak menghasilkan galat kompilasi — hanya nilai yang tertukar diam-diam.
func TestKeduaKueriMasterMemakaiKolomYangSama(t *testing.T) {
	require.Equal(t, selectedAliases(query("master_list")),
		selectedAliases(query("master_pending_committee")),
		"master_list dan master_pending_committee tidak mengembalikan kolom yang sama")
}

// TestJumlahKolomSesuaiDenganPemindainya memastikan setiap kueri mengembalikan sebanyak
// kolom yang dipindai kodenya.
func TestJumlahKolomSesuaiDenganPemindainya(t *testing.T) {
	cases := map[string]int{
		"master_list":              10,
		"master_pending_committee": 10,
		"master_business_list":     3,
		"claim_summary":            4,
		"breakdown_business":       5,
		"breakdown_treaty_inward":  5,
		"advice_list_pla":          21,
		"advice_list_dla":          21,
		"approval_advice_queue":    4,
		"cause_of_loss_list":       2,
	}
	for name, wanted := range cases {
		t.Run(name, func(t *testing.T) {
			require.Len(t, selectedAliases(query(name)), wanted)
		})
	}
}

// TestPenandaIDsHanyaAdaDiKueriYangMembutuhkannya menjaga penanda tidak tertinggal di
// kueri yang tidak pernah melewati expandIDs.
//
// Penanda yang tertinggal akan terkirim ke basis data sebagai `IN ()` — sah secara teks,
// tidak sah secara SQL, dan gagalnya baru terlihat saat layar dibuka.
func TestPenandaIDsHanyaAdaDiKueriYangMembutuhkannya(t *testing.T) {
	needsIDs := map[string]bool{"claim_summary": true, "breakdown_business": true}
	for name, text := range queries {
		if needsIDs[name] {
			require.Contains(t, text, idsMarker, "kueri %q kehilangan penanda %s", name, idsMarker)
			continue
		}
		require.NotContains(t, text, idsMarker,
			"kueri %q memuat penanda %s padahal tidak pernah melewati expandIDs", name, idsMarker)
	}
}

func TestExpandIDsMenomoriPlaceholderBerurutan(t *testing.T) {
	text, arguments, err := expandIDs("claim_summary", []any{"2024"}, []string{"A", "B", "C"})
	require.NoError(t, err)

	require.Contains(t, text, "IN (:2, :3, :4)")
	require.NotContains(t, text, idsMarker)
	require.Equal(t, []any{"2024", "A", "B", "C"}, arguments)
}

func TestExpandIDsPadaKueriRincianMelanjutkanDuaParameterPertama(t *testing.T) {
	text, arguments, err := expandIDs("breakdown_business",
		[]any{"01/02/2024", "BANJIR"}, []string{"10", "20"})
	require.NoError(t, err)

	require.Contains(t, text, "IN (:3, :4)")
	require.Equal(t, []any{"01/02/2024", "BANJIR", "10", "20"}, arguments)
}

// TestExpandIDsMenolakSenaraiKosong menjaga `IN ()` tidak pernah terbentuk.
//
// Di Oracle ia galat sintaksis; di sebagian dialek lain ia mengembalikan SELURUH baris.
// Yang kedua jauh lebih berbahaya: ia menampilkan klaim group business yang tidak
// ditanggung perjanjian itu, tanpa satu pun pesan galat.
func TestExpandIDsMenolakSenaraiKosong(t *testing.T) {
	_, _, err := expandIDs("claim_summary", []any{"2024"}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sekurangnya satu kode group business")
}

func TestExpandIDsMenolakKueriTanpaPenanda(t *testing.T) {
	_, _, err := expandIDs("cause_of_loss_list", nil, []string{"A"})
	require.Error(t, err)
	require.Contains(t, err.Error(), idsMarker)
}

// TestKomentarTidakIkutTerkirim memastikan penjelasan panjang di berkas .sql berhenti di
// berkas itu.
//
// Pengecualiannya satu: penanda /*:ids*/, yang memang harus sampai ke expandIDs.
func TestKomentarTidakIkutTerkirim(t *testing.T) {
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			require.False(t, strings.HasPrefix(strings.TrimSpace(line), "--"),
				"kueri %q masih membawa baris komentar: %s", name, line)
		}
	}
}

// TestTabelYangDisentuhSemuanyaDiketahui menjaga modul ini tidak diam-diam membaca tabel
// di luar yang sudah dianalisis.
//
// Tabel baru yang muncul tanpa tercatat berarti ada bagian sistem lama yang dipakai tanpa
// pernah dibaca rule-nya — persis yang `P-5` tuntut jangan terjadi.
func TestTabelYangDisentuhSemuanyaDiketahui(t *testing.T) {
	known := map[string]bool{
		"POOLDATA.MST_XOL_PNC":         true,
		"POOLDATA.MST_XOL_BUSINESS":    true,
		"POOLDATA.BUSINESSGROUP":       true,
		"POOLDATA.MST_USER_TEKNIK":     true,
		"POOLDATA.XOL_TABLE_ALL_KLAIM": true,
		"POOLDATA.V_D_CAUSE_OF_LOSS":   true,
		"POOLDATA.T_CLAIM_XOL":         true,
		"POOLDATA.T_CLAIM_INWARD_XOL":  true,
		"POOLDATA.M_CURRENCYSTANDARD":  true,
		"POOLDATA.T_PLA_XOL":           true,
		"POOLDATA.T_DLA_XOL":           true,
		"POOLDATA.T_REINSURER":         true,
	}
	pattern := regexp.MustCompile(`(?i)\bFROM\s+(POOLDATA\.[A-Z_][A-Z0-9_]*)`)
	for name, text := range queries {
		for _, match := range pattern.FindAllStringSubmatch(text, -1) {
			table := strings.ToUpper(match[1])
			require.True(t, known[table],
				"kueri %q membaca tabel %s yang belum tercatat", name, table)
		}
	}
}

// TestTipePemberitahuanTerpetakanSeluruhnya menjaga setiap tipe punya kuerinya.
func TestTipePemberitahuanTerpetakanSeluruhnya(t *testing.T) {
	for _, adviceType := range []inboxxol.AdviceType{inboxxol.AdvicePLA, inboxxol.AdviceDLA} {
		name, known := adviceQueryByType[adviceType]
		require.True(t, known, "tipe %q tidak punya kueri", adviceType)
		require.NotEmpty(t, query(name))
	}
	require.Len(t, adviceQueryByType, 2)
}

// TestMataUangDasarJatuhKeBawaanSaatKosong menjaga treaty inward tidak kehilangan kurs
// hanya karena konfigurasinya belum diisi.
func TestMataUangDasarJatuhKeBawaanSaatKosong(t *testing.T) {
	require.Equal(t, DefaultBaseCurrencyID, NewRepoWithBaseCurrency(nil, "   ").baseCurrencyID)
	require.Equal(t, "20002", NewRepoWithBaseCurrency(nil, " 20002 ").baseCurrencyID)
	require.Equal(t, DefaultBaseCurrencyID, NewRepo(nil).baseCurrencyID)
}

// selectedAliases mengembalikan nama kolom hasil sebuah kueri, dalam urutan aslinya.
//
// Ia membaca alias setelah `AS`, dan untuk kolom tanpa alias membaca nama kolomnya. Ia
// sengaja sederhana: yang diperiksa adalah kesesuaian JUMLAH dan URUTAN antara kueri dan
// pemindainya, bukan penguraian SQL yang utuh.
func selectedAliases(text string) []string {
	head := text
	// Hanya blok SELECT terluar yang dibaca. Subkueri di dalam FROM punya SELECT-nya
	// sendiri, dan ikut menghitungnya akan membuat jumlah kolom tampak lebih banyak
	// daripada yang benar-benar dikembalikan.
	if index := strings.Index(strings.ToUpper(head), "\n  FROM "); index > 0 {
		head = head[:index]
	}

	var aliases []string
	depth := 0
	current := strings.Builder{}
	flush := func() {
		part := strings.TrimSpace(current.String())
		current.Reset()
		if part == "" {
			return
		}
		fields := strings.Fields(part)
		last := fields[len(fields)-1]
		aliases = append(aliases, strings.ToUpper(strings.TrimSuffix(last, ",")))
	}

	body := strings.TrimSpace(head)
	body = strings.TrimPrefix(body, "SELECT")
	for _, char := range body {
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				flush()
				continue
			}
		}
		current.WriteRune(char)
	}
	flush()
	return aliases
}
