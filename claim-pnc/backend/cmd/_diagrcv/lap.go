package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	lapsql "claim-pnc/internal/inboxlaporanklaim/repo/sqlstore"
)

type clk struct{}

func (clk) Now() time.Time { return time.Now() }

func lap(ctx context.Context, conn *sql.DB, id string) {
	r := lapsql.NewRepo(conn, clk{})
	for _, x := range []string{id, "RCVN.26.86"} {
		rep, err := r.Get(ctx, x)
		fmt.Printf("LAP Get %s err=%v\n  %+v\n", x, err, rep)
	}
}
