// Package usecase mengorkestrasi perkara master grouping sparepart.
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

	"claim-pnc/internal/mastergroupingsparepart"
)

// Service adalah pintu masuk seluruh perkara master grouping sparepart.
type Service struct {
	repoSelector mastergroupingsparepart.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastergroupingsparepart.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang: rakitan
// yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
//
// # Kenapa TANPA Clock, berbeda dari Master Sparepart
//
// Kedua tabel modul ini tidak punya satu pun kolom waktu. `GetDataMasterGrouping` membaca
// empat belas kolom dan tidak satu pun menampung stempel waktu, dan
// `Activity/UpdateGroupingSparepartHE_act` tidak memanggil `@DateTime.CurrentDateTime()` sama
// sekali. Menerima jam yang tidak akan pernah dipakai hanya akan menyesatkan pembaca
// berikutnya.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastergroupingsparepart/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field, dan berbeda dari Master Sparepart ia TIDAK TERSIMPAN: kedua tabel modul ini
// tidak punya kolom pencatat pelaku. Ia dibawa hanya untuk dicatat di log — sehingga siapa
// yang mengubah apa tetap dapat ditelusuri di sana, satu-satunya tempat yang tersedia sampai
// `S-5` Jejak Audit dibangun.
type Actor struct {
	Login string
}

// LookupSet adalah kedua daftar acuan yang dibutuhkan form saat dibuka.
//
// Keduanya dikirim bersamaan, bukan lewat dua endpoint terpisah, karena keduanya dibutuhkan
// bersamaan: form tidak dapat digambar tanpa daftar panel maupun daftar tipe kendaraan. Dua
// perjalanan jaringan untuk membuka satu form adalah biaya yang tidak perlu dibayar.
//
// Daftar Sisi TIDAK ikut di sini: ia bergantung pada panel yang dipilih, sehingga baru dapat
// dibaca setelah pengguna memilih. Lihat Sides.
type LookupSet struct {
	Panel       []mastergroupingsparepart.Panel
	VehicleType []mastergroupingsparepart.VehicleType
}

// List mengembalikan baris grouping satu portal yang cocok dengan penyaring.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	status mastergroupingsparepart.ApprovalStatus,
	keyword string,
) ([]mastergroupingsparepart.Grouping, error) {
	if !status.Known() {
		return nil, mastergroupingsparepart.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx, mastergroupingsparepart.Filter{
		Status:  status,
		Keyword: strings.TrimSpace(keyword),
	})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
func (l *Service) Get(
	ctx context.Context,
	portalAlias, id string,
) (mastergroupingsparepart.Grouping, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Options mengembalikan kedua daftar acuan yang dipakai form saat dibuka.
//
// Keduanya dibaca dari basis data ENTITAS yang sedang dibuka, bukan dari daftar tetap milik
// aplikasi — panel dan tipe kendaraan berbeda antarentitas, dan menyajikan daftar satu
// entitas kepada entitas lain adalah kebocoran yang justru dicegah `R-20`.
func (l *Service) Options(ctx context.Context, portalAlias string) (LookupSet, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return LookupSet{}, err
	}

	panel, err := store.ListPanels(ctx)
	if err != nil {
		return LookupSet{}, fmt.Errorf("mastergroupingsparepart/usecase: membaca panel: %w", err)
	}
	vehicle, err := store.ListVehicleTypes(ctx)
	if err != nil {
		return LookupSet{}, fmt.Errorf(
			"mastergroupingsparepart/usecase: membaca tipe kendaraan: %w", err)
	}

	return LookupSet{Panel: panel, VehicleType: vehicle}, nil
}

// Sides mengembalikan pilihan Sisi pada sebuah panel.
//
// Padanan `Activity/GetSisiPanel-Act.xml`, yang berjalan setelah pengguna memilih Nama Panel.
//
// Panel yang tidak punya satu pun baris lokasi mengembalikan daftar KOSONG, bukan galat.
// Sistem lama menjawabnya dengan pesan "Data Panel tidak ditemukan"; di sini daftar kosong
// yang dikirim, dan layar yang mengatakan panel itu belum punya sisi. Alasannya: itu bukan
// kegagalan permintaan melainkan keadaan data, dan menjadikannya galat membuat layar
// menampilkan kotak merah untuk sesuatu yang sepenuhnya sah.
func (l *Service) Sides(
	ctx context.Context,
	portalAlias string,
	key mastergroupingsparepart.SideKey,
) ([]mastergroupingsparepart.Side, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	clean := mastergroupingsparepart.SideKey{
		PanelID:   strings.TrimSpace(key.PanelID),
		PanelName: strings.TrimSpace(key.PanelName),
	}
	if clean.PanelID == "" && clean.PanelName == "" {
		// Tanpa keduanya, kueri aslinya akan mencocoki SELURUH baris lokasi di basis data.
		// Menolaknya di sini lebih jujur daripada mengirim daftar yang isinya sisi milik panel
		// mana pun.
		return nil, mastergroupingsparepart.OneViolation("id_panel",
			"Pilih dulu panelnya sebelum memilih sisi.")
	}

	return store.ListSides(ctx, clean)
}

// LookupPart mencari satu sparepart menurut nomornya.
//
// Padanan `Activity/SetDataSparepart-Act.xml`, yang berjalan saat isian Nomor Sparepart
// kehilangan fokus dan mengisi kelima isian turunannya.
//
// Ia endpoint tersendiri, bukan bagian dari penyimpanan, karena di sistem lama pun ia berjalan
// LEBIH DULU — pengguna melihat nama sparepartnya muncul sebelum menekan Simpan, dan itulah
// yang memberitahunya bahwa nomor yang diketik sudah benar. Penyimpanan tetap mencarinya
// sekali lagi; lihat resolvePart.
func (l *Service) LookupPart(
	ctx context.Context,
	portalAlias, number string,
) (mastergroupingsparepart.PartRef, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastergroupingsparepart.PartRef{}, err
	}

	clean := strings.TrimSpace(number)
	if clean == "" {
		return mastergroupingsparepart.PartRef{}, mastergroupingsparepart.OneViolation(
			"nomor_sparepart", "Nomor sparepart wajib diisi.")
	}
	return store.FindPart(ctx, clean)
}

