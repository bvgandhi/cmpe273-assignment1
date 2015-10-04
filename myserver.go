package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strconv"
)

var tradingid = 0

var tradeMap = make(map[int]Trade)

//Args struct to passed to server
type Args struct {
	Budget          float64
	StockpercentMap map[string]int
}

//Buyresponse to be send to client
type Buyresponse struct {
	TradeID      int
	Stocksbought string
	// (E.g. “GOOG:100:$500.25”, “YHOO:200:$31.40”)
	UnvestedAmount float64
	Status         string
}

//PortfolioResp to be send from server to client
type PortfolioResp struct {
	//E.g. “GOOG:100:+$520.25”, “YHOO:200:-$30.40”
	Stocksbought       string
	CurrentMarketValue float64
	UnvestedAmount     float64
}

//Stock cntg stock's name buying price count
type Stock struct {
	name        string
	buyingPrice float64
	//currmktprice float64
	boughtcount int
}

//Trade for a trade_id it will hold trade details
type Trade struct {
	//id of the trade, static counter increasing per trade, to be used in trades_donemap as key
	Tradingid      int
	UnvestedAmount float64
	Stocks         []Stock
}

//MyResponse structure from Yahoo
type MyResponse struct {
	List struct {
		Meta struct {
			Type  string `json:"type"`
			Start int    `json:"start"`
			Count int    `json:"count"`
		} `json:"meta"`
		Resources []struct {
			Resource struct {
				Classname string `json:"classname"`
				Fields    struct {
					Name    string `json:"name"`
					Price   string `json:"price"`
					Symbol  string `json:"symbol"`
					Ts      string `json:"ts"`
					Type    string `json:"type"`
					Utctime string `json:"utctime"`
					Volume  string `json:"volume"`
				} `json:"fields"`
			} `json:"resource"`
		} `json:"resources"`
	} `json:"list"`
}

//StockCstmr structure does buying and dispalying of portfolio
type StockCstmr struct{}

//BuyingStocks does buying of stocks called by client
func (t *StockCstmr) BuyingStocks(args *Args, reply *Buyresponse) error {
	//fmt.Println("buy reqt to server", args)
	buyCapacityMap := calperstockallocation(args)
	priceMap := returnStockValue(buyCapacityMap)
	buystocks(priceMap, buyCapacityMap, reply)
	//fmt.Println("$$$$$$$$$$$$$Response from buystocks", reply)
	//fmt.Println("TRADEMAP", tradeMap)
	return nil
}

//DisplayingPortfolio to display portfolio loss or gain
func (t *StockCstmr) DisplayingPortfolio(X *int, portfolioResp *PortfolioResp) error {

	//fmt.Println("trading id passed to server:", X)
	//	fmt.Println("*** trading id passed to server:", (*X))
	buyCapacityMap := make(map[string]float64)

	trade := tradeMap[(*X)]

	for istock := range trade.Stocks {
		buyCapacityMap[trade.Stocks[istock].name] = 0.00

	}
	priceMap := returnStockValue(buyCapacityMap)

	var buffer bytes.Buffer
	currMktPrice := 0.00
	for istock := range trade.Stocks {
		//buyCapacityMap[trade.Stocks[istock].name] = 0.00
		if istock > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(trade.Stocks[istock].name)
		buffer.WriteString(":")

		buffer.WriteString(strconv.Itoa(trade.Stocks[istock].boughtcount))
		buffer.WriteString(":")

		currPrice := priceMap[trade.Stocks[istock].name]
		currMktPrice = (currPrice * float64(trade.Stocks[istock].boughtcount)) + currMktPrice
		if currPrice > (trade.Stocks[istock].buyingPrice) {
			buffer.WriteString("+")
		} else if currPrice < (trade.Stocks[istock].buyingPrice) {
			buffer.WriteString("-")
		} else {
			buffer.WriteString("=")
		}
		buffer.WriteString(strconv.FormatFloat(currPrice, 'f', 2, 64))

	}

	portfolioResp.CurrentMarketValue = currMktPrice
	portfolioResp.UnvestedAmount = trade.UnvestedAmount
	portfolioResp.Stocksbought = buffer.String()

	return nil
}

