package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	go_ora "github.com/sijms/go-ora/v2"
)

func envY(k string) string { return strings.TrimSpace(os.Getenv(k)) }

// keysOnly mencetak STRUKTUR dokumen JSON tanpa satu pun nilainya — data nasabah tidak
// boleh tertulis ke keluaran mana pun (`D-69`).
func keysOnly(v any, prefix string, out map[string]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, vv := range t {
			keysOnly(vv, prefix+"."+k, out)
		}
	case []any:
		out[prefix+"[]"] = fmt.Sprintf("senarai %d baris", len(t))
		if len(t) > 0 {
			keysOnly(t[0], prefix+"[0]", out)
		}
	default:
		out[prefix] = fmt.Sprintf("%T", v)
	}
}

func TestProbeY(t *testing.T) {
	if os.Getenv("DDLPROBE") == "" {
		t.Skip("")
	}
	url := go_ora.BuildUrl(envY("POOLDATA_ASM_HOST"), 1521, envY("POOLDATA_ASM_SERVICE"), envY("POOLDATA_ASM_PENGGUNA"), envY("POOLDATA_ASM_SANDI"), nil)
	db, _ := sql.Open("oracle", url)
	defer db.Close()

	var raw string
	if err := db.QueryRow(`SELECT JSON_POLIS FROM POOLDATA.MST_RECOVERY_ASM_PENJAMINAN WHERE BATCH = 1`).Scan(&raw); err != nil {
		fmt.Println("ERR", err)
		return
	}
	fmt.Println("panjang:", len(raw))

	var doc any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		fmt.Println("BUKAN JSON yang dapat diurai:", err)
		fmt.Println("50 karakter pertama (struktur saja):", strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return 'N'
			}
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				return 'x'
			}
			return r
		}, raw[:50]))
		return
	}
	out := map[string]string{}
	keysOnly(doc, "", out)
	fmt.Println("--- struktur (KUNCI saja, tanpa nilai) ---")
	for k, v := range out {
		fmt.Printf("  %-60s %s\n", k, v)
	}
}
