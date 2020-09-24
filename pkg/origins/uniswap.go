//  Copyright (C) 2020 Maker Ecosystem Growth Holdings, INC.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package origins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/makerdao/gofer/internal/query"
)

const uniswapURL = "https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v2"

type uniswapResponse struct {
	Data struct {
		Pairs []uniswapPairResponse
	}
}

type uniswapTokenResponse struct {
	Symbol string `json:"symbol"`
}

type uniswapPairResponse struct {
	Id      string               `json:"id"`
	Price0  stringAsFloat64      `json:"token0Price"`
	Price1  stringAsFloat64      `json:"token1Price"`
	Volume0 stringAsFloat64      `json:"volumeToken0"`
	Volume1 stringAsFloat64      `json:"volumeToken1"`
	Token0  uniswapTokenResponse `json:"token0"`
	Token1  uniswapTokenResponse `json:"token1"`
}

// Uniswap origin handler
type Uniswap struct {
	Pool query.WorkerPool
}

func (u *Uniswap) pairsToContractAddresses(pairs []Pair) []string {
	var names []string

	// We're checking for reverse pairs because the same contract is used to
	// trade in both directions.
	match := func(a, b Pair) bool {
		if a.Quote == b.Quote && a.Base == b.Base {
			return true
		}

		if a.Quote == b.Base && a.Base == b.Quote {
			return true
		}

		return false
	}

	for _, pair := range pairs {
		p := Pair{Base: u.renameSymbol(pair.Base), Quote: u.renameSymbol(pair.Quote)}

		switch {
		case match(p, Pair{Base: "COMP", Quote: "WETH"}):
			names = append(names, "0xcffdded873554f362ac02f8fb1f02e5ada10516f")
		case match(p, Pair{Base: "LRC", Quote: "WETH"}):
			names = append(names, "0x8878df9e1a7c87dcbf6d3999d997f262c05d8c70")
		case match(p, Pair{Base: "KNC", Quote: "WETH"}):
			names = append(names, "0xf49c43ae0faf37217bdcb00df478cf793edd6687")
		}
	}

	return names
}

// TODO: We should find better solution for this.
func (u *Uniswap) renameSymbol(symbol string) string {
	switch symbol {
	case "ETH":
		return "WETH"
	case "BTC":
		return "WBTC"
	}

	return symbol
}

func (u *Uniswap) Fetch(pairs []Pair) []FetchResult {
	var err error

	pairsJson, _ := json.Marshal(u.pairsToContractAddresses(pairs))
	body := fmt.Sprintf(
		`{"query":"query($ids:[String]){pairs(where:{id_in:$ids}){id token0Price token1Price volumeToken0 volumeToken1 token0 { symbol } token1 { symbol }}}","variables":{"ids":%s}}`,
		pairsJson,
	)

	req := &query.HTTPRequest{
		URL:    uniswapURL,
		Method: "POST",
		Body:   bytes.NewBuffer([]byte(body)),
	}

	// make query
	res := u.Pool.Query(req)
	if res == nil {
		return fetchResultListWithErrors(pairs, errEmptyOriginResponse)
	}
	if res.Error != nil {
		return fetchResultListWithErrors(pairs, res.Error)
	}

	// parse JSON
	var resp uniswapResponse
	err = json.Unmarshal(res.Body, &resp)
	if err != nil {
		return fetchResultListWithErrors(pairs, fmt.Errorf("failed to parse Uniswap response: %w", err))
	}

	// convert response from a slice to a map
	respMap := map[string]uniswapPairResponse{}
	for _, pairResp := range resp.Data.Pairs {
		respMap[pairResp.Token0.Symbol + "/" + pairResp.Token1.Symbol] = pairResp
	}

	// prepare result
	results := make([]FetchResult, 0)
	for _, pair := range pairs {
		b := u.renameSymbol(pair.Base)
		q := u.renameSymbol(pair.Quote)

		pair0 := b + "/" + q
		pair1 := q + "/" + b

		if r, ok := respMap[pair0]; ok {
			results = append(results, FetchResult{
				Tick: Tick{
					Pair:      pair,
					Price:     r.Price1.val(),
					Bid:       r.Price1.val(),
					Ask:       r.Price1.val(),
					Volume24h: r.Volume0.val(),
					Timestamp: time.Now(),
				},
			})
		} else if r, ok := respMap[pair1]; ok {
			results = append(results, FetchResult{
				Tick: Tick{
					Pair:      pair,
					Price:     r.Price0.val(),
					Bid:       r.Price0.val(),
					Ask:       r.Price0.val(),
					Volume24h: r.Volume1.val(),
					Timestamp: time.Now(),
				},
			})
		} else {
			results = append(results, FetchResult{
				Tick:  Tick{Pair: pair},
				Error: errMissingResponseForPair,
			})
		}
	}

	return results
}
