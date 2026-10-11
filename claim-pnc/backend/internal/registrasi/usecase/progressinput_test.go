package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// fakeProgressInput adalah ProgressInputStore di memori.
type fakeProgressInput struct {
	positions []registrasi.ProgressPosition
	needs     map[string]string
	today     []string
	inserted  []registrasi.ProgressInput
	broker    []registrasi.BrokerEmail
	brokerErr error
}

func (f *fakeProgressInput) Positions(context.Context, string) ([]registrasi.ProgressPosition, error) {
	return f.positions, nil
}
func (f *fakeProgressInput) Status1Options(context.Context) ([]registrasi.ProgressOption, error) {
	return []registrasi.ProgressOption{{ID: "004", Name: "SURVEY"}}, nil
}
func (f *fakeProgressInput) StatusOptions(context.Context) ([]string, error) {
	return []string{"SUDAH BERHASIL DIHUBUNGKAN"}, nil
}
func (f *fakeProgressInput) DocTypes(context.Context) ([]string, error) { return []string{"ALL"}, nil }
func (f *fakeProgressInput) NeedOptions(_ context.Context, p1, doc string) ([]registrasi.ProgressOption, error) {
	return []registrasi.ProgressOption{{ID: p1 + "/" + doc}}, nil
}
func (f *fakeProgressInput) NeedName(_ context.Context, id string) (string, bool, error) {
	n, ok := f.needs[id]
	return n, ok, nil
}
func (f *fakeProgressInput) InputToday(context.Context, string, time.Time) ([]string, error) {
	return f.today, nil
}
func (f *fakeProgressInput) Insert(_ context.Context, p registrasi.ProgressInput) error {
	f.inserted = append(f.inserted, p)
	return nil
}
func (f *fakeProgressInput) BrokerEmail(context.Context, registrasi.BrokerEmail) (string, error) {
	return "lama@contoh.internal", nil
}
func (f *fakeProgressInput) SaveBrokerEmail(_ context.Context, b registrasi.BrokerEmail) error {
	f.broker = append(f.broker, b)
	return f.brokerErr
}

func progressEnv(t *testing.T) (environment, *fakeProgressInput, registrasi.Claim) {
	t.Helper()
	fake := &fakeProgressInput{
		positions: []registrasi.ProgressPosition{{ID: "1", Name: "REGISTER", Status: "Done"}, {ID: "2", Name: "SURVEY", Status: "On Progress"}},
		needs:     map[string]string{"59": "HASIL SURVEY", "14": "BELUM DI FU"},
	}
	l := setupWith(t, func(o *usecase.Options) { o.ProgressInput = fake })
	claim, _ := l.upToInputRegister(t, firePolicy)
	return l, fake, claim
}

func violationMessages(t *testing.T, err error) []string {
	t.Helper()
	var v *registrasi.ValidationError
	require.ErrorAs(t, err, &v)
	out := []string{}
	for _, x := range v.Violation {
		out = append(out, x.Message)
	}
	return out
}

func TestInputProgress(t *testing.T) {
	l, fake, claim := progressEnv(t)
	ctx := context.Background()

	form, err := l.service.ProgressInputForm(ctx, claim.ID)
	require.NoError(t, err)
	require.Len(t, form.Positions, 2)
	require.Equal(t, "lama@contoh.internal", form.EmailBroker)

	// Tipe Dokumen hanya menyaring Status Progress 1 = 003.
	opts, err := l.service.ProgressNeedOptions(ctx, "004", "ALL")
	require.NoError(t, err)
	require.Equal(t, "004/", opts[0].ID)
	opts, err = l.service.ProgressNeedOptions(ctx, "003", "ALL")
	require.NoError(t, err)
	require.Equal(t, "003/ALL", opts[0].ID)

	cmd := usecase.ProgressInputCommand{ClaimID: claim.ID, PositionID: "2", Progress1: "004", NeedIDs: []string{"14", "59"},
		Status: "SUDAH BERHASIL DIHUBUNGKAN", Remarks: "catatan", DocType: "ALL", EmailBroker: "baru@contoh.internal"}

	// Posisi Done tidak menampilkan tombol; posisi yang tidak ada ditolak.
	closed := cmd
	closed.PositionID = "1"
	_, err = l.service.InputProgress(ctx, closed, l.caller)
	require.Equal(t, []string{registrasi.MsgProgressPositionClosed}, violationMessages(t, err))
	missing := cmd
	missing.PositionID = "9"
	_, err = l.service.InputProgress(ctx, missing, l.caller)
	require.Equal(t, []string{registrasi.MsgProgressPositionMissing}, violationMessages(t, err))

	// STATUS PERLU di luar master dan STATUS PERLU baris terakhir yang sudah diinput hari ini.
	fake.today = []string{"X, HASIL SURVEY"}
	bad := cmd
	bad.NeedIDs = []string{"77", "59"}
	_, err = l.service.InputProgress(ctx, bad, l.caller)
	require.Equal(t, []string{registrasi.MsgProgressAlreadyToday, registrasi.MsgProgressNeedFromMaster}, violationMessages(t, err))
	require.Empty(t, fake.inserted)

	fake.today = nil
	res, err := l.service.InputProgress(ctx, cmd, l.caller)
	require.NoError(t, err)
	require.Empty(t, res.Warnings)
	require.Len(t, fake.inserted, 1)
	got := fake.inserted[0]
	require.Equal(t, claim.Number, got.ClaimNumber)
	require.Equal(t, "2", got.PositionID)
	require.Equal(t, "59", got.LastNeed().ID)
	require.Equal(t, "HASIL SURVEY", got.LastNeed().Name)
	require.Empty(t, got.DocType, "Tipe Dokumen hanya untuk Status Progress 1 = 003")
	require.Equal(t, "baru@contoh.internal", fake.broker[0].Email)

	// Galat simpan master email broker tidak membatalkan progres.
	fake.brokerErr = errors.New("gagal")
	res, err = l.service.InputProgress(ctx, cmd, l.caller)
	require.NoError(t, err)
	require.Len(t, res.Warnings, 1)
	require.Len(t, fake.inserted, 2)
}

func TestInputProgressUnavailable(t *testing.T) {
	l := setup(t)
	_, err := l.service.ProgressInputForm(context.Background(), "x")
	require.ErrorIs(t, err, registrasi.ErrProgressInputUnavailable)
	_, err = l.service.InputProgress(context.Background(), usecase.ProgressInputCommand{}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrProgressInputUnavailable)
	_, err = l.service.ProgressNeedOptions(context.Background(), "004", "")
	require.ErrorIs(t, err, registrasi.ErrProgressInputUnavailable)
}
