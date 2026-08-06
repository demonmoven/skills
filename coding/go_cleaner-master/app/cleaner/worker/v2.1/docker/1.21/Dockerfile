ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/bytedance_go1.21_ftf_tango.1.21:1.0.0.16

COPY . /root/project

ENV GoVersionRange='go,go1.21,1.21'
ENV PATH="$PATH:/root/tools:/opt/tiger/compile_path/bin/1.21"

WORKDIR /root/project

ENTRYPOINT ["sh", "/root/project/entrypoint.sh"]