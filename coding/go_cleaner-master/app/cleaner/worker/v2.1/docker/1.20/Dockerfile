ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/bytedance_go1.20_ftf_tango.1.20:1.0.0.30

COPY . /root/project

ENV GoVersionRange='go,go1.20,1.20'
ENV PATH="$PATH:/root/tools:/opt/tiger/compile_path/bin/1.20"

WORKDIR /root/project

ENTRYPOINT ["sh", "/root/project/entrypoint.sh"]