// Create menyisipkan satu baris grouping baru.
//
// # Urutannya mengikuti sistem lama
//
//	SetDataSparepart                          isi kelima kolom turunan dari Master Sparepart
//	UpdateGroupingSparepartHE_act langkah 2   ID := "UnknownID" bila kosong
//	                             langkah 3–5  tolak bila keempat kunci alaminya sudah ada
//	                             langkah 6–11 terbitkan atau ikuti nomor grup
//	PEGA_M_GROUPING_SPAREPART_HE.prc:11       terbitkan ID
//	                             langkah 13   simpan
//
// Satu perbedaan urutan yang disengaja: pemeriksaan kunci alami di sini berada DI DALAM
// Repo.Insert, bukan sebagai langkah terpisah sebelumnya. Alasannya ada pada doc comment
// mastergroupingsparepart.Repo.Insert — memecahnya membuka kembali lubang balapan yang justru
// sedang dipersempit.
//
// # Dua hal yang TIDAK dibawa dari sistem lama, dan keduanya disengaja
//
//  1. **Surel pemberitahuan.** `UpdateGroupingSparepartHE_act` langkah 17–18 mengirimnya lewat
//     `Call SendEmailNotification` dengan penerima yang DI-HARDCODE sebagai satu akun surel
//     pribadi di jalur produksi. Seam Notifier (`S-3`) belum ada, `D-15` melarang nilai bisnis
//     di-hardcode, dan `D-67` melarang akun pribadi dibawa apa adanya — tiga alasan yang
//     ketiganya berdiri sendiri. Peristiwanya dicatat di log supaya ketiadaannya terlihat,
//     bukan tersamar.
//  2. **`Property-Set-HTML` ke TempMaster.** Ia menyusun badan surel yang tidak jadi dikirim.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input mastergroupingsparepart.Input,
	by Actor,
	logger *slog.Logger,
) (mastergroupingsparepart.Grouping, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	part, err := resolvePart(ctx, store, clean.PartNumber)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	group, err := l.resolveGroupNumber(ctx, store, clean, "")
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	id, err := store.NextID(ctx)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, fmt.Errorf(
			"mastergroupingsparepart/usecase: menerbitkan ID grouping: %w", err)
	}
	if strings.TrimSpace(id) == "" {
		// Bukan keadaan yang dapat diperbaiki pengguna, dan tidak boleh diteruskan diam-diam:
		// baris tanpa kunci tidak dapat dibuka, disunting, maupun disetujui.
		return mastergroupingsparepart.Grouping{}, errors.New(
			"mastergroupingsparepart/usecase: ID grouping yang diterbitkan kosong")
	}

	fresh := apply(mastergroupingsparepart.Grouping{ID: strings.TrimSpace(id)}, clean, part)
	fresh.GroupNumber = group
	fresh.Status = mastergroupingsparepart.StatusPending

	if err := store.Insert(ctx, fresh); err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	notifyOmitted(logger, portalAlias, fresh, by, "grouping baru diajukan")
	return fresh, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
