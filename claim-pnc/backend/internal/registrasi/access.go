package registrasi

import (
	"context"
	"strings"
)

// # Siapa boleh mengerjakan sebuah tugas
//
// Tugas Worklist punya pemilik — orang yang dipilih router saat tugas lahir. Work Owner
// menetapkan (2026-09-28) bahwa tugas juga boleh dikerjakan pengguna yang memegang GRUP
// tahap itu, dibaca dari POOLDATA.M_LOGIN_GROUP_PNC:
//
//	PNCAdminRouter   → PncAdmin          (Input Register, Input Estimasi)
//	PNCTeknikRouter  → PNCKomiteTeknik   (Choose Surveyor, Send To PIC Teknik, Send To Analis)
//
// `PNCKomiteTeknik` dianggap PIC Teknik atas keputusan Work Owner. Access group Pega untuk
// PIC Teknik sebenarnya `PncPICTeknik`, tetapi grup itu belum ada di tabel.
//
// Tahap antrean bersama tetap diatur Workbasket-nya (diambil lebih dulu), dan tahap yang
// dirutekan ke orang bernama atau ke pemanggil tidak punya grup.

// GroupPrefix adalah awalan ruleset access group Pega. M_LOGIN_GROUP_PNC menyimpan nama
// grup tanpa awalan (`PncAdmin`), sedangkan peran di alur memakai literal Pega
// (`GCNMFW:PncAdmin`); keduanya disamakan saat grup dibaca.
const GroupPrefix = "GCNMFW:"

// Grup penentu kewenangan tahap.
const (
	RoleAdmin      = GroupPrefix + "PncAdmin"
	RoleTechnicPIC = GroupPrefix + "PNCKomiteTeknik"
)

// routerRole memetakan router tahap ke grup yang boleh mengerjakannya.
var routerRole = map[string]string{
	RouterPNCAdmin:     RoleAdmin,
	RouterPNCTechnical: RoleTechnicPIC,
}

// GroupSource adalah seam ke keanggotaan grup pengguna (POOLDATA.M_LOGIN_GROUP_PNC).
type GroupSource interface {
	// GroupsOf mengembalikan GROUP_ID seorang login, apa adanya.
	GroupsOf(ctx context.Context, login string) ([]string, error)
}

// RolesOfGroups mengubah GROUP_ID menjadi peran berawalan ruleset, tanpa duplikat.
func RolesOfGroups(groups []string) []string {
	seen := map[string]bool{}
	var roles []string
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(g), strings.ToUpper(GroupPrefix)) {
			g = GroupPrefix + g
		}
		key := strings.ToUpper(g)
		if !seen[key] {
			seen[key] = true
			roles = append(roles, g)
		}
	}
	return roles
}

// StageRole adalah grup yang boleh mengerjakan tahap ini; kosong bila tahap tidak diatur
// grup.
func StageRole(s Stage) string {
	if s.Queue != QueueWorklist {
		return ""
	}
	return routerRole[s.Router]
}

// RoleAnalyst adalah When `IsAnalisator`: operator anggota workgroup `KlaimAnalisator`
// (`@Utilities.countInPageList("KlaimAnalisator","pyWorkGroupName",OperatorID.pyWorkGroupList) > 0`).
// Di sini dibaca dari M_LOGIN_GROUP_PNC seperti grup lain; per 2026-10-03 belum ada satu pun
// anggota grup itu di tabel.
const RoleAnalyst = GroupPrefix + "KlaimAnalisator"

// IsAnalyst menyatakan pemegang peran-peran ini anggota grup Analyst (When `IsAnalisator`).
func IsAnalyst(roles []string) bool {
	return FlowContext{CallerRoles: roles}.HasRole(RoleAnalyst)
}

// CanWork menyatakan seseorang boleh mengerjakan tugas: tugasnya belum bertuan, ia
// pemiliknya, atau ia memegang grup tahap itu.
func CanWork(task Task, stage Stage, identity string, roles []string) bool {
	if !task.Owned() || equalFold(task.Owner, identity) {
		return true
	}
	role := StageRole(stage)
	if role == "" {
		return false
	}
	return FlowContext{CallerRoles: roles}.HasRole(role)
}

// GroupStages adalah tahap yang boleh dikerjakan pemegang peran-peran ini — dipakai Inbox
// untuk menampilkan tugas grup, bukan hanya tugas milik sendiri.
func (d Definition) GroupStages(roles []string) []string {
	fctx := FlowContext{CallerRoles: roles}
	var result []string
	for _, s := range d.Stages() {
		if role := StageRole(s); role != "" && fctx.HasRole(role) {
			result = append(result, s.ID)
		}
	}
	return result
}
