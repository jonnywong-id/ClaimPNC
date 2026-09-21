package directory_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/masterpicteknik/directory"
)

// errNotRegistered meniru provider.ErrServiceNotRegistered milik modul auth.
//
// Ia dideklarasikan ulang di sini, bukan diimpor, karena itulah bentuk pemakaiannya yang
// sebenarnya: adapter menerima galat itu sebagai NILAI dari cmd, sehingga kedua modul
// tidak saling mengimpor.
var errNotRegistered = errors.New("uji: alamat layanan tidak terdaftar")

// catalogStub merekam apa yang ditanyakan adapter ke katalog layanan.
//
// Rekaman itulah inti berkas ini: yang diuji bukan hanya "adapter mengembalikan nama",
// melainkan bahwa ia mencari alamatnya dengan APP dan TYPESERVICE yang benar.
type catalogStub struct {
	address string
	err     error

	askedApp  string
	askedKind string
	calls     int
}

func (c *catalogStub) ServiceAddress(_ context.Context, app, serviceKind string) (string, error) {
	c.calls++
	c.askedApp = app
	c.askedKind = serviceKind
	if c.err != nil {
		return "", c.err
	}
	return c.address, nil
}

// employeeResponse adalah bentuk respons HCQ yang dipakai seluruh uji di sini.
//
// Ia DISALIN dari contoh respons yang diberikan Work Owner 2026-09-16 dan tersimpan di
// `internal/auth/provider/hcq_test.go:52` — termasuk blok `EmpLeader`, yang justru bagian
// yang membuat satu pencarian mengisi dua isian sekaligus.
const employeeResponse = `{
  "Response": {"pyErrorCode":"200","pyStatusMessage":"OK"},
  "EmpResponse": {
    "Person": {"NIK":"90000002","Login":"PICTEKNIK05","Name":"Contoh Petugas Baru","pyEmail1":"contoh.baru@example.invalid"},
    "Placement": {"NIK":"90000002","Name":"Contoh Petugas Baru","BranchName":"KANTOR PUSAT"}
  },
  "EmpLeader": {"Person":{"NIK":"88880000","Login":"PICTEKNIK01","Name":"Contoh Kepala Teknik"}},
  "Login": "PICTEKNIK05"
}`

// newAdapter merakit adapter di atas peladen tiruan, dan mengembalikan katalognya supaya
// uji dapat memeriksa apa yang ditanyakan.
func newAdapter(t *testing.T, handler http.HandlerFunc) (*directory.HCQ, *catalogStub, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	catalog := &catalogStub{address: server.URL}
	adapter, err := directory.New(directory.Options{
		Catalog:       catalog,
		User:          "pengguna-aplikasi",
		Password:      "sandi-aplikasi",
		NotRegistered: errNotRegistered,
	})
	require.NoError(t, err)
	return adapter, catalog, server
}

func TestNewRejectsIncompleteOptions(t *testing.T) {
	_, err := directory.New(directory.Options{User: "a", Password: "b"})
	require.Error(t, err)

	_, err = directory.New(directory.Options{Catalog: &catalogStub{}, Password: "b"})
	require.Error(t, err)

	_, err = directory.New(directory.Options{Catalog: &catalogStub{}, User: "a"})
	require.Error(t, err)
}

// Inilah uji yang menjawab instruksi Work Owner secara langsung: alamat layanan dibaca
// dari katalog dengan APP = alias portal dan TYPESERVICE = 'HCQ-LOGIN'.
//
//	SELECT servicename FROM POOLDATA.GCNM_CONNECT_REST
//	 WHERE app = <portal_alias> AND typeservice = 'HCQ-LOGIN';
//
// Kueri itu sendiri hidup di internal/auth/repo/sqlstore/legacy.sql; yang dibuktikan di
// sini adalah kedua NILAI yang dikirim adapter ke sana.
func TestLookupAsksCatalogWithPortalAliasAndHCQLogin(t *testing.T) {
	adapter, catalog, _ := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(employeeResponse))
	})

	_, err := adapter.Lookup(context.Background(), "asm", "PICTEKNIK05")
	require.NoError(t, err)

	require.Equal(t, 1, catalog.calls)
	// Alias dinaikkan menjadi huruf besar: kolom APP menyimpannya demikian, dan pengguna
	// dapat memilih portal dengan huruf kecil.
	require.Equal(t, "ASM", catalog.askedApp)
	require.Equal(t, "HCQ-LOGIN", catalog.askedKind)
	require.Equal(t, "HCQ-LOGIN", directory.DefaultServiceKind)
}

