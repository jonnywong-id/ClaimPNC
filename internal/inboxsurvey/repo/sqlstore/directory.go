package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxsurvey"
)

// Directory memenuhi seam inboxsurvey.Directory — jembatan login menjadi identitas surveyor.
//
// Ia membaca `POOLDATA.MST_LOGIN_SURVEYOR`, master login surveyor yang sudah punya modulnya
// sendiri (Master Login, MENU_ID 37). Modul ini hanya MEMBACA master itu; yang memiliki dan
// menulisnya adalah modul masterlogin.
type Directory struct {
	db *sql.DB
}

// NewDirectory membentuk jembatan identitas di atas satu koneksi.
func NewDirectory(db *sql.DB) *Directory {
	return &Directory{db: db}
}

// LoginActive adalah nilai `STSLOGIN` yang berarti login surveyor masih berlaku.
//
// # Kenapa ia TIDAK dipakai menyaring
//
// Karena nilainya tidak terbaca dari export: tidak satu pun rule membandingkan `STSLOGIN`
// dengan apa pun, dan `GetLoginLeaderSurveyor` mencari HANYA dengan `login`. Menambahkan
// penyaring status berarti menebak — dan tebakan yang salah di sini mengunci seseorang dari
// pekerjaannya sendiri tanpa pesan apa pun.
//
// Nilainya tetap DIBACA dan dibawa, supaya domainnya terlihat dari data nyata dan
// penyaringnya dapat ditambahkan kemudian sebagai keputusan, bukan sebagai tebakan.
const LoginActive = "1"

// ResolveSurveyor menerjemahkan login menjadi identitas surveyor beserta cakupannya.
//
// # Dua perjalanan, dan yang kedua bersyarat
//
// Perjalanan pertama membaca barisnya sendiri. Perjalanan kedua — daftar anggota — hanya
// dijalankan bila baris pertama menunjukkan pemanggil TIDAK punya leader, yaitu ia sendiri
// yang menjadi leader.
//
// Menggabungkannya menjadi satu kueri berbiaya lebih murah, tetapi menghasilkan jawaban yang
// tidak dapat dibedakan: "login tidak terdaftar" dan "terdaftar tetapi tanpa anggota"
// keduanya mengembalikan nol baris. Yang pertama harus menjadi ErrNotSurveyor, yang kedua
// tidak — dan membedakannya jauh lebih berharga daripada satu perjalanan yang dihemat, pada
// kueri yang dijalankan sekali per permintaan terhadap tabel berukuran ratusan baris.
func (d *Directory) ResolveSurveyor(
	ctx context.Context,
	login string,
) (inboxsurvey.SurveyorIdentity, error) {
	trimmed := strings.TrimSpace(login)
	if trimmed == "" {
		return inboxsurvey.SurveyorIdentity{}, inboxsurvey.ErrCallerUnknown
	}

	var (
		surveyorLogin sql.NullString
		surveyorName  sql.NullString
		leaderLogin   sql.NullString
		loginStatus   sql.NullString
	)

	err := d.db.QueryRowContext(ctx, query("resolve_surveyor"), sql.Named("login", trimmed)).
		Scan(&surveyorLogin, &surveyorName, &leaderLogin, &loginStatus)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Pemanggil TERBACA tetapi tidak terdaftar sebagai surveyor. Itu galat tersendiri,
		// bukan antrean kosong — lihat inboxsurvey.ErrNotSurveyor.
		return inboxsurvey.SurveyorIdentity{}, inboxsurvey.ErrNotSurveyor
	case err != nil:
		return inboxsurvey.SurveyorIdentity{},
			fmt.Errorf("menjalankan kueri resolve_surveyor: %w", err)
	}

	name := strings.TrimSpace(surveyorName.String)
	if name == "" {
		// Barisnya ADA tetapi tanpa nama. Cakupan dibentuk dari NAMA, sehingga baris seperti
		// ini tidak dapat menghasilkan antrean apa pun — dan menjawabnya dengan daftar
		// kosong akan menyembunyikan data master yang cacat.
		return inboxsurvey.SurveyorIdentity{}, inboxsurvey.ErrNotSurveyor
	}

	identity := inboxsurvey.SurveyorIdentity{
		Login: strings.TrimSpace(surveyorLogin.String),
		Name:  name,
		Scope: []string{name},
	}
	if identity.Login == "" {
		identity.Login = trimmed
	}

	// Pemanggil adalah LEADER bila ia sendiri tidak menunjuk leader lain.
	//
	// Itu bacaan `LOGINLEADER` apa adanya: kolom itu menyimpan login ATASAN seseorang, dan
	// `GetLoginLeaderSurveyor` membacanya justru untuk mengetahui kepada siapa seorang
	// anggota bernaung. Baris yang kolomnya kosong karena itu adalah puncaknya.
	if strings.TrimSpace(leaderLogin.String) != "" {
		return identity, nil
	}

	members, err := d.members(ctx, identity.Login, identity.Name)
	if err != nil {
		return inboxsurvey.SurveyorIdentity{}, err
	}

	identity.IsLeader = len(members) > 0
	identity.Scope = append(identity.Scope, members...)
	return identity, nil
}

// members membaca nama seluruh surveyor yang bernaung pada satu leader.
//
// # Dua parameter untuk satu orang, dan keduanya dibutuhkan
//
// `leaderLogin` yang dikirim ke kueri, karena `LOGINLEADER` menyimpan LOGIN atasan.
// `leaderName` yang dipakai membuang duplikat, karena yang dikembalikan kueri adalah NAMA.
//
// Menyamakan keduanya adalah kesalahan yang mudah terjadi dan tidak menghasilkan galat: bila
// baris seorang leader menunjuk dirinya sendiri sebagai atasannya — bentuk yang tidak
// dilarang apa pun di master itu — namanya akan masuk cakupan DUA KALI. `INSTR` tidak
// terganggu olehnya, tetapi daftar cakupan yang dibaca saat menelusuri keluhan menjadi
// membingungkan.
func (d *Directory) members(
	ctx context.Context,
	leaderLogin string,
	leaderName string,
) ([]string, error) {
	rows, err := d.db.QueryContext(ctx, query("resolve_members"), sql.Named("leader_login", leaderLogin))
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri resolve_members: %w", err)
	}
	defer rows.Close()

	// Nama leader sudah masuk cakupan lebih dulu di pemanggil, sehingga ia ikut menjadi
	// penanda "sudah pernah dilihat" di sini.
	seen := map[string]bool{strings.ToUpper(strings.TrimSpace(leaderName)): true}

	result := []string{}
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("membaca baris kueri resolve_members: %w", err)
		}

		trimmed := strings.TrimSpace(name.String)
		key := strings.ToUpper(trimmed)
		if trimmed == "" || seen[key] {
			continue
		}

		seen[key] = true
		result = append(result, trimmed)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri resolve_members: %w", err)
	}

	return result, nil
}
