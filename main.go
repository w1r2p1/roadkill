package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/CuteQ/roadkill/orderbook"

	"github.com/gorilla/websocket"
	_ "github.com/pebbe/zmq4"
)

func main() {
	var (
		headers http.Header
		c       websocket.Dialer
		msgRecv []interface{}
	)

	entryMsg := map[string]string{
		"command": "subscribe",
		"channel": "BTC_ETH",
	}

	connStr := "wss://api2.poloniex.com"
	conn, _, err := c.Dial(connStr, headers)

	if err != nil {
		fmt.Println("Error in connection")
	}
	defer conn.Close()
	conn.WriteJSON(entryMsg)

	// TODO: Get poloniex current tick count by asking the database itself

	for {
		conn.ReadJSON(&msgRecv)
		msgLength := len(msgRecv)

		if msgLength < 3 {
			continue
		}

		msgData := msgRecv[2].([]interface{})
		dataLength := len(msgData)
		deltas := make([]orderbook.Delta, dataLength, dataLength)

		st := time.Now()
	dataIter:
		for i := 0; i < dataLength; i++ {
			var (
				eventType uint8
				price     float64
				size      float64
			)

			tickData := msgData[i].([]interface{})

			switch tickData[0] {
			case "o": // Orderbook updates
				// Poloniex update format:
				//	[<MARKET_ID>, <MARKET_TICK>, [
				//		[<TICK_TYPE>, <BOOK_SIDE>, <PRICE>, <NEW_PRICE>],
				//		...
				//	]]
				price, err = strconv.ParseFloat(tickData[2].(string), 32)
				size, err = strconv.ParseFloat(tickData[3].(string), 32)

				switch tickData[1] { // Book side
				case 0: // Ask
					eventType = orderbook.IsUpdate | orderbook.IsAsk
				case 1: // Bid
					eventType = orderbook.IsUpdate | orderbook.IsBid
				}

			case "t": // Trade event
				price, err = strconv.ParseFloat(tickData[3].(string), 32)
				size, err = strconv.ParseFloat(tickData[4].(string), 32)

				switch tickData[2] { // Book side
				case 0: // Ask
					eventType = orderbook.IsTrade | orderbook.IsAsk
				case 1: // Bid
					eventType = orderbook.IsTrade | orderbook.IsBid
				}

			case "i": // Base orderbook event
				// The Poloniex orderbook tick is formatted as follows:
				//	[<MARKET_ID>, <MARKET_TICK>, {
				//		currencyPair: <MARKET>_<ASSET>,
				//		orderBook: [
				//			<ASK>{<ASK_PRICE>: <AMOUNT_ASSET>, ...},
				//			<BID>{<BID_PRICE>: <AMOUNT_ASSET>, ...}
				//		]
				//	}]

				// snapshotTick converts the orderbook data into a parsable format
				snapshotTick := tickData[1].(map[string]interface{})["orderBook"].([]interface{})
				snapshot := orderbook.Snapshot{
					Timestamp: uint32(time.Now().UnixNano() / 1000),
					StartSeq:  0,
					AskSide:   snapshotTick[0],
					BidSide:   snapshotTick[1],
				}
				break dataIter
			}

			deltas[i] = orderbook.Delta{
				Timestamp: uint32(time.Now().UnixNano() / 1000),
				Tick:      0,
				Event:     eventType,
				Price:     float32(price),
				Size:      float32(size),
			}
		}
		en := time.Now().Sub(st)
		fmt.Println("Time elapsed: ", en)
	}
}
