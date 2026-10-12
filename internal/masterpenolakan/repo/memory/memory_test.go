package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenolakan/repo/memory"
)

func submission(name, parentID, parentName string) masterpenolakan.Submission {
	return masterpenolakan.Submission{
		Input: masterpenolakan.Input{Name: name, ParentID: parentID, ParentName: parentName},
		By:    "adminpnc",
		At:    time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC),
	}
}

// INILAH cacat yang diperbaiki, dinyatakan sebagai uji.
//
// `MASTERPENOLAKANKLAIM1.prc` menyisipkan baris baru pada KEDUA cabang IF-nya, sehingga
// memilih induk yang sudah ada pun menerbitkan duplikat. Work Owner memutuskan
// memperbaikinya pada 2026-09-19; uji ini yang menjaga perbaikannya tidak hilang.
func TestChoosingExistingParentDoesNotCreateDuplicateParent(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents())
	ctx := context.Background()

	before, err := repo.ListParent(ctx)
	require.NoError(t, err)

	_, err = repo.InsertNew(ctx, submission("PREMI BELUM DIBAYAR", "1", ""))
	require.NoError(t, err)

	after, err := repo.ListParent(ctx)
	require.NoError(t, err)
	require.Len(t, after, len(before),
		"memilih Status Penolakan 1 yang sudah ada TIDAK boleh menerbitkan baris baru")
}

func TestRequestingNewParentCreatesExactlyOneParent(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents())
	ctx := context.Background()

	before, err := repo.ListParent(ctx)
	require.NoError(t, err)

	saved, err := repo.InsertNew(ctx, submission("KLAIM DI LUAR WILAYAH", "", "WILAYAH TIDAK DIJAMIN"))
	require.NoError(t, err)

	after, err := repo.ListParent(ctx)
	require.NoError(t, err)
	require.Len(t, after, len(before)+1)

	// Nama induk DISALIN ke baris tingkat 2, bukan dibiarkan kosong — kolom NOTE_ST
	// memang ditulis `MASTERPENOLAKANKLAIM2.prc:9`.
	require.Equal(t, "WILAYAH TIDAK DIJAMIN", saved.ParentName)
	require.NotEmpty(t, saved.ParentID)
}

func TestUnknownParentIsRejectedWithItsOwnError(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents())

	_, err := repo.InsertNew(context.Background(), submission("X", "999", ""))
	require.ErrorIs(t, err, masterpenolakan.ErrParentNotFound,
		"induk yang hilang harus dibedakan dari baris yang hilang: yang satu isian, yang lain sumber daya")
}

func TestNewRowAlwaysStartsWaitingForApproval(t *testing.T) {
	// Status baris baru SELALU menunggu, tidak pernah dari pemanggil — itu yang ditulis
	// `MASTERPENOLAKANKLAIM2.prc:10`. Yang berwenang mengubahnya adalah layar Inbox
	// Manager, bukan layar ini.
	repo := memory.NewRepo(memory.SampleParents())

	saved, err := repo.InsertNew(context.Background(), submission("X", "1", ""))
	require.NoError(t, err)

	require.Equal(t, masterpenolakan.StatusPending, saved.Status)
	require.Empty(t, saved.ApprovedBy)
	require.Nil(t, saved.ApprovedAt)
}

func TestSubmitterAndTimeComeFromServerNotFromInput(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents())
	at := time.Date(2026, 9, 19, 10, 30, 0, 0, time.UTC)

	saved, err := repo.InsertNew(context.Background(), masterpenolakan.Submission{
		Input: masterpenolakan.Input{Name: "X", ParentID: "1"},
		By:    "pictekniks",
		At:    at,
	})
	require.NoError(t, err)

	require.Equal(t, "pictekniks", saved.SubmittedBy)
	require.Equal(t, at, saved.SubmittedAt)
	require.Equal(t, time.UTC, saved.SubmittedAt.Location(), "waktu disimpan UTC")
}

// Pengubahan MENGEMBALIKAN baris ke antrean persetujuan.
//
// `MASTERPENOLAKANKLAIM2.prc:14` menyetel STATUS='0' dan TANGGALKIRIM=sysdate pada setiap
// pengubahan. Perilaku itu dipertahankan atas keputusan Work Owner 2026-09-19.
func TestUpdateSendsRowBackToApprovalQueue(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	ctx := context.Background()

	before, err := repo.Get(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.StatusApproved, before.Status, "prasyarat uji: baris ini sudah disetujui")

	after, err := repo.Update(ctx, "1", submission("TEKS BARU", "1", ""))
	require.NoError(t, err)

	require.Equal(t, masterpenolakan.StatusPending, after.Status)
	require.Equal(t, "TEKS BARU", after.Name)
}

func TestUpdateKeepsPreviousApprovalTrace(t *testing.T) {
	// Ketiga kolom persetujuan TIDAK ikut dibersihkan, persis seperti procedure lama.
	// Akibatnya baris berstatus MENUNGGU masih memuat nama penyetuju sebelumnya — jejak
	// keputusan yang pernah ada, bukan keadaan yang berlaku.
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	ctx := context.Background()

	before, err := repo.Get(ctx, "1")
	require.NoError(t, err)
	require.NotEmpty(t, before.ApprovedBy)

	after, err := repo.Update(ctx, "1", submission("TEKS BARU", "1", ""))
	require.NoError(t, err)

	require.Equal(t, before.ApprovedBy, after.ApprovedBy)
	require.Equal(t, before.ApprovedAt, after.ApprovedAt)
	require.Equal(t, before.ApprovalNote, after.ApprovalNote)
}

