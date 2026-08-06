#!/bin/bash

if [ "X$3" == "X" ]; then
    echo "usage: $0 <user_name> <go_version> <branch>"
    exit 1
fi

user_name=$1
branch=$3
go_version=$2
if [ "$go_version" == "all" ]; then
    GO_VERSION="1.18 1.19 1.20 1.21 1.22"
else
    for valid_go_version in 1.18 1.19 1.20 1.21 1.22; do
        if [ "X$2" == "X$valid_go_version" ]; then
            GO_VERSION=$2
            break
        fi
    done
fi
if [ "X$GO_VERSION" == "X" ]; then
    echo "invalid go_version: $2"
    exit 1
fi

set -x -e

cleaner_repo_dir=$HOME/$user_name/go_cleaner
if [ ! -d "$cleaner_repo_dir" ]; then
    echo "idl_cleaner not found, cloning to $cleaner_repo_dir..."
    mkdir -p "$cleaner_repo_dir"
    git clone git@code.byted.org:analyzers/go_cleaner.git "$cleaner_repo_dir"
fi


tmp_branch=tmp_$(openssl rand -hex 4)
cd "$cleaner_repo_dir"
git checkout -b "$tmp_branch"
git branch -D $branch > /dev/null 2>&1 || true
git fetch origin $branch:$branch
git checkout $branch
git branch -D $tmp_branch
BUILD_BRANCH=$(git rev-parse --abbrev-ref HEAD)
BUILD_COMMIT=$(git rev-parse HEAD)

echo "cleaner branch $BUILD_BRANCH commit $BUILD_COMMIT"

docker_list=$(docker ps -aq --filter "name=worker_v2.1_${user_name}_go*")
if [ -n "${docker_list}" ];then
    docker rm -f ${docker_list}
    sleep 30
fi

rm -rf ~/_tmp_tools/${user_name} || true
mkdir -p ~/_tmp_tools/${user_name} || true
mkdir -p ~/.tools/${user_name} || true
curl https://tosv.byted.org/obj/webcast-room-pack-cores/idl_cleaner/stable.latest.tar.gz -o ~/_tmp_tools/${user_name}/idl_cleaner.tar.gz
cd ~/_tmp_tools/${user_name}
tar -zxvf idl_cleaner.tar.gz
mv idl_cleaner ~/.tools/${user_name}/
cd -
rm -rf ~/_tmp_tools/${user_name} || true

worker_name=worker_v2.1_${user_name}
for go_version in $GO_VERSION; do
    my_docker_name=${worker_name}_go${go_version}
    my_image_name=${my_docker_name}_image
    tmp_docker=$HOME/$user_name/_tmp_docker/
    rm -rf $tmp_docker || true
    mkdir -p $tmp_docker/deps/
    if [ "X$GOROOT_DIR" == "X" ]; then
        GOROOT_DIR=/data00/go
    fi
    cp -r /home/tiger/deps/git $tmp_docker/deps/git || true
    IDC=lf
    cd $cleaner_repo_dir/app/cleaner/worker/v2.1
    FOR_CLEANER_TEST=1
    if [ "X$4" == "X-b" ]; then
        FOR_CLEANER_TEST=0 # run background
    fi
   
    ls . | grep -v docker | grep -v output | xargs -I {} cp -r ./{} $tmp_docker
    ls docker/$go_version | xargs -I {} cp -r docker/$go_version/{} $tmp_docker
    echo "${BUILD_BRANCH}@${BUILD_COMMIT}" > $tmp_docker/version.txt
    cd $tmp_docker
    docker build -t ${my_image_name} .
    # docker rm -f $my_docker_name || true
    docker run --network host -d \
            -e "RUNTIME_IDC_NAME=$IDC" -e "BYTED_HOST_IPV6=$BYTED_HOST_IPV6" \
            -e "FOR_CLEANER_TEST=$FOR_CLEANER_TEST" \
            -e "WORKER_NAME=$worker_name" \
            -e "GOBIN=/go/bin/$go_version" \
            -e "BUILD_BRANCH=$BUILD_BRANCH" \
            -e "BUILD_COMMIT=$BUILD_COMMIT" \
            -e "CLEANER_REVISION=$BUILD_BRANCH" \
            -e "WORKER_ENV=$WORKER_ENV" \
            -e "WKIP=$WKIP" \
            -v $HOME/gitconfig/tiktok/.gitconfig:/root/.gitconfig \
            -v $HOME/gitconfig/git_ssh:/root/.ssh \
            -v $GOROOT_DIR:/go \
            -v $HOME/.tools/${user_name}:/root/tools \
            --name $my_docker_name $my_image_name
    # docker exec $my_docker_name bash output/script/upgrade_bin.sh $branch
    # docker exec $my_docker_name bash -c "cp /root/project/output/script/upgrade_bin.sh /root/ && chmod +x /root/upgrade_bin.sh"
    if [ "X$4" == "X-t" ]; then
        docker exec -it $my_docker_name bash
        exit 0
    fi
    if [ "X$5" == "X-t" ]; then
        docker exec -it $my_docker_name bash
        exit 0
    fi
done