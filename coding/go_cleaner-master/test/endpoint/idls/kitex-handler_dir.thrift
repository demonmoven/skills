namespace go endpoint.kitex.handler_dir

struct MyReq{
    1:required string input
}

struct ToCleanReq{
    1:required string input
}

struct ToCleanResp{
    1:required string output
}

service MyService{
    string Hello(1:required MyReq req)
    ToCleanResp ToClean1(1:required ToCleanReq req)
    ToCleanResp ToClean2(1:required ToCleanReq req)
}