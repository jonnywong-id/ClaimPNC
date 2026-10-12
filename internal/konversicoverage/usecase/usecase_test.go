package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kc "claim-pnc/internal/konversicoverage"
)

type fakeSource map[string][]kc.ObjectRow // kunci: tabel objek + "|" + polis

func (f fakeSource) Objects(_ context.Context, line kc.Line, policyNo string, filter kc.Filter) ([]kc.ObjectRow, error) {
	if policyNo == "RUSAK" {
		return nil, errors.New("ORA-12541")
	}
	var result []kc.ObjectRow
	for _, o := range f[line.ObjectTable+"|"+policyNo] {
		if (filter.ProdKe == "" || o.ProdKe == filter.ProdKe) && (filter.IndexObject == "" || o.IndexObject == filter.IndexObject) {
			result = append(result, o)
		}
	}
	return result, nil
}

type fakeTarget struct {
	batches []Batch
	commits []bool
}

func (f *fakeTarget) Apply(_ context.Context, batch Batch, commit bool) (Applied, error) {
	f.batches = append(f.batches, batch)
	f.commits = append(f.commits, commit)
	return Applied{
		ObjectsUpdated:   len(batch.Lines[0].Objects),
		CoverageInserted: len(batch.Lines[0].Coverage),
		SpreadInserted:   len(batch.Spreading),
	}, nil
}

func TestSetiapPolisDilaporkanTerpisah(t *testing.T) {
	source := fakeSource{
		"T_CARGOLIST|C1": {{IDPega: "X", PolicyNo: "C1", ProdKe: "0", IndexObject: "1",
			Document: []byte(`{"CoverageList":[{"Coverage":"ICC-A","SpreadingList":[{"TreatyType":"1"}]}]}`)}},
	}
	target := &fakeTarget{}
	report, err := New(source, target, func() time.Time { return time.Unix(0, 0) }).
		Run(context.Background(), []string{"C1", "TIDAKADA", "RUSAK"}, kc.Filter{}, false)
	require.NoError(t, err)

	require.True(t, report.UjiCoba)
	require.Equal(t, []bool{false}, target.commits, "uji coba tidak meng-commit")
	require.Equal(t, 1, report.Berhasil)
	require.Equal(t, 1, report.TidakAda)
	require.Equal(t, 1, report.Gagal)
	require.Equal(t, []string{"T_CARGOLIST"}, report.Polis[0].Lini)
	require.Equal(t, 1, report.Polis[0].CoverageDitulis)
	require.Equal(t, 1, report.Polis[0].SpreadingDitulis)
	require.Contains(t, report.Polis[2].Galat, "ORA-12541")
}

func TestDaftarKosongDitolak(t *testing.T) {
	_, err := New(fakeSource{}, &fakeTarget{}, nil).Run(context.Background(), nil, kc.Filter{}, true)
	require.ErrorIs(t, err, ErrEmptyList)
}

func TestFilterMembatasiObjekDanDiteruskanKePenulis(t *testing.T) {
	doc := []byte(`{"CoverageList":[{"Coverage":"ICC-A"}]}`)
	source := fakeSource{
		"T_CARGOLIST|C1": {
			{PolicyNo: "C1", ProdKe: "0", IndexObject: "1", Document: doc},
			{PolicyNo: "C1", ProdKe: "1", IndexObject: "1", Document: doc},
			{PolicyNo: "C1", ProdKe: "1", IndexObject: "2", Document: doc},
		},
	}
	target := &fakeTarget{}
	filter := kc.Filter{ProdKe: "1", IndexObject: "2"}
	report, err := New(source, target, nil).Run(context.Background(), []string{"C1"}, filter, true)
	require.NoError(t, err)

	require.Equal(t, filter, target.batches[0].Filter)
	require.Len(t, target.batches[0].Lines[0].Objects, 1)
	require.Equal(t, []string{"1"}, report.Polis[0].Versi)
	require.Equal(t, "1", report.FilterProd)
	require.Equal(t, "2", report.FilterIndex)
}

func TestFilterTanpaObjekCocokDilaporkanTidakDitemukan(t *testing.T) {
	source := fakeSource{"T_CARGOLIST|C1": {{PolicyNo: "C1", ProdKe: "0", IndexObject: "1"}}}
	report, err := New(source, &fakeTarget{}, nil).
		Run(context.Background(), []string{"C1"}, kc.Filter{IndexObject: "9"}, false)
	require.NoError(t, err)
	require.Equal(t, 1, report.TidakAda)
	require.Contains(t, report.Polis[0].Galat, "INDEXOBJECT 9")
}
