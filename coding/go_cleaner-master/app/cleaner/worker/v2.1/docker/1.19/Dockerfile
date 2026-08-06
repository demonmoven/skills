ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/bytedance_go1.19_ftf_tango.1.19:83135681ed2f5aa21a99eef8e337ae47

COPY . /root/project

ENV GoVersionRange='go,go1.19,1.19'
ENV PATH="$PATH:/root/tools:/opt/tiger/compile_path/bin/1.19"

WORKDIR /root/project

ENTRYPOINT ["sh", "/root/project/entrypoint.sh"]