func calperstockallocation(args *Args) map[string]float64 {

	buyCapacityMap := make(map[string]float64)

	for stock, percent := range args.StockpercentMap {
		buyCapacityMap[stock] = (float64(percent) / 100) * args.Budget
	}
	return buyCapacityMap
}

func main() {
	stk := new(StockCstmr)
	server := rpc.NewServer()
	server.Register(stk)
	server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	listener, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	for {
		if conn, err := listener.Accept(); err != nil {
			log.Fatal("accept error: " + err.Error())
		} else {
			log.Printf("new connection established\n")
			go server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}
}

func buystocks(priceMap map[string]float64, buyCapacityMap map[string]float64, reply *Buyresponse) {
	//fmt.Println("************BUY STOCKS ENTERED***************")

	unvested := 0.00
	var trade Trade
	var buffer bytes.Buffer
	//to count diff individual stock in trade
	counter := 0
	stockArr := make([]Stock, len(buyCapacityMap))
	for stock, capacity := range buyCapacityMap {
		//fmt.Println("stock:", stock, "capacity:", capacity)
		//	fmt.Println("length of priceMap", len(priceMap))
		price := priceMap[stock]
		//fmt.Println("Curr price", price, "of stock", stock)
		if price == 0 {
			reply.Status = "Failure"
			fmt.Println("Failure Occurred")
			return
		}
		//fmt.Println("@@@@@@@@@@@@@@@@price:", price, "capacity:", capacity)
		if capacity > price {
			bought, _ := math.Modf(capacity / price)
			unvested = unvested + (capacity - (bought * price))
			//	fmt.Println("bought:", bought, "unvested:", unvested)
			// name      buyingPrice  currmktprice boughtcount
			stockArr[counter] = Stock{name: stock, buyingPrice: price, boughtcount: int(bought)}
			counter++
			if counter > 1 {
				buffer.WriteString(",")
			}
			buffer.WriteString(stock)
			buffer.WriteString(":")
			buffer.WriteString(strconv.Itoa(int(bought)))
			buffer.WriteString(":$")
			buffer.WriteString(strconv.FormatFloat(price, 'f', 2, 64))

		} else {
			unvested = unvested + capacity
		}
		//buffer

	}
	if counter > 0 {
		tradingid++
		trade.Tradingid = tradingid
		trade.UnvestedAmount = unvested
		trade.Stocks = stockArr
		tradeMap[tradingid] = trade
	}

	//fmt.Println("trade", trade)
	reply.TradeID = tradingid
	reply.UnvestedAmount = unvested
	reply.Stocksbought = buffer.String()
	reply.Status = "Success"
}

func returnStockValue(buyCapacityMap map[string]float64) map[string]float64 {
	var s MyResponse
	var priceMap map[string]float64
	var buffer bytes.Buffer
	//left part of url
	buffer.WriteString("http://finance.yahoo.com/webservice/v1/symbols/")
	//adding the stocks reqd
	stockCounter := 0
	for stock := range buyCapacityMap {
		//buyCapacityMap[stock] = (float64(percent) / 100) * args.Budget
		if stockCounter > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(stock)
		stockCounter++
	}

	buffer.WriteString("/quote?format=json")

	//fmt.Println(buffer.String())
	response, err := http.Get(buffer.String())
	if err != nil {
		fmt.Printf("error occured")
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)

		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		json.Unmarshal([]byte(contents), &s)

		priceMap = make(map[string]float64)

		for i := 0; i < s.List.Meta.Count; i++ {
			f, err1 := strconv.ParseFloat(s.List.Resources[i].Resource.Fields.Price, 64)
			priceMap[s.List.Resources[i].Resource.Fields.Symbol] = f
			if err1 != nil {
				fmt.Printf("%s", err1)
				os.Exit(1)
			}
		}
	}
	return priceMap
}
