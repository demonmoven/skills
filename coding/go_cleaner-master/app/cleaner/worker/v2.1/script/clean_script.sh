#! /usr/bin/env bash

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
  --no-commit)
      no_commit=1
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

# Main function to extract Go version from param、scm and go.mod
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
    echo "The go version $go_version:(${go_version_main}) is within the GoVersionRange($GoVersionRange) variable of this worker."
else
    echo "The go version $go_version:(${go_version_main}) is not within the GoVersionRange($GoVersionRange) variable of this worker. This is because the backend is currently under heavy computational pressure and there are no suitable workers to handle your task. Your task has entered the queue and you may need to wait for dozens of minutes."
    exit 111
fi

# 存在指定的base commit
if [ -n "$assigned_base_commit" ]; then
    echo "assigned_base_commit: ${assigned_base_commit}"
    git reset --hard "${assigned_base_commit}"
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

scopeFile="${rootDir}/scope.txt"
rm -rf "${scopeFile}"
if [ "X${repoAppMains}" != "X" ]; then
    cd "${repoDir}"
    cleaner scope --output="${scopeFile}" --main-dir="${main_dir}" --repo-app-mains="${repoAppMains}"
    cd "${rootDir}"
fi

IFS=',' read -ra downstream_repo_array <<< "${downstream_repos}"
for downstream_repo in "${downstream_repo_array[@]}"
do
  if [ -z "${downstream_repo}" ]; then
    continue
  else
    (
      cd "${rootDir}" &&
      rm -rf "${rootDir}/$(basename "${downstream_repo}")" &&
      git clone --single-branch -b master "git@code.byted.org:${downstream_repo}.git" &&
      cd "$(basename "${downstream_repo}")" &&
      ( cleaner search --target="${go_module}" --out="${importBy}" || cleaner search --target="${go_module}" --out="${importBy}" --method=ast ) &&
      echo "_downstream_repo_checked:${downstream_repo}"
    ) || true
  fi
done

cd "${appDir}"
pwd

if [ -n "$pre_script" ]; then
  echo "run pre_script:${pre_script}"
  eval "${pre_script}"
elif [ -e "pre_code_clean.sh" ]; then # 隐式约定：如果存在pre_code_clean.sh脚本，则执行该脚本
  echo "run pre_code_clean.sh"
  bash pre_code_clean.sh
fi

total_line=$(find . -type f -name "*.go" |grep -v _gen/|grep -v '^$'|xargs cat|wc -l)
echo "_total_lines: ${total_line}"

modified=0

# 注释标注代码
cmdStr="cleaner comment_retained --annotate_retained_code='${annotate_retained_code}'"
echo "${cmdStr}"
eval "${cmdStr}"
if [[ $(git status --porcelain) ]]; then
  modified=1
fi

# 删除不可达代码
cmdStr="cleaner ${cleaner_flag} --extra_import=${importBy} --scope_file=${scopeFile} --main-dir=${main_dir}"
echo "${cmdStr}"
eval "${cmdStr}"

if [[ $(git status --porcelain) ]]; then
  # 统计结果
  git diff --numstat | awk '{ add += $1; subs += $2; loc += $1 - $2 } END { printf "_removed_lines: %s\n", subs }'

  if [ "X${no_commit}" == "X" ]; then
    git add .
    git commit -m "chore: clean unused code ${meego_id}"
  fi

  modified=1
else
  echo "no changes to commit, delete static code stage"
fi



if [ $modified -ne 0 ]; then
  if [ -n "${commit_branch}" ]
  then
    git push -f origin "${branch}:${commit_branch}"
  fi
fi

cd "${rootDir}"
if [ "X${repo_name}" != "X" ]; then
  rm -rf "${repoDir}" # 清理
fi