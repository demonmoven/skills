ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/scm_llvm16_blade_cleaner:29bba45b4f6aebdb863070edcae39c15

COPY . /root/project

RUN go env -w GONOPROXY="*.byted.org,*.everphoto.cn,git.smartisan.com" && \
    go env -w GOPRIVATE="*.byted.org,*.everphoto.cn,git.smartisan.com" && \
    go env -w GOPROXY="https://go-mod-proxy.byted.org|https://goproxy.cn|https://proxy.golang.org|direct" && \
    go env -w GONOSUMDB="*.byted.org,*.everphoto.cn,git.smartisan.com"

ENV GoVersionRange='c++'

WORKDIR /root/project
ENTRYPOINT ["sh", "/root/project/cxx/entrypoint.sh"]