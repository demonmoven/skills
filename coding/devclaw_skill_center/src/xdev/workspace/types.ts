export interface SubRepoSpec {
  name: string;
  url: string;
}

export interface ScaffoldFlags {
  agents_md?: boolean;
  docs_tree?: boolean;
  exec_plans?: boolean;
  onboarding?: boolean;
  setup_script?: boolean;
}

export interface SceneSpec {
  scene: string;
  main_repo: {
    url: string;
    dir: string;
  };
  sub_repos_dir: string;
  sub_repos: SubRepoSpec[];
  scaffold?: ScaffoldFlags;
}
