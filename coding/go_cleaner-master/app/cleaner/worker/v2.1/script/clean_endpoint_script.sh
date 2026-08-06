#! /usr/bin/env bash

# 错误码
# 110 参数错误
# 111 go version不匹配
# 112 endpoint没有可删除代码


set -e

while (("$#")); do
  case "$1" in
  --repo)
    repo_name=${2}
    shift 2
    ;;
  --set_go_version)
    set_go_version=${2}
    shift 2
    ;;
  --scm_go_version)
    scm_go_version=${2}
    shift 2
    ;;
  --branch)
    branch=${2:-master}
    shift 2
    ;;
  --subdir)
    subdir=${2:-.}
    shift 2
    ;;
  --commit_branch)
    commit_branch=${2}
    shift 2
    ;;
  --meego_id)
      meego_id=${2}
      shift 2
      ;;
  --cleaner_flag)
    cleaner_flag=${2}
    shift 2
    ;;
  --endpoint)
      endpoint=${2}
      shift 2
      ;;
  --handler_path)
      handler_path=${2}
      shift 2
      ;;
  --downstream_repos)
      downstream_repos=${2}
      shift 2
      ;;
  --annotate_retained_code)
      annotate_retained_code=${2}
      shift 2
      ;;
  --assigned_base_commit)
      assigned_base_commit=${2}
      shift 2
      ;;
  --pre-script)
      pre_script=${2}
      shift 2
      ;;
  --main-dir)
      main_dir=${2}
      shift 2
      ;;
  --repo-dir)
      repoDir=${2}
      shift 2
      ;;
  --repo-app-mains)
      repoAppMains=${2}
      shift 2
      ;;
  --) # 结束解析
    shift
    break
    ;;
  -* | --*=)
    echo "Error: Unsupported flag $1" >&2
    exit 110
    ;;
  *)
    echo "Error: Positional parameters should be specified before named parameters." >&2
    exit 110
    ;;
  esac
done

echo "repo_name: ${repo_name}, branch: ${branch}, subdir: ${subdir}, commit_branch: ${commit_branch}, meego_id: ${meego_id} cleaner_flag: ${cleaner_flag}"

go version $(command -v cleaner) # print the cleaner build go version

rootDir="$(pwd)"
if [ "X${repoDir}" = "X" ]; then # support specify local repo
  repoDir="${rootDir}/$(basename "${repo_name}")"
fi
if [ "X${repo_name}" != "X" ]; then 
  rm -rf "${repoDir}"
  git clone --single-branch -b "${branch}" "git@code.byted.org:${repo_name}.git" "${repoDir}"
fi
appDir="${repoDir}"
if [ -n "${subdir}" ]; then
    appDir="${appDir}/${subdir}"
fi
echo "root: ${rootDir}, repo: ${repoDir}, app: ${appDir}"
cd "${appDir}"
pwd

