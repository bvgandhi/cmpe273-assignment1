package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strconv"
	"strings"
)

//Args struct to passed to server
type Args struct {
	Budget          float64
	StockpercentMap map[string]int
}

//PortfolioResp to be send from server to client
type PortfolioResp struct {
	//E.g. “GOOG:100:+$520.25”, “YHOO:200:-$30.40”
	Stocksbought       string
	CurrentMarketValue float64
	UnvestedAmount     float64
}

//Buyresponse to be send to client
type Buyresponse struct {
	TradeID      int
	Stocksbought string
	// (E.g. “GOOG:100:$500.25”, “YHOO:200:$31.40”)
	UnvestedAmount float64
	Status         string
}

//X for passing trading id
var X int

var err error

/*
type Args struct {
	X, Y int
}
*/
func buystocks(client net.Conn, c *rpc.Client) {
	var stockipstr string
	var Budget float64

	fmt.Printf("Enter the stock and allocation ")
	fmt.Scanln(&stockipstr)
	fmt.Printf("Enter the budget ")
	fmt.Scanln(&Budget)

	sStocknum := strings.Split(stockipstr, ",")
	//fmt.Println(s1.len)
	//keep a for loop
	// step 1 populate  2 arrays  , 1 for names other for %'s
	//step 2 keep a count to add %'s, after loop is done chk if its 100
	count := 0
	//BuyallocationMap consists of stocks n % for eahc stock
	StockpercentMap := make(map[string]int)
	for _, v := range sStocknum {
		sSplited := strings.Split(v, ":")
		sSplitnumper := strings.Split(sSplited[1], "%")
		i, err := strconv.Atoi(sSplitnumper[0])
		if err != nil {
			// handle error
			fmt.Println(err)
			os.Exit(2)
		}
		StockpercentMap[strings.ToUpper(sSplited[0])] = i
		//sSplited[1]
		count = count + i
	}
	if count != 100 {
		fmt.Println("Sum of Stock Percentages should be 100")
		os.Exit(2)
	}
	//fmt.Println("stockpercentMap", StockpercentMap)
	args := &Args{Budget, StockpercentMap}

	//fmt.Println("args on clientside", args)
	var reply Buyresponse

	err = c.Call("StockCstmr.BuyingStocks", args, &reply)
	if err != nil {
		log.Fatal("Error While Buying Stocks:", err)
	}
	//fmt.Println("reply from server:", reply)
	if reply.Status == "Success" {
		fmt.Println("TradeId", (float64(int(reply.TradeID*100)) / 100))
		fmt.Println("Stocks", reply.Stocksbought)
		fmt.Println("UnvestedAmount", (float64(int(reply.UnvestedAmount*100)) / 100))
	} else {
		fmt.Println("Error in buying stocks")
	}
	fmt.Println("#########################################")
}

func dispPortfolio(client net.Conn, c *rpc.Client) {
	fmt.Printf("Enter the trading id ")
	fmt.Scanln(&X)
	var portfolioResp PortfolioResp
	err = c.Call("StockCstmr.DisplayingPortfolio", &X, &portfolioResp)
	if err != nil {
		log.Fatal("Error While DisplayingPortfolio:", err)
	}
	//fmt.Println("reply from server for DisplayingPortfolio:", portfolioResp)
	fmt.Println("Stocks:", portfolioResp.Stocksbought)
	fmt.Println("CurrentMarketValue:", (float64(int(portfolioResp.CurrentMarketValue*100)) / 100))
	fmt.Println("UnvestedAmount:", (float64(int(portfolioResp.UnvestedAmount*100)) / 100))
	fmt.Println("#########################################")
}

func main() {

	var input string

	var c *rpc.Client

	var client net.Conn

	client, err = net.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	c = jsonrpc.NewClient(client)

	for {

		fmt.Println("Enter A to Buy stocks")
		fmt.Println("Enter B to check portfolio stocks")
		fmt.Println("Enter C to Exit")
		fmt.Println("#########################################")
		fmt.Printf("Enter the input ")
		fmt.Scanln(&input)

		switch input {

		case "A":

			buystocks(client, c)

			//Case 2 Code for trading id
		case "B":
			dispPortfolio(client, c)
		case "C":
			os.Exit(0)
		}
	}
}
