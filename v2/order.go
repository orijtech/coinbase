// Copyright 2017 orijtech, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coinbase

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Order lifecycle
// The HTTP Request will respond when an order is either
// rejected (insufficient funds, invalid parameters, etc)
// or received (accepted by the matching engine). A 200
// response indicates that the order was received and is active.
// Active orders may execute immediately depending on price
// and market conditions, either partially or fully.
// A partial execution will put the remaining size of the
// order in the "open" state. An order that is filled completely
// will go into the "done" state.
//
// Users listening to streaming market data are encouraged to
// use the CustomOrderID (or "client_oid" in JSON) field to
// identify their received messages in the feed. The REST
// response with a server OrderID("order_id" in JSON) may
// come after the received message in the public data feed.
//
// Response:
// A successful order will be assigned an order id. A successful
// order is defined as one that has been accepted by the matching engine.

type Order struct {
	Side Side `json:"side"`

	// Product must be the valid list of currency pairs
	Product string `json:"product_id,omitempty"`

	// Price must be specified in quote increment product units.
	// The quote increment is the smallest unit of price.
	// For the BTC-USD product, the quote increment is 0.01 or 1 penny.
	// Prices less than 1 penny will not be accepted, and no
	// fractionaly penny prices will be accepted. It is not required
	// for Market Orders.
	Price float64 `json:"price,string,omitempty"`

	// Size must be greater than the base_min_size for the
	// product and no larger than the base_max_size.
	// The size can be in any increment of the base currency
	// (BTC for the BTC-USD product), which includes Satoshi units.
	// Size indicates the amount of BTC to buy or sell.
	Size float64 `json:"size,string,omitempty"`

	// SelfTradePrevention is an optional field which if
	// set ensures that you avoid performing self trades.
	SelfTradePrevention SelfTradePrevention `json:"stp,omitempty"`

	// CustomerID is an optional OrderID selected
	// by the user to identify your order.
	// It must be a UUID generated by your trading application.
	// This field value will be broadcast in the public feed
	// for received messages. You can use this field to
	// identify your orders in the public feed.
	// CustomOrderID is different from GDAX's server assigned
	// ID. If you are consuming the public feed and see
	// a record message with your CustomOrderID, you should record
	// the server assigned order_id as it will be used for future
	// order updates. The CustomOrderID will NOT be used after
	// the received message is sent.
	CustomOrderID string `json:"client_oid,omitempty"`

	// Limit Order Parameters
	// TimeInForce is an optional field enumerated by:
	//  * GTC
	//  * GTT
	//  * IOC
	//  * FOK
	// Default is GTC
	TimeInForce TimeInForce `json:"time_in_force,omitempty"`

	// CancelAfter is an optional field that determines
	// how long until an unmatched order should be cancelled.
	// Requires TimeInForce to be GTT.
	CancelAfter Period `json:"cancel_after,omitempty"`

	// PostOnly is an optional field that increases market
	// participants' ability to control their provision, or
	// taking of market liquidity, and thus better anticipate
	// trading costs.
	// See https://www.nasdaqtrader.com/content/ProductsServices/Trading/postonly_factsheet.pdf
	// PostOnly is invalid when TimeInForce is IOC or FOK.
	PostOnly bool `json:"post_only,omitempty"`
	// End of Limit Order Parameters

	// Market Order Parameters
	// Market orders differ from limit orders in that they provide no
	// pricing guarantees. They however do provide a way to buy or sell
	// specific amounts of Bitcoin or Fiat without having to specify
	// the price. Market orders execute immediately and no part of
	// the market order will go on the open order book. Market orders
	// are always considered takers and iincur taker fees. When placing
	// a market order you can specify funds and/or size. Funds will limit
	// how much of your quote currency account balance is used, and size
	// will limit the Bitcoin amount transacted.
	//
	// Funds is an optional field that relays the
	// desired amount of quote currency to use.
	// Either Funds or Size but not both, should be set.
	// Funds is optionally used for Market orders. When specified,
	// it indicates how much of the product quote currency to buy or sell.
	// For example, a market buy for BTC-USD with funds specified as 150.00
	// will spend 150 USD to buy BTC (including any fees). If the funds field
	// is not specified for a market buy order, Size must be specified
	// and GDAX will use available funds in your account to buy Bitcoin.
	Funds float64 `json:"funds,string,omitempty"`
	// End of Market Order Parameters

	// Stop Order Parameters
	// Price:
	// Size:
	// Funds:
	//
	//  Stop orders become active and wait to trigger based on the movement
	// of the last trade price. There are two types of stop orders:
	// * sell stop
	// * buy stop
	// The Side parameter is important:
	// * Side: 'sell': Place a sell stop order, which triggers when the
	//    last trade price changes to a value at or below Price.
	// * Side: 'buy': Place a buy stop order, which triggers when the
	//    last trade price changes to a value at or above Price.
	// The last trade price is the last price at which an order was filled.
	// This price can be found in the latest Match message
	// i.e. https://docs.gdax.com/#the-code-classprettyprintfullcode-channel.
	// Note that not all match messages may be received due to dropped message.
	// Note that when triggered, stop orders execute as market orders
	// and are therefore subject to Market Order holds https://docs.gdax.com/#holds
	// End of Stop Order Parameters

	// Margin Parameters
	// OverdraftEnabled if set specifies that funding will be provided
	// if order's cost cannot be covered by the account's balance.
	// Once set, it'll help GDAX automatically determine FundingAmount
	// such that you can place the order. If you have enough funds to
	// cover the order in your account, FundingAmount will be 0. If
	// you do not have enough funds to cover the order's cost,
	// GDAx will set FundingAmount to be the difference. For example
	// if you have 100 USD in your margin account and place a PostOnly
	// limit to buy 2 BTC @800USD, they'll set FundingAmount to be
	//    2 * 800 - 100 = 1500USD
	OverdraftEnabled bool `json:"overdraft_enabled,string,omitempty"`

	// FundingAmount is the amount of funding to be provided
	// for the order. It is the amount of funding that you wish
	// to be credited to your account at the time of order placement.
	// For buy orders this value is denominated in the quote currency
	// and for sell orders it is denominated in the base currency.
	// On the BTC-USD product, this would be USD for buy orders
	// and BTC for sell orders. This amount cannot be larger
	// than the cost of the order.
	FundingAmount float64 `json:"funding_amount,string,omitempty"`
}

