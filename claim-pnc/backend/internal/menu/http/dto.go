// Package menuhttp adalah lapisan transport modul menu.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `menuhttp` supaya tidak menutupi `net/http`.
package menuhttp

import "claim-pnc/internal/menu"

// ItemDTO adalah satu butir menu yang dikirim ke peramban.
//
// Terpisah dari menu.Node supaya perubahan internal tidak bocor ke klien dan sebaliknya
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
type ItemDTO struct {
	// ID adalah MENU_ID. Dikirim supaya layar punya kunci yang stabil untuk setiap
	// butir — nama menu dapat berubah, nomornya tidak.
	ID int `json:"id"`

	Nama string `json:"nama"`

	// Program adalah MENU_PROGRAM: nama harness Pega yang dituju butir ini.
	//
	// Dikirim APA ADANYA, termasuk ketika kosong. Yang memutuskan butir ini dapat
	// diklik atau belum adalah frontend, karena hanya ia yang tahu layar mana yang
	// sudah dibangun. Backend tidak menyimpan peta rute antarmuka.
	Program string `json:"program"`

	// Submenu selalu ada sebagai senarai, tidak pernah null — layar yang menerima
	// `null` harus menjaganya sendiri, dan satu layar yang lupa akan gagal pada butir
	// yang tidak punya anak.
	Submenu []ItemDTO `json:"submenu"`
}

// ListResponse adalah jawaban GET /api/menu.
type ListResponse struct {
	Menu []ItemDTO `json:"menu"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul auth — `{kode, pesan}`. Klien membedakan jenis galat
// lewat `kode`, tidak pernah dengan mencocokkan teks `pesan`.
type ErrorResponse struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`
}

// toDTO mengubah satu simpul domain beserta anak-anaknya.
func toDTO(node menu.Node) ItemDTO {
	return ItemDTO{
		ID:      node.ID,
		Nama:    node.Description,
		Program: node.Program,
		Submenu: toListDTO(node.Children),
	}
}

// toListDTO mengubah sekumpulan simpul domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya menu kosong terkirim
// sebagai `[]` dan bukan `null`.
func toListDTO(nodes []menu.Node) []ItemDTO {
	result := make([]ItemDTO, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, toDTO(node))
	}
	return result
}
