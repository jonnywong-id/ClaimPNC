package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxacceptopenprotection/usecase"
)

// Uji di berkas ini menjaga akibat akseptasi pada DATA KLAIM.
//
// Pega mengubahnya di `Activity/InsertOpenProtectionCase-Act.xml`:
//
//	TempPNCOPEN.ClaimData.DateOfLoss <- @substring(.ClaimDataProtect.DateOfLoss,0,8)+"T000000.000 GMT"
//
// `TempPNCOPEN` adalah work object klaim. Sasarannya di sistem baru adalah
// **`POOLDATA.T_CLAIM_PNC`**, dikunci `CLAIMID` (Work Owner, 2026-09-26) —
// `T_CLAIMLIST_ADMIN` hanya tabel baca untuk dashboard dan tidak ditulisi.
//
// Tanpa uji ini, hilangnya penerapan TIDAK menghasilkan gejala apa pun: keputusan tersimpan,
// layar menampilkan "disetujui", dan klaim tetap memakai DOL lama.

// proteksiDOL membentuk permintaan perubahan DOL yang lengkap.
func proteksiDOL(number string, at time.Time, baru *time.Time) inboxacceptopenprotection.Protection {
	p := lengkap(number, inboxacceptopenprotection.TypeChangeLossDate, at)
	sebelum := at.AddDate(0, 0, -5)
	p.Change = inboxacceptopenprotection.ChangeDetail{
		LossDateBefore: &sebelum,
		LossDateAfter:  baru,
	}
	return p
}

func TestPersetujuanPerubahanDOLMengubahTanggalKejadianKlaim(t *testing.T) {
	at := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	dolBaru := time.Date(2026, time.August, 5, 14, 30, 0, 0, wib)

	service, repo := bangun(t, at)
	p := proteksiDOL("OPCN.26.0100", at, &dolBaru)
	repo.Add(p)
	repo.AddClaim(p.ClaimReference, time.Date(2026, time.August, 3, 0, 0, 0, 0, wib))

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0100",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          loginBerwenang,
	})
	require.NoError(t, err)

	tersimpan, ada := repo.LossDateOf(p.ClaimReference)
	require.True(t, ada)

	// Jam DIBUANG: Pega memotong `@substring(...,0,8)` lalu menempel `T000000.000`.
	// Menyimpan jam akan menggeser aturan "Tanggal Lapor <= DOL + 7 hari" tanpa terlihat.
	require.Equal(t, 2026, tersimpan.Year())
	require.Equal(t, time.August, tersimpan.Month())
	require.Equal(t, 5, tersimpan.Day())
	require.Zero(t, tersimpan.Hour(), "jam harus dibuang")
	require.Zero(t, tersimpan.Minute())
}

func TestPenolakanTIDAKMengubahTanggalKejadianKlaim(t *testing.T) {
	// SELISIH TERENCANA dari Pega. Di sana perubahan DOL hanya berprakondisi
	// `Local.typeprotection=="7"` — `Local.acceptstatus` dipakai SEKALI saja, untuk menyusun
	// teks catatan. Dibaca apa adanya, menolak permintaan tetap mengubah DOL klaim.
	//
	// Perilaku itu tidak dibawa: permintaan yang ditolak tidak boleh terjadi.
	at := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	dolBaru := time.Date(2026, time.August, 5, 0, 0, 0, 0, wib)
	dolAwal := time.Date(2026, time.August, 3, 0, 0, 0, 0, wib)

	service, repo := bangun(t, at)
	p := proteksiDOL("OPCN.26.0101", at, &dolBaru)
	repo.Add(p)
	repo.AddClaim(p.ClaimReference, dolAwal)

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0101",
		Decision:    inboxacceptopenprotection.DecisionReject,
		By:          loginBerwenang,
	})
	require.NoError(t, err)

	tersimpan, _ := repo.LossDateOf(p.ClaimReference)
	require.True(t, tersimpan.Equal(dolAwal),
		"penolakan tidak boleh mengubah Tanggal Kejadian klaim")
}

func TestTipeSelainTujuhTidakMenyentuhKlaim(t *testing.T) {
	at := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	dolAwal := time.Date(2026, time.August, 3, 0, 0, 0, 0, wib)

	service, repo := bangun(t, at)
	p := lengkap("OPCN.26.0102", "1", at)
	repo.Add(p)
	repo.AddClaim(p.ClaimReference, dolAwal)

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0102",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          loginBerwenang,
	})
	require.NoError(t, err)

	tersimpan, _ := repo.LossDateOf(p.ClaimReference)
	require.True(t, tersimpan.Equal(dolAwal), "hanya tipe '7' yang mengubah DOL klaim")
}

