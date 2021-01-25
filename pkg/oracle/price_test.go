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

package oracle

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/makerdao/gofer/pkg/ethereum"
	"github.com/makerdao/gofer/pkg/ethereum/mocks"
)

// Hash for the AAABBB asset pair, with the price set to 42 and the age to 1605371361:
var priceHash = "c678b27c20ef30b95452d8d61f8f3916899717692d8a01c595971035b25a00ff"

func TestPrice_SetFloat64Price(t *testing.T) {
	tests := []struct {
		name  string
		price float64
	}{
		{
			// Float64 with 1.0 precision:
			name:  "2^52",
			price: math.Pow(2, 52),
		},
		{
			// Smallest possible price but greater than 0:
			name:  "1/PriceMultiplier",
			price: float64(1) / PriceMultiplier,
		},
		{
			// Zero:
			name:  "0",
			price: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Price is stored internally in different format so we want to
			// be sure, that the price is not changed during that conversion.
			p := &Price{Wat: "AAABBB"}
			p.SetFloat64Price(tt.price)
			assert.Equal(t, tt.price, p.Float64Price())
		})
	}
}

func TestPrice_Sign(t *testing.T) {
	s := &mocks.Signer{}
	p := &Price{Wat: "AAABBB"}
	p.Age = time.Unix(1605371361, 0)
	p.SetFloat64Price(42)

	// Generate a random signature and address:
	sig := ethereum.Signature{}
	var addr ethereum.Address
	rand.Read(sig[:])
	rand.Read(addr[:])

	// Test Sign:
	//
	// Hash passed to the Signature function *must* be exactly the same as in
	// the priceHash var.
	hash, _ := hex.DecodeString(priceHash)
	s.On("Signature", hash).Return(sig, nil)
	err := p.Sign(s)
	assert.NoError(t, err)

	// Test From:
	//
	// Here, we're just checking if the signature and the hash passed to
	// the Recover function are the same as generated above.
	s.On("Recover", sig, hash).Return(&addr, nil)
	retAddr, err := p.From(s)
	assert.NoError(t, err)
	assert.Equal(t, addr, *retAddr)
}

func TestPrice_Sign_NoPrice(t *testing.T) {
	s := &mocks.Signer{}
	p := &Price{Wat: "AAABBB"}

	err := p.Sign(s)
	assert.Equal(t, ErrPriceNotSet, err)
}

func TestPrice_Marshall(t *testing.T) {
	p := &Price{Wat: "AAABBB"}
	p.Age = time.Unix(1605371361, 0)
	p.SetFloat64Price(42)
	p.V = 0xAA
	p.R = [32]byte{0x01}
	p.S = [32]byte{0x02}

	// Marshall to JSON:
	j, err := p.MarshalJSON()
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.JSONEq(t, `
		{
		   "wat":"AAABBB",
		   "val":"42000000000000000000",
		   "age":1605371361,
		   "v":"aa",
		   "r":"0100000000000000000000000000000000000000000000000000000000000000",
		   "s":"0200000000000000000000000000000000000000000000000000000000000000"
		}`,
		string(j),
	)

	// Unmarshall from JSON:
	var p2 Price
	err = p2.UnmarshalJSON(j)
	assert.NoError(t, err)
	assert.Equal(t, p.Wat, p2.Wat)
	assert.Equal(t, p.Age, p2.Age)
	assert.Equal(t, p.Val, p2.Val)
	assert.Equal(t, p.V, p2.V)
	assert.Equal(t, p.R, p2.R)
	assert.Equal(t, p.S, p2.S)
}