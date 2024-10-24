package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/jackpal/gateway"
	"github.com/miekg/dns"
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
	resolve2("www.google.com")
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

func resolve(domain string) {
	var resolver *net.Resolver
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ips, err := resolver.LookupIP(ctx, "ip", domain)
	if err != nil {
		fmt.Printf("NG resolve from local (%v)\n", err)
		return
	}
	fmt.Printf("OK resolve from local (%s -> %s)\n", domain, ips[0])
}

func resolve2(domain string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client := &dns.Client{Timeout: time.Second}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		msg := &dns.Msg{}
		msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
		dnsServer := "1.1.1.1:53"
		res, _, err := client.ExchangeContext(ctx, msg, dnsServer)
		if err != nil {
			fmt.Printf("NG resolve from %s (%v)\n", dnsServer, err)
			return
		}
		for _, answer := range res.Answer {
			if a, ok := answer.(*dns.A); ok {
				fmt.Printf("OK resolve from %s (%s -> %s)\n", dnsServer, domain, a.A.String())
			}
		}
	}()
	go func() {
		defer wg.Done()
		msg := &dns.Msg{}
		msg.SetQuestion(dns.Fqdn(domain), dns.TypeAAAA)
		dnsServer := "[2606:4700:4700::1111]:53"
		res, _, err := client.ExchangeContext(ctx, msg, dnsServer)
		if err != nil {
			fmt.Printf("NG resolve from %s (%v)\n", dnsServer, err)
			return
		}
		for _, answer := range res.Answer {
			if a, ok := answer.(*dns.AAAA); ok {
				fmt.Printf("OK resolve from %s (%s -> %s)\n", dnsServer, domain, a.AAAA.String())
			}
		}
	}()
	wg.Wait()
}
