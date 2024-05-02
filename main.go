package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jackpal/gateway"
	probing "github.com/prometheus-community/pro-bing"
)

var version string = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help":
			fmt.Println("Usage: network-check")
			os.Exit(0)
		case "-v", "--version":
			fmt.Println(version)
			os.Exit(0)
		default:
			fmt.Printf("unknown option: %s\n", os.Args[1])
			os.Exit(1)
		}
	}

	gw, err := gateway.DiscoverGateway()
	if err != nil {
		fmt.Printf("cannot find gateway: %v\n", err)
		os.Exit(1)
	}
	myip, err := gateway.DiscoverInterface()
	if err != nil {
		fmt.Printf("cannot find local ip: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("IP %s\n", myip)
	fmt.Printf("GW %s\n", gw.String())
	fmt.Println("")

	ping(gw.String())
	ping("8.8.8.8")
	resolve("www.google.com")
}

func ping(addr string) {
	pinger := probing.New(addr)
	pinger.Count = 1
	pinger.Timeout = time.Second
	err := pinger.Run()
	if err != nil {
		fmt.Printf("NG ping %s (%v)\n", addr, err)
		return
	}
	stats := pinger.Statistics()
	fmt.Printf("OK ping %s (%s)\n", addr, stats.AvgRtt.String())
}

func resolve(addr string) {
	var resolver *net.Resolver
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ips, err := resolver.LookupIP(ctx, "ip", addr)
	if err != nil {
		fmt.Printf("NG resolve %s (%v)\n", addr, err)
		return
	}
	fmt.Printf("OK resolve %s (%s)\n", addr, ips[0])
}