func TestKlaimYangBelumAdaDiDaftarMenolakKeputusan(t *testing.T) {
	// `T_CLAIM_PNC` TIDAK memuat setiap klaim: dari 2.639 klaim berkelas
	// `ASM-FW-GCNMFW-Work-PNC` di tabel kerja Pega, 1.242 — 47% — tidak punya baris di sana
	// (hitungan langsung ke Oracle, 2026-09-26). Jadi klaim yang sah dapat belum ada.
	//
	// Keputusannya DIBATALKAN, bukan diteruskan: menyimpan persetujuan atas perubahan yang
	// tidak pernah diterapkan menghasilkan klaim ber-DOL berbeda dari yang disetujui, tanpa
	// satu pun gejala.
	at := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	dolBaru := time.Date(2026, time.August, 5, 0, 0, 0, 0, wib)

	service, repo := bangun(t, at)
	repo.Add(proteksiDOL("OPCN.26.0103", at, &dolBaru))
	// Klaimnya sengaja TIDAK didaftarkan.

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0103",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          loginBerwenang,
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)

	// Keputusannya tidak boleh tersimpan separuh: barisnya harus tetap MENUNGGU.
	tetap, err := service.Get(context.Background(), portal, "OPCN.26.0103", loginBerwenang)
	require.NoError(t, err)
	require.True(t, tetap.Pending(),
		"keputusan yang gagal diterapkan tidak boleh menyisakan status tersimpan")
}

func TestBarisWarisanTanpaTanggalBaruTetapDapatDisetujui(t *testing.T) {
	// Baris warisan Pega tidak punya kolom asal untuk OLD_DATA/NEW_DATA, sehingga tanggal
	// barunya kosong. Menyetujuinya tetap sah — yang tidak terjadi hanyalah perubahan DOL,
	// karena tidak ada nilai yang hendak diterapkan.
	at := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	dolAwal := time.Date(2026, time.August, 3, 0, 0, 0, 0, wib)

	service, repo := bangun(t, at)
	p := proteksiDOL("OPCN.26.0104", at, nil)
	repo.Add(p)
	repo.AddClaim(p.ClaimReference, dolAwal)

	saved, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0104",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          loginBerwenang,
	})
	require.NoError(t, err)
	require.True(t, saved.Approved())

	tersimpan, _ := repo.LossDateOf(p.ClaimReference)
	require.True(t, tersimpan.Equal(dolAwal))
}

func TestKunciPenerapanAdalahIDClaimBukanNomorKlaim(t *testing.T) {
	// `T_CLAIM_PNC` dikunci `CLAIMID`, yang bagi klaim WARISAN berbentuk
	// `ASM-FW-GCNMFW-WORK PNC-1865` — berbeda dari nomor klaimnya.
	//
	// Memakai nomor klaim sebagai kunci tidak akan menemukan baris mana pun, dan tidak
	// menemukan apa pun TIDAK menghasilkan galat basis data. Uji ini yang membedakan
	// keduanya: klaim didaftarkan HANYA di bawah CLAIMID-nya.
	at := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	dolBaru := time.Date(2026, time.August, 5, 0, 0, 0, 0, wib)
	dolAwal := time.Date(2026, time.August, 3, 0, 0, 0, 0, wib)

	service, repo := bangun(t, at)

	p := proteksiDOL("OPCN.26.0105", at, &dolBaru)
	p.ClaimNumber = "PNC-1865"
	p.ClaimReference = "ASM-FW-GCNMFW-WORK PNC-1865"
	repo.Add(p)
	repo.AddClaim(p.ClaimReference, dolAwal)

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0105",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          loginBerwenang,
	})
	require.NoError(t, err)

	tersimpan, ada := repo.LossDateOf("ASM-FW-GCNMFW-WORK PNC-1865")
	require.True(t, ada)
	require.Equal(t, 5, tersimpan.Day(), "perubahan harus mendarat di baris ber-CLAIMID")

	_, adaSalah := repo.LossDateOf("PNC-1865")
	require.False(t, adaSalah, "nomor klaim BUKAN kunci T_CLAIM_PNC")
}

func TestProteksiTanpaIDClaimMenolakKeputusan(t *testing.T) {
	// Tanpa ID_CLAIM, barisnya tidak dapat ditautkan ke klaim mana pun. Diperlakukan sama
	// dengan klaim yang tidak ditemukan — dibatalkan, bukan diteruskan.
	at := time.Date(2026, time.September, 26, 10, 0, 0, 0, wib)
	dolBaru := time.Date(2026, time.August, 5, 0, 0, 0, 0, wib)

	service, repo := bangun(t, at)
	p := proteksiDOL("OPCN.26.0106", at, &dolBaru)
	p.ClaimReference = ""
	repo.Add(p)

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0106",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          loginBerwenang,
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)
}
