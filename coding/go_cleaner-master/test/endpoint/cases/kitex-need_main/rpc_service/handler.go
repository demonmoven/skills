package main

import (
	"context"

	need_main "code.byted.org/webcast/endpoint_kitex_need_main/kitex_gen/endpoint/kitex/need_main"
)

type MyServiceImpl struct{}

func (s *MyServiceImpl) Hello(ctx context.Context, req *need_main.MyReq) (resp string, err error) {
	return "Hello " + req.GetInput(), nil
}

func (s *MyServiceImpl) ToClean1(ctx context.Context, req *need_main.ToCleanReq) (resp *need_main.ToCleanResp, err error) {
	resp = need_main.NewToCleanResp()
	resp.Output = req.GetInput()
	return
}

func (s *MyServiceImpl) ToClean2(ctx context.Context, req *need_main.ToCleanReq) (resp *need_main.ToCleanResp, err error) {
	resp = need_main.NewToCleanResp()
	resp.Output = req.GetInput()
	return
}
