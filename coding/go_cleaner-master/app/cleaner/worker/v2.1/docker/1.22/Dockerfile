ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/bytedance_go1.22_ftf_tango.1.22:1.0.0.10

COPY . /root/project

ENV GoVersionRange='go,go1.22,1.22,1.22.0'
ENV PATH="$PATH:/root/tools:/opt/tiger/compile_path/bin/1.22"

WORKDIR /root/project

ENTRYPOINT ["sh", "/root/project/entrypoint.sh"]