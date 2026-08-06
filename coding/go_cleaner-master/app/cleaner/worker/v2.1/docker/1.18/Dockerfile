ARG HUB_DOMAIN=hub.byted.org

FROM ${HUB_DOMAIN}/compile/bytedance_go1.18_ftf_tango.1.18:5a9999480674b6ee5586d5438215e5be

COPY . /root/project

ENV GoVersionRange='go,go1.10,go1.11,go1.12,go1.13,go1.14,go1.15,go1.16,go1.17,go1.18,1.10,1.11,1.12,1.13,1.14,1.15,1.16,1.17,1.18'
ENV PATH="$PATH:/root/tools:/opt/tiger/compile_path/bin/1.18"

WORKDIR /root/project

ENTRYPOINT ["sh", "/root/project/entrypoint.sh"]