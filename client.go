// client.go
package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type BuyRequest struct {
	STOCKS string
	BUDGET float64
}

type BuyResponse struct {
	TRADEID int
	STOCKS string
	UNVESTEDAMOUNT float64
}

func BuyStocks(c *rpc.Client) {
	var stocks string
	var budget float64
	fmt.Print("STOCKS:")
	fmt.Scanf("%s", &stocks)
	fmt.Print("BUDGET:")
	fmt.Scanf("%f", &budget)

	request := &BuyRequest{stocks, budget}
	var response BuyResponse
	
	err := c.Call("VirtualTradingPlatform.Buy", request, &response)

	if err != nil {
		log.Fatal("error:", err)
	}

	fmt.Println("TRADEID:", response.TRADEID)
	fmt.Println("STOCKS:", response.STOCKS)
	fmt.Println(fmt.Sprintf("UNVESTEDAMOUNT:%.2f", response.UNVESTEDAMOUNT))
}

type GetRequest struct {
	TRADEID int
}

type GetResponse struct {
	STOCKS string
	CURRENTMARKETVALUE float64
	UNVESTEDAMOUNT float64
}


func GetPortfolio(c *rpc.Client) {
	var tradeid int
	fmt.Print("tradeid:")
	fmt.Scanln(&tradeid)
	
	request := &GetRequest{tradeid}
	var response GetResponse

	err := c.Call("VirtualTradingPlatform.Get",  request, &response)

	if err != nil {
		log.Fatal("error:", err)
	}	

	fmt.Println("STOCKS:", response.STOCKS)
	fmt.Println(fmt.Sprintf("CURRENTMARKETVALUE:%.2f", response.CURRENTMARKETVALUE))
	fmt.Println(fmt.Sprintf("UNVESTEDAMOUNT:%.2f", response.UNVESTEDAMOUNT))
}

func main() {

	client, err := net.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	c := jsonrpc.NewClient(client)

	for {
		var option int
		fmt.Println("1. Buy stocks")
		fmt.Println("2. Check Portfolio")
		fmt.Scanln(&option)

		if option == 1 {
			BuyStocks(c)
		} else if option == 2 {
			GetPortfolio(c)
		} else {
			fmt.Println("Invalid option")
		}
	}
}