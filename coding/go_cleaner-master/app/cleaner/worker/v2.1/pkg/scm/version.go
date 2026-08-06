package scm

func GoVersion(psm string, gitRepoName string) (version string, err error) {
	if psm == "" {
		return getCompilerVersionFromSCMByGitName(gitRepoName, extractGoVersion)
	}

	mainScmRepo, err := getTCEMainScmRepoByPSM(psm)
	if err != nil && err != ErrCompilerVersionNotFoundInImage {
		return "", err
	}

	if mainScmRepo == "" {
		return getCompilerVersionFromSCMByGitName(gitRepoName, extractGoVersion)
	}

	return getCompilerVersionFromSCMBySCMRepoName(mainScmRepo, extractGoVersion)
}

func CxxVersion(psm string, gitRepoName string) (version string, err error) {
	if psm == "" {
		return getCompilerVersionFromSCMByGitName(gitRepoName, extractCxxVersion)
	}

	mainScmRepo, err := getTCEMainScmRepoByPSM(psm)
	if err != nil && err != ErrCompilerVersionNotFoundInImage {
		return "", err
	}

	if mainScmRepo == "" {
		return getCompilerVersionFromSCMByGitName(gitRepoName, extractCxxVersion)
	}

	return getCompilerVersionFromSCMBySCMRepoName(mainScmRepo, extractCxxVersion)
}

func CxxBuildScript(gitRepoName string) (script string, err error) {
	return getBuildScriptFromSCMByGitName(gitRepoName)
}