//
// Di sistem lama status yang disimpan datang dari pemanggil —
// `TempSparepart.APPROVAL := Param.Approval` — dan layar penyuntingannya selalu mengirim "0".
// Di sini nilainya tidak lagi dapat dikirim klien: jalur simpan selalu StatusPending, dan
// keputusan komite menempuh Decide.
//
// Itu BUKAN perubahan hasil melainkan perubahan permukaan. Yang dicegahnya nyata: pada
// bentuk lama, satu permintaan simpan dapat sekaligus menyetujui dirinya sendiri hanya dengan
// mengirim `Approval=1`.
//
// # Yang TIDAK ikut tersimpan dari badan permintaan
//
//	ID               kunci baris, diambil dari jalur URL, bukan dari badan permintaan
//	APPROVAL         bukan isian melainkan akibat — selalu StatusPending di sini
//	NO_GROUP_RANGKA  diterbitkan atau diwarisi; lihat resolveGroupNumber
//	kelima turunan   dibaca ulang dari Master Sparepart menurut Nomor Sparepart
func (l *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input mastergroupingsparepart.Input,
	by Actor,
	logger *slog.Logger,
) (mastergroupingsparepart.Grouping, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return mastergroupingsparepart.Grouping{}, mastergroupingsparepart.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	part, err := resolvePart(ctx, store, clean.PartNumber)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	if err := ensureUnique(ctx, store, key, clean); err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	group, err := l.resolveGroupNumber(ctx, store, clean, stored.GroupNumber)
	if err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	updated := apply(stored, clean, part)
	updated.GroupNumber = group
	updated.Status = mastergroupingsparepart.StatusPending

	if err := store.Update(ctx, updated); err != nil {
		return mastergroupingsparepart.Grouping{}, err
	}

	notifyOmitted(logger, portalAlias, updated, by, "grouping diubah")
	return updated, nil
}

