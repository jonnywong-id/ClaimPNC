package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/registrasi/repo/sqlstore"
)

func main() {
	id := os.Args[1]
	_ = config.LoadEnvFile(".env")
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("config:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	var ps []db.Parameter
	keys := []string{}
	for k := range cfg.Portal {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys { fmt.Println("cfg", k, cfg.Portal[k].Alias, cfg.Portal[k].Complete(), cfg.Portal[k].Missing())
		b := cfg.Portal[k]
		if !b.Complete() {
			continue
		}
		ps = append(ps, db.Parameter{Alias: b.Alias, Host: b.Host, Port: b.Port, Service: b.Service, User: b.User, Password: b.Password})
	}
	pool, err := db.NewPool(ctx, cfg.PrimaryPortal, ps, func(a string, e error) { fmt.Println("skip", a, e) })
	if err != nil {
		fmt.Println("pool:", err)
		return
	}
	for _, alias := range pool.Available() {
		conn, _ := pool.For(alias)
		var n int
		if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM POOLDATA.T_CLAIM_RECIVEDCLAIM WHERE CLAIMID = :1", id).Scan(&n); err != nil {
			fmt.Println(alias, "count err:", err)
			continue
		}
		fmt.Println("portal", alias, "rows:", n)
		if n == 0 {
			continue
		}
		dumpRow(ctx, conn, "POOLDATA.T_CLAIM_RECIVEDCLAIM", "CLAIMID = :1", id)
		dumpRow(ctx, conn, "POOLDATA.T_CLAIM_PNC", "CLAIMID = :1", id)
		claims(ctx, conn); return
		near(ctx, conn); for _, x := range []string{"RCVN.26.64","RCVN.26.70","RCVN.26.86"} { dumpRow(ctx, conn, "POOLDATA.T_CLAIM_RECIVEDCLAIM", "CLAIMID = :1", x) }
		snap, err := sqlstore.NewClaimReportLink(conn).Snapshot(ctx, id)
		fmt.Printf("snapshot: %+v err=%v\n", snap, err)
		if err != nil || snap.PolicyNumber == "" {
			continue
		}
		pol, err := sqlstore.NewPolicyRepo(conn).Get(ctx, snap.PolicyNumber)
		if err != nil {
			fmt.Println("policy err:", err)
			continue
		}
		fmt.Printf("policy: no=%s line=%v biz=%s cur=%s\n", pol.Number, pol.Line, pol.BusinessCode, pol.Currency)
		items, err := sqlstore.NewPolicyItems(conn).Items(ctx, pol)
		fmt.Println("items:", len(items), "err:", err)
		causes, err := sqlstore.NewAreaDirectory(conn).CauseOfLossOptions(ctx, pol.BusinessCode)
		fmt.Println("causes:", len(causes), "err:", err)
	}
}
