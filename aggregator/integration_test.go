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

package aggregator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "makerdao/gofer/model"
	"makerdao/gofer/pather"
)

func TestPathWithSetzerPatherAndMedianIntegration(t *testing.T) {
	pas := []*PriceAggregate{
		newTestPricePointAggregate(6, "exchange1", "ETH", "BTC", 2, 1),
		newTestPricePointAggregate(7, "exchange2", "ETH", "BTC", 4, 1),
		// median( ()=>ETH/BTC$2, ()=>ETH/BTC$4 )=>ETH/BTC$3

		// Should be filtered due to outside time window
		newTestPricePointAggregate(-1000, "exchange0", "BTC", "USD", 1000, 1),
		// Should be overwritten by entry 3 due to same exchange but older
		newTestPricePointAggregate(1, "exchange1", "BTC", "USD", 2000, 1),
		newTestPricePointAggregate(2, "exchange2", "BTC", "USD", 20, 1),
		newTestPricePointAggregate(3, "exchange1", "BTC", "USD", 3, 1),
		// Should be skipped due to non-matching pair
		newTestPricePointAggregate(4, "exchange4", "n", "o", 4, 1),
		newTestPricePointAggregate(5, "exchange5", "BTC", "USD", 5, 1),
		// median( ()=>BTC/USD$3, ()=>BTC/USD$5, ()=>BTC/USD$20 )=>BTC/USD$5
		// trade( median(..)=>ETH/BTC$3, median(..)=>BTC/USD$5 )=>ETH/USD$15

		newTestPricePointAggregate(8, "exchange1", "ETH", "USDT", 3, 1),
		newTestPricePointAggregate(9, "exchange2", "USDT", "USD", 3, 1),
		// trade( median(..)=>ETH/USDT$3, medain(..)=>USDT/USD$3 )=>ETH/USD$9
	}
	// indirect-median( trade(..)=>ETH/USD$9, trade(..)=>ETH/USD$15 )=>ETH/USD$12

	for i := 0; i < 100; i++ {
		pathAggregator := NewPathWithDefaultTrade(
			pather.NewSetzer(),
			func (pair *Pair) Aggregator {
				return NewMedian(pair, 1000)
			},
			func (pair *Pair) Aggregator {
				return NewIndirectMedian(pair)
			},
		)

		randomReduce(pathAggregator, &Pair{Base: "ETH", Quote: "USD"}, pas)

		res_ETH_USD := pathAggregator.Aggregate(&Pair{Base: "ETH", Quote: "USD"})
		assert.NotNil(t, res_ETH_USD)
		assert.Equal(t, &Pair{Base: "ETH", Quote: "USD"}, res_ETH_USD.Pair)
		assert.Equal(t, "indirect-median", res_ETH_USD.PriceModelName)
		assert.Equal(t, uint64(12), res_ETH_USD.Price)

		res_ETH_BTC := pathAggregator.Aggregate(&Pair{Base: "ETH", Quote: "BTC"})
		assert.NotNil(t, res_ETH_BTC)
		assert.Equal(t, &Pair{Base: "ETH", Quote: "BTC"}, res_ETH_BTC.Pair)
		assert.Equal(t, "indirect-median", res_ETH_BTC.PriceModelName)
		assert.Equal(t, uint64(3), res_ETH_BTC.Price)

		res_BTC_USD := pathAggregator.Aggregate(&Pair{Base: "BTC", Quote: "USD"})
		assert.NotNil(t, res_BTC_USD)
		assert.Equal(t, &Pair{Base: "BTC", Quote: "USD"}, res_BTC_USD.Pair)
		assert.Equal(t, "indirect-median", res_BTC_USD.PriceModelName)
		assert.Equal(t, uint64(5), res_BTC_USD.Price)

		res_ETH_KRW := pathAggregator.Aggregate(&Pair{Base: "ETH", Quote: "KRW"})
		assert.Nil(t, res_ETH_KRW, "Pair not existing in Pather")

		res_REP_USD := pathAggregator.Aggregate(&Pair{Base: "REP", Quote: "USD"})
		assert.NotNil(t, res_REP_USD, "Pair existis in Pather but no price points yet")
		assert.Equal(t, &Pair{Base: "REP", Quote: "USD"}, res_REP_USD.Pair)
		assert.Equal(t, "indirect-median", res_REP_USD.PriceModelName)
		assert.Equal(t, uint64(0), res_REP_USD.Price)
	}
}