// resolveGroupNumber menentukan nomor grup kendaraan sebuah baris.
//
// # Ketiga cabangnya dibaca dari Pega apa adanya
//
//	langkah 6–7   BERAT=="" && pyID==""  -> GetNewNoGroup, lalu pyID := "000" + hasilnya
//	langkah 8–11  BERAT!=""              -> GetNoGroup menurut nomor rangka itu
//	              (tidak keduanya)       -> pyID yang sudah ada dipertahankan
//
// `BERAT` adalah isian "Grouping Dengan No Rangka" dan `pyID` adalah nomor grup yang sudah
// tersimpan pada baris itu. Syarat langkah 6 menyebut KEDUANYA, sehingga baris lama yang
// sudah punya nomor grup TIDAK mendapat nomor baru hanya karena disimpan ulang.
//
// # Satu departure yang disengaja, dan ia perlu persetujuan D-54
//
// `RDB List/GetNoGroup-SQL.xml` mengembalikan `TO_NUMBER(NO_GROUP_RANGKA)`, sehingga baris
// yang BERGABUNG ke sebuah grup menyimpan `"1"` sementara baris yang MEMBUKA grup itu
// menyimpan `"0001"`. Dua bentuk teks berbeda untuk grup yang sama, pada kolom yang sama.
//
// Akibatnya nyata: `GetNewNoGroup` mengambil `max(NO_GROUP_RANGKA)` secara TEKS, sehingga
// satu baris bernilai `"9"` membuat nomor grup berikutnya menjadi `"00010"` — melompati
// puluhan nomor — dan menghitung "berapa baris di grup ini" menuntut kedua bentuk dicoba.
//
// Yang dikerjakan di sini: nomor grup disimpan APA ADANYA seperti yang tersimpan pada grup
// yang diikuti, sehingga baris yang bergabung membawa nomor yang SAMA PERSIS dengan grupnya.
// Itu tujuan grouping, dan ia tidak mengubah satu pun isian yang dilihat pengguna — nomor
// grup tidak digambar di layar mana pun.
//
// Ia TIDAK ada pada tiga belas butir `P-5` (`D-49`), sehingga selisih yang muncul pada uji
// kesetaraan gerbang 1 menuntut persetujuan Work Owner tertulis (`D-54`). Dicatat di
// docs/keputusan-implementasi.md supaya ia diputuskan, bukan ditemukan.
func (l *Service) resolveGroupNumber(
	ctx context.Context,
	store mastergroupingsparepart.Store,
	in mastergroupingsparepart.Input,
	existing string,
) (string, error) {
	if in.GroupWithChassis != "" {
		group, err := store.FindGroupByChassis(ctx, in.GroupWithChassis)
		switch {
		case errors.Is(err, mastergroupingsparepart.ErrGroupChassisNotFound):
			// Pesannya diambil apa adanya dari `local.err2` pada
			// `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 2, ditambah kalimat yang
			// menuntun — sistem lama hanya menyebut masalahnya tanpa memberi tahu apa yang
			// harus dilakukan.
			return "", mastergroupingsparepart.OneViolation("grouping_dengan_no_rangka",
				"Nomor rangka dalam grouping tidak ditemukan. "+
					"Isi dengan nomor rangka yang sudah pernah didaftarkan, atau kosongkan untuk membuka grup baru.")
		case err != nil:
			return "", fmt.Errorf(
				"mastergroupingsparepart/usecase: mencari grup nomor rangka %q: %w",
				in.GroupWithChassis, err)
		}
		return group, nil
	}

	if strings.TrimSpace(existing) != "" {
		return strings.TrimSpace(existing), nil
	}

	group, err := store.NextGroupNumber(ctx)
	if err != nil {
		return "", fmt.Errorf(
			"mastergroupingsparepart/usecase: menerbitkan nomor grup: %w", err)
	}
	return group, nil
}

