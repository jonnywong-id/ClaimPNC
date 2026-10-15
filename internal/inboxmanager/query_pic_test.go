package inboxmanager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewQueryMembetulkanPenyaringPIC(t *testing.T) {
	caller := Caller{Login: "MGR"}

	cases := map[string]string{
		"":            "",
		"   ":         "",
		"ALL":         "",
		" all ":       "",
		" YOSUA ":     "YOSUA",
		"a@gmail.com": "a@gmail.com",
	}
	for input, want := range cases {
		q, err := NewQuery(QueryInput{Tab: TabOutstanding, PIC: input}, caller)
		require.NoError(t, err)
		require.Equal(t, want, q.PIC, "masukan %q", input)
	}
}

func TestHanyaGridPICOutstandingYangMenjadiTautan(t *testing.T) {
	for _, tab := range Tabs() {
		for _, panel := range tab.Panels {
			if tab.Code == TabOutstanding && panel.Key == "pic" {
				require.Equal(t, FilterPIC, panel.RowFilter)
				continue
			}
			require.Empty(t, panel.RowFilter, "tab %s panel %s", tab.Code, panel.Key)
		}
	}
}
