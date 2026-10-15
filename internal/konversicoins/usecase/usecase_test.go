package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kc "claim-pnc/internal/konversicoins"
)

type fakeSource map[string][]kc.PolicyDoc // kunci: nomor polis

func (f fakeSource) Documents(_ context.Context, policyNo string) ([]kc.PolicyDoc, error) {
	if policyNo == "RUSAK" {
		return nil, errors.New("ORA-12541")
	}
	return f[policyNo], nil
}

type fakeTarget struct {
	batches []Batch
	commits []bool
}

func (f *fakeTarget) Apply(_ context.Context, batch Batch, commit bool) (Applied, error) {
	f.batches = append(f.batches, batch)
	f.commits = append(f.commits, commit)
	return Applied{Inserted: len(batch.Rows)}, nil
}

func TestSetiapPolisDilaporkanTerpisah(t *testing.T) {
	source := fakeSource{
		"P1": {
			{IDPega: "X0", PolicyNo: "P1", ProdKe: "0",
				Document: []byte(`{"CoinsList":[{"CoinsID":"A"},{"CoinsID":"B"}]}`)},
			{IDPega: "X1", PolicyNo: "P1", ProdKe: "1", Document: []byte(`{"TypeOfCoins":"0"}`)},
		},
	}
	target := &fakeTarget{}
	report, err := New(source, target, func() time.Time { return time.Unix(0, 0) }).
		Run(context.Background(), []string{"P1", "TIDAKADA", "RUSAK"}, false)
	require.NoError(t, err)

	require.True(t, report.UjiCoba)
	require.Equal(t, []bool{false}, target.commits, "uji coba tidak meng-commit")
	require.Equal(t, 1, report.Berhasil)
	require.Equal(t, 1, report.TidakAda)
	require.Equal(t, 1, report.Gagal)
	require.Equal(t, 2, report.TotalBaris)

	p1 := report.Polis[0]
	require.Equal(t, []string{"0", "1"}, p1.Versi)
	require.Equal(t, 1, p1.TanpaCoins)
	require.Equal(t, []string{"0", "1"}, target.batches[0].Versions,
		"versi tanpa koasuransi tetap diganti, supaya TEST sama dengan LIVE")
	require.Contains(t, report.Polis[2].Galat, "ORA-12541")
}

func TestDokumenRusakMenggagalkanPolisItuSaja(t *testing.T) {
	source := fakeSource{
		"P1": {{PolicyNo: "P1", ProdKe: "0", Document: []byte(`{rusak`)}},
		"P2": {{PolicyNo: "P2", ProdKe: "0", Document: []byte(`{"CoinsList":[{"CoinsID":"A"}]}`)}},
	}
	target := &fakeTarget{}
	report, err := New(source, target, nil).Run(context.Background(), []string{"P1", "P2"}, true)
	require.NoError(t, err)
	require.Equal(t, "gagal", report.Polis[0].Status)
	require.Equal(t, "berhasil", report.Polis[1].Status)
	require.Equal(t, []bool{true}, target.commits)
}

func TestDaftarKosongDanTerlaluPanjangDitolak(t *testing.T) {
	_, err := New(fakeSource{}, &fakeTarget{}, nil).Run(context.Background(), nil, true)
	require.ErrorIs(t, err, ErrEmptyList)
	_, err = New(fakeSource{}, &fakeTarget{}, nil).Run(context.Background(), make([]string, MaxPolicies+1), true)
	require.ErrorIs(t, err, ErrTooMany)
}
