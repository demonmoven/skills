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
  --commit-branch)
    commit_branch=${2}
    shift 2
    ;;
  --storage-psms)
      storage_psms=${2}
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

echo "repo_name: ${repo_name}, branch: ${branch}, commit_branch: ${commit_branch}, psm-list: ${storage_psms}"

rootDir="$(pwd)"

if [ "X${repoDir}" = "X" ]; then # support specify local repo
  repoDir="${rootDir}/$(basename "${repo_name}")"
fi

if [ "X${repo_name}" != "X" ]; then 
  rm -rf "${repoDir}"
  git clone --single-branch -b "${branch}" "git@code.byted.org:${repo_name}.git" "${repoDir}"
fi
appDir="${repoDir}"
cd $appDir

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
    echo "The go version $go_version is within the GoVersionRange variable of this worker."
else
    echo "The go version $go_version is not within the GoVersionRange variable of this worker. This is because the backend is currently under heavy computational pressure and there are no suitable workers to handle your task. Your task has entered the queue and you may need to wait for dozens of minutes."
    exit 111
fi

go_module=$(extract_go_module)
echo "_go_module:${go_module}"

# 执行清理
cmdStr="cleaner storage --psm-list=${storage_psms}"
echo "${cmdStr}"
eval "${cmdStr}"

modified=0
if [[ $(git status --porcelain) ]]; then
  # 统计结果
  git diff --numstat | awk '{ add += $1; subs += $2; loc += $1 - $2 } END { printf "_removed_lines: %s\n", subs }'

  if [ "X${no_commit}" == "X" ]; then
    git add .
    git commit -m "chore: clean storage code for psm ${storage_psms}"
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