package inboxadmin

import (
	"context"
	"strings"
)

// Batas data cabang dan kanwil layar Inbox Admin.
//
// # Aturan sistem lama
//
// `Activity/SetTempClaimRegistandNotRegist` (isi tab) dan `GetReportClaimRegistList`
// (daftar Status Register) menyusun dua potongan penyaring yang ditempelkan ke kueri:
//
//	langkah 2  manajer + kanwil dipilih  AND a.KODECABANG_1 IN (SELECT ID FROM branch
//	                                          WHERE basterritory = <kanwil>)
//	langkah 8  BUKAN manajer, cabang     AND a.KODECABANG_1 IN (SELECT ID FROM branch
//	           bukan kantor pusat             WHERE ID = '<cabang petugas>')
//
// "Manajer" adalah access group `GCNMFW:CaseManager` atau `GCNMFW:PncManagerAdmin`.
// Cabang petugas dibaca `GetIDCabang` lewat DB Link HRD; kantor pusat (`100081`) dan
// cabang yang tidak terbaca (kosong) MELEWATI penyaring cabang — keduanya syarat eksplisit
// pada prakondisi langkah 8, bukan akibat perangkaian teks.
//
// Kueri mana yang memakai potongan mana dibaca dari `{ASIS:...}` di setiap RDB rule:
// cabang dipakai tab Outstanding, Unregistered RCV (dan Online), LOD, Request Survey, dan
// All Case Admin; kanwil dipakai Outstanding, Unregistered RCV (dan Online), dan LOD.
//
// # Sejak kapan berlaku
//
// Sempat ditunda (keputusan Work Owner 2026-09-20) sampai API pengganti DB Link HRD ada.
// Dipasang 2026-10-07 atas permintaan Work Owner, memakai DB Link yang sama dengan sistem
// lama di balik seam ScopeRepo — pola yang sudah dipakai modul Inbox Laporan Klaim. Peran
// dibaca dari POOLDATA.M_LOGIN_GROUP_PNC (keputusan Work Owner 2026-09-25).

// HeadOfficeBranch adalah kode cabang kantor pusat, yang melewati penyaring cabang.
const HeadOfficeBranch = "100081"

// managerGroups adalah access group yang melewati penyaring cabang dan boleh memilih
// kanwil — huruf kecil, tanpa awalan `GCNMFW:`.
var managerGroups = map[string]bool{"casemanager": true, "pncmanageradmin": true}

// IsManager menyatakan salah satu group termasuk manajer.
//
// Kapitalisasi dan awalan `GCNMFW:` diabaikan: barisnya diisi manusia, dan Pega sendiri
// tidak konsisten kapitalisasinya (`D-58`).
func IsManager(groups []string) bool {
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if i := strings.LastIndex(g, ":"); i >= 0 {
			g = g[i+1:]
		}
		if managerGroups[strings.ToLower(strings.TrimSpace(g))] {
			return true
		}
	}
	return false
}

// Scope adalah batas data yang berlaku pada satu permintaan.
type Scope struct {
	// BranchCode menyaring KODECABANG_1 ke satu cabang; kosong = tidak menyaring.
	BranchCode string

	// RegionCode menyaring KODECABANG_1 ke cabang-cabang satu kanwil; kosong = tidak.
	RegionCode string
}

// Viewer adalah keterangan batas data pemanggil, untuk layar.
type Viewer struct {
	// Manager menyatakan pemanggil boleh memilih kanwil dan tidak dibatasi cabang.
	Manager bool

	// BranchCode adalah cabang yang membatasi pemanggil; kosong bila tidak dibatasi.
	BranchCode string
}

// ResolveScope menerapkan aturan di atas.
//
// regionChoice hanya berlaku bagi manajer: langkah 2 Pega memasang penyaring kanwil HANYA
// bila access group-nya manajer. Pilihan kanwil dari petugas lain diabaikan, dan layar
// karena itu hanya menggambar dropdown-nya bagi manajer — dropdown yang tidak mengubah
// apa pun lebih menyesatkan daripada tidak ada.
func ResolveScope(manager bool, branch string, regionChoice string) (Scope, Viewer) {
	if manager {
		return Scope{RegionCode: strings.TrimSpace(regionChoice)}, Viewer{Manager: true}
	}
	branch = strings.TrimSpace(branch)
	if branch == "" || branch == HeadOfficeBranch {
		return Scope{}, Viewer{}
	}
	return Scope{BranchCode: branch}, Viewer{BranchCode: branch}
}

// ScopeRepo membaca bahan batas data: cabang petugas, access group-nya, dan daftar kanwil.
type ScopeRepo interface {
	// BranchOfLogin mengembalikan kode cabang (POOLDATA.BRANCH.ID) petugas. found=false
	// berarti petugasnya tidak terdaftar — diperlakukan sama dengan cabang kosong.
	BranchOfLogin(ctx context.Context, login string) (code string, found bool, err error)

	// GroupsOf mengembalikan access group login itu.
	GroupsOf(ctx context.Context, login string) ([]string, error)

	// Regions mengembalikan isi dropdown "Pilih Kanwil".
	Regions(ctx context.Context) ([]string, error)
}
