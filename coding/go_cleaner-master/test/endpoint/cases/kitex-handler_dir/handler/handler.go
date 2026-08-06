package handler

import (
	"context"

	handler_dir "code.byted.org/webcast/endpoint_kitex_handler_dir/kitex_gen/endpoint/kitex/handler_dir"
)

type MyServiceImpl struct{}

func (s *MyServiceImpl) Hello(ctx context.Context, req *handler_dir.MyReq) (resp string, err error) {
	return "Hello " + req.GetInput(), nil
}

func (s *MyServiceImpl) ToClean1(ctx context.Context, req *handler_dir.ToCleanReq) (resp *handler_dir.ToCleanResp, err error) {
	resp = handler_dir.NewToCleanResp()
	resp.Output = req.GetInput()
	return
}

func (s *MyServiceImpl) ToClean2(ctx context.Context, req *handler_dir.ToCleanReq) (resp *handler_dir.ToCleanResp, err error) {
	resp = handler_dir.NewToCleanResp()
	resp.Output = req.GetInput()
	return
}
