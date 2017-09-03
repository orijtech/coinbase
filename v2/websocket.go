package coinbase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/orijtech/wsu"
)

type Type string

const (
	TypeReceived  Type = "received"
	TypeMarket    Type = "market"
	TypeLimit     Type = "limit"
	TypeOpen      Type = "open"
	TypeActivate  Type = "activate"
	TypeEntry     Type = "entry"
	TypeHeartbeat Type = "heartbeat"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Message struct {
	Time           time.Time `json:"time,omitempty"`
	Type           Type      `json:"type,omitempty"`
	ProductID      string    `json:"product_id,omitempty"`
	SequenceNumber int       `json:"sequence,omitempty"`
	OrderID        string    `json:"order_id,omitempty"`
	Size           float64   `json:"size,string,omitempty"`
	Price          float64   `json:"price,string,omitempty"`
	OrderType      string    `json:"order_type,omitempty"`
	Funds          float64   `json:"funds,strings,omitempty"`
	Side           Side      `json:"side,omitempty"`
	RemainingSize  float64   `json:"remaining_size,string,omitempty"`
	Reason         Reason    `json:"reason,omitempty"`
	MakerOrderID   string    `json:"maker_order_id,omitempty"`
	TakerOrderID   string    `json:"taker_order_id,omitempty"`

	OldFunds           float64 `json:"old_funds,string,omitempty"`
	NewFunds           float64 `json:"new_funds,string,omitempty"`
	Nonce              uint64  `json:"nonce,omitempty"`
	Position           string  `json:"position,omitempty"`
	PositionSize       float64 `json:"position_size,string,omitempty"`
	PositionCompliment float64 `json:"position_compliment,string,omitempty"`
	PositionMaxSize    float64 `json:"position_max_size,string,omitempty"`

	CallSide       Side      `json:"call_side,omitempty"`
	CallPrice      float64   `json:"call_price,string,omitempty"`
	CallFunds      float64   `json:"call_funds,string,omitempty"`
	Covered        bool      `json:"covered,omitempty"`
	NextExpireTime time.Time `json:"next_expire_time,omitempty"`
	BaseBalance    float64   `json:"base_balance,string,omitempty"`
	BaseFunding    float64   `json:"base_funding,string,omitempty"`
	QuoteBalance   float64   `json:"quote_balance,string,omitempty"`
	QuoteFunding   float64   `json:"quote_funding,string,omitempty"`
	Private        bool      `json:"private,omitempty"`

	StopPrice    float64 `json:"stop_price,omitempty"`
	StopType     Type    `json:"stop_type,omitempty"`
	TakerFeeRate float64 `json:"taker_fee_rate,string,omitempty"`
	LastTradeID  string  `json:"last_trade_id,omitempty"`

	Message string `json:"message,omitempty"`

	// These fields are only set if authenticated
	TakerUserID    string `json:"taker_user_id,omitempty"`
	TakerProfileID string `json:"taker_profile_id,omitempty"`
	MyProfileID    string `json:"profile_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`

	Err error `json:"err,omitempty"`
}

type Reason string

const (
	ReasonFilled   Reason = "filled"
	ReasonCanceled Reason = "canceled"
)

type Subscription struct {
	Authenticate bool     `json:"authenticate,omitempty"`
	Currencies   []string `json:"currencies,omitempty"`
}

type SubscriptionResponse struct {
	MessagesChan <-chan *Message
	cancelFn     func() error
}

func (sr *SubscriptionResponse) Close() error {
	if fn := sr.cancelFn; fn != nil {
		return fn()
	}
	return nil
}

type subscribeMessage struct {
	Type       string   `json:"type,omitempty"`
	ProductIDs []string `json:"product_ids,omitempty"`

	// The fields below are necessary when making
	// an authenticated subscription for products.
	APIKey     string `json:"key,omitempty"`
	Signature  string `json:"signature,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

var defaultProductIDs = []string{
	fmt.Sprintf("%s-%s", BTC, USD),
	fmt.Sprintf("%s-%s", ETH, USD),
	fmt.Sprintf("%s-%s", LTC, USD),
}

const (
	websocketFeedURL = "wss://ws-feed.gdax.com"
)

func (c *Client) Subscribe(sin *Subscription) (*SubscriptionResponse, error) {
	if sin == nil {
		sin = new(Subscription)
	}

	wsConn, err := wsu.NewClientConnection(&wsu.ClientSetup{
		URL: websocketFeedURL,
	})
	if err != nil {
		return nil, err
	}

	s := new(Subscription)
	*s = *sin
	if len(s.Currencies) == 0 {
		s.Currencies = defaultProductIDs[:]
	}

	sm := &subscribeMessage{
		Type:       "subscribe",
		ProductIDs: s.Currencies[:],
	}

	if s.Authenticate {
		fullURL := fmt.Sprintf("%s/users/self", unversionedBaseURL)
		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return nil, err
		}
		c.signAndSetHeaders(req)
		hdr := req.Header
		sm.Signature = hdr.Get(hdrSignatureKey)
		sm.Timestamp = hdr.Get(hdrTimestampKey)
		sm.APIKey = hdr.Get(hdrAPIKeyKey)
	}

	subscriptionBlob, err := json.Marshal(sm)
	if err != nil {
		return nil, err
	}
	// Send that subscription blob to kick off the entire process.
	wsConn.Send(&wsu.Message{Frame: subscriptionBlob})

	msgsChan := make(chan *Message)
	go func() {
		defer close(msgsChan)

		for {
			recvMsg, ok := wsConn.Receive()
			if !ok {
				return
			}
			msg := new(Message)

			if err := recvMsg.Err; err != nil {
				msg.Err = err
			} else if err := json.Unmarshal(recvMsg.Frame, msg); err != nil {
				msg.Err = err
			}
			msgsChan <- msg
		}
	}()

	sres := &SubscriptionResponse{
		cancelFn:     wsConn.Close,
		MessagesChan: msgsChan,
	}

	return sres, nil
}

const (
	wsURL = "wss://ws-feed.gdax.com"
)
