package cashier_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/cashier"
)

func account() masterrekening.Account {
	return masterrekening.Account{
		Number: "123", OwnerName: "PT A", BankCode: "014", BankName: "BANK BCA",
		BankBranch: "Thamrin", BankAddress: "Jakarta", AccountType: "Giro",
		NIK: "317", Email: "a@contoh.example", Phone: "021", DocumentID: "DOK",
		PreviousBankCode: "002", PreviousNumber: "999", PreviousOwnerName: "PT Lama",
	}
}

func TestConfigIsCompleteOnlyWithBothAddresses(t *testing.T) {
	require.True(t, cashier.Config{RegisterURL: "http://a", UpdateURL: "http://b"}.Complete())
	require.False(t, cashier.Config{RegisterURL: "http://a", UpdateURL: " "}.Complete())
	require.False(t, cashier.Config{}.Complete())
}

// server merekam badan permintaan dan menjawab dengan status dan badan yang ditentukan.
func server(t *testing.T, status int, reply string, seen *map[string]any, auth *[2]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		if seen != nil {
			require.NoError(t, json.NewDecoder(r.Body).Decode(seen))
		}
		if auth != nil {
			user, password, _ := r.BasicAuth()
			*auth = [2]string{user, password}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRegisterSendsTheAccountWithoutPreviousValues(t *testing.T) {
	var seen map[string]any
	var auth [2]string
	srv := server(t, http.StatusOK,
		`{"ReponseCode":" 0 ","ResponseMessage":" ok ","IdRekening":" ID-1 "}`, &seen, &auth)

	client := cashier.NewClient(cashier.Config{
		RegisterURL: srv.URL, UpdateURL: srv.URL, User: "u", Password: "p",
	})
	result, err := client.Register(context.Background(), account())
	require.NoError(t, err)
	require.Equal(t, masterrekening.CashierResult{
		Succeeded: true, AccountID: "ID-1", Message: "ok", Code: "0",
	}, result)

	require.Equal(t, "123", seen["no_rekening"])
	require.Equal(t, "014", seen["kode_bank"])
	require.NotContains(t, seen, "kode_bank_lama")
	require.Equal(t, [2]string{"u", "p"}, auth)
}

func TestUpdateSendsThePreviousValues(t *testing.T) {
	var seen map[string]any
	var auth [2]string
	srv := server(t, http.StatusOK, `{"ReponseCode":"1","ResponseMessage":"ditolak"}`, &seen, &auth)

	client := cashier.NewClient(cashier.Config{RegisterURL: srv.URL, UpdateURL: srv.URL})
	result, err := client.Update(context.Background(), account())
	require.NoError(t, err)
	require.False(t, result.Succeeded, "kode 1 berarti gagal")
	require.Equal(t, "ditolak", result.Message)
	require.Equal(t, "002", seen["kode_bank_lama"])
	require.Equal(t, "999", seen["no_rekening_lama"])
	require.Equal(t, "PT Lama", seen["nama_rekening_lama"])
	require.Equal(t, [2]string{"", ""}, auth, "tanpa user tidak ada basic auth")
}

func TestCode9AndNon2xxAreFailures(t *testing.T) {
	srv := server(t, http.StatusOK, `{"ReponseCode":"9"}`, nil, nil)
	result, err := cashier.NewClient(cashier.Config{RegisterURL: srv.URL}).
		Register(context.Background(), account())
	require.NoError(t, err)
	require.False(t, result.Succeeded)

	srv = server(t, http.StatusBadGateway, `{"ReponseCode":"5","ResponseMessage":"down"}`, nil, nil)
	result, err = cashier.NewClient(cashier.Config{RegisterURL: srv.URL}).
		Register(context.Background(), account())
	require.NoError(t, err)
	require.Equal(t, masterrekening.CashierResult{Succeeded: false, Code: "5", Message: "down"}, result)
}

func TestSendFailures(t *testing.T) {
	_, err := cashier.NewClient(cashier.Config{}).Register(context.Background(), account())
	require.ErrorContains(t, err, "alamat service belum dikonfigurasi")

	_, err = cashier.NewClient(cashier.Config{UpdateURL: "://bukan-alamat"}).
		Update(context.Background(), account())
	require.ErrorContains(t, err, "menyusun permintaan HTTP")

	srv := server(t, http.StatusOK, "bukan json", nil, nil)
	_, err = cashier.NewClient(cashier.Config{RegisterURL: srv.URL}).
		Register(context.Background(), account())
	require.ErrorContains(t, err, "membaca jawaban Cashier")

	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	_, err = cashier.NewClient(cashier.Config{RegisterURL: dead.URL, Timeout: 1}).
		Register(context.Background(), account())
	require.ErrorContains(t, err, "menghubungi sistem Kasir")
}

func TestFakeRecordsAndAnswers(t *testing.T) {
	fake := cashier.NewFake()

	result, err := fake.Register(context.Background(), account())
	require.NoError(t, err)
	require.True(t, result.Succeeded)
	require.Equal(t, "TIRUAN-0001", result.AccountID)

	result, err = fake.Update(context.Background(), account())
	require.NoError(t, err)
	require.True(t, result.Succeeded)

	fake.Error = errors.New("putus")
	_, err = fake.Register(context.Background(), account())
	require.EqualError(t, err, "putus")
	_, err = fake.Update(context.Background(), account())
	require.EqualError(t, err, "putus")

	require.Len(t, fake.Registered, 2)
	require.Len(t, fake.Updated, 2)
}
