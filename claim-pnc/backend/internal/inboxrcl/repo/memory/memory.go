// Package memory memenuhi seam inboxrcl.Repo dengan penyimpanan di memori.
//
// Dipakai pengujian aturan modul tanpa basis data dan pengembangan lokal saat `PENYIMPANAN`
// tidak menunjuk basis data mana pun.
//
// # Kenapa penyaring dan urutannya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Seluruhnya ditiru apa adanya:
//
//	identitas lama hanya dari grup Administrators/PNCKomite/CaseManager yang aktif
//	pxAssignedOperatorID          = identitas lama      penyaring A
//	pyStatusWork                 != Resolved-Completed  penyaring B
//	TanggalAnalystSendRCL IS NOT NULL                   penyaring C
//	NamaDokterRCL                 = identitas lama      penyaring D
//	urutan pxCreateDateTime DESC, pyID DESC
package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/inboxrcl"
)

// AccessRow adalah satu baris `POOLDATA.T_ACCESS_GROUP_PNC` — satu grup akses satu orang.
type AccessRow struct {
	Login       string
	LegacyID    string
	AccessGroup string
	Active      bool
}

// Store adalah pembaca antrean RCL Dokter di memori.
type Store struct {
	access []AccessRow
	tasks  []inboxrcl.RCLTask
}

// NewStore membentuk pembaca dari baris yang diberikan.
func NewStore(access []AccessRow, tasks []inboxrcl.RCLTask) *Store {
	return &Store{access: access, tasks: tasks}
}

// LegacyOperatorFor meniru `legacy_operator_for` — MAX atas baris yang lolos.
func (s *Store) LegacyOperatorFor(_ context.Context, loginID string) (string, error) {
	id := normalize(loginID)
	if id == "" {
		return "", nil
	}

	best := ""
	for _, row := range s.access {
		if normalize(row.Login) != id || !row.Active || !allowedGroup(row.AccessGroup) {
			continue
		}
		if legacy := normalize(row.LegacyID); legacy > best {
			best = legacy
		}
	}
	return best, nil
}

// allowedGroup meniru `ACCESS_GROUP IN (…) AND ACCESS_GROUP <> 'GCNMFW:ViewClaimPNC'`.
func allowedGroup(group string) bool {
	if group == inboxrcl.ExcludedAccessGroup {
		return false
	}
	for _, allowed := range inboxrcl.LegacyAccessGroups {
		if group == allowed {
			return true
		}
	}
	return false
}

// List mengambil satu halaman antrean milik sebuah identitas lama.
func (s *Store) List(
	_ context.Context,
	operator string,
	f inboxrcl.Filter,
) (inboxrcl.Page, error) {
	clean := f.Normalize()

	matched := []inboxrcl.RCLTask{}
	for _, task := range s.tasks {
		if matches(task, operator, clean.Search) {
			matched = append(matched, task)
		}
	}

	sortTasks(matched)

	result := inboxrcl.Page{Tasks: []inboxrcl.RCLTask{}, Total: len(matched)}
	if clean.Offset >= len(matched) {
		return result, nil
	}

	end := clean.Offset + clean.Limit
	if end > len(matched) {
		end = len(matched)
	}
	result.Tasks = matched[clean.Offset:end]
	return result, nil
}

// matches meniru keempat penyaring Report Definition ditambah kotak cari.
//
// Identitas dibandingkan tanpa peka huruf besar-kecil, sama seperti `UPPER(...)` di kueri.
func matches(task inboxrcl.RCLTask, operator, search string) bool {
	op := normalize(operator)
	if op == "" {
		return false
	}
	if normalize(task.AssignedOperator) != op {
		return false
	}
	if strings.TrimSpace(task.ProcessStatus) == inboxrcl.StatusKerjaSelesai {
		return false
	}
	if task.SentToRCLAt.IsZero() {
		return false
	}
	if normalize(task.RCLDoctor) != op {
		return false
	}

	needle := strings.ToUpper(strings.TrimSpace(search))
	if needle == "" {
		return true
	}
	for _, field := range []string{task.ClaimNumber, task.PolicyNumber} {
		if strings.Contains(strings.ToUpper(field), needle) {
			return true
		}
	}
	return false
}

// sortTasks mengurutkan seperti kuerinya: pxCreateDateTime MENURUN, pyID MENURUN sebagai
// pemutus seri.
func sortTasks(items []inboxrcl.RCLTask) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if left.RegisteredAt.Equal(right.RegisteredAt) {
			return left.ClaimNumber > right.ClaimNumber
		}
		return left.RegisteredAt.After(right.RegisteredAt)
	})
}

func normalize(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
