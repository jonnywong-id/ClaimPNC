package reportklaim

import (
	"encoding/json"
	"strings"
)

// ProgressNames memetakan kode tahapan progres ke namanya.
//
// Sumbernya `POOLDATA.GCNM_MST_PROGRESS`, kolom `ID_MST` → `STS_PROGRESS2`.
type ProgressNames struct {
	name   map[string]string
	loaded bool
}

// NewProgressNames membentuk peta dari master.
func NewProgressNames(name map[string]string) *ProgressNames {
	if name == nil {
		name = map[string]string{}
	}
	return &ProgressNames{name: name, loaded: true}
}

// UnavailableProgressNames adalah peta yang sumbernya tidak dapat dibaca.
func UnavailableProgressNames() *ProgressNames { return &ProgressNames{} }

// Available menyatakan apakah peta ini terisi dari sumbernya.
func (p *ProgressNames) Available() bool { return p != nil && p.loaded }

// Count mengembalikan banyaknya tahapan yang dikenal.
func (p *ProgressNames) Count() int {
	if p == nil {
		return 0
	}
	return len(p.name)
}

// progressDocument adalah bentuk JSON yang dibaca.
//
// Hanya satu field yang diambil dari setiap objek, dan namanya MENYESATKAN: `BranchName`
// berisi kode tahapan progres, bukan nama cabang. Nama itu tidak diubah di sini karena ia
// nama field pada data yang sudah tersimpan; yang dapat dilakukan adalah menyebutnya.
type progressDocument struct {
	ObjectList []struct {
		BranchName string `json:"BranchName"`
	} `json:"ObjectList"`
}

// Positions menyusun daftar nama tahapan progres dari satu dokumen JSON.
//
// # Aturannya, dibaca dari GET_POSISI_PROGRESS2.fnc
//
// Fungsi aslinya mengurai `JSONSTATUS_PROGRESS2`, mengambil `ObjectList[].BranchName`,
// mencari namanya di `GCNM_MST_PROGRESS`, lalu merangkainya dipisah `", "` menurut urutan
// kemunculan di dalam JSON.
//
// # Tiga hal yang diperlakukan berbeda, dan sebabnya
//
//  1. **Kode yang tidak ada di master.** Sistem lama memakai `SELECT … INTO`, sehingga
//     kode asing melempar `NO_DATA_FOUND` dan **menggagalkan seluruh laporan** — satu
//     baris rusak membuat puluhan ribu baris lain tidak terunduh. Di sini kodenya
//     dikeluarkan apa adanya. Ia terlihat tidak wajar di antara nama-nama tahapan, dan itu
//     memang yang dikehendaki: kekurangan master terbaca, tanpa menjatuhkan laporan.
//
//  2. **JSON yang tidak dapat diurai.** Sistem lama melempar galat parse. Di sini selnya
//     kosong. Alasannya sama.
//
//  3. **Dua parameter pertama yang tidak dipakai.** `p_claimno` dan `id` diterima fungsi
//     aslinya lalu **tidak pernah disentuh**. Keduanya tidak dibawa ke sini — ini kelas
//     yang sama dengan `TTGLPLADLA` pada `D-49` butir 7: parameter yang diterima lalu
//     dibuang bukan cacat yang harus ditiru, melainkan sisa yang boleh dihapus.
//
// # Yang TIDAK diubah
//
// Duplikat tidak dibuang dan urutannya tidak diubah. Dua objek bertahapan sama
// menghasilkan nama yang sama dua kali, persis seperti sistem lama.
func (p *ProgressNames) Positions(rawJSON string) string {
	if !p.Available() || strings.TrimSpace(rawJSON) == "" {
		return ""
	}

	var doc progressDocument
	if err := json.Unmarshal([]byte(rawJSON), &doc); err != nil {
		return ""
	}

	parts := make([]string, 0, len(doc.ObjectList))
	for _, item := range doc.ObjectList {
		if item.BranchName == "" {
			continue
		}
		if name, ada := p.name[item.BranchName]; ada {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, item.BranchName)
	}
	return strings.Join(parts, ", ")
}
