include "dir1/srv2.thrift"

namespace go cleaner.service.app2

struct Srv1Req1 {
    1: required i64 id,
}
struct Srv1Resp1 {
    1: required string data,
}

struct Srv1Req2 {
    1: required i64 id,
}
struct Srv1Resp2 {
    1: required string data,
}

struct Srv1Req3 {
    1: required i64 id,
}
struct Srv1Resp3 {
    1: required string data,
}

struct Srv1Req4 {
    1: required i64 id,
}
struct Srv1Resp4 {
    1: required string data,
}

struct Srv1Req5 {
    1: required i64 id,
}
struct Srv1Resp5 {
    1: required string data,
}

service Srv1_1 extends srv2.Srv2 {
    Srv1Resp3 GetSrv1Func3(1: Srv1Req3 req),
    // some comment
    Srv1Resp4 GetSrv1Func4(1: Srv1Req4 req), // some commet2
    /*
        some comment for GetSrv1Func5
    */
    Srv1Resp5 GetSrv1Func5(1: Srv1Req5 req),
}

service Srv1_0 extends Srv1_1 {
    Srv1Resp1 GetSrv1Func1(1: Srv1Req1 req), // some comment
    Srv1Resp2 GetSrv1Func2(1: Srv1Req2 req),
}

service Srv1_2 {
    Srv1Resp1 GetSrv1Func6(1: Srv1Req1 req),
    Srv1Resp1 GetSrv1Func7(1: Srv1Req1 req),
}