// Satu pencarian menjawab Nama DAN Atasan — blok EmpLeader pada respons.
func TestLookupReturnsNameAndLeaderFromOneCall(t *testing.T) {
	adapter, _, _ := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(employeeResponse))
	})

	employee, err := adapter.Lookup(context.Background(), "ASM", "picteknik05")
	require.NoError(t, err)

	require.Equal(t, "Contoh Petugas Baru", employee.Name)
	require.Equal(t, "contoh.baru@example.invalid", employee.Email)
	require.Equal(t, "PICTEKNIK01", employee.SupervisorID)
	require.Equal(t, "Contoh Kepala Teknik", employee.SupervisorName)
	// Identitas dikembalikan sebagaimana dikenal HCQ, bukan sebagaimana diketik pengguna.
	require.Equal(t, "PICTEKNIK05", employee.OperatorID)
}

// Permintaan membawa Login huruf besar dan sandi pengisi, meniru
// `SetMstUserTeknisMstUser_act` langkah 6 (`@toUpperCase(...)` dan `pyPwdCurrent = "1"`),
// ditambah Basic Auth kredensial APLIKASI.
func TestLookupSendsUppercaseLoginWithFillerPasswordAndBasicAuth(t *testing.T) {
	var sent struct {
		login    string
		password string
		user     string
		pass     string
		hasAuth  bool
	}

	adapter, _, _ := newAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Login    string `json:"Login"`
			Password string `json:"Password"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		sent.login, sent.password = payload.Login, payload.Password
		sent.user, sent.pass, sent.hasAuth = r.BasicAuth()
		_, _ = w.Write([]byte(employeeResponse))
	})

	_, err := adapter.Lookup(context.Background(), "ASM", "  picteknik05  ")
	require.NoError(t, err)

	require.Equal(t, "PICTEKNIK05", sent.login)
	require.Equal(t, directory.DefaultLookupPassword, sent.password)
	require.True(t, sent.hasAuth)
	require.Equal(t, "pengguna-aplikasi", sent.user)
	require.Equal(t, "sandi-aplikasi", sent.pass)
}

// Verdict sandi TIDAK dipakai sebagai penentu.
//
// Ini perilaku yang paling mudah salah dibaca, dan justru yang membuat pencarian mungkin:
// sandi yang dikirim adalah pengisi, sehingga `pyErrorCode` hampir pasti bukan "200".
// Yang menentukan adalah ada-tidaknya nama di dalam EmpResponse.
func TestLookupIgnoresErrorCodeWhenEmployeeBlockIsPresent(t *testing.T) {
	const wrongPasswordButEmployeePresent = `{
	  "Response": {"pyErrorCode":"403","pyStatusMessage":"Invalid password"},
	  "EmpResponse": {
	    "Person": {"NIK":"90000002","Login":"PICTEKNIK05","Name":"Contoh Petugas Baru"}
	  },
	  "EmpLeader": {"Person":{"Login":"PICTEKNIK01","Name":"Contoh Kepala Teknik"}}
	}`

	adapter, _, _ := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(wrongPasswordButEmployeePresent))
	})

	employee, err := adapter.Lookup(context.Background(), "ASM", "PICTEKNIK05")
	require.NoError(t, err)
	require.Equal(t, "Contoh Petugas Baru", employee.Name)
}

// Tidak ada nama berarti pegawainya tidak dikenal — keadaan yang di sistem lama membuat
// `TempDcol.MCL_NAME` tetap kosong lalu ditolak langkah 2.
func TestLookupWithoutNameIsUnknownEmployee(t *testing.T) {
	adapter, _, _ := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Response":{"pyErrorCode":"200"},"EmpResponse":{}}`))
	})

	_, err := adapter.Lookup(context.Background(), "ASM", "TIDAKTERDAFTAR")
	require.ErrorIs(t, err, masterpicteknik.ErrEmployeeUnknown)
}

