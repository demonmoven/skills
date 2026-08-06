package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/model"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/log"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
)

const (
	acquireJobRoute = "/asset/worker/job/acquire"
	saveJobRoute    = "/asset/worker/job/save"
	getByIDRoute    = "/asset/worker/job/_id"

	domainCN = "https://4uael4vr.fn.bytedance.net"
	domainSG = "https://5mtu1qsj.sg-fn.bytedance.net"
)

type httpCli struct{}

type AdminClient interface {
	AcquireCodeCleanTask(ctx context.Context, request *model.AcquireCodeCleanTaskRequest) (r *model.AcquireCodeCleanTaskResponse, err error)
	SaveCodeCleanTask(ctx context.Context, request *model.SaveCodeCleanTaskRequest) (r *model.SaveCodeCleanTaskResponse, err error)
	GetCodeCleanTaskByID(ctx context.Context, taskID int64) (r *model.AcquireCodeCleanTaskResponse, err error)
}
type iResp interface {
	GetBaseResp() *model.BaseResp
}

func (c *httpCli) SaveCodeCleanTask(ctx context.Context, request *model.SaveCodeCleanTaskRequest) (r *model.SaveCodeCleanTaskResponse, err error) {
	info := log.NewInfoWriter(ctx).Str("save task over https").KV("task_id", request.Task.GetID())
	defer info.Emit()
	realResp, err := processRequest[*model.SaveCodeCleanTaskResponse](ctx, request, saveJobRoute, nil, info)
	if err != nil {
		return nil, err
	}
	return realResp, nil
}
func (c *httpCli) AcquireCodeCleanTask(ctx context.Context, request *model.AcquireCodeCleanTaskRequest) (r *model.AcquireCodeCleanTaskResponse, err error) {
	info := log.NewInfoWriter(ctx).Str("acquire task over https").KV("go_version", request.GoVersion).KV("worker", request.GetWorker())
	defer info.Emit()

	realResp, err := processRequest[*model.AcquireCodeCleanTaskResponse](ctx, request, acquireJobRoute, nil, info)
	if err != nil {
		return nil, err
	}
	return realResp, nil
}

func (c *httpCli) GetCodeCleanTaskByID(ctx context.Context, taskID int64) (r *model.AcquireCodeCleanTaskResponse, err error) {
	info := log.NewInfoWriter(ctx).Str("get task by id over https").KV("task_id", taskID)
	defer info.Emit()

	realResp, err := processRequest[*model.AcquireCodeCleanTaskResponse](ctx, nil, getByIDRoute, map[string]string{
		"id": strconv.FormatInt(taskID, 10),
	}, info)
	if err != nil {
		return nil, err
	}
	return realResp, nil
}

func processRequest[T iResp](ctx context.Context, req any, route string, queries map[string]string, info log.InfoWriter) (T, error) {
	domain := domainCN
	isI18N := false
	if isI18N = os.Getenv("I18NWORKER") == "1"; isI18N {
		domain = domainSG
		info.KV("domain", "sg")
	} else {
		info.KV("domain", "cn")
	}
	if queries == nil {
		queries = make(map[string]string)
	}
	url := domain + route
	if workerEnv == "" && !isI18N {
		workerEnv = "ppe_bag_asset_manager" // 泳道发布便于日常更新，非i18n还没搞泳道
	}
	if workerEnv != "" {
		info.KV("worker_env", workerEnv)
		queries["__env"] = workerEnv
	}
	querySegs := []string{}
	for k, v := range queries {
		querySegs = append(querySegs, fmt.Sprintf("%s=%s", k, v))
	}
	if len(querySegs) > 0 {
		url += ("?" + strings.Join(querySegs, "&"))
	}
	var zeroResp T
	body, err := json.Marshal(req)
	if err != nil {
		return zeroResp, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		info.KV("result", "new_request_failed")
		return zeroResp, err
	}

	if logID := ctxvalues.LogIDDefault(ctx); logID == "" {
		logID = logid.GenLogID()
		httpReq.Header.Set("x-tt-logid", logID)
		info.KV("set-log-id", logID)
	} else {
		httpReq.Header.Set("x-tt-logid", logID)
		info.KV("ctx-log-id", logID)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		info.KV("result", "do_request_failed")
		return zeroResp, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		cnt, _ := io.ReadAll(resp.Body)
		if len(cnt) > 0 {
			info.KV("result", "status_code_not_200")
			return zeroResp, fmt.Errorf("status code not 200, status code: %d, status message: %s", resp.StatusCode, string(cnt))
		}
		info.KV("result", "status_code_not_200_no_body")
		return zeroResp, fmt.Errorf("status code not 200, status code: %d", resp.StatusCode)
	}
	realResp := reflect.New(reflect.TypeOf(zeroResp).Elem()).Interface().(T)
	err = json.NewDecoder(resp.Body).Decode(&realResp)
	if err != nil {
		info.KV("result", "decode_response_failed")
		return realResp, err
	}
	if realResp.GetBaseResp() != nil && realResp.GetBaseResp().GetExtra()["machine_env"] != "" {
		info.KV("machine_env", realResp.GetBaseResp().GetExtra()["machine_env"])
		info.KV("machine_dc", realResp.GetBaseResp().GetExtra()["machine_dc"])
		if realResp.GetBaseResp().GetExtra()["machine_ip_v4"] != "" {
			info.KV("machine_ip", realResp.GetBaseResp().GetExtra()["machine_ip_v4"])
		} else {
			info.KV("machine_ip", realResp.GetBaseResp().GetExtra()["machine_ip_v6"])
		}
	}
	return realResp, nil
}
