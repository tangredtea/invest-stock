package data

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func quoteURL(secid string) string {
	q := url.Values{}
	q.Set("secid", secid)
	q.Set("fields", "f43,f44,f45,f46,f60")
	return "https://push2.eastmoney.com/api/qt/stock/get?" + q.Encode()
}

type quoteResp struct {
	Data struct {
		F43 int `json:"f43"` // 最新价 *1000
		F44 int `json:"f44"` // 最高
		F45 int `json:"f45"` // 最低
		F46 int `json:"f46"` // 开盘
		F60 int `json:"f60"` // 昨收
	} `json:"data"`
}

func FetchQuote(secid string) (Quote, error) {
	rawSecID := secid
	secid, ok := normalizeSecID(secid)
	if !ok {
		return Quote{}, fmt.Errorf("无效证券ID: %q", rawSecID)
	}

	body, err := httpGet(quoteURL(secid))
	if err != nil {
		return Quote{}, err
	}
	var r quoteResp
	if err := json.Unmarshal(body, &r); err != nil {
		return Quote{}, err
	}
	q := Quote{
		Price:    float64(r.Data.F43) / 1000,
		High:     float64(r.Data.F44) / 1000,
		Low:      float64(r.Data.F45) / 1000,
		Open:     float64(r.Data.F46) / 1000,
		PreClose: float64(r.Data.F60) / 1000,
	}
	if err := validateQuote(q); err != nil {
		return Quote{}, err
	}
	return q, nil
}
