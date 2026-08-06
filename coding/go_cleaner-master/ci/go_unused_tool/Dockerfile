FROM hub.byted.org/codebase/ci_go_1_19

RUN apt-get update &&apt-get -y install \
        gcc \
        ssh \
        krb5-user \
        libgssapi-krb5-2 \
        libkrb5-dev \
        libmagickwand-dev \
        libsasl2-modules-gssapi-mit \
        libsnmp-dev \
        wget \
        expect \
        git \
        unzip && apt-get clean

# install bvc
RUN mkdir -p /bvc_wget/bvc/
RUN wget http://luban-source.byted.org/repository/scm/tao.modules.bvc_1.0.0.103.tar.gz -O /bvc_wget/bvc.tar.gz
RUN cd /bvc_wget/ && tar -zxvf bvc.tar.gz -C /bvc_wget/bvc && cp /bvc_wget/bvc/bin/bvc /usr/local/bin

ARG SCM_VERSION
ARG HOME=/root/

RUN bvc clone codebase/analysis/go_unused_code ./tmp --version ${SCM_VERSION} && cp ./tmp/go_unused_code /usr/local/bin/
RUN cat ./tmp/current_revision || true

WORKDIR $HOME

COPY ./ci/go_unused_tool/entry.sh /usr/local/bin/entry.sh
RUN chmod +x /usr/local/bin/entry.sh
ENTRYPOINT "bash"





