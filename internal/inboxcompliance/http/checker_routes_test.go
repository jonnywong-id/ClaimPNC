package inboxcompliancehttp_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

// Rute form Compliance Checker.
//
// Berkas tersendiri dari routes_test.go, bukan tambahan di ujungnya: berkas itu sudah
// menguji dua jalur baca dan satu jalur tulis, dan menambahkan jalur keempat ke dalamnya
// membuat satu berkas menguji dua layar yang berbeda.

// claimInQueue adalah klaim contoh yang sedang menunggu di antrean Compliance.
//
// Kunci teknis Pega, lengkap dengan SPASI di dalamnya — dan spasi itulah yang membuat uji
// di bawah bermakna: ia memaksa jalur permintaan dikodekan, dan pengkodean yang salah di
// salah satu ujung tidak akan pernah ketahuan pada kunci tanpa spasi.
const claimInQueue = "ASM-FW-GCNMFW-WORK PNC-900101"

func checkerRoute(reference string) string {
	return route + "/" + url.PathEscape(reference)
}

// Membuka form lewat HTTP: klaimnya terbawa, keempat pilihan terbawa, keterbatasan terbawa.
//
// Keterbatasan ikut diuji, dan itu bukan kelebihan: kalimat itu satu-satunya tempat
// petugas diberi tahu bahwa keputusannya BELUM mengubah klaim di Pega. Menghilangkannya
// membuat layar menjanjikan sesuatu yang tidak terjadi.
func TestMembukaFormComplianceChecker(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, checkerRoute(claimInQueue), "ASM", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)

	klaim, ok := content["klaim"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, claimInQueue, klaim["referensi"])

	pilihan, ok := content["pilihan"].([]any)
	require.True(t, ok)
	require.Len(t, pilihan, 4, "keempat nilai PilihanCompliance")

	pertama, ok := pilihan[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "0", pertama["nilai"])
	require.Equal(t, "Fraud / Tolak", pertama["label"])

	// Belum pernah diputuskan: `null`, bukan objek kosong. Layar membedakan keduanya —
	// objek kosong akan terbaca sebagai "diputuskan dengan pilihan kosong".
	require.Nil(t, content["keputusan"])

	require.NotEmpty(t, content["keterbatasan"])
	require.Equal(t, "ASM", content["portal"])
}

// Jalur `{referensi}` TIDAK menelan jalur harfiah yang sudah ada.
//
// `/tab` dan `/post-audit` keduanya cocok dengan pola `{referensi}`. chi memang
// mengutamakan jalur harfiah, tetapi itu sifat pustaka — bukan sesuatu yang terlihat saat
// membaca berkas rute. Uji ini yang menahan perubahan urutan atau pola yang diam-diam
// mengalihkan keduanya ke handler form.
func TestJalurHarfiahTidakTertelanPolaReferensi(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, route+"/tab", "ASM", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotNil(t, content["tab_bawaan"], "/tab harus tetap dijawab keterangan layar")
	require.Nil(t, content["klaim"], "/tab tidak boleh dijawab badan form")
}

