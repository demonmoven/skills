package tqs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"code.byted.org/dp/gotqs"
	"code.byted.org/dp/gotqs/client"
	"code.byted.org/gdp/log"
	"code.byted.org/gopkg/logs"
)

var (
	DefaultCNCli = MustNew(
		"sDOd013Tmwc0ImRpB7WJ70G1LYsT0xkf6yEMpB0lYtNSC8s2",
		"odXnWGX6eJy7Pj2SpYr7AMgnXC8iUEpHeu02XK8V8Uav6Bhp",
		"xiaoxing.sn",
		2*time.Hour,
		"default",
		"hdfs://haruna/home/byte_tiktok_serverarch_gdp/warehouse",
	)
)

type Client struct {
	APPID    string
	APPKey   string
	UserName string
	TimeOut  time.Duration
	Cluster  string

	HDFSPathPrefix string

	cli *client.TqsClient
}

func MustNew(appID, appKey, userName string, timeout time.Duration, cluster, hdfsPathFile string) *Client {
	c := &Client{
		APPID:          appID,
		APPKey:         appKey,
		UserName:       userName,
		TimeOut:        timeout,
		Cluster:        cluster,
		HDFSPathPrefix: hdfsPathFile,
		cli:            nil,
	}

	cli, err := gotqs.MakeTqsClient(context.TODO(), appID, appKey, userName, timeout, cluster)
	if err != nil {
		panic(err)
	}
	c.cli = cli

	return c
}

func (c *Client) Query(ctx context.Context, sql string, param map[string]interface{}) ([][]string, error) {
	logs.CtxInfo(ctx, "tqs query start", log.KVPair("sql", sql), log.KVPair("param", param))
	status, jobPreview, err := gotqs.SyncQuery(ctx, c.cli, sql, param)
	if err != nil {
		logs.CtxError(ctx, "tqs job err", log.KVPair("err", err))
		return nil, err
	}
	if status == "Completed" {
		fullResult, fullResultErr := c.cli.FetchQueryResultsByUrl(ctx, jobPreview.Url)
		if fullResultErr != nil {
			return nil, err
		}
		logs.CtxInfo(ctx, "full result rows: %v", len(fullResult))
		return fullResult, nil
	} else {
		jobPreviewStr, _ := json.Marshal(jobPreview)
		logs.CtxError(ctx, "tqs job err, status: %s, err: %s", status, jobPreviewStr)
		return nil, fmt.Errorf("tqs job err: %s", jobPreviewStr)
	}
}
