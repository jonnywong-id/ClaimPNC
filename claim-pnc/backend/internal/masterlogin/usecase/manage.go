// Package usecase mengorkestrasi perkara master login surveyor.
//
// Ia yang mengetahui urutan langkah; aturan isian ada di paket domain, dan cara membacanya
// dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/masterlogin"
)

// Service adalah pintu masuk seluruh perkara master login surveyor.
type Service struct {
	repoSelector masterlogin.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterlogin.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
//
// TANPA Clock. `POOLDATA.MST_LOGIN_SURVEYOR` tidak punya satu pun kolom waktu, sehingga
// tidak ada yang perlu distempel. Menerimanya "untuk jaga-jaga" berarti menerima bahan yang
// tidak pernah dipakai, dan itu menyesatkan pembaca berikutnya.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterlogin/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field, dan ia dipakai DUA hal yang berbeda — perbedaan yang mudah terlewat:
//
//  1. **Menurunkan LOGINLEADER** pada penambahan. Ini bukan pencatatan melainkan DATA:
//     `Activity/CNMInsertMstLoginSurveyor_act` mencari leader milik pengguna yang menyimpan,
//     lalu menuliskannya ke baris yang baru. Lihat Create.
//  2. **Mengisi log.** Tabelnya tidak punya kolom pencatat pelaku sama sekali, sehingga log
//     adalah satu-satunya tempat "siapa yang menambah login ini" terekam. Itu bukan
//     pengganti jejak audit `S-5`, dan tidak diklaim demikian.
type Actor struct {
	Login string
}

// List mengembalikan baris satu portal yang cocok dengan penyaring.
//
// Cakupannya SELURUH baris tabel, bukan hanya milik satu tim; alasannya beserta bacaan lain
// yang mungkin ada pada doc comment masterlogin.Filter.
func (l *Service) List(
	ctx context.Context,
	portalAlias, keyword string,
) ([]masterlogin.SurveyorLogin, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, masterlogin.Filter{Keyword: strings.TrimSpace(keyword)})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
//
// Padanan `Activity/SetLoginSurveyorValue_act`, yang dijalankan tombol Ubah pada setiap
// baris grid.
func (l *Service) Get(
	ctx context.Context,
	portalAlias, login string,
) (masterlogin.SurveyorLogin, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	key := strings.TrimSpace(login)
	if key == "" {
		return masterlogin.SurveyorLogin{}, masterlogin.ErrNotFound
	}
	return repo.Get(ctx, key)
}

// Create menyisipkan satu login surveyor baru.
//
// # Urutannya mengikuti sistem lama
//
//	SetLoginSurveyor_act            turunkan LOGIN dari NAMA, tolak login ganda
//	CNMInsertMstLoginSurveyor_act   tolak LOGIN kosong
//	  langkah 3 GetLoginLeaderSurveyor    cari leader milik pengguna yang menyimpan
//	  langkah 4 Property-Set              LOGINLEADER := leader itu
//	  langkah 1 Property-Set              STSLOGIN := "Member"
//	  langkah 5 RDB-Save                  sisipkan
//
// Dua perbedaan urutan yang disengaja, keduanya dijelaskan di tempatnya: penurunan LOGIN
// pindah ke server (masterlogin.DeriveLogin), dan pemeriksaan login ganda pindah ke dalam
// Repo.Insert (masterlogin.ErrLoginTaken).
//
// # LOGINLEADER diturunkan dari PENYIMPAN, bukan dari baris yang dibuat
//
// Ini bagian yang paling mudah salah dibaca. `GetLoginLeaderSurveyor` berbunyi:
//
//	select loginleader from pooldata.mst_login_surveyor
//	 where login = {OperatorID.pyUserIdentifier}
//
// Yang dicari adalah baris milik **pengguna yang sedang menekan Simpan**, dan yang diambil
// adalah kolom LOGINLEADER-nya — bukan login pengguna itu sendiri. Akibatnya: surveyor yang
// menambahkan rekannya memberi rekan itu **leader yang sama dengan dirinya**, bukan menjadi
// leader rekannya.
//
// # Pengguna yang tidak punya baris di tabel ini
//
// Kuerinya tidak mengembalikan apa pun, dan langkah berikutnya menyalin nilai yang tidak
// ada — di Pega hasilnya LOGINLEADER kosong, tanpa satu pun galat. Perilaku itu ditiru apa
// adanya (`P-5`): baris tetap tersimpan dengan LOGINLEADER kosong.
//
// Yang DITAMBAHKAN hanyalah catatan log, supaya keadaan itu terlihat. Baris ber-LOGINLEADER
// kosong tidak dapat dibedakan dari baris yang memang tidak bertim, dan tidak ada apa pun
// di layar yang akan menunjukkannya.
//
// # Satu hal yang TIDAK dibawa, dan itu tidak dapat dibawa
//
// `Activity/SetLoginSurveyor_act` memanggil `GCNMCreateOperator` untuk **menerbitkan akun
// operator Pega** bagi surveyor yang baru, dengan kata sandi yang sama untuk setiap orang
// (`local.pass`), dan pesan suksesnya menyebut kata sandi itu terang-terangan.
//
// Tiga hal menghalanginya, dan ketiganya bukan selera:
//
//  1. **Tidak ada tempat menerbitkannya.** Sistem baru tidak punya operator Pega, dan
//     kontrak identitas `F-3` belum ada (`R-14`, `ADR-0024`).
//  2. **Kata sandinya tidak boleh ditulis di artefak yang di-commit** (`D-69`), dan tidak
//     boleh di-hardcode (`D-15`).
//  3. **Mengumumkan kata sandi bagi akun yang tidak diterbitkan adalah keterangan yang
//     salah.** Petugas akan menyampaikannya kepada surveyor, dan surveyor tidak akan dapat
//     masuk.
//
// Baris MST_LOGIN_SURVEYOR tetap tersimpan — itulah yang dikerjakan layar ini — dan
// ketiadaan akunnya dicatat di log serta DINYATAKAN di layar, bukan tersamar. Perlakuan
// yang sama dipakai Master Bengkel, yang menghadapi `GCNMCreateOperator` yang sama persis.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input masterlogin.Input,
	by Actor,
	logger *slog.Logger,
) (masterlogin.SurveyorLogin, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	leader, err := leaderOf(ctx, repo, by.Login)
	if err != nil {
		return masterlogin.SurveyorLogin{}, err
	}
	if leader == "" {
		noteLeaderMissing(logger, portalAlias, by)
	}

	saved, err := repo.Insert(ctx, masterlogin.SurveyorLogin{
		Name:        clean.Name,
		Login:       masterlogin.DeriveLogin(clean.Name),
		Email:       clean.Email,
		Phone:       clean.Phone,
		Address:     clean.Address,
		LoginStatus: masterlogin.LoginStatusMember,
		LeaderLogin: leader,
	})
	if err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	noteAccountNotIssued(logger, portalAlias, saved, by)
	return saved, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Nama TIDAK dapat diubah, dan itu ditegakkan di sini
//
// Layar Pega menguncinya; lihat masterlogin.ErrNameLocked untuk alasan lengkapnya. Yang
// dibandingkan adalah nilai yang sudah dipangkas di kedua sisi — menolak penyimpanan hanya
// karena ada satu spasi ujung yang berbeda adalah kegagalan yang tidak dapat dijelaskan
// kepada siapa pun.
//
// Perbandingannya **peka huruf besar-kecil**, berbeda dari pemeriksaan login ganda. Alasan:
// NAMA adalah teks yang dibaca manusia dan ditampilkan apa adanya, sehingga "Budi" dan
// "BUDI" memang dua tulisan yang berbeda — sedangkan LOGIN adalah kunci, dan dua kunci yang
// hanya berbeda huruf besar-kecilnya akan bertabrakan pada basis data yang membandingkannya
// tanpa membedakan.
//
// # KETUJUH kolom ditulis, bukan hanya yang berubah
//
// `RDB List/UpdateMasterLoginSurvey-SQL.xml` menulis ketujuhnya sekaligus. Ditiru apa
// adanya — termasuk menuliskan kembali LOGIN, STSLOGIN, dan LOGINLEADER dengan nilai yang
// sudah tersimpan.
//
// Ketiganya diambil dari BARIS YANG TERSIMPAN, bukan dari permintaan. Dengan begitu
// penyuntingan tidak pernah dapat memindahkan seseorang ke tim lain, mengubah perannya,
// maupun memindahkan kunci barisnya — tiga hal yang di sistem lama pun tidak pernah berada
// di tangan pengguna, karena ketiganya tidak digambar di layar.
//
// Yang berubah hanyalah Email, Telp, dan Alamat. Itu memang seluruh yang dapat disunting:
// Nama terkunci, dan keempat kolom lain tidak berasal dari form.
func (l *Service) Save(
	ctx context.Context,
	portalAlias, login string,
	input masterlogin.Input,
	by Actor,
	logger *slog.Logger,
) (masterlogin.SurveyorLogin, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	key := strings.TrimSpace(login)
	if key == "" {
		return masterlogin.SurveyorLogin{}, masterlogin.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	stored, err := repo.Get(ctx, key)
	if err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	if strings.TrimSpace(stored.Name) != clean.Name {
		return masterlogin.SurveyorLogin{}, masterlogin.ErrNameLocked
	}

	updated := masterlogin.SurveyorLogin{
		Name:        stored.Name,
		Login:       stored.Login,
		Email:       clean.Email,
		Phone:       clean.Phone,
		Address:     clean.Address,
		LoginStatus: stored.LoginStatus,
		LeaderLogin: stored.LeaderLogin,
	}
	if err := repo.Update(ctx, updated); err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	noteSaved(logger, portalAlias, updated, by)
	return updated, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterlogin/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}

// leaderOf mencari isi LOGINLEADER milik satu login.
//
// Baris yang tidak ada BUKAN galat: pengguna yang menambahkan login surveyor belum tentu
// terdaftar sebagai surveyor sendiri — petugas admin, misalnya. Di Pega keadaan itu
// menghasilkan LOGINLEADER kosong tanpa satu pun tanda, dan itu ditiru apa adanya.
//
// Pemanggil yang loginnya kosong pun dilayani tanpa menyentuh basis data: tidak ada baris
// yang dapat ditemukan dengan kunci kosong, dan menembakkannya hanya akan menghasilkan
// perjalanan yang pasti sia-sia.
func leaderOf(
	ctx context.Context,
	repo masterlogin.Repo,
	login string,
) (string, error) {
	key := strings.TrimSpace(login)
	if key == "" {
		return "", nil
	}

	leader, err := repo.FindLeaderOf(ctx, key)
	switch {
	case errors.Is(err, masterlogin.ErrNotFound):
		return "", nil
	case err != nil:
		return "", err
	default:
		return strings.TrimSpace(leader), nil
	}
}

// noteAccountNotIssued mencatat akun aplikasi yang TIDAK diterbitkan.
//
// Ia sengaja bertingkat Info dan bukan Warn: ketiadaannya adalah keadaan yang DIKETAHUI dan
// diputuskan (`F-3` belum ada, `D-69` melarang kata sandi tertulis), bukan gangguan yang
// perlu ditindaklanjuti seseorang malam ini.
//
// Yang dicatat adalah peristiwanya, supaya saat kontrak identitas tiba, daftar login yang
// akunnya belum pernah diterbitkan dapat ditarik tanpa menebak baris mana saja.
//
// Kata sandi yang di-hardcode pada `Activity/SetLoginSurveyor_act` TIDAK ditulis ke log
// maupun ke berkas mana pun (`D-69`).
func noteAccountNotIssued(
	logger *slog.Logger,
	portalAlias string,
	one masterlogin.SurveyorLogin,
	by Actor,
) {
	if logger == nil {
		return
	}
	logger.Info("login surveyor tersimpan; akun aplikasinya tidak diterbitkan karena seam identitas belum ada",
		slog.String("portal", portalAlias),
		slog.String("login", one.Login),
		slog.String("nama", one.Name),
		slog.String("login_leader", one.LeaderLogin),
		slog.String("oleh", by.Login))
}

// noteSaved mencatat penyimpanan.
//
// Terpisah dari noteAccountNotIssued karena penyuntingan TIDAK menyentuh penerbitan akun
// sama sekali — menyatukan keduanya akan membuat setiap penyimpanan tampak seperti
// percobaan menerbitkan akun yang gagal.
func noteSaved(
	logger *slog.Logger,
	portalAlias string,
	one masterlogin.SurveyorLogin,
	by Actor,
) {
	if logger == nil {
		return
	}
	logger.Info("login surveyor diubah",
		slog.String("portal", portalAlias),
		slog.String("login", one.Login),
		slog.String("nama", one.Name),
		slog.String("oleh", by.Login))
}

// noteLeaderMissing mencatat penambahan yang menghasilkan LOGINLEADER kosong.
//
// Warn, bukan Info: berbeda dari akun yang sengaja tidak diterbitkan, ini keadaan yang
// TIDAK dikehendaki siapa pun — baris yang tersimpan tidak bertaut ke tim mana pun, dan
// tabelnya tidak punya cara membedakan "tidak bertim" dari "gagal menemukan timnya".
//
// Ia tetap tidak menggagalkan penyimpanan, meniru Pega apa adanya (`P-5`).
func noteLeaderMissing(logger *slog.Logger, portalAlias string, by Actor) {
	if logger == nil {
		return
	}
	logger.Warn("login leader tidak dapat diturunkan; baris baru akan tersimpan tanpa LOGINLEADER",
		slog.String("portal", portalAlias),
		slog.String("oleh", by.Login),
		slog.String("sebab", "pemanggil tidak punya baris di MST_LOGIN_SURVEYOR portal ini"))
}
