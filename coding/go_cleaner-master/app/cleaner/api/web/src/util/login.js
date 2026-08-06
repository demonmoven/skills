import {BDSSO, BDSSOType} from "@byted-sdk/bdsso";

function login() {
    if (process.env.NODE_ENV === 'development') {
        return;
    }
    let bdsso = BDSSO.config({
        type: BDSSOType.CAS,
        aid: "6287",
        redirectUrl: "https://go-unused-code-analyzer.byted.org/",
    });
    bdsso.isLogin((login) => {
        console.log(login)
        if (!login.data.is_login) {
            bdsso.login()
        }
    });

}

export {login}