# 解析Go版本
# Find the directory containing go.mod starting from the current directory and going up
find_go_mod_dir() {
    local dir=$PWD
    while [ "$dir" != "" ] && [ ! -e "$dir/go.mod" ]; do
        dir=${dir%/*}
    done
    echo "$dir"
}

# Main function to extract Go version from go.mod
extract_go_version() {
    if [ -n "$set_go_version" ]; then
        echo $set_go_version
        return 0
    fi

    if [ -n "$scm_go_version" ]; then
        echo $scm_go_version
        return 0
    fi

    local go_mod_dir=$(find_go_mod_dir)

    if [ -z "$go_mod_dir" ]; then
        echo "Error: go.mod file not found in the current directory tree." >&2
        return 1
    fi

    local go_mod_path="$go_mod_dir/go.mod"
    local go_version_line=$(grep "^go " "$go_mod_path")

    if [ -z "$go_version_line" ]; then
        echo "Error: No Go version specified in go.mod file." >&2
        return 1
    fi

    # Extract and print the Go version
    local go_version=$(echo "$go_version_line" | awk '{print $2}')
    echo $go_version
}

# Main function to extract Go module from go.mod
extract_go_module() {
    local go_mod_dir=$(find_go_mod_dir)

    if [ -z "$go_mod_dir" ]; then
        echo "Error: go.mod file not found in the current directory tree." >&2
        return 1
    fi

    local go_mod_path="$go_mod_dir/go.mod"
    local go_module_line=$(grep "^module " "$go_mod_path")

    if [ -z "$go_module_line" ]; then
        echo "Error: No Go version specified in go.mod file." >&2
        return 1
    fi

    # Extract and print the Go version
    local go_module=$(echo "$go_module_line" | awk '{print $2}')
    echo $go_module
}

# 判断go version是否匹配
go_version=$(extract_go_version)
echo "_go_version:${go_version}"
go_version_main=$(echo "${go_version}" | awk -F. '{print $1"."$2}')
if [[ "$GoVersionRange" == *"${go_version_main}"* ]]; then
    echo "The go version $go_version is within the GoVersionRange variable of this worker."
else
    echo "The go version $go_version is not within the GoVersionRange variable of this worker. This is because the backend is currently under heavy computational pressure and there are no suitable workers to handle your task. Your task has entered the queue and you may need to wait for dozens of minutes."
    exit 111
fi

go_module=$(extract_go_module)
echo "_go_module:${go_module}"

base_commit_id=$(git rev-parse HEAD)
echo "_base_commit_id:${base_commit_id}"
# go mod tidy

# 如果存在其它仓库依赖这个仓库，把其它仓库的依赖项记录到文件中，并后续传给cleaner工具
cd "${rootDir}"
importBy="${rootDir}/importBy.txt"
rm -rf "${importBy}"

IFS=',' read -ra downstream_repo_array <<< "${downstream_repos}"
for downstream_repo in "${downstream_repo_array[@]}"
do
  if [ -z "${downstream_repo}" ]; then
    continue
  else
    (
      cd "${rootDir}" &&
      rm -rf "${rootDir}/$(basename "${downstream_repo}")" &&
      git clone --single-branch -b "${branch}" "git@code.byted.org:${downstream_repo}.git" &&
      cd "$(basename "${downstream_repo}")" &&
      ( cleaner search --target="${go_module}" --out="${importBy}" || cleaner search --target="${go_module}" --out="${importBy}" --method=ast ) &&
      echo "_downstream_repo_checked:${downstream_repo}"
    ) || true
  fi
done

if [ -n "$pre_script" ]; then
  echo "run pre_script:${pre_script}"
  eval "${pre_script}"
elif [ -e "pre_code_clean.sh" ]; then # 隐式约定：如果存在pre_code_clean.sh脚本，则执行该脚本
  echo "run pre_code_clean.sh"
  bash pre_code_clean.sh
fi

cd "${appDir}"
pwd

total_line=$(find . -type f -name "*.go" |grep -v _gen/|grep -v '^$'|xargs cat|wc -l)
echo "_total_lines: ${total_line}"

temp_branch="${commit_branch}_temp_for_cherry"
git checkout -b ${temp_branch}

# 1.提前执行一次删除不可达代码
cmdStr="cleaner ${cleaner_flag} --extra_import=${importBy}"
echo "eval pre clean unused code: ${cmdStr}"
eval "${cmdStr}"
if [[ $(git status --porcelain) ]]; then
  # pre_unused_paths=$(git status --porcelain|awk -F ' ' '{print $2}' | tr '\n' ',' | sed 's/,$//')
  git add .
  git commit -m "chore: pre clean unused code ${meego_id}"
  commit_output_pre_clean=$(git log -1 --pretty=format:%H)
  commit_hash_pre_clean=$(echo "$commit_output_pre_clean" | grep -o -E '[a-f0-9]{40}')
fi

# 2.删除目标移除的入口代码
if [ "$endpoint" == "__UseMarkUnusedScript" ]; then
  cmdStrForCleanEndpoint="bash ${rootDir}/../script/custom_script/recycle.sh ./"
else
  cmdStrForCleanEndpoint="cleaner endpoint --ep=\"${endpoint}\" --handler_path=\"${handler_path}\" --main-dir=\"${main_dir}\""
fi
echo "eval clean endpoint: ${cmdStrForCleanEndpoint}"
eval "${cmdStrForCleanEndpoint}"
if [[ $(git status --porcelain) ]]; then
  git add .
  git commit -m "chore: clean unused endpoint ${meego_id}"
  commit_output_del_endpoint=$(git log -1 --pretty=format:%H)
  commit_hash_del_endpoint=$(echo "$commit_output_del_endpoint" | grep -o -E '[a-f0-9]{40}')
else
  echo "no changes after clean endpoint"
  git checkout ${branch}
  git push -f origin "${branch}:${commit_branch}"
  echo "_removed_lines: 0"
  exit 0
fi

# 3.删除增量不可达代码
echo "incremental eval clean unused code ${cmdStr}"
eval "${cmdStr}"
if [[ $(git status --porcelain) ]]; then
  git add .
  git commit -m "chore: incremental clean unused endpoint ${meego_id}"
  commit_output_incremental_clean=$(git log -1 --pretty=format:%H)
  commit_hash_incremental_clean=$(echo "$commit_output_incremental_clean" | grep -o -E '[a-f0-9]{40}')
fi

echo "commit hash pre_clean=$commit_hash_pre_clean del_endpoint=$commit_hash_del_endpoint incremental_clean=$commit_hash_incremental_clean"

git checkout ${branch}

initial_commit=$(git log -1 --pretty=format:%H)
initial_commit_hash=$(echo "$initial_commit" | grep -o -E '[a-f0-9]{40}')

# 应用删除代码入口提交
if [ -n "$commit_hash_del_endpoint" ]; then
  echo "git cherry-pick ${commit_hash_del_endpoint}"
  git cherry-pick ${commit_hash_del_endpoint}
fi

# 定义一个函数来处理 git cherry-pick 的失败情况
handle_cherry_pick_failure() {
    # 检查是否存在合并冲突
    if git status --porcelain | grep -q '^UU\|^UD'; then
        # 获取所有冲突文件的列表
        local conflicted_files=$(git status --porcelain | grep '^UU\|^UD' | cut -d ' ' -f 2| tr '\n' ',' | sed 's/,$//')
        # 冲突文件列表
        echo "$conflicted_files"
    else
        echo "git cherry-pick failure not due to conflict，please check"
        exit 1
    fi
}

# 应用删除增量不可达代码提交
if [ -n "$commit_hash_incremental_clean" ]; then
  echo "git cherry-pick ${commit_hash_incremental_clean}"
  git cherry-pick ${commit_hash_incremental_clean} || conflicted_files=$(handle_cherry_pick_failure)

  # 处理冲突
  if [ -n "$conflicted_files" ]; then
      echo "cherry-pick incremental clean find conflicted files: $conflicted_files"
      # 退出并回滚 cherry-pick
      git cherry-pick --abort
      # 切回临时分支，使用冲突文件作为排除文件参数，重跑清理任务
      git checkout ${temp_branch}
      git reset --hard ${commit_hash_del_endpoint}
      cmdIncrCleanRetry="cleaner ${cleaner_flag} --extra_import=${importBy} --exclude='kitex_gen/,model_gen/,thrift_gen/,mock_gen/,rpc_gen,wcc/,opp_gen/,${conflicted_files}'"
      echo "incremental eval clean unused code ${cmdIncrCleanRetry}"
      eval "${cmdIncrCleanRetry}"
      if [[ $(git status --porcelain) ]]; then
        git add .
        git commit -m "chore: incremental clean unused endpoint ${meego_id}"
        commit_output_incremental_clean_retry=$(git log -1 --pretty=format:%H)
        commit_hash_incremental_clean_retry=$(echo "$commit_output_incremental_clean_retry" | grep -o -E '[a-f0-9]{40}')
        echo "commit_hash_incremental_clean_retry: $commit_hash_incremental_clean_retry"
      fi
      # 应用重新生成的增量代码清理
      if [ -n "$commit_hash_incremental_clean_retry" ]; then
        git checkout ${branch}
        echo "git cherry-pick retry ${commit_hash_incremental_clean_retry}"
        git cherry-pick ${commit_hash_incremental_clean_retry}
      else
        echo "no change by incremental_clean_retry"
        git checkout ${branch}
      fi
  fi
fi

if [[ -n "$commit_hash_del_endpoint" ]]; then
  git diff $initial_commit_hash --numstat | awk '{ add += $1; subs += $2; loc += $1 - $2 } END { printf "_removed_lines: %s\n", subs }'
  git push -f origin "${branch}:${commit_branch}"
else
  echo "no changes to commit, delete static code stage"
fi
git branch -D ${temp_branch}

cd "${rootDir}"
if [ "X${repo_name}" != "X" ]; then
  rm -rf "${repoDir}" # 清理
fi