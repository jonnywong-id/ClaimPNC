package memory

import (
	_ "embed"

	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SamplePrincipal adalah isi master Virtual Account untuk dipakai tanpa Oracle.
//
// # Kenapa isinya dikarang, bukan disalin
//
// Kedua baris nyata pada POOLDATA.MST_VIRTUAL_ACCOUNT_PNC portal ASM memuat nomor
// rekening virtual dan alamat surel pegawai yang benar-benar ada. `D-69` menetapkan data
// seperti itu TIDAK PERNAH ditulis ke berkas yang di-commit, dan aturan itu berlaku untuk
// kode contoh sama seperti untuk dokumen.
//
// Yang ditiru karena itu adalah BENTUKNYA — panjang nomor VA, pola Client ID, dan
// keadaan STATUS serta MESSAGE yang keduanya kosong pada baris nyata — bukan nilainya.
// Nomor di bawah sengaja berawalan yang jelas bukan awalan bank mana pun.
func SamplePrincipal() []masterrecovery.Principal {
	return sampledata.Must[[]masterrecovery.Principal](sampleJSON, "SamplePrincipal")
}

// SamplePolicy adalah acuan polis untuk mencoba pencarian identitas tanpa DB Link.
//
// Di produksi keempat nilainya dibaca dari MST_DET_SALES@ASMD, dan link itu tidak ada di
// lingkungan pengembangan. Tanpa contoh ini, bagian layar yang mencari identitas polis
// tidak dapat dicoba sama sekali.
//
// Satu nomor sengaja TIDAK didaftarkan: apa pun selain kedua nomor di bawah akan menjawab
// "polis tidak ditemukan", sehingga perilaku penolakannya ikut teruji saat pengembangan —
// bukan baru ditemukan di produksi.
func SamplePolicy() map[string]masterrecovery.PolicyReference {
	return sampledata.Must[map[string]masterrecovery.PolicyReference](sampleJSON, "SamplePolicy")
}

// NewSampleRepo membentuk repo berisi seluruh contoh di atas.
func NewSampleRepo() *Repo {
	return NewRepo(SamplePrincipal(), SamplePolicy())
}
