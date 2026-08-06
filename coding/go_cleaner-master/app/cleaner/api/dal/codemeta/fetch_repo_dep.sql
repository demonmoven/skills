select  regexp_replace(dep_name, 'code.byted.org/', '') as lib_name,
        concat_ws(',', collect_set(repo_name)) as git_name
from    iesarch_codemeta.repo_pkgs_dependencies_latest_dwd_d
where   date='${date}'
  and     indirect=0
group by
    dep_name