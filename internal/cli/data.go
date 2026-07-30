package cli

import (
	"fmt"

	"invest/pkg/data"
)

func FetchKLines(secid string) ([]data.KLine, error) {
	klines, err := data.FetchKLines(secid)
	if err != nil {
		return nil, wrapKLineError(err)
	}
	return klines, nil
}

func FetchQuote(secid string) (data.Quote, error) {
	quote, err := data.FetchQuote(secid)
	if err != nil {
		return data.Quote{}, wrapQuoteError(err)
	}
	return quote, nil
}

func wrapKLineError(err error) error {
	return fmt.Errorf("获取K线失败: %w", err)
}

func wrapQuoteError(err error) error {
	return fmt.Errorf("获取实时行情失败: %w", err)
}
