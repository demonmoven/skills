ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/bytedance.golang.compile_tango_1_23_bookworm:1.0.0.7

COPY . /root/project

ENV GoVersionRange='go,go1.23,1.23'
ENV PATH="$PATH:/root/tools:/opt/tiger/compile_path/bin/1.23"

WORKDIR /root/project

ENTRYPOINT ["sh", "/root/project/entrypoint.sh"]