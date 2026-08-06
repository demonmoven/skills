package util

import (
	"context"
	"fmt"

	"code.byted.org/gopkg/tccclient"
	"github.com/sirupsen/logrus"
)

var (
	cli *tccclient.ClientV2
)

func init() {
	var err error
	cli, err = tccclient.NewClientV2("byte.arch.govern", tccclient.NewConfigV2())
	if err != nil {
		logrus.Errorf("init tcc client failed, err=%+v\n", err)
	}
}

func GetIesOpsOauth() string {
	if cli == nil {
		return ""
	}
	secret, err := cli.Get(context.Background(), "oauth_ies_ops")
	if err != nil {
		return ""
	}
	return secret
}

func GetRepoOriginURL(repo string) (string, error) {
	if cli == nil {
		return "", fmt.Errorf("tcc client not init")
	}
	oauth := GetIesOpsOauth()
	if oauth == "" {
		return "", fmt.Errorf("get oauth failed")
	}
	return fmt.Sprintf("https://oauth2:%s@code.byted.org/%s.git", oauth, repo), nil
}
