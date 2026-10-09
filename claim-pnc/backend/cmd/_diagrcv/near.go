package main

import (
	"context"
	"database/sql"
	"fmt"
)

func near(ctx context.Context, conn *sql.DB) {
	rows, err := conn.QueryContext(ctx, `SELECT CLAIMID, NOPOLIS, NOKLAIM, TRANSFERASM, USERINPUT, TANGGALINPUTDOKUMEN
	  FROM POOLDATA.T_CLAIM_RECIVEDCLAIM WHERE CLAIMID LIKE 'RCVN.26.%' AND TANGGALINPUTDOKUMEN > SYSDATE - 2 ORDER BY TANGGALINPUTDOKUMEN`)
	if err != nil { fmt.Println(err); return }
	defer rows.Close()
	for rows.Next() {
		var a, b, c, d, e, f sql.NullString
		rows.Scan(&a, &b, &c, &d, &e, &f)
		fmt.Printf("%-12s pol=%-22s klaim=%-14s trf=%-10.10s user=%-22s %s\n", a.String, b.String, c.String, d.String, e.String, f.String)
	}
}
