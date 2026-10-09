package main

import (
	"context"
	"database/sql"
	"fmt"
)

func dumpRow(ctx context.Context, conn *sql.DB, table, where string, arg any) {
	rows, err := conn.QueryContext(ctx, "SELECT * FROM "+table+" WHERE "+where, arg)
	if err != nil {
		fmt.Println(table, "err:", err)
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	n := 0
	for rows.Next() {
		n++
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Println("scan:", err)
			return
		}
		fmt.Println("==", table, "row", n)
		for i, c := range cols {
			if vals[i] != nil && fmt.Sprint(vals[i]) != "" {
				s := fmt.Sprint(vals[i])
				if len(s) > 120 {
					s = s[:120] + "…"
				}
				fmt.Printf("  %s = %s\n", c, s)
			}
		}
	}
	fmt.Println(table, "rows:", n)
}
