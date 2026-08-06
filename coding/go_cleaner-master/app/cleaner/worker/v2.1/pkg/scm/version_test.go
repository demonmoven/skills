package scm

import (
	"testing"

	"code.byted.org/lang/gg/gslice"
)

func TestGetGoVersionFromSCMByPsm(t *testing.T) {
	mainScmRepo, err := getTCEMainScmRepoByPSM("capcut.cloud_edit.render_service")
	if err != nil {
		t.Fatal(err)
	}

	if mainScmRepo == "" {
		t.Fatal("not found MainScmRepo from tce")
	}

	scmGoVersion, err := getCompilerVersionFromSCMBySCMRepoName(mainScmRepo, extractGoVersion)
	if err != nil {
		t.Fatalf("getCompilerVersionFromSCMBySCMRepoName err: %v", err)
	}

	if scmGoVersion == "" {
		t.Fatal("not found go version from scm")
	}
}

func TestGoVersion(t *testing.T) {
	type Repo struct {
		PSM           string
		Name          string
		ExpectVersion string
		Platfrom      string
	}

	tests := []Repo{
		{
			"capcut.cloud_edit.render_service",
			"", // 即使不提供git仓库名也应该成功
			"1.20",
			"TCE",
		},
		{
			"ulike.search.sync",
			"ulike-server/ulike-search-sync",
			"1.22",
			"ByteFaaS",
		},
		{
			"toutiao.zebra.whitebox_rule_cmd",
			"zebra/whitebox_rule",
			"1.20",
			"CronJob",
		},
		{
			"ttgame.mgplatform.plat_tcs_callback",
			"ttgame/mgplatform_plat_tcs_callback",
			"1.20",
			"Unknown",
		},
		{
			"flow.secure.standard_center_api",
			"flow/secure.standard_center_api",
			"1.23",
			"Unknown",
		},
	}

	for _, test := range tests {
		version, err := GoVersion(test.PSM, test.Name)
		if err != nil {
			t.Fatalf("%s %s: %v", test.Platfrom, test.PSM, err)
			return
		}

		if version != test.ExpectVersion {
			t.Fatalf("Expect %s %s's version is %s, but found %s", test.Platfrom, test.PSM, test.ExpectVersion, version)
		}
	}
}

func TestCxxVersion(t *testing.T) {
	type Repo struct {
		PSM           string
		Name          string
		ExpectVersion string
		Platfrom      string
	}

	tests := []Repo{
		{
			"ad.shuttle.product",
			"data/tiktok_shuttle",
			"llvm16",
			"Unknown",
		},
		{
			"",
			"ad/docking_i18n",
			"llvm16",
			"TCE",
		},
		{
			"data.ad.arbiterunion",
			"ad/arbitercpp_union",
			"llvm16",
			"Unknown",
		},
		{
			"data.union_ad.union_predict_lite",
			"data/union_predict_lite",
			"llvm16",
			"Unknown",
		},
		{
			"flow.app_audio.podcast_monad_content_node",
			"monad/monad_extra",
			"llvm16",
			"Unknown",
		},
		{
			"flow.app_audio.podcast_monad_compute_layer",
			"monad/monad_engine",
			"llvm16",
			"Unknown",
		},
		{
			"data.union.targeting_tensorflow",
			"data/tensorflow_bin",
			"llvm16",
			"Unknown",
		},
		{
			"data.groot.cold_strategy_union",
			"lagrange/groot",
			"llvm16",
			"Unknown",
		},
		/*
		{
			"acache.ad.union_ad_sequence",
			"ad/acache",
			"llvm16",
			"Unknown",
		},*/
		{
			"search.nlp.aweme_intent_commerce_bert_v2",
			"lab/quicksilver",
			"llvm16",
			"Unknown",
		},
		{
			"data.search.spider_css_cache_test",
			"data/spider_cache_proxy",
			"llvm16",
			"Unknown",
		},
		{
			"search.nlp.matx_tvm",
			"nlp/matx_service",
			"llvm16",
			"Unknown",
		},
	}

	for _, test := range tests {
		version, err := CxxVersion(test.PSM, test.Name)
		if err != nil {
			t.Fatalf("%s %s: %v", test.Name, test.PSM, err)
			return
		}

		if version != test.ExpectVersion {
			t.Fatalf("Expect %s %s's version is %s, but found %s", test.Platfrom, test.PSM, test.ExpectVersion, version)
		}
	}
}

func TestCxxBuildScript(t *testing.T) {
	type Repo struct {
		PSM          string
		Name         string
		BuildScripts []string
	}

	tests := []Repo{
		{
			"ad.shuttle.product",
			"data/tiktok_shuttle",
			[]string{
				"build_tiktok.sh",
			},
		},
		{
			"",
			"ad/docking_i18n",
			[]string{
				"build.sh",
			},
		},
		{
			"data.ad.arbiterunion",
			"ad/arbitercpp_union",
			[]string{
				"build.sh",
			},
		},
		{
			"data.union_ad.union_predict_lite",
			"data/union_predict_lite",
			[]string{
				"build.sh",
			},
		},
		{
			"ad.feature.union",
			"ad/ad_feature",
			[]string{
				"build_union_service.sh",
				"build_native_ad_service.sh",
				"build_service.sh",
			},
		},
		{
			"data.union_ad.union_feature_predict",
			"data/union_predict",
			[]string{
				"build.sh",
			},
		},
	}

	for _, test := range tests {
		script, err := CxxBuildScript(test.Name)
		if err != nil {
			t.Fatalf("%s %s: %v", test.Name, test.PSM, err)
			return
		}

		if !gslice.Contains(test.BuildScripts, script) {
			t.Fatalf("Expect %s's build script is any one of %v, but found %s", test.PSM, test.BuildScripts, script)
		}
	}
}
