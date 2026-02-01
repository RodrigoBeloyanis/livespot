package binance

import "encoding/json"

type AccountInfo struct {
	Balances []AccountBalance `json:"balances"`
}

type AccountBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

func ParseAccountInfo(body []byte) (AccountInfo, error) {
	var info AccountInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return AccountInfo{}, err
	}
	return info, nil
}

func FindBalance(info AccountInfo, asset string) (AccountBalance, bool) {
	for _, balance := range info.Balances {
		if balance.Asset == asset {
			return balance, true
		}
	}
	return AccountBalance{}, false
}

