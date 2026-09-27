package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxkomunikasicabang/repo/memory"
)

// messageTo menyusun perintah pesan baru.
func messageTo(destination, branch string) inboxkomunikasicabang.NewMessageCommand {
	command, err := inboxkomunikasicabang.NewMessageCommandOf(
		inboxkomunikasicabang.NewMessageInput{
			Destination: destination,
			BranchCode:  branch,
			Message:     "Mohon konfirmasi kelengkapan dokumen.",
		},
		inboxkomunikasicabang.Caller{Login: "pictekniks", Name: "PIC Teknik Surabaya"},
		time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC),
	)
	if err != nil {
		panic("uji menyusun pesan yang tidak sah: " + err.Error())
	}
	return command
}

func TestTheBranchListIsNotFilteredByTheCallerBranch(t *testing.T) {
	// Yang dibatasi adalah PERCAKAPAN, bukan daftar cabang. Petugas cabang 1001 boleh
	// mengirim pesan ke cabang mana pun — yang tidak boleh adalah MEMBACA percakapan cabang
	// lain dengan pihak ketiga.
	//
	// `1003` sengaja tidak dimiliki login contoh mana pun, dan ia HARUS tetap dapat dipilih.
	store := memory.NewSampleStore()

	branches, err := store.Branches(context.Background())
	require.NoError(t, err)

	codes := []string{}
	for _, branch := range branches {
		codes = append(codes, branch.Code)
	}
	require.Contains(t, codes, "1003")
}

func TestTheBranchListIsOrderedByName(t *testing.T) {
	// Urutannya ditiru dari `ORDER BY BRANCHNAME ASC` pada kueri SQL. Penyimpanan memori yang
	// mengembalikan urutan berbeda akan membuat uji urutan lulus di sini dan gagal di Oracle.
	store := memory.NewSampleStore()

	branches, err := store.Branches(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, branches)

	for i := 1; i < len(branches); i++ {
		require.LessOrEqual(t, branches[i-1].Name, branches[i].Name)
	}
}

func TestAMessageFromHeadOfficeReachesTheChosenBranch(t *testing.T) {
	// Arah percakapan ditentukan asal DAN tujuan, dan keduanya harus benar sekaligus:
	// pesan dari pusat ke cabang 1002 berasal dari `"1"` dan menuju `"1002"`.
	//
	// Yang diuji di sini BUKAN nilainya melainkan akibatnya — barisnya harus terbaca oleh
	// cabang tujuan, dan itu dibuktikan lewat daftarnya sendiri.
	store := memory.NewSampleStore()

	id, err := store.SendMessage(context.Background(),
		messageTo("CABANG", "1002"), inboxkomunikasicabang.HeadOfficeCode)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	branch1002 := inboxkomunikasicabang.ResolveBranch("1002", true)
	require.Contains(t, idsOf(listVia(t, store, "1", branch1002)), id)
}

func TestAMessageFromABranchReachesHeadOffice(t *testing.T) {
	store := memory.NewSampleStore()

	id, err := store.SendMessage(context.Background(), messageTo("PUSAT", ""), "1001")
	require.NoError(t, err)

	require.Contains(t, idsOf(listVia(t, store, "1", headOffice())), id)

	// Dan pengirimnya sendiri melihatnya pula — penyaringnya `OR`, bukan `AND`.
	require.Contains(t, idsOf(listVia(t, store, "1", branch1001())), id)
}

func TestANewMessageLandsInTheNotAnsweredTabNotTheAnsweredOne(t *testing.T) {
	// `KOMUNIKASISTATUS = '0'` pada INSERT-nya, dan REPLYMESSAGE kosong. Pesan baru adalah
	// pekerjaan yang menunggu dijawab — itulah arti tab pertama.
	store := memory.NewSampleStore()

	id, err := store.SendMessage(context.Background(), messageTo("PUSAT", ""), "1001")
	require.NoError(t, err)

	require.Contains(t, idsOf(listVia(t, store, "1", headOffice())), id)
	require.NotContains(t, idsOf(listVia(t, store, "2", headOffice())), id)
}

func TestANewMessageAlsoLeavesARowInTheBranchHistoryTable(t *testing.T) {
	// `PNCSendMessageKomunikasiCabang` menjalankan TIGA penulisan; yang ketiga mudah
	// terlupakan karena tidak ada satu pun layar yang menampilkannya.
	//
	// Kolom `kodecabang`-nya menerima KODE CABANG di sini — berbeda dari jalur balasan, yang
	// mengisinya dengan penanda kanal. Dua konvensi dalam satu kolom, dan keduanya
	// direplikasi.
	store := memory.NewSampleStore()

	id, err := store.SendMessage(context.Background(),
		messageTo("CABANG", "1002"), inboxkomunikasicabang.HeadOfficeCode)
	require.NoError(t, err)

	// Percakapan BARU, sehingga riwayatnya tepat satu baris — tidak bercampur dengan utas
	// percakapan contoh.
	history := historyOf(store, id)
	require.Len(t, history, 1)
	require.Equal(t, "pictekniks", history[0].Sender)
	require.Equal(t, "1002", history[0].Channel,
		"dari Kirim Pesan, kolom kodecabang berisi KODE CABANG — bukan penanda kanal")
}

func TestANewMessageCanBeRepliedToImmediately(t *testing.T) {
	// Uji rantai: pesan baru harus benar-benar menjadi percakapan yang utuh, bukan baris
	// yang bentuknya berbeda dari baris warisan. Kalau nomornya, kanalnya, atau arahnya
	// salah, balasannya akan ditolak.
	store := memory.NewSampleStore()

	id, err := store.SendMessage(context.Background(), messageTo("PUSAT", ""), "1001")
	require.NoError(t, err)

	require.NoError(t, store.Reply(context.Background(), replyTo(id), branch1001()))
	require.Contains(t, idsOf(listVia(t, store, "2", branch1001())), id)
}

func TestTwoMessagesGetDifferentNumbers(t *testing.T) {
	// Nomornya ditiru dari `MAX(KOMUNIKASIID)` sesudah penyisipan. Dua pesan yang berbagi
	// satu nomor akan menautkan riwayat yang satu ke percakapan yang lain — cacat yang di
	// sistem lama nyata, karena ketiga langkahnya berjalan tanpa transaksi.
	store := memory.NewSampleStore()

	first, err := store.SendMessage(context.Background(), messageTo("PUSAT", ""), "1001")
	require.NoError(t, err)

	second, err := store.SendMessage(context.Background(), messageTo("PUSAT", ""), "1001")
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestAStoreWithoutBranchesOffersNoChoice(t *testing.T) {
	// Keadaan yang tidak dapat dibuat lewat data contoh, tetapi nyata di produksi bila
	// `V_D_SURVEYORS` kosong atau kuerinya salah sasaran. Layar harus dapat menyatakannya,
	// bukan menampilkan pemilih kosong tanpa penjelasan.
	store := memory.NewSampleStore().WithBranches()

	branches, err := store.Branches(context.Background())
	require.NoError(t, err)
	require.Empty(t, branches)
}
