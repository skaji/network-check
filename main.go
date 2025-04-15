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

	ping("1.1.1.1", "2606:4700:4700::1111", gw.String())
	udp("1.1.1.1:53", "[2606:4700:4700::1111]:53", gw.String()+":53")
	tcp("1.1.1.1:443", "[2606:4700:4700::1111]:443", gw.String()+":80")
}

type result struct {
	OK   bool
	Type string
	Addr string
}

func (r *result) String() string {
	ok := "OK"
	if !r.OK {
		ok = "NG"
	}
	return fmt.Sprintf("%s %s %s", ok, r.Type, r.Addr)
}

func ping(addrs ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(addrs))
	out := make([]*result, len(addrs))
	for i, addr := range addrs {
		go func() {
			defer wg.Done()
			pinger := probing.New(addr)
			pinger.Count = 1
			pinger.Timeout = time.Second
			err := pinger.RunWithContext(ctx)
			out[i] = &result{Type: "ping", OK: err == nil, Addr: addr}
		}()
	}
	wg.Wait()
	for _, str := range out {
		fmt.Println(str)
	}
}

func udp(addrs ...string) {
	domain := "www.google.com"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client := &dns.Client{Timeout: time.Second}

	var wg sync.WaitGroup
	wg.Add(len(addrs))
	out := make([]*result, len(addrs))
	for i, addr := range addrs {
		go func() {
			defer wg.Done()
			msg := &dns.Msg{}
			msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
			_, _, err := client.ExchangeContext(ctx, msg, addr)
			out[i] = &result{Type: "udp", OK: err == nil, Addr: addr}
		}()
	}
	wg.Wait()
	for _, str := range out {
		fmt.Println(str)
	}
}

func tcp(addrs ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(addrs))
	out := make([]*result, len(addrs))
	for i, addr := range addrs {
		go func() {
			defer wg.Done()
			conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
			if err == nil {
				conn.Close()
			}
			out[i] = &result{Type: "tcp", OK: err == nil, Addr: addr}
		}()
	}
	wg.Wait()
	for _, str := range out {
		fmt.Println(str)
	}
}