func TestLookupWithEmptyIDDoesNotCallAnything(t *testing.T) {
	adapter, catalog, _ := newAdapter(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("layanan tidak boleh ditembak untuk ID kosong")
	})

	_, err := adapter.Lookup(context.Background(), "ASM", "   ")
	require.ErrorIs(t, err, masterpicteknik.ErrEmployeeUnknown)
	require.Zero(t, catalog.calls)
}

// Katalog yang belum memuat barisnya DIBEDAKAN dari jaringan yang putus: yang satu tidak
// akan pulih sendiri, dan menyuruh pengguna mencoba lagi hanya membuang waktunya.
func TestUnregisteredServiceIsDistinctFromOutage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)

	catalog := &catalogStub{err: errNotRegistered}
	adapter, err := directory.New(directory.Options{
		Catalog:       catalog,
		User:          "a",
		Password:      "b",
		NotRegistered: errNotRegistered,
	})
	require.NoError(t, err)

	_, err = adapter.Lookup(context.Background(), "ASI", "PICTEKNIK05")
	require.ErrorIs(t, err, masterpicteknik.ErrDirectoryNotConfigured)
	require.NotErrorIs(t, err, masterpicteknik.ErrDirectoryUnreachable)
	// Pesannya menyebut APP dan TYPESERVICE yang dicari, supaya administrator tahu baris
	// mana yang harus ditambahkan.
	require.Contains(t, err.Error(), "ASI")
	require.Contains(t, err.Error(), "HCQ-LOGIN")
}

func TestCatalogFailureBecomesUnreachable(t *testing.T) {
	catalog := &catalogStub{err: errors.New("basis data sedang tidak dapat dihubungi")}
	adapter, err := directory.New(directory.Options{
		Catalog:       catalog,
		User:          "a",
		Password:      "b",
		NotRegistered: errNotRegistered,
	})
	require.NoError(t, err)

	_, err = adapter.Lookup(context.Background(), "ASM", "PICTEKNIK05")
	require.ErrorIs(t, err, masterpicteknik.ErrDirectoryUnreachable)
}

func TestNonJSONResponseBecomesUnreachable(t *testing.T) {
	adapter, _, _ := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>gateway error</html>"))
	})

	_, err := adapter.Lookup(context.Background(), "ASM", "PICTEKNIK05")
	require.ErrorIs(t, err, masterpicteknik.ErrDirectoryUnreachable)
}

// ServiceKind dapat ditimpa, sehingga pemindahan ke layanan pencarian tersendiri kelak
// menjadi perubahan konfigurasi — bukan rilis ulang.
func TestServiceKindIsConfigurable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(employeeResponse))
	}))
	t.Cleanup(server.Close)

	catalog := &catalogStub{address: server.URL}
	adapter, err := directory.New(directory.Options{
		Catalog:        catalog,
		User:           "a",
		Password:       "b",
		ServiceKind:    "GetEmployee",
		LookupPassword: "x",
	})
	require.NoError(t, err)

	_, err = adapter.Lookup(context.Background(), "ASM", "PICTEKNIK05")
	require.NoError(t, err)
	require.Equal(t, "GetEmployee", catalog.askedKind)
}

// Direktori tiruan memenuhi seam yang sama, sehingga jalur simpan dapat diuji tanpa
// layanan luar sama sekali.
func TestFakeSatisfiesTheSameSeam(t *testing.T) {
	fake := directory.NewFake(directory.SampleEmployees()...)

	employee, err := fake.Lookup(context.Background(), "ASM", "picteknik02")
	require.NoError(t, err)
	require.Equal(t, "Contoh Adjuster Madya", employee.Name)
	require.Equal(t, "PICTEKNIK01", employee.SupervisorID)

	_, err = fake.Lookup(context.Background(), "ASM", "TIDAKTERDAFTAR")
	require.ErrorIs(t, err, masterpicteknik.ErrEmployeeUnknown)
}
