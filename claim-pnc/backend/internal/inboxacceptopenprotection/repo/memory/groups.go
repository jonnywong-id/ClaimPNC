package memory

import (
	"context"
	"strings"
)

// GroupRepo adalah pembaca access group untuk pengujian dan pengembangan lokal.
//
// Memenuhi seam `inboxacceptopenprotection.GroupReader`.
type GroupRepo struct {
	byLogin map[string][]string
}

// NewGroupRepo membentuk pembaca dari peta login -> daftar GROUP_ID.
//
// Peta kosong berarti tidak seorang pun berwenang — dan itu keadaan yang memang perlu dapat
// diuji: tabel `M_LOGIN_GROUP_PNC` yang belum diisi harus MENUTUP layar, bukan membukanya.
func NewGroupRepo(byLogin map[string][]string) *GroupRepo {
	salinan := make(map[string][]string, len(byLogin))
	for login, groups := range byLogin {
		salinan[kunciLogin(login)] = append([]string(nil), groups...)
	}
	return &GroupRepo{byLogin: salinan}
}

// NewSampleGroupRepo membentuk pembaca berisi contoh untuk pengembangan lokal.
func NewSampleGroupRepo() *GroupRepo { return NewGroupRepo(SampleGroups()) }

// SampleGroups adalah contoh keanggotaan group.
//
// Seluruhnya KARANGAN (`D-69`), dan sengaja mewakili keempat keadaan yang berbeda
// penanganannya:
//
//	ADMINCONTOH     PncAdmin                     -> TIDAK berwenang; bukan satu dari lima
//	KOLEKSICONTOH   PncCollection                -> antrean PREMI saja
//	TEKNIKCONTOH    CaseManager                  -> antrean NON PREMI saja
//	KEDUACONTOH     CaseManager + PncCollection  -> KEDUA antrean
//
// Baris pertama ada justru karena ia harus DITOLAK. Contoh yang seluruhnya berwenang tidak
// membuktikan pemeriksaannya bekerja.
//
// Kapitalisasi `pnccollection` pada baris terakhir juga disengaja: `D-58` mencatat Pega
// tidak konsisten kapitalisasinya, dan normalisasi di domain harus tetap mengenalinya.
func SampleGroups() map[string][]string {
	return map[string][]string{
		"ADMINCONTOH":   {"PncAdmin"},
		"KOLEKSICONTOH": {"PncCollection"},
		"TEKNIKCONTOH":  {"CaseManager"},
		"KEDUACONTOH":   {"CaseManager", "pnccollection"},
	}
}

// GroupsOf mengembalikan GROUP_ID yang diikuti sebuah login.
func (r *GroupRepo) GroupsOf(_ context.Context, login string) ([]string, error) {
	groups, ada := r.byLogin[kunciLogin(login)]
	if !ada {
		return nil, nil
	}
	return append([]string(nil), groups...), nil
}

// kunciLogin merapikan login menjadi bentuk yang dibandingkan.
//
// Sama dengan `UPPER(TRIM(...))` pada kuerinya di Oracle — dua tempat yang tidak sepakat
// menghasilkan uji yang lulus sementara produksi menolak pengguna yang sah.
func kunciLogin(login string) string {
	return strings.ToUpper(strings.TrimSpace(login))
}
