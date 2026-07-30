package main

import (
	"invest/internal/market"
	"invest/pkg/data"
)

type klineClientError int

const (
	klineClientErrorNone klineClientError = iota
	klineClientErrorInvalidCode
	klineClientErrorFetchFailed
)

type loadedKLines struct {
	SecID  string
	KLines []data.KLine
}

func resolveSecIDForCode(code string) (string, klineClientError) {
	secid := market.ResolveSecID(code)
	if secid == "" {
		return "", klineClientErrorInvalidCode
	}
	return secid, klineClientErrorNone
}

func loadKLinesForCode(code string) (loadedKLines, klineClientError, error) {
	secid, clientErr := resolveSecIDForCode(code)
	if clientErr != klineClientErrorNone {
		return loadedKLines{}, klineClientErrorInvalidCode, nil
	}
	klines, err := fetchKLines(secid)
	if err != nil {
		return loadedKLines{}, klineClientErrorFetchFailed, err
	}
	return loadedKLines{SecID: secid, KLines: klines}, klineClientErrorNone, nil
}
