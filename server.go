// server.go
package main

import (
	"fmt"
	"errors"
	"container/list"
	"log"
	"net"
	"strings"
	"strconv"
	"net/rpc"
	"net/http"
	"net/rpc/jsonrpc"
	"io/ioutil"
	"regexp"
)


// -------------------------------------------------

func GetHttp(url string) (error, string) {
	res, err := http.Get(url)
	if err != nil {
		return err, ""
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if  err != nil {
		return err, ""
	}

	return nil, string(data)
}

func LookupYahooFinance(ticker string) (error, float64) {
	url := fmt.Sprintf("http://finance.yahoo.com/webservice/v1/symbols/%s/quote?format=json", ticker)
	
	err, json := GetHttp(url)
	if err != nil {
		return err, 0
	}

	regex, err := regexp.Compile("\"price\".*:.*\"(.*)\".*")
	matches := regex.FindStringSubmatch(json)
	
	if len(matches) != 2 {
		return errors.New("Error parsing data from Yahoo"), 0
	}

	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return err, 0
	}

	return nil, val
}

// -------------------------------------------------

type VirtualTradingPlatform struct
{
	tradeid int
	db list.List
}

type StockRecord struct {
	ticker string
	perStockValue float64
	count int
}

type DBRecord struct {
	tradeid int
	stocks list.List
	unvestedAmount float64
}

func (t *VirtualTradingPlatform) GetDBRecord(tradeid int) (error, DBRecord) {
	for it := t.db.Front(); it != nil; it = it.Next() {
		record := it.Value.(DBRecord)
		if record.tradeid == tradeid {
			return nil, record
		}
	}

	return errors.New("TradeID not found"), DBRecord{}
}

//
// Buying stocks
//
type BuyRequest struct {
	STOCKS string
	BUDGET float64
}

type BuyResponse struct {
	TRADEID int
	STOCKS string
	UNVESTEDAMOUNT float64
}

type Stock struct {
	ticker string
	pct float64
}

func parse(data string, list *list.List) error {
	stocks := strings.Split(data, ",")

	var totalpct float64 = 0

	for i := 0; i < len(stocks); i += 1 {
		stock := stocks[i]

		tokens := strings.Split(stock, ":")
		if len(tokens) != 2 {
			return errors.New("Invalid token")
		}

		ticker := tokens[0]
		pct,err := strconv.ParseFloat(tokens[1], 64)

		if err != nil {
			return errors.New("Error converting to float")
		}

		totalpct += pct

		list.PushBack(Stock{ticker, pct})
	}

	if totalpct != 100 {
		return errors.New("Percentage is not adding to 100%")
	}

	return nil
}

func (t *VirtualTradingPlatform) Buy(request *BuyRequest, response *BuyResponse) error {
	fmt.Println("VirtualTradingPlatform::Buy STOCKS=", request.STOCKS)
	fmt.Println("BUDGET=", request.BUDGET)

	response.TRADEID = t.tradeid
	response.UNVESTEDAMOUNT = 0

	t.tradeid += 1

	var err error

	stocks := list.New()
	err = parse(request.STOCKS, stocks)

	if err != nil {
		return err
	}

	stocklist := list.New()

	for s := stocks.Front(); s != nil; s = s.Next() {
		stock := s.Value.(Stock)
		err, perStockValue := LookupYahooFinance(stock.ticker)

		if err != nil {
			return err
		}

		amount := stock.pct / 100 * request.BUDGET
		count := int(amount / perStockValue)
		unvestedAmount := amount - (float64(count) * perStockValue)
		if len(response.STOCKS) > 0 {
			response.STOCKS += ", "
		}
		
		response.STOCKS += fmt.Sprintf("%s:%d:$%.2f", stock.ticker, count, perStockValue)
		response.UNVESTEDAMOUNT += unvestedAmount

		stocklist.PushBack(StockRecord{stock.ticker, perStockValue, count})
	}

	t.db.PushBack(DBRecord{response.TRADEID, *stocklist, response.UNVESTEDAMOUNT})

	return nil
}


//
// Checking the portfolio
//

type GetRequest struct {
	TRADEID int
}

type GetResponse struct {
	STOCKS string
	CURRENTMARKETVALUE float64
	UNVESTEDAMOUNT float64
}

func makeCurrentMarketValue(stocks list.List) float64 {
	var ret float64

	ret = 0

	for it := stocks.Front(); it != nil; it = it.Next() {
		var rec StockRecord
		rec = it.Value.(StockRecord)

		_, perStockValue := LookupYahooFinance(rec.ticker)
		ret += float64(rec.count) * perStockValue
	}

	return ret
}

func makeStockString(stocks list.List) string {
	var stockstr string

	for it := stocks.Front(); it != nil; it = it.Next() {
		var rec StockRecord
		rec = it.Value.(StockRecord)

		_, currentPerStockValue := LookupYahooFinance(rec.ticker)
		var symbol string
		if currentPerStockValue > rec.perStockValue {
			symbol = "+"
		} else if currentPerStockValue < rec.perStockValue {
			symbol = "-"
		} else {
			symbol = "="
		}
		if len(stockstr) > 0 {
			stockstr += ", "
		}
		stockstr += fmt.Sprintf("%s:%d:%s$%.2f", rec.ticker, rec.count, symbol, currentPerStockValue)
	}

	return stockstr
}

func (t *VirtualTradingPlatform) Get(request *GetRequest, response *GetResponse) error {
	fmt.Println("VirtualTradingPlatform::Get request.tradeid= ", request.TRADEID)

	var err error
	var rec DBRecord
	err, rec = t.GetDBRecord(request.TRADEID)

	if err != nil {
		return err
	}

	response.CURRENTMARKETVALUE = makeCurrentMarketValue(rec.stocks)
	response.STOCKS = makeStockString(rec.stocks)
	response.UNVESTEDAMOUNT = rec.unvestedAmount

	return nil
}

func main() {
	vtrade := new(VirtualTradingPlatform)
	vtrade.tradeid = 0
	server := rpc.NewServer()
	server.Register(vtrade)
	server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	listener, e := net.Listen("tcp", ":1234")

	if e != nil {
		fmt.Println("listen error:", e)
		return
	}

	fmt.Println("Server started")

	for {
		if conn, err := listener.Accept(); err != nil {
			fmt.Println("accept error: " + err.Error())
			log.Fatal("accept error: " + err.Error())
		} else {
			log.Printf("new connection established\n")
			go server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}
}