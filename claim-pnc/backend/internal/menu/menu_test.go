package menu_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/menu"
)

func parent(id int) *int { return &id }

// sample meniru bentuk POOLDATA.M_MENU_APLIKASI_PNC yang diterima: empat kelompok
// tingkat atas tanpa program, lalu butir-butir di bawahnya.
func sample() []menu.Item {
	return []menu.Item{
		{ID: 1, Description: "MASTER", Sequence: 1},
		{ID: 2, Description: "INBOX", Sequence: 2},
		{ID: 4, Description: "REPORT", Sequence: 4},
		{ID: 11, Description: "Master Status Klaim", Program: "StatusClaimInbox", ParentID: parent(1), Sequence: 1101},
		{ID: 12, Description: "Master Rekening", Program: "MasterRekening", ParentID: parent(1), Sequence: 1102},
		{ID: 23, Description: "Master Status Progress 1", Program: "StatusProgress", ParentID: parent(1), Sequence: 1113},
		{ID: 51, Description: "My Inbox", Program: "InboxRegister_Harness", ParentID: parent(2), Sequence: 1141},
		{ID: 83, Description: "Report Adjuster", ParentID: parent(4), Sequence: 1173},
	}
}

func TestGroupIsShownWhenAtLeastOneChildIsAllowed(t *testing.T) {
	tree := menu.BuildTree(sample(), []int{11})

	require.Len(t, tree, 1, "hanya kelompok MASTER yang punya anak terizinkan")
	require.Equal(t, "MASTER", tree[0].Description)
	require.Len(t, tree[0].Children, 1)
	require.Equal(t, "Master Status Klaim", tree[0].Children[0].Description)
}

// Isi M_OTORISASI_PNC yang diterima TIDAK memberi izin atas MENU_ID 1..4 sama sekali,
// padahal anak-anaknya diberi. Menuntut kelompoknya punya baris izin sendiri akan
// menghapus seluruh menu group IT.
func TestGroupNeedsNoGrantOfItsOwn(t *testing.T) {
	tree := menu.BuildTree(sample(), []int{11, 12, 23, 51})

	require.Len(t, tree, 2)
	require.Equal(t, "MASTER", tree[0].Description)
	require.Equal(t, "INBOX", tree[1].Description)
}

// Login `JONNY` diberi izin atas MENU_ID 4 (REPORT). Bila seluruh anaknya tersaring,
// judul kelompoknya tidak boleh tersisa sendirian.
func TestGrantedGroupWithoutVisibleChildIsHidden(t *testing.T) {
	tree := menu.BuildTree(sample(), []int{4})

	require.Empty(t, tree)
}

// MENU_ID 83 ada di master tanpa MENU_PROGRAM. Ia tetap tampil supaya kekosongan
// datanya terlihat, bukan terkubur.
func TestAllowedLeafWithoutProgramIsStillShown(t *testing.T) {
	tree := menu.BuildTree(sample(), []int{4, 83})

	require.Len(t, tree, 1)
	require.Equal(t, "REPORT", tree[0].Description)
	require.Len(t, tree[0].Children, 1)
	require.Equal(t, "Report Adjuster", tree[0].Children[0].Description)
	require.Empty(t, tree[0].Children[0].Program)
}

func TestMenuWithoutAnyGrantIsEmpty(t *testing.T) {
	require.Empty(t, menu.BuildTree(sample(), nil))
}

func TestOrderFollowsMenuSequence(t *testing.T) {
	// Sengaja diberikan dalam urutan terbalik: pengurutan adalah tugas BuildTree, bukan
	// tugas pemanggil — dan kueri yang mengurutkannya pun dapat berubah kelak.
	items := sample()
	for a, b := 0, len(items)-1; a < b; a, b = a+1, b-1 {
		items[a], items[b] = items[b], items[a]
	}

	tree := menu.BuildTree(items, []int{11, 12, 23, 51})

	require.Equal(t, []string{"MASTER", "INBOX"}, []string{tree[0].Description, tree[1].Description})
	require.Equal(t, []string{
		"Master Status Klaim", "Master Rekening", "Master Status Progress 1",
	}, []string{
		tree[0].Children[0].Description,
		tree[0].Children[1].Description,
		tree[0].Children[2].Description,
	})
}

// Urutan langkah ditetapkan Work Owner: dari login yang diketik, cari GROUP_ID-nya,
// lalu cari izin untuk group-group itu DAN untuk loginnya sendiri.
func TestSubjectsCombinesGroupsAndLogin(t *testing.T) {
	require.Equal(t, []string{"IT", "JONNY"}, menu.Subjects("JONNY", []string{"IT"}))
}

func TestSubjectsWorksWithoutAnyGroup(t *testing.T) {
	require.Equal(t, []string{"JONNY"}, menu.Subjects("JONNY", nil))
}

// Kolomnya VARCHAR2 tanpa penyeragaman apa pun. Satu spasi di ujung akan membuat baris
// yang sah tidak pernah cocok, dan kegagalannya diam — menunya hanya kosong.
func TestSubjectsTrimsAndNormalisesCase(t *testing.T) {
	require.Equal(t, []string{"IT", "JONNY"}, menu.Subjects("  jonny ", []string{" it "}))
}

func TestSubjectsDropsDuplicatesAndBlanks(t *testing.T) {
	require.Equal(t, []string{"IT", "JONNY"}, menu.Subjects("JONNY", []string{"IT", "it", "", "   ", "JONNY"}))
}
