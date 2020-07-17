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
	"github.com/makerdao/gofer/model"
)

// Pather describes a set of asset pairs and how to trade between them
type Pather interface {
	// Pairs returns a list of Pairs that are tradeable
	Pairs() []*model.Pair
	// Path returns PricePaths describing how to trade between two assets
	Path(*model.Pair) []*model.PricePath
}

// FilterPotentialPricePoints returns the PotentialPricePoints that are required
// to complete the PricePaths given and nil if no paths are possible to complete
// with the given PotentialPricePoints
func FilterPotentialPricePoints(ppaths []*model.PricePath, ppps []*model.PotentialPricePoint) ([]*model.PricePath, []*model.PotentialPricePoint) {
	// Group all PotentialPricePoints by pair
	pppIndex := make(map[model.Pair][]*model.PotentialPricePoint)
	for _, ppp := range ppps {
		pair := *ppp.Pair
		pppIndex[pair] = append(pppIndex[pair], ppp)
	}

	var resPricePaths []*model.PricePath
	pairs := make(map[model.Pair]bool)
	outer:
	for _, ppath := range ppaths {
		// Check that each PricePath has all of its Pairs in PotentialPricePoints
		for _, pair := range *ppath {
			if _, ok := pppIndex[*pair]; !ok {
				// Continue with next PricePath and don't add pairs to list
				continue outer
			}
		}
		// Add each uniqe Pair to a list
		for _, pair := range *ppath {
			pairs[*pair] = true
		}
		resPricePaths = append(resPricePaths, ppath)
	}

	// Add each uniqe PotentialPricePoint by completed PricePath Pair
	var resPPP []*model.PotentialPricePoint
	for pair := range pairs {
		resPPP = append(resPPP, pppIndex[pair]...)
	}

	return resPricePaths, resPPP
}