func TestUpdateNeverChangesTheKey(t *testing.T) {
	// ID_ND adalah kunci, dan procedure lama pun hanya memakainya sebagai penyaring WHERE.
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)

	after, err := repo.Update(context.Background(), "2", submission("TEKS BARU", "1", ""))
	require.NoError(t, err)
	require.Equal(t, "2", after.ID)
}

func TestUpdateOnMissingRowDoesNotCreateOrphanParent(t *testing.T) {
	// Induk diselesaikan SESUDAH barisnya ditemukan. Kalau dibalik, permintaan yang
	// menyebut ID tidak dikenal akan sempat menerbitkan baris tingkat 1 baru yang tidak
	// pernah dipakai siapa pun.
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	ctx := context.Background()

	before, err := repo.ListParent(ctx)
	require.NoError(t, err)

	_, err = repo.Update(ctx, "999", submission("X", "", "INDUK BARU"))
	require.ErrorIs(t, err, masterpenolakan.ErrNotFound)

	after, err := repo.ListParent(ctx)
	require.NoError(t, err)
	require.Len(t, after, len(before))
}

func TestListIsOrderedByParentThenRow(t *testing.T) {
	// `ORDER BY ID_ST ASC, ID_ND ASC` — pengurutan TEKS karena kedua kolomnya bertipe
	// teks. Ditiru apa adanya supaya urutan di layar tidak berubah saat berpindah dari
	// memori ke Oracle.
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, list)

	for i := 1; i < len(list); i++ {
		previous, current := list[i-1], list[i]
		if previous.ParentID == current.ParentID {
			require.LessOrEqual(t, previous.ID, current.ID)
			continue
		}
		require.Less(t, previous.ParentID, current.ParentID)
	}
}

func TestSeamProvidesNoDeleteOperation(t *testing.T) {
	// Pernyataan yang disengaja, bukan uji perilaku: seluruh export tidak memuat satu pun
	// DELETE terhadap ketiga tabel modul ini, dan tidak satu pun punya kolom penanda
	// terhapus yang dapat dipakai `D-66`. Bila kelak ada yang menambahkan Delete ke seam,
	// baris ini yang membuatnya terlihat sebagai keputusan.
	var repo masterpenolakan.Repo = memory.NewRepo(memory.SampleParents())
	require.NotNil(t, repo)

	var komite masterpenolakan.RepoKomite = memory.NewRepoKomite()
	require.NotNil(t, komite)
}

func TestFirstCommitteeRowOnEmptyTableIsNumberedOneHundredEleven(t *testing.T) {
	// Keanehan yang direplikasi dari `INSERTMASTERREJECTEDKOMITE.prc:9`. Ia hanya berlaku
	// sekali seumur tabel, dan justru karena itu mudah terlewat.
	repo := memory.NewRepoKomite()

	saved, err := repo.InsertNew(context.Background(), masterpenolakan.InputKomite{Note: "X"})
	require.NoError(t, err)
	require.Equal(t, "111", saved.ID)
}

func TestNextCommitteeRowContinuesFromHighest(t *testing.T) {
	repo := memory.NewRepoKomite(memory.SampleListKomite()...)

	saved, err := repo.InsertNew(context.Background(), masterpenolakan.InputKomite{Note: "X"})
	require.NoError(t, err)
	require.Equal(t, "114", saved.ID)
}

func TestCommitteeUpdateChangesOnlyTheNote(t *testing.T) {
	repo := memory.NewRepoKomite(memory.SampleListKomite()...)

	after, err := repo.Update(context.Background(), "112", masterpenolakan.InputKomite{Note: "TEKS BARU"})
	require.NoError(t, err)
	require.Equal(t, "112", after.ID)
	require.Equal(t, "TEKS BARU", after.Note)
}

func TestCommitteeUpdateOnMissingRowIsReported(t *testing.T) {
	repo := memory.NewRepoKomite(memory.SampleListKomite()...)

	_, err := repo.Update(context.Background(), "999", masterpenolakan.InputKomite{Note: "X"})
	require.ErrorIs(t, err, masterpenolakan.ErrKomiteNotFound,
		"galatnya milik tab komite, bukan tab sebelah — pesannya menyebut master yang benar")
}

func TestCommitteeListIsOrderedNumerically(t *testing.T) {
	// `ORDER BY IDMASTER` di sini pengurutan ANGKA, bukan teks: kolomnya NUMBER. Bedanya
	// terlihat begitu tabel memuat lebih dari sembilan baris — "111" sebelum "9" secara
	// teks, tetapi sesudahnya secara angka.
	repo := memory.NewRepoKomite(
		masterpenolakan.CommitteeRejection{ID: "111", Note: "A"},
		masterpenolakan.CommitteeRejection{ID: "9", Note: "B"},
	)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "9", list[0].ID)
	require.Equal(t, "111", list[1].ID)
}

func TestEachPortalKeepsItsOwnRows(t *testing.T) {
	// Setiap portal mendapat instans sendiri. Menyatukannya akan menyembunyikan kelas
	// cacat yang paling ingin dicegah ADR-0030 dan R-20.
	asm := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	asi := memory.NewRepo(nil)
	ctx := context.Background()

	_, err := asm.InsertNew(ctx, submission("HANYA MILIK ASM", "1", ""))
	require.NoError(t, err)

	other, err := asi.List(ctx)
	require.NoError(t, err)
	require.Empty(t, other, "baris satu entitas tidak boleh terlihat dari entitas lain")
}
