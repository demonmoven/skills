namespace go cleaner.service.app1

struct Req {
    1: i32 a
}
struct Rsp {
    1: i32 b
}

service Srv3 {
    Rsp Srv3Fn1(1: Req req)
    Rsp Srv3Fn2(1: Req req)
    Rsp Srv3Fn3(1: Req req)
}