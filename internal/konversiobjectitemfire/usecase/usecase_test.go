package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ki "claim-pnc/internal/konversiobjectitemfire"
)

type fakeSource map[string][]ki.ObjectRow // kunci: nomor polis

func (f fakeSource) Objects(_ context.Context, policyNo string) ([]ki.ObjectRow, error) {
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
	return Applied{ObjectsUpdated: len(batch.Objects), ItemsInserted: len(batch.Items)}, nil
}

func TestSetiapPolisDilaporkanTerpisah(t *testing.T) {
	source := fakeSource{
		"F1": {
			{IDPega: "X", PolicyNo: "F1", ProdKe: "0", IndexObject: "1",
				Document: []byte(`{"PropertyItemList":[{"ItemType":"BUILDING"},{"ItemType":"CONTENTS"}]}`)},
			{IDPega: "X", PolicyNo: "F1", ProdKe: "1", IndexObject: "1"}, // tanpa dokumen
		},
	}
	target := &fakeTarget{}
	report, err := New(source, target, func() time.Time { return time.Unix(0, 0) }).
		Run(context.Background(), []string{"F1", "TIDAKADA", "RUSAK"}, false)
	require.NoError(t, err)

	require.True(t, report.UjiCoba)
	require.Equal(t, []bool{false}, target.commits, "uji coba tidak meng-commit")
	require.Equal(t, 1, report.Berhasil)
	require.Equal(t, 1, report.TidakAda)
	require.Equal(t, 1, report.Gagal)
	require.Equal(t, 2, report.TotalItem)

	f1 := report.Polis[0]
	require.Equal(t, []string{"0", "1"}, f1.Versi)
	require.Equal(t, 2, f1.Objek)
	require.Equal(t, 1, f1.DokumenKosong)
	require.Len(t, target.batches[0].Objects, 2, "BLOB kosong tetap disalin, supaya TEST sama dengan LIVE")
	require.Contains(t, report.Polis[2].Galat, "ORA-12541")
}

func TestDokumenRusakMenggagalkanPolisItuSaja(t *testing.T) {
	source := fakeSource{
		"F1": {{PolicyNo: "F1", ProdKe: "0", IndexObject: "1", Document: []byte(`{"Lain":1}`)}},
		"F2": {{PolicyNo: "F2", ProdKe: "0", IndexObject: "1", Document: []byte(`[{"ItemType":"STOCK"}]`)}},
	}
	target := &fakeTarget{}
	report, err := New(source, target, nil).Run(context.Background(), []string{"F1", "F2"}, true)
	require.NoError(t, err)
	require.Equal(t, "gagal", report.Polis[0].Status)
	require.Contains(t, report.Polis[0].Galat, "PropertyItemList")
	require.Equal(t, "berhasil", report.Polis[1].Status)
	require.Equal(t, []bool{true}, target.commits)
}

func TestDaftarKosongDanTerlaluPanjangDitolak(t *testing.T) {
	_, err := New(fakeSource{}, &fakeTarget{}, nil).Run(context.Background(), nil, true)
	require.ErrorIs(t, err, ErrEmptyList)
	_, err = New(fakeSource{}, &fakeTarget{}, nil).Run(context.Background(), make([]string, MaxPolicies+1), true)
	require.ErrorIs(t, err, ErrTooMany)
}
