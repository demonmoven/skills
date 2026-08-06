include "srv3.thrift"

namespace go cleaner.service.app1

struct Srv2Req1 {
    1: string name,
}

struct Srv2Resp1 {
    1: string data,
}

struct Srv2Req2 {
    1: string name,
}
struct Srv2Resp2 {
    1: string data,
}

struct Srv2Req3 {
    1: string name,
}
struct Srv2Resp3 {
    1: string data,
}

service Srv2 extends srv3.Srv3 {
    Srv2Resp1 Srv2Func1(1: Srv2Req1 req)
    Srv2Resp2 Srv2Func2(1: Srv2Req2 req)
    Srv2Resp3 Srv2Func3(1: Srv2Req3 req)
}