package memory

import (
	_ "embed"

	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleMaster adalah isi contoh POOLDATA.M_AUTO_CLAIM_PNC.
//
// # Dari mana nama-nama ini berasal
//
// TIDAK dari data nyata. Isi tabelnya tidak pernah diterima — hanya kolomnya yang terbaca
// dari RDB List/BrowseAutoKlaim-SQL.xml dan DDL-nya. Nama di bawah KARANGAN, dan sengaja
// terdengar seperti nama lembaga pembiayaan supaya layarnya terasa nyata tanpa menyalin
// satu pun data nasabah ke dalam repository (D-69).
//
// `SRVY` sengaja ada tetapi tidak punya batch satu pun: ia yang membuktikan penyaring
// perusahaan membaca MASTER, bukan tabel batch — dengan sumber yang salah, ia tidak akan
// muncul di dropdown.
func SampleMaster() map[string]string {
	return sampledata.Must[map[string]string](sampleJSON, "SampleMaster")
}

// berlaku2026 adalah data polis yang periodenya mencakup seluruh tahun 2026, bermata uang IDR.
var berlaku2026 = inboxautoclaim.PolicyDetail{StartDate: "01/01/2026", EndDate: "31/12/2026", Currency: "IDR"}

// SamplePolicy meniru hasil pencarian polis pada rantai unggah.
//
// Di basis data ia dua kueri berbeda — `GetReceiverClaimAsuransiKredit` atas T_GENERAL dan
// `BrowsePolisAso` atas JSON_POLIS. Di sini keduanya disatukan karena yang ditiru adalah
// HASILNYA.
//
// Keempat baris memperlihatkan empat keadaan yang masing-masing menghasilkan perilaku
// berbeda, dan ketiga yang terakhir mudah terlewat saat menguji dengan tangan:
//
//	polis lengkap                    -> baris lolos
//	perusahaan kosong                -> baris DITOLAK, tidak disimpan sama sekali
//	prodke kosong                    -> baris disimpan bertanda "No Polis tidak di temukan"
//	polis tidak terdaftar sama sekali -> sama dengan perusahaan kosong
//
// Empat polis tambahan (…600 sampai …603) menguji periode polis, produk hewan, polis
// batal, dan Open Protection.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Polis baris Kredit yang Sukses Klaim; bisnis dan sumber bisnisnya dipakai contoh
// tab Cek Premi (Total Klaim).
// Polis yang periodenya sudah lewat — tanggal kejadian 2026 berada di luarnya.
// Produk hewan (10166) dengan periode yang sudah lewat — ANEKA memeriksanya.
// Polis yang sudah dibatalkan.
// Polis ber-Open Protection tipe 3 — membebaskan premi belum lunas di Travel/ANEKA.
// Polis ada di T_GENERAL tetapi sumber bisnisnya tidak menunjuk perusahaan
// rekanan yang aktif.
// Polis dikenali perusahaan tetapi tidak ada di JSON_POLIS.
func SamplePolicy() map[string]PolicyRow {
	return sampledata.Must[map[string]PolicyRow](sampleJSON, "SamplePolicy")
}

// SampleLines adalah isi contoh POOLDATA.TMP_BATCH_AUTO_CLAIM.
//
// Empat batch yang sengaja memperlihatkan empat keadaan berbeda:
//
//	MFIN batch 2  seluruhnya selesai, ada yang berhasil dan ada yang gagal
//	MFIN batch 1  belum diproses sama sekali  -> kedua tombol ekspor menghasilkan berkas kosong
//	BPRC batch 1  baru diproses sebagian      -> "di upload" dan "telah diproses" berselisih
//	ZZZZ batch 1  kode perusahaan TIDAK ADA di master -> nama perusahaannya kosong
//
// Batch terakhir ada dengan sengaja. Ia yang membuktikan LEFT JOIN benar-benar berlaku —
// dengan INNER JOIN seperti kueri Pega aslinya, barisnya hilang dari layar, dan hilangnya
// tidak akan terlihat kecuali ada data seperti ini saat mencoba.
//
// CURRENCY diisi ID beserta kodenya, mengikuti bentuk aslinya: kolomnya menyimpan id, dan
// kode yang dibaca pengguna adalah hasil lookup ke POOLDATA.CURRENCY.
//
// Nomor polis, nilai, dan tanggalnya karangan. Tanggalnya berformat dd/mm/yyyy mengikuti
// bentuk yang dibaca Activity/CreateCasePNC_AutoClaim-Act.xml.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// MFIN batch 2 — selesai seluruhnya.
// Baris gagal: ketiga kolom penanda membawa teks galat yang sama, persis
// seperti yang dilakukan InsertKlaimToTable_Other.
// MFIN batch 1 — belum diproses sama sekali.
// BPRC batch 1 — baru sebagian diproses.
// ZZZZ batch 1 — perusahaan yang TIDAK ADA di master.
func SampleLines() []inboxautoclaim.Line {
	return sampledata.Must[[]inboxautoclaim.Line](sampleJSON, "SampleLines")
}

// SampleKreditLines adalah isi contoh POOLDATA.TMP_BATCH_CLAIM_KREDIT — tab Asuransi
// Kredit.
//
// Isinya SENGAJA berbeda dari tab ANEKA, bukan salinannya. Dengan isi yang sama, layar
// yang lupa mengirim `?sumber=` akan tampak benar di kedua tab — persis cacat yang paling
// mungkin terjadi dan paling sulit terlihat, karena tabel tetap terisi dan angkanya tetap
// masuk akal.
func SampleKreditLines() []inboxautoclaim.Line {
	return sampledata.Must[[]inboxautoclaim.Line](sampleJSON, "SampleKreditLines")
}

// SampleTravelLines adalah isi contoh POOLDATA.TMP_BATCH_AUTO_TRAVEL — tab Travel.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Travel banyak bervaluta asing; baris ini yang memperlihatkan kolom mata
// uang benar-benar dipakai, bukan selalu IDR.
func SampleTravelLines() []inboxautoclaim.Line {
	return sampledata.Must[[]inboxautoclaim.Line](sampleJSON, "SampleTravelLines")
}

// NewSampleRepo membentuk penyimpanan memori beserta isi contoh KETIGA tab.
//
// Ketiganya diisi, bukan hanya tab bawaan: tab yang kosong di lingkungan pengembangan
// tidak dapat dibedakan dari tab yang gagal memuat, dan keduanya tampak sama di layar.
func NewSampleRepo() *Repo {
	repo := NewRepo(SampleMaster(), SamplePolicy(), SampleLines()...)
	repo.Seed(inboxautoclaim.SourceKredit, SampleKreditLines()...)
	repo.Seed(inboxautoclaim.SourceTravel, SampleTravelLines()...)
	repo.SetBusiness(SampleBusiness())
	return repo
}

// SampleBusiness adalah isi contoh POOLDATA.BUSINESS (ID -> NOTE) untuk tab Cek Premi.
func SampleBusiness() map[string]string {
	return sampledata.Must[map[string]string](sampleJSON, "SampleBusiness")
}
