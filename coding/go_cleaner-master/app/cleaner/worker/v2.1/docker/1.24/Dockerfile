ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/bytedance.golang.compile_tango_1_24_bookworm:1.0.0.4

COPY . /root/project

ENV GoVersionRange='go,go1.24,1.24'
ENV PATH="$PATH:/root/tools:/opt/tiger/compile_path/bin/1.24"

WORKDIR /root/project

ENTRYPOINT ["sh", "/root/project/entrypoint.sh"]