// Menyimpan keputusan lewat HTTP, lalu membacanya kembali lewat HTTP.
//
// Dua permintaan, bukan satu, dengan alasan yang sama seperti pada uji Post Audit:
// permintaan pertama hanya membuktikan handler menjawab 200. Yang dipertaruhkan adalah
// apakah keputusan itu terbaca saat petugas membuka formnya lagi — dan di antara keduanya
// ada pemetaan DTO dan penyimpanan yang masing-masing bisa menjatuhkannya tanpa galat.
func TestMenyimpanKeputusanLaluMembacanyaUlang(t *testing.T) {
	server := newTestServer(t)

	response, content := server.post(
		t, checkerRoute(claimInQueue)+"/keputusan", "ASM", server.token,
		`{"pilihan":"1","note":"Dokumen lengkap",`+
			`"komentar":[{"komentar":"Tidak ada temuan"}]}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	keputusan, ok := content["keputusan"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "1", keputusan["pilihan"])
	require.Equal(t, "Bayar / Valid", keputusan["pilihan_label"])
	require.NotEmpty(t, keputusan["tanggal_valid"])
	require.Nil(t, keputusan["tanggal_kirim_post_audit"])

	// Bayar/Valid TIDAK menerbitkan baris Post Audit.
	require.Nil(t, content["post_audit"])

	_, reopened := server.call(t, checkerRoute(claimInQueue), "ASM", server.token)
	tersimpan, ok := reopened["keputusan"].(map[string]any)
	require.True(t, ok, "keputusan harus terbaca saat form dibuka ulang")
	require.Equal(t, "1", tersimpan["pilihan"])
	require.Equal(t, "Dokumen lengkap", tersimpan["note"])

	// Komentarnya terbaca kembali, bernomor. Inilah isian petugas — bukan
	// `catatan_investigator`, yang read-only di Pega.
	komentar, ok := tersimpan["komentar"].([]any)
	require.True(t, ok)
	require.Len(t, komentar, 1)
	require.Equal(t, "Tidak ada temuan", komentar[0].(map[string]any)["komentar"])
	require.Equal(t, float64(1), komentar[0].(map[string]any)["urutan"])
}

// Bayar/PostAudit menerbitkan baris Post Audit, dan barisnya tampil di tabnya.
//
// Inilah yang membuat form ini MENCAKUP tombol "Kirim ke Post Audit" yang berdiri sendiri:
// di Pega, pengiriman ke Post Audit bukan aksi tersendiri melainkan SALAH SATU dari empat
// hasil form ini — `SetComplianceResult` langkah 10, prasyarat `pilihan == 2`.
func TestKeputusanPostAuditMenerbitkanBarisDanTampilDiTabnya(t *testing.T) {
	server := newTestServer(t)

	response, content := server.post(
		t, checkerRoute(claimInQueue)+"/keputusan", "ASM", server.token,
		// `aksi` WAJIB "kirim": baris Post Audit terbit dari `SetComplianceResult`
		// langkah 20, dan tombol "Simpan Data" tidak memanggil activity itu.
		`{"pilihan":"2","aksi":"kirim","komentar":[{"komentar":"to compilance"}]}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	postAudit, ok := content["post_audit"].(map[string]any)
	require.True(t, ok, "pilihan 2 harus menerbitkan baris Post Audit")
	require.Equal(t, "CPL.26.1", postAudit["nomor_case"])
	// Catatan baris Post Audit datang dari KLAIM — `.ClaimData.ComplianceRemark` yang
	// read-only — bukan dari komentar yang baru diketik. Ia kosong karena kueri tab
	// Compliance belum membawa kolom itu; lihat catatan pada uji usecase sejenis.
	require.Empty(t, postAudit["catatan"])

	listing, listed := server.call(
		t, route+"?tab="+inboxcompliance.TabPostAudit, "ASM", server.token)
	require.Equal(t, http.StatusOK, listing.StatusCode)

	rows, ok := listed["baris"].([]any)
	require.True(t, ok)

	found := false
	for _, row := range rows {
		if row.(map[string]any)["nomor_case"] == "CPL.26.1" {
			found = true
		}
	}
	require.True(t, found, "baris Post Audit harus tampil di tabnya")
}

// Pilihan kosong dijawab 422 yang MENUNJUK isiannya.
//
// 422, bukan 400: permintaannya terbentuk benar, isinya yang melanggar aturan bisnis
// (`10-API-STRATEGY.md` §5). Layar menanganinya berbeda — yang satu bug frontend, yang
// lain kesalahan pengguna yang dapat diperbaiki di tempat.
func TestKeputusanTanpaPilihanDijawab422(t *testing.T) {
	server := newTestServer(t)

	response, content := server.post(
		t, checkerRoute(claimInQueue)+"/keputusan", "ASM", server.token,
		`{"pilihan":"","catatan":"apa pun"}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	detail, ok := content["detail"].([]any)
	require.True(t, ok, "pelanggaran harus menunjuk isiannya, bukan hanya satu pesan")
	require.Len(t, detail, 1)
	require.Equal(t, "pilihan", detail[0].(map[string]any)["field"])
}

// Klaim yang TIDAK di antrean tidak dapat dibuka maupun diputuskan.
//
// Keduanya 409, bukan 404, dan pembedaan itu yang penting: klaimnya boleh jadi ada dan
// sehat, hanya sudah diputuskan petugas lain. "Tidak ditemukan" akan membuat petugas
// mencari klaimnya, padahal yang perlu dilakukan hanyalah menyegarkan daftar.
func TestFormMenolakKlaimDiLuarAntrean(t *testing.T) {
	server := newTestServer(t)
	hilang := checkerRoute("ASM-FW-GCNMFW-WORK PNC-TIDAK-ADA")

	response, _ := server.call(t, hilang, "ASM", server.token)
	require.Equal(t, http.StatusConflict, response.StatusCode)

	response, _ = server.post(t, hilang+"/keputusan", "ASM", server.token,
		`{"pilihan":"1"}`)
	require.Equal(t, http.StatusConflict, response.StatusCode)
}

// Kedua rute form berada di balik sesi DAN di balik portal.
//
// Diuji tersendiri, tidak dianggap tercakup uji rute lain: keduanya dipasang lewat
// pemanggilan Mount yang sama, tetapi menyimpang satu baris sudah cukup membuatnya
// terbuka. Pada rute yang MEMUTUSKAN klaim — pilihan `0` berarti Fraud/Tolak — akibatnya
// bukan kebocoran bacaan melainkan penolakan klaim lintas badan hukum (`R-20`).
func TestFormMenuntutSesiDanPortal(t *testing.T) {
	server := newTestServer(t)
	open := checkerRoute(claimInQueue)
	submit := open + "/keputusan"
	body := `{"pilihan":"1"}`

	response, _ := server.call(t, open, "", server.token)
	require.Equal(t, http.StatusBadRequest, response.StatusCode, "buka tanpa portal")

	response, _ = server.call(t, open, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode, "buka tanpa sesi")

	response, _ = server.post(t, submit, "", server.token, body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode, "simpan tanpa portal")

	response, _ = server.post(t, submit, "ASM", "", body)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode, "simpan tanpa sesi")
}