// Decide menetapkan status persetujuan sejumlah baris sekaligus.
//
// # Kenapa borongan, dan bukan menyimpan ulang seluruh barisnya
//
// Di sistem lama keputusan menempuh activity yang SAMA dengan penyimpanan —
// `Section/ApprovalPNCMasterGroupingSparepartHE-Section.xml` memanggil
// `UpdateGroupingSparepartHE_act` dengan `Param.Approval` yang berbeda — sehingga menyetujui
// berarti menjalankan ulang seluruh langkahnya: pencarian duplikat, penerbitan nomor grup,
// dan penulisan keempat belas kolomnya.
//
// Yang dikerjakan di sini hanya menyentuh kolom APPROVAL. Hasil yang DILIHAT pengguna sama
// persis; yang berbeda adalah dua hal yang keduanya cacat pada bentuk lama:
//
//  1. **Persetujuan dapat GAGAL karena validasi.** Baris yang sudah terlanjur tersimpan
//     dengan isian yang kini dianggap tidak sah tidak akan pernah dapat disetujui — pesan
//     yang muncul bicara tentang isian, padahal yang sedang dilakukan adalah menyetujui.
//  2. **Menyetujui dapat MENIMPA isi baris** dengan apa pun yang sedang ada di form
//     persetujuan, termasuk nilai yang sudah basi.
//
// Bentuknya — daftar baris bercentang ditambah satu status — mengikuti Master Bengkel, Master
// Panel, dan Master Sparepart, sehingga keempat master alat berat diputuskan dengan cara yang
// sama.
//
// TANPA alasan penolakan: kedua tabel modul ini tidak punya kolom penampungnya. Layar
// persetujuan Pega pun tidak punya isian catatan.
func (l *Service) Decide(
	ctx context.Context,
	portalAlias string,
	id []string,
	status mastergroupingsparepart.ApprovalStatus,
	by Actor,
	logger *slog.Logger,
) (int, error) {
	if !status.Known() {
		return 0, mastergroupingsparepart.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return 0, err
	}

	wanted := make([]string, 0, len(id))
	seen := make(map[string]bool, len(id))
	for _, one := range id {
		clean := strings.TrimSpace(one)
		// Kunci ganda dibuang: layar dapat mengirim baris yang sama dua kali bila daftarnya
		// dimuat ulang saat centang masih terpasang, dan jumlah baris berubah yang dilaporkan
		// harus mencerminkan baris, bukan centang.
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		wanted = append(wanted, clean)
	}
	if len(wanted) == 0 {
		return 0, mastergroupingsparepart.OneViolation("id_grouping",
			"Pilih dulu grouping yang akan diputuskan.")
	}

	changed, err := store.SetStatus(ctx, wanted, status)
	if err != nil {
		return 0, err
	}

	if changed != len(wanted) && logger != nil {
		// Bukan galat: baris yang tidak berubah adalah baris yang sudah tidak ada, atau sudah
		// berstatus itu. Tetapi selisihnya berarti layar menampilkan daftar yang sudah basi,
		// dan itu layak terbaca.
		logger.Warn("keputusan master grouping sparepart tidak menyentuh seluruh baris yang dipilih",
			slog.String("portal", portalAlias),
			slog.Int("dipilih", len(wanted)),
			slog.Int("berubah", changed),
			slog.String("oleh", by.Login))
	}

	notifyDecisionOmitted(logger, portalAlias, changed, status, by)
	return changed, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("mastergroupingsparepart/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}

// resolvePart membaca kelima isian turunan dari Master Sparepart.
//
// Ia dijalankan pada jalur simpan MESKIPUN layar sudah memanggil LookupPart lebih dulu.
// Alasannya bukan kehati-hatian berlebih: nilai turunan datang dari klien pada bentuk lama,
// sehingga klien dapat mengirim nama sparepart yang sama sekali tidak berhubungan dengan
// nomornya. Membacanya ulang di server membuat nomor sparepart menjadi satu-satunya sumber
// kelimanya.
//
// Nomor yang tidak ketemu menghasilkan pelanggaran yang menempel pada isiannya, bukan galat
// 404: yang salah adalah isian pengguna, dan pesannya harus muncul di bawah kotak yang
// diketiknya.
func resolvePart(
	ctx context.Context,
	store mastergroupingsparepart.Store,
	number string,
) (mastergroupingsparepart.PartRef, error) {
	part, err := store.FindPart(ctx, number)
	switch {
	case errors.Is(err, mastergroupingsparepart.ErrPartNotFound):
		// Pesannya diambil apa adanya dari `local.mssg` pada
		// `Activity/SetDataSparepart-Act.xml`, ditambah kalimat yang menuntun.
		return mastergroupingsparepart.PartRef{}, mastergroupingsparepart.OneViolation(
			"nomor_sparepart",
			"Data Sparepart tidak ditemukan. Periksa nomornya di Master Sparepart.")
	case err != nil:
		return mastergroupingsparepart.PartRef{}, fmt.Errorf(
			"mastergroupingsparepart/usecase: mencari sparepart %q: %w", number, err)
	}
	return part, nil
}

// apply menyalin isian yang dikirim layar DAN kelima turunan ke atas baris yang sudah ada.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya kedua jalur tidak pernah berbeda soal
// kolom mana yang ikut berubah. Tiga kolom sengaja tidak disentuh — ID, GroupNumber, dan
// Status — dan ketiganya diatur pemanggil.
func apply(
	base mastergroupingsparepart.Grouping,
	in mastergroupingsparepart.Input,
	part mastergroupingsparepart.PartRef,
) mastergroupingsparepart.Grouping {
	base.PartNumber = in.PartNumber
	base.PanelID = in.PanelID
	base.PanelName = in.PanelName
	base.PanelSide = in.PanelSide
	base.ChassisNumber = in.ChassisNumber
	base.VehicleType = in.VehicleType
	base.GroupWithChassis = in.GroupWithChassis
	base.Note = in.Note

	base.PartName = part.Name
	base.CategoryID = part.CategoryID
	base.TypeID = part.TypeID
	base.PartCode = part.Code
	base.ProductionDate = part.ProductionDate

	return base
}

// ensureUnique menolak kunci alami yang sudah dipakai BARIS LAIN.
//
// Ia dipakai jalur simpan saja. Jalur tambah memeriksanya di dalam Repo.Insert, yang dapat
// menguncinya lebih dulu; di sini penguncian itu tidak diperlukan karena kunci barisnya sudah
// ada, dan bentrok yang muncul di antara pemeriksaan dan penyimpanan hanya dapat berasal dari
// petugas lain yang menyimpan nilai yang sama pada saat yang sama — keadaan yang tertutup
// constraint unik, bukan oleh urutan langkah (`R-08`, `D-63`).
//
// # Kenapa pemeriksaannya ada di jalur simpan juga
//
// Sistem lama JUSTRU TIDAK memeriksanya di sana: `UpdateGroupingSparepartHE_act` langkah 3
// bersyarat `TempSparepart.ID=="UnknownID"`, yang hanya benar pada penambahan. Akibatnya dua
// baris berkunci sama dapat lahir cukup dengan menyunting salah satunya menjadi kunci milik
// yang lain — dan pemeriksaan pada penambahan menjadi tidak ada artinya.
//
// SELISIH YANG DIRENCANAKAN. Ia menolak penyimpanan yang di sistem lama diterima, dan baris
// lama yang sudah berkunci ganda tetap DIBACA apa adanya — penolakan hanya terjadi saat
// barisnya disimpan ulang.
//
// Baris yang sedang disunting dikecualikan: menyimpan tanpa mengubah kuncinya tidak boleh
// ditolak karena kuncinya sendiri sudah dipakai oleh dirinya sendiri.
func ensureUnique(
	ctx context.Context,
	store mastergroupingsparepart.Store,
	id string,
	in mastergroupingsparepart.Input,
) error {
	key := mastergroupingsparepart.NaturalKey{
		PartNumber:    in.PartNumber,
		PanelName:     in.PanelName,
		ChassisNumber: in.ChassisNumber,
		PanelSide:     in.PanelSide,
	}

	other, err := store.FindByKey(ctx, key)
	switch {
	case errors.Is(err, mastergroupingsparepart.ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("mastergroupingsparepart/usecase: memeriksa kunci grouping: %w", err)
	case strings.TrimSpace(other.ID) == strings.TrimSpace(id):
		return nil
	}

	// Pesannya diambil dari `local.err` — "Data sudah ada" — ditambah keterangan baris mana
	// yang memakainya, supaya petugas dapat membukanya alih-alih menebak.
	return mastergroupingsparepart.OneViolation("nomor_sparepart",
		"Data sudah ada. Kombinasi nomor sparepart, nama panel, no rangka, dan sisi ini "+
			"sudah dipakai grouping "+strings.TrimSpace(other.ID)+".")
}

// notifyOmitted mencatat pemberitahuan yang di sistem lama dikirim, dan di sini tidak.
//
// Ia sengaja dicatat sebagai Info dan bukan didiamkan: yang hilang adalah satu langkah proses
// yang nyata — PIC tidak lagi diberi tahu bahwa ada grouping menunggu persetujuan — dan
// ketiadaannya harus terbaca di log alih-alih ditemukan berbulan kemudian oleh petugas yang
// bertanya kenapa antreannya menumpuk.
//
// Pelaku ikut dicatat DI SINI, bukan di basis data: kedua tabel modul ini tidak punya kolom
// pencatat pelaku, sehingga log adalah satu-satunya tempat yang tersedia sampai `S-5` Jejak
// Audit dibangun.
func notifyOmitted(
	logger *slog.Logger,
	portalAlias string,
	g mastergroupingsparepart.Grouping,
	by Actor,
	event string,
) {
	if logger == nil {
		return
	}
	logger.Info("pemberitahuan master grouping sparepart tidak dikirim",
		slog.String("peristiwa", event),
		slog.String("portal", portalAlias),
		slog.String("id_grouping", g.ID),
		slog.String("nomor_sparepart", g.PartNumber),
		slog.String("no_rangka", g.ChassisNumber),
		slog.String("nomor_grup", g.GroupNumber),
		slog.String("oleh", by.Login),
		slog.String("sebab", "seam Notifier (S-3) belum ada, dan penerima di rule lama berupa akun surel pribadi yang D-15 dan D-67 larang dibawa"))
}

// notifyDecisionOmitted mencatat hal yang sama untuk jalur keputusan.
func notifyDecisionOmitted(
	logger *slog.Logger,
	portalAlias string,
	changed int,
	status mastergroupingsparepart.ApprovalStatus,
	by Actor,
) {
	if logger == nil || changed == 0 {
		return
	}
	logger.Info("keputusan master grouping sparepart tersimpan tanpa pemberitahuan",
		slog.String("portal", portalAlias),
		slog.Int("berubah", changed),
		slog.String("status", string(status)),
		slog.String("status_label", status.Label()),
		slog.String("oleh", by.Login))
}