var (
	errBlankPriceOrSize = errors.New("expecting either price or size to have been set")

	errBlankSide = errors.New("expecting side to be set")

	errCancelAfterWithoutGTT = errors.New("CancelAfter if set requires TimeInForce to be GTT")
)

func (o *Order) Validate() error {
	if o == nil || o.Product == "" {
		return errBlankProduct
	}
	if o.Price <= 0 && o.Size <= 0 {
		return errBlankPriceOrSize
	}
	if o.Side == "" {
		return errBlankSide
	}
	if o.CancelAfter != "" && o.TimeInForce != GTT {
		return errCancelAfterWithoutGTT
	}
	return nil
}

type OrderResponse struct {
	ID            string    `json:"id,omitempty"`
	Price         float64   `json:"price,string,omitempty"`
	Size          float64   `json:"size,string,omitempty"`
	ProductID     string    `json:"product_id,omitempty"`
	Side          Side      `json:"side,omitempty"`
	STP           string    `json:"stp,omitempty"`
	Type          Type      `json:"type,omitempty"`
	TimeInForce   string    `json:"time_in_force,omitempty"`
	PostOnly      bool      `json:"post_only,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	FillFees      float64   `json:"fill_fees,string,omitempty"`
	ExecutedValue float64   `json:"executed_value,string,omitempty"`
	Status        Status    `json:"status,omitempty"`
	Settled       bool      `json:"settled,omitempty"`
}

type Status string

const (
	// Active signifies that the profile can be used for trading.
	Active Status = "active"
	Locked Status = "locked"
	// Default signifies that you were not able repay funding
	// after a margin call or expired funding and now have a default.
	Default Status = "locked"
	Pending Status = "pending"
)

// TimeInForce policies provide guarantees about the lifetime
// of an order. There are four policies:
//  * Good Till Time	    GTT
//  * Immediate Or Cancel   IOC
//  * Fill Or Kill	    FOK
type TimeInForce string

const (
	//  * Good Till Canceled
	// These orders remain open on the book until canceled.
	// This is the default behavior if no policy is specified.
	GTC TimeInForce = "GTC"

	// Good Till Time
	//  These orders remain open on the book until canceled or the alloted
	//  CancelAfter is depleted on the matching engine. GTT orders are
	// guaranteed to cancel before any other order is processed after
	// the CancelAfter timestamp which is returned by the API.
	GTT TimeInForce = "GTT"

	// Immediate Or Cancel orders instantly cancel the remaining
	// size of the limit order instead of opening it on the book.
	IOC TimeInForce = "IOC"

	// Fill Or Kill orders are rejected if the entire size cannot be matched.
	FOK TimeInForce = "FOK"
)

type Period string

const (
	Minute Period = "min"
	Hour   Period = "hour"
	Day    Period = "day"
)

type SelfTradePrevention string

const (
	DecreaseAndCancel SelfTradePrevention = "dc"
	CancelOldest      SelfTradePrevention = "co"
	CancelNewest      SelfTradePrevention = "cn"
	CancelBoth        SelfTradePrevention = "cb"
)

func (c *Client) Order(o *Order) (*OrderResponse, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	blob, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	fullURL := "https://api.gdax.com/orders"
	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	blob, _, err = c.doAuthAndReq(req)
	if err != nil {
		return nil, err
	}
	ores := new(OrderResponse)
	if err := json.Unmarshal(blob, ores); err != nil {
		return nil, err
	}
	return ores, nil
}
