package inboxadminhttp_test

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/repo/memory"
)

// readCSV membuang BOM lalu mengurai berkas CSV jawaban.
func readCSV(t *testing.T, body string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\xEF\xBB\xBF"))).ReadAll()
	require.NoError(t, err)
	return records
}

func TestExportsUseLegacyHeadersAndFileNames(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	cases := []struct {
		path, file string
		header     []string
	}{
		{"/api/inbox-admin/ekspor/lod", "Export Data LOD Reminder.csv", []string{
			"NOKLAIM", "No Polis", "Nama Tertanggung", "Nama Bisnis", "Sumbis",
			"Nama Cabang", "DOL", "Tanggal Report", "User Regist",
		}},
		{"/api/inbox-admin/ekspor/hasil-auto-claim", "Laporan Hasil Auto Claim.csv", []string{
			"Inisial", "No Polis", "No Klaim", "No Aksep", "Nilai Klaim", "No Objek", "Keterangan",
		}},
		{"/api/inbox-admin/ekspor/klaim-gagal", "Laporan Hasil Auto Claim Gagal.csv", []string{
			"NOPOLIS", "CLIENTID", "EDMNO", "EDMTYPE", "ACCUMCODE", "REGISTERID", "STATUSBUSINESS",
		}},
	}
	for _, c := range cases {
		recorder := get(t, server, c.path)
		require.Equalf(t, http.StatusOK, recorder.Code, "%s: %s", c.path, recorder.Body.String())
		require.Contains(t, recorder.Header().Get("Content-Type"), "text/csv")
		require.Contains(t, recorder.Header().Get("Content-Disposition"), c.file)
		require.Equal(t, c.header, readCSV(t, recorder.Body.String())[0], c.path)
	}
}

func TestExportLODListsBranchClaimRows(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})
	store := memory.NewSampleStore()
	tab, _ := inboxadmin.FindTab(inboxadmin.TabBranchClaim)
	expected, err := store.List(t.Context(), inboxadmin.Query{Tab: tab, Business: inboxadmin.BusinessAll})
	require.NoError(t, err)

	records := readCSV(t, get(t, server, "/api/inbox-admin/ekspor/lod").Body.String())
	require.Len(t, records, len(expected)+1, "satu baris judul ditambah seluruh baris Branch Claim")
}

func TestExportsRequirePortalAndCaller(t *testing.T) {
	server := buildServer(t, serverOptions{})
	recorder := get(t, server, "/api/inbox-admin/ekspor/hasil-auto-claim")
	require.Equal(t, http.StatusConflict, recorder.Code, "tanpa identitas pemanggil")
}

func TestCountsListsEveryTabWithItsRowCount(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})
	recorder := get(t, server, "/api/inbox-admin/jumlah")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	counts := decode(t, recorder)["jumlah"].([]any)
	require.Len(t, counts, len(inboxadmin.Tabs()))

	store := memory.NewSampleStore()
	for _, raw := range counts {
		entry := raw.(map[string]any)
		tab, ok := inboxadmin.FindTab(entry["kode"].(string))
		require.True(t, ok)
		query, err := inboxadmin.NewQuery(inboxadmin.QueryInput{Tab: tab.Code}, inboxadmin.Caller{Login: memory.SampleOwner})
		require.NoError(t, err)
		rows, err := store.List(t.Context(), query)
		require.NoError(t, err)
		require.Equal(t, float64(len(rows)), entry["jumlah"], "tab %s: jumlah harus sama dengan isi tabnya", tab.Name)
		require.Equal(t, false, entry["gagal"])
	}
	require.Equal(t, "Outstanding", counts[1].(map[string]any)["nama"])
}

func TestViewerDescribesScope(t *testing.T) {
	store := memory.NewSampleStore()
	store.SetGroups(memory.SampleOwner, "PncManagerAdmin")
	server := buildServer(t, serverOptions{
		getCaller: knownCaller(memory.SampleOwner),
		selector:  func(string) (inboxadmin.Repo, error) { return store, nil },
	})

	body := decode(t, get(t, server, "/api/inbox-admin/batas"))
	require.Equal(t, true, body["manajer"])
	regions := body["kanwil"].([]any)
	require.Len(t, regions, len(memory.SampleRegions))
	require.Equal(t, "Kanwil 1", regions[0].(map[string]any)["label"])
}
