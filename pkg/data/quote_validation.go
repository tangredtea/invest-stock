package data

import "fmt"

func validateQuote(q Quote) error {
	if q.Price <= 0 || q.PreClose <= 0 {
		return fmt.Errorf("实时行情无效: 最新价和昨收必须为正数")
	}
	if q.Open < 0 || q.High < 0 || q.Low < 0 {
		return fmt.Errorf("实时行情无效: 开高低不能为负数")
	}
	if q.High > 0 && q.High < q.Price {
		return fmt.Errorf("实时行情无效: 最高价低于最新价")
	}
	if q.Low > 0 && q.Low > q.Price {
		return fmt.Errorf("实时行情无效: 最低价高于最新价")
	}
	if q.High > 0 && q.Low > 0 && q.High < q.Low {
		return fmt.Errorf("实时行情无效: 最高价低于最低价")
	}
	if q.Open > 0 {
		if q.High > 0 && q.High < q.Open {
			return fmt.Errorf("实时行情无效: 最高价低于开盘价")
		}
		if q.Low > 0 && q.Low > q.Open {
			return fmt.Errorf("实时行情无效: 最低价高于开盘价")
		}
	}
	return nil
}
