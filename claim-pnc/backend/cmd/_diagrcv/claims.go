package main

import (
	"context"
	"database/sql"
	"fmt"
)

func claims(ctx context.Context, conn *sql.DB) {
	rows, err := conn.QueryContext(ctx, `SELECT CLAIMID, NOPOLIS, RCVID, GROUPPANEL, PICTEKNIK, QQNAME FROM POOLDATA.T_CLAIM_PNC
	  WHERE CLAIMID IN ('PNCN.26.40','PNCN.26.41','PNCN.26.43','PNCN.26.57')`)
	if err != nil { fmt.Println("claims err:", err); return }
	defer rows.Close()
	for rows.Next() {
		var a, b, c, d, e, f sql.NullString
		if err := rows.Scan(&a, &b, &c, &d, &e, &f); err != nil { fmt.Println(err) }
		fmt.Println("CLAIM", a.String, "pol=", b.String, "rcv=", c.String, "gp=", d.String, e.String, f.String)
	}
}
