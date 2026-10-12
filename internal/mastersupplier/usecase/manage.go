// Package usecase mengorkestrasi perkara master supplier.
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
	"time"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/platform/clock"
)

// Service adalah pintu masuk seluruh perkara master supplier.
type Service struct {
	repoSelector mastersupplier.RepoSelector

	// clock mengisi kunci TGL_INSERT pada dokumen supplier dan kolom TGL_INPUT pada baris
	// permintaan persetujuan.
	//
	// Ia seam, bukan time.Now() langsung, supaya jejak waktunya dapat diuji deterministik
	// dan supaya tidak ada satu pun penambahan tujuh jam manual yang menyelinap masuk
	// (`08-TECHNICAL-STRATEGY.md` §4.4). Konversi ke WIB terjadi di satu tempat, yaitu
	// mastersupplier.FormatJakartaDate.
	clock clock.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastersupplier.RepoSelector

	// Clock menjadi sumber TGL_INSERT dan TGL_INPUT. Wajib.
	Clock clock.Clock
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastersupplier/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("mastersupplier/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Berbeda dari Master Bengkel, identitas pemanggil di sini BENAR-BENAR TERSIMPAN: ia
// menjadi kunci `USERKLAIMID` di dalam dokumen supplier dan kolom `USER_REQ` pada baris
// permintaan persetujuan. Keduanya ditulis sistem lama juga, sehingga jejaknya bukan
// tambahan melainkan bagian dari perilaku yang ditiru.
type Actor struct {
	Login string
}

// List mengembalikan baris master supplier satu portal yang cocok dengan penyaring.
func (l *Service) List(
	ctx context.Context,
	portalAlias, keyword string,
) ([]mastersupplier.Supplier, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx, mastersupplier.Filter{Keyword: strings.TrimSpace(keyword)})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
//
// Ia padanan `Activity/GetDataSupplier_pre`, termasuk penurunan JENIS_STATUS dari
// SUPPLIER_HE — yang dikerjakan adapter saat membaca, supaya kedua adapter tidak pernah
// berbeda soal nilai mana yang menang.
func (l *Service) Get(ctx context.Context, portalAlias, id string) (mastersupplier.Supplier, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersupplier.Supplier{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Create menyisipkan satu baris master supplier baru.
//
// # Urutannya mengikuti Activity/CreateNewMasterSupplier_post
//
//	step 6   STS_AKTIF := "0" · USERKLAIMID := pemanggil · TGL_INSERT := hari ini
//	step 7   SUPPLIER_HE := "1" bila JENIS_STATUS = "1"
//	step 9   JSONDATA := seluruh halaman
//	step 10  KonversiMasterSupplier_SQL -> PEGA_M_SUPPLIER, IDPega kosong -> INSERT
//	step 12  hanya bila penyimpanan berhasil, permintaan persetujuan dibentuk
//	step 17  InsertProteksiKlaimMBU_SQL, POSISI := "1"
//
// Satu perbedaan urutan yang disengaja: pemeriksaan nama ganda berada DI DALAM
// Repo.Insert, bukan sebagai langkah terpisah sebelumnya. Alasannya ada pada doc comment
// mastersupplier.Repo.Insert.
//
// # Tiga hal yang TIDAK dibawa dari sistem lama, dan ketiganya disengaja
//
//  1. **Pemanggilan `POOLDATA.PEGA_M_SUPPLIER`.** `D-02` menetapkan seluruh logika
//     procedure naik ke Go, dan procedure itu melakukan `COMMIT` sendiri dua kali
//     (`PEGA_M_SUPPLIER.prc:25,37`) sehingga penyimpanan dan permintaan persetujuannya
//     tidak dapat dibuat atomik selama ia dipanggil.
//  2. **Case `ASM-FW-GKM-Work-Protection`.** Ketiga langkah yang membuatnya —
//     `CreateWorkPage`, `AddWork`, `commitWithErrorHandling` — adalah mekanisme work
//     object Pega yang tidak punya padanan. Yang dibutuhkan darinya hanyalah ID-nya, dan
//     itu dibentuk mastersupplier.ComposeApprovalID.
//  3. **Penggantian teks `'UnknownID'`.** `PEGA_M_SUPPLIER.prc:24` mengganti literal itu
//     di dalam dokumen dengan ID yang baru terbit, TETAPI tidak ada satu pun langkah di
//     `CreateNewMasterSupplier_post` yang pernah menaruh `"UnknownID"` ke halamannya.
//     Akibatnya kunci ID di dalam dokumen tersimpan KOSONG sementara kolom ID terisi
//     benar. Di sini ID yang sebenarnya ditulis ke keduanya — selisih yang direncanakan,
//     dan tidak terlihat pengguna karena layar membaca kolomnya.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input mastersupplier.Input,
	by Actor,
	logger *slog.Logger,
) (mastersupplier.Supplier, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersupplier.Supplier{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastersupplier.Supplier{}, err
	}

	id, err := store.NextID(ctx)
	if err != nil {
		return mastersupplier.Supplier{}, fmt.Errorf("mastersupplier/usecase: menerbitkan ID supplier: %w", err)
	}
	if strings.TrimSpace(id) == "" {
		// Bukan keadaan yang dapat diperbaiki pengguna, dan tidak boleh diteruskan
		// diam-diam: baris tanpa kunci tidak dapat dibuka maupun disunting.
		return mastersupplier.Supplier{}, errors.New("mastersupplier/usecase: ID supplier yang diterbitkan kosong")
	}

	now := l.clock.Now()
	fresh := apply(mastersupplier.Supplier{ID: strings.TrimSpace(id)}, clean)
	fresh.UpdatedBy = by.Login
	fresh.UpdatedAt = mastersupplier.FormatJakartaDate(now)

	// STS_AKTIF selalu "0" pada penambahan, berapa pun yang dipilih di layar
	// (`CreateNewMasterSupplier_post` step 6). Supplier baru memang belum boleh dipakai
	// sebelum permintaannya diputuskan.
	fresh.Active = mastersupplier.ActiveNo

	if err := store.Insert(ctx, fresh); err != nil {
		return mastersupplier.Supplier{}, err
	}

	// Permintaan persetujuan TANPA syarat pada jalur penambahan. Prasyarat Pega di
	// step 12 hanyalah "penyimpanannya berhasil" (`@equals(OutputKonversi.ERRMSG,"")`),
	// dan itu sudah dipenuhi baris di atas yang mengembalikan galat bila gagal.
	if err := l.requestApproval(ctx, store, fresh, by, now); err != nil {
		return mastersupplier.Supplier{}, err
	}

	logSaved(logger, portalAlias, fresh, by, "supplier baru ditambahkan", true)
	return fresh, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Nama TIDAK dapat diubah
//
// `Section/CreateMasterSupplier_Sec-Section.xml` memasang syarat read-only pada isian
// NAMA — satu-satunya isian di form itu yang punya syarat semacam itu:
//
//	pyReadOnlyCondition: MasterSupplier.ID != ''
//
// Layar Pega menegakkannya dengan mengunci isiannya, dan hanya itu.
//
// Di sini server ikut memeriksanya. Penguncian di antarmuka adalah kenyamanan tampilan;
// permintaan yang tidak datang dari layar itu — dan setiap permintaan HTTP dapat
// dibentuk tanpa melewatinya — tidak tersentuh olehnya sama sekali.
//
// # Yang TIDAK ikut tersimpan
//
//	ID           kunci baris, diambil dari jalur URL, bukan dari badan permintaan
//	OLDID        kolom warisan; dipertahankan dari baris yang tersimpan
//	SUPPLIER_HE  diturunkan dari JENIS_STATUS, bukan dikirim layar
//	STS_AKTIF    diisi dari STS_AKTIF_PROMLIST (`EditMasterSupplier_post` step 7)
func (l *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input mastersupplier.Input,
	by Actor,
	logger *slog.Logger,
) (mastersupplier.Supplier, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersupplier.Supplier{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return mastersupplier.Supplier{}, mastersupplier.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastersupplier.Supplier{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return mastersupplier.Supplier{}, err
	}

	// Perbandingannya mengabaikan besar-kecil huruf dan spasi di ujung: yang dilarang
	// adalah MENGGANTI namanya, bukan mengirimkannya kembali dengan huruf yang sedikit
	// berbeda. Layar mengirim nilai yang dimuatnya apa adanya, tetapi klien lain belum
	// tentu.
	if !sameName(clean.Name, stored.Name) {
		return mastersupplier.Supplier{}, mastersupplier.ErrNameLocked
	}

	now := l.clock.Now()
	updated := apply(stored, clean)
	updated.Name = stored.Name // nama yang tersimpan yang menang, bukan yang dikirim
	updated.UpdatedBy = by.Login
	updated.UpdatedAt = mastersupplier.FormatJakartaDate(now)

	// STS_AKTIF menyusul pilihan di layar (`EditMasterSupplier_post` step 7) — berbeda
	// dari jalur penambahan yang selalu "0".
	updated.Active = clean.ActiveRequested

	if err := store.Update(ctx, updated); err != nil {
		return mastersupplier.Supplier{}, err
	}

	// # Kenapa persetujuan TIDAK selalu diminta pada jalur ini
	//
	// `EditMasterSupplier_post` step 12 memasang prasyarat
	// `@String.equals(STS_AKTIF_PROMLIST,"1") || @String.equals(STS_AKTIF_PROMLIST,"")`.
	// Artinya menonaktifkan sebuah supplier tersimpan LANGSUNG tanpa persetujuan siapa
	// pun, sedangkan mengaktifkannya harus menunggu.
	//
	// Itu tampak disengaja dan arahnya masuk akal — menutup kerja sama tidak perlu izin,
	// membukanya perlu — sehingga ditiru apa adanya. Nilai kosong ikut diterima persis
	// seperti prasyaratnya, meski form mewajibkan isian itu: permintaan yang tidak datang
	// dari form dapat mengirimkannya kosong, dan Pega memperlakukannya sebagai
	// "perlu persetujuan", bukan sebagai "lewati".
	requested := needApproval(clean.ActiveRequested)
	if requested {
		if err := l.requestApproval(ctx, store, updated, by, now); err != nil {
			return mastersupplier.Supplier{}, err
		}
	}

	logSaved(logger, portalAlias, updated, by, "supplier diubah", requested)
	return updated, nil
}

// ListBranches melayani dropdown Cabang pada form.
func (l *Service) ListBranches(ctx context.Context, portalAlias string) ([]mastersupplier.Branch, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.ListBranches(ctx)
}

// SearchCities melayani lookup Kota pada form.
//
// Kata kunci yang terlalu pendek dijawab daftar kosong, BUKAN galat: pengguna yang baru
// mengetik satu huruf belum melakukan kesalahan apa pun, dan pesan galat di bawah kotak
// pencarian yang sedang diketik hanya akan mengganggu.
func (l *Service) SearchCities(ctx context.Context, portalAlias, keyword string) ([]mastersupplier.City, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	clean := strings.TrimSpace(keyword)
	if len(clean) < mastersupplier.MinLookupKeyword {
		return nil, nil
	}
	return store.SearchCities(ctx, clean)
}

// ListCountries melayani lookup Negara pada form.
func (l *Service) ListCountries(ctx context.Context, portalAlias string) ([]mastersupplier.Country, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.ListCountries(ctx)
}

// ListBanks melayani dropdown Bank pada form.
func (l *Service) ListBanks(ctx context.Context, portalAlias string) ([]mastersupplier.Bank, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.ListBanks(ctx)
}

// ListCodes melayani kelima dropdown bersandi.
//
// Ia MENGGABUNGKAN dua sumber, dan penggabungannya ada di sini — bukan di adapter —
// karena ia aturan, bukan penyimpanan:
//
//	mastersupplier.DefaultCodeOption()  sandi yang artinya TERBUKTI dari export
//	store.ListCodes(ctx)                sandi yang BENAR-BENAR dipakai baris yang ada
//
// Yang pertama lebih dulu supaya nilai yang artinya diketahui muncul di puncak daftar,
// dan yang kedua melengkapi dengan nilai yang tidak terbaca dari export mana pun.
//
// Tanpa penggabungan ini, basis data yang masih kosong akan menyajikan lima dropdown
// tanpa satu pun pilihan — dan supplier pertama tidak akan pernah dapat ditambahkan.
func (l *Service) ListCodes(ctx context.Context, portalAlias string) (mastersupplier.CodeSet, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersupplier.CodeSet{}, err
	}

	stored, err := store.ListCodes(ctx)
	if err != nil {
		return mastersupplier.CodeSet{}, err
	}

	known := mastersupplier.DefaultCodeOption()
	return mastersupplier.CodeSet{
		PartnerStatus: mergeOption(known.PartnerStatus, stored.PartnerStatus),
		SupplyType:    mergeOption(known.SupplyType, stored.SupplyType),
		SupplierType:  mergeOption(known.SupplierType, stored.SupplierType),
		Active:        mergeOption(known.Active, stored.Active),
		AutoPayment:   mergeOption(known.AutoPayment, stored.AutoPayment),
	}, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("mastersupplier/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// requestApproval menyisipkan satu baris permintaan persetujuan.
//
// Isinya meniru `RDB List/InsertProteksiKlaimMBU_SQL-SQL.xml` kolom per kolom, termasuk
// dua kolom yang sistem lama isi kosong tanpa syarat — CATATAN dan APPROVAL. Keduanya
// tetap disebut eksplisit alih-alih dibiarkan bernilai bawaan yang tidak diketahui
// (`R-08`).
//
// Dokumen supplier ikut disalin ke barisnya, persis seperti sistem lama — lihat
// mastersupplier.ApprovalRequest.Snapshot untuk alasannya.
func (l *Service) requestApproval(
	ctx context.Context,
	store mastersupplier.Store,
	saved mastersupplier.Supplier,
	by Actor,
	at time.Time,
) error {
	request := mastersupplier.ApprovalRequest{
		ID:          mastersupplier.ComposeApprovalID(saved.ID, at),
		SupplierID:  saved.ID,
		RequestedBy: by.Login,
		RequestedAt: at,
		Reason:      saved.Note,
		Note:        "",
		Decision:    "",
		Position:    mastersupplier.PositionRequested,
		Snapshot:    saved,
	}

	if err := store.RequestApproval(ctx, request); err != nil {
		// Galatnya dibungkus, bukan diteruskan apa adanya: penyimpanan master-nya sudah
		// berhasil pada titik ini, dan pesan yang sampai ke petugas harus membedakan
		// "supplier tidak tersimpan" dari "supplier tersimpan tetapi tidak masuk antrean
		// persetujuan". Keduanya menuntut tindakan yang berbeda.
		return fmt.Errorf("mastersupplier/usecase: supplier %q tersimpan tetapi permintaan persetujuannya gagal: %w",
			saved.ID, err)
	}
	return nil
}

// apply menyalin isian yang dikirim layar ke atas baris yang sudah ada.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya kedua jalur tidak pernah berbeda soal
// kunci mana yang ikut berubah. Empat nilai sengaja tidak disentuh — ID, OldID, Active,
// dan jejak pemanggil — dan keempatnya diatur pemanggil.
//
// SUPPLIER_HE diturunkan di sini, bukan di pemanggil, supaya tidak ada satu pun jalur
// yang dapat menyimpan JENIS_STATUS tanpa memperbarui pasangannya.
func apply(base mastersupplier.Supplier, in mastersupplier.Input) mastersupplier.Supplier {
	base.Name = in.Name
	base.Address = in.Address
	base.City = in.City
	base.BranchName = in.BranchName
	base.PostalCode = in.PostalCode
	base.Country = in.Country

	base.Phone = in.Phone
	base.Fax = in.Fax
	base.Email = in.Email

	base.TaxNumber = in.TaxNumber
	base.ContactPerson = in.ContactPerson

	base.PartnerStatus = in.PartnerStatus
	base.SupplyType = in.SupplyType
	base.HeavyEquipment = mastersupplier.DeriveHeavyEquipment(in.SupplyType)

	base.TermOfPayment = in.TermOfPayment
	base.TermOfDelivery = in.TermOfDelivery
	base.Note = in.Note

	base.Bank = in.Bank
	base.AccountNumber = in.AccountNumber
	base.AccountName = in.AccountName
	base.BankBranch = in.BankBranch

	base.SupplierType = in.SupplierType
	base.ActiveRequested = in.ActiveRequested
	base.AutoPayment = in.AutoPayment

	return base
}

// sameName membandingkan dua nama supplier tanpa memandang besar-kecil huruf dan spasi.
func sameName(sent, stored string) bool {
	return strings.EqualFold(strings.TrimSpace(sent), strings.TrimSpace(stored))
}

// needApproval menyatakan sebuah penyimpanan meminta persetujuan.
//
// Meniru prasyarat `EditMasterSupplier_post` step 12 apa adanya, termasuk nilai kosong
// yang ikut diterima.
func needApproval(activeRequested string) bool {
	clean := strings.TrimSpace(activeRequested)
	return clean == mastersupplier.ActiveYes || clean == ""
}

// mergeOption menggabungkan dua daftar pilihan tanpa nilai kembar.
//
// Urutannya dijaga: yang pertama tetap di depan, dan yang kedua menyusul pada urutan yang
// dikembalikan adapter. Nilai yang sudah ada TIDAK ditimpa labelnya — label yang artinya
// terbukti lebih berguna daripada sandi mentah yang kebetulan ditemukan di data.
func mergeOption(known, stored []mastersupplier.CodeOption) []mastersupplier.CodeOption {
	result := make([]mastersupplier.CodeOption, 0, len(known)+len(stored))
	seen := make(map[string]bool, len(known)+len(stored))

	for _, one := range append(append([]mastersupplier.CodeOption{}, known...), stored...) {
		value := strings.TrimSpace(one.Value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		label := strings.TrimSpace(one.Label)
		if label == "" {
			label = value
		}
		result = append(result, mastersupplier.CodeOption{Value: value, Label: label})
	}
	return result
}

// logSaved mencatat penyimpanan beserta apakah persetujuannya ikut diminta.
//
// Penanda approval sengaja ikut dicatat: menonaktifkan sebuah supplier tersimpan tanpa
// melewati persetujuan siapa pun, dan itu satu-satunya jalur di modul ini yang mengubah
// keadaan tanpa jejak di antrean mana pun.
func logSaved(
	logger *slog.Logger,
	portalAlias string,
	saved mastersupplier.Supplier,
	by Actor,
	event string,
	approvalRequested bool,
) {
	if logger == nil {
		return
	}
	logger.Info("master supplier tersimpan",
		slog.String("peristiwa", event),
		slog.String("portal", portalAlias),
		slog.String("id_supplier", saved.ID),
		slog.String("nama", saved.Name),
		slog.String("status_aktif", saved.Active),
		slog.Bool("persetujuan_diminta", approvalRequested),
		slog.String("oleh", by.Login))
}
