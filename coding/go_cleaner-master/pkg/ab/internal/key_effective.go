package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type KeyEffectChecker interface {
	MustMultiGet(ks []string)
	Effectiveness(k string) KeyEffectiveness
}

type KeyEffectiveness int

func (k KeyEffectiveness) Expired() bool {
	return k == KeyExpired
}

const (
	KeyEffective KeyEffectiveness = 1
	KeyExpired                    = 2
	KeyUncertain                  = 3
)

type KeyEffectClickHouseChecker struct {
	data map[string]KeyEffectiveness
}

func newKeyEffectClickHouseChecker() *KeyEffectClickHouseChecker {
	return &KeyEffectClickHouseChecker{data: map[string]KeyEffectiveness{}}
}

func (c *KeyEffectClickHouseChecker) MustMultiGet(ks []string) {
	e, err := keyEffective0(ks)
	if err != nil {
		fmt.Println("get ab key effective info err:", err)
		panic(err)
	}
	for _, k := range ks {
		if effective, ok := e[k]; !ok {
			c.data[k] = KeyUncertain
		} else if effective {
			c.data[k] = KeyEffective
		} else {
			c.data[k] = KeyExpired
		}
	}
	return
}

func (c *KeyEffectClickHouseChecker) Effectiveness(k string) KeyEffectiveness {
	if _, ok := c.data[k]; !ok {
		c.MustMultiGet([]string{k})
	}
	return c.data[k]
}

func keyEffective0(ks []string) (map[string]bool, error) {
	if len(ks) == 0 {
		return map[string]bool{}, nil
	}
	effective := map[string]bool{}
	var keys []string
	for i := range ks {
		keys = append(keys, "'"+ks[i]+"'")
	}
	query := fmt.Sprintf("SELECT ab_key,effective from tiktok_serverarch_gdp.ab_key where date=(select max(date) from tiktok_serverarch_gdp.ab_key) and ab_key in (%s)", strings.Join(keys, ",")) // ignore_security_alert
	data := bytes.NewBuffer([]byte(query))
	resp, err := http.Post("http://clickhouse.bytedance.net/cnch_gamma_yg?user=db711734-54f9-4c9d-88f3-0ced4bb9d713&password=c5305d5c-e79b-40f0-b3a8-9de09dbcfe9a&query_id=dewfrbwdgrtqswdw", "text/plain", data)
	if err != nil {
		return nil, err
	}
	rows, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(rows))
	for _, row := range bytes.Split(rows, []byte{'\n'}) {
		if len(row) == 0 {
			continue
		}
		items := bytes.Split(row, []byte{'\t'})
		if len(items) < 2 {
			return nil, fmt.Errorf("invalid result: %v", items)
		}
		if items[1][0] == '1' {
			effective[strings.Trim(string(items[0]), "'")] = true
		} else if items[1][0] == '0' {
			effective[strings.Trim(string(items[0]), "'")] = false
		}
	}
	return effective, nil
}

type KeyEffectAssignChecker struct {
	expiredKey map[string]bool
}

func newKeyEffectAssignChecker(expiredKey []string) *KeyEffectAssignChecker {
	c := &KeyEffectAssignChecker{expiredKey: map[string]bool{}}
	for _, k := range expiredKey {
		c.expiredKey[k] = true
	}
	return c
}

func (c *KeyEffectAssignChecker) MustMultiGet(ks []string) {
}

func (c *KeyEffectAssignChecker) Effectiveness(k string) KeyEffectiveness {
	if _, ok := c.expiredKey[k]; ok {
		return KeyExpired
	}
	return KeyEffective
}
