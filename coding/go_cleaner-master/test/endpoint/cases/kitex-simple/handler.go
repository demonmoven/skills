package main

import (
	"context"

	simple "code.byted.org/webcast/endpoint_kitex_simple/kitex_gen/endpoint/kitex/simple"
)

type MyServiceImpl struct{}

func (s *MyServiceImpl) Hello(ctx context.Context, req *simple.MyReq) (resp string, err error) {
	return "Hello " + req.GetInput(), nil
}

func (s *MyServiceImpl) ToClean1(ctx context.Context, req *simple.ToCleanReq) (resp *simple.ToCleanResp, err error) {
	resp = simple.NewToCleanResp()
	resp.Output = req.GetInput()
	return
}

func (s *MyServiceImpl) ToClean2(ctx context.Context, req *simple.ToCleanReq) (resp *simple.ToCleanResp, err error) {
	resp = simple.NewToCleanResp()
	resp.Output = req.GetInput()
	return
}
