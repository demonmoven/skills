package codemeta

import (
	"context"
	_ "embed"
	"strings"
	"sync/atomic"
	"time"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/api/dal/tqs"
	"code.byted.org/gdp/log"
	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/metrics"
)

//go:embed fetch_repo_dep.sql
var sql string

var DependentMgr = newMetaMgr()

func init() {
	DependentMgr.mustAutoReload()
}

type meta = map[string][]string

type dependentMgr struct {
	meta atomic.Value

	cli *metrics.MetricsClientV2
}

func newMetaMgr() *dependentMgr {
	return &dependentMgr{
		meta: atomic.Value{},
		cli:  metrics.NewDefaultMetricsClientV2("tiktok.serverarch.unused_code_api", true),
	}
}

func (m *dependentMgr) reload() error {
	ctx := context.Background()
	date := time.Now().AddDate(0, 0, -2).Format("20060102")

	log.Warnf(ctx, "dependentMgr reload start")
	table, err := tqs.DefaultCNCli.Query(ctx, sql, map[string]interface{}{"date": date})

	if err != nil {
		logs.CtxError(ctx, "meta mgr reload err", log.KVPair("err", err))
		_ = m.cli.EmitCounter("repo_dependent_sync", 1, metrics.T{Name: "ok", Value: "false"})
		return err
	}

	meta := map[string][]string{}

	for _, record := range table {
		if len(record) > 1 {
			meta[record[0]] = strings.Split(record[1], ",")
		}
	}

	m.meta.Store(meta)
	_ = m.cli.EmitCounter("repo_dependent_sync", 1, metrics.T{Name: "ok", Value: "true"})
	log.Warnf(ctx, "dependentMgr reload success")
	return nil
}

func (m *dependentMgr) Get(repo string) []string {
	v, ok := m.meta.Load().(meta)
	if !ok || v == nil {
		return nil
	}

	return v[repo]
}

func (m *dependentMgr) mustAutoReload() {
	go func() {
		_ = m.reload()
	}()

	go func() {
		ticker := time.Tick(12 * time.Hour)

		for {
			select {
			case <-ticker:
				_ = m.reload()
			}
		}
	}()
}
