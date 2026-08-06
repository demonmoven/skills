#! /usr/bin/env bash

# 错误码
# 110 参数错误
# 111 compiler version不匹配


set -e

while (("$#")); do
  case "$1" in
  --repo)
    repo_name=${2}
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
  --annotate_retained_code)
      annotate_retained_code=${2}
      shift 2
      ;;
  --assigned_base_commit)
      assigned_base_commit=${2}
      shift 2
      ;;
  --main-dir)
      main_dir=${2}
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
echo "workspace: ${WORKSPACE}"

rootDir="$(pwd)"

if [ "X${repoDir}" = "X" ]; then # support specify local repo
  repoDir="${WORKSPACE}/$(basename "${repo_name}")"
fi

if [ "X${repo_name}" != "X" ]; then 
  rm -rf "${WORKSPACE}"
  mkdir -p "${WORKSPACE}"
  git clone --single-branch -b "${branch}" "git@code.byted.org:${repo_name}.git" "${repoDir}"
fi

appDir="${repoDir}"
if [ -n "${subdir}" ]; then
    appDir="${appDir}/${subdir}"
fi
echo "workspace: ${WORKSPACE}, repo: ${repoDir}, app: ${appDir}"
cd "${appDir}"
pwd

# 存在指定的base commit
if [ -n "$assigned_base_commit" ]; then
    echo "assigned_base_commit: ${assigned_base_commit}"
    git reset --hard "${assigned_base_commit}"
fi

cd "${appDir}"
pwd

total_line=$(find . -type f -name "*.cxx" -o -name "*.cpp" -o -name "*.cc" -o -name "*.h" -o -name "*.c" |grep -v _gen/|grep -v '^$'|xargs cat|wc -l)
echo "_total_lines: ${total_line}"

modified=0

# 删除不可达代码
cd "${WORKSPACE}"
cmdStr="../sopt_cxx_cleaner/scripts/run_cleaner_in_docker.py --workdir ${WORKSPACE} --repo ${repo_name} ${cleaner_flag}"
echo "${cmdStr}"
eval "${cmdStr}"

cd "${appDir}"
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