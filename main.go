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
	fmt.Println("IP", myip)
	fmt.Println("GW", gw.String())

	inputs := []Input{
		{Type: "ping", Addr: "1.1.1.1"},
		{Type: "ping", Addr: "2606:4700:4700::1111"},
		{Type: "ping", Addr: gw.String()},
		{Type: "tcp", Addr: "1.1.1.1:443"},
		{Type: "tcp", Addr: "[2606:4700:4700::1111]:443"},
		{Type: "tcp", Addr: gw.String() + ":80"},
		{Type: "udp", Addr: "1.1.1.1:53"},
		{Type: "udp", Addr: "[2606:4700:4700::1111]:53"},
		{Type: "udp", Addr: gw.String() + ":53"},
	}
	errors := make([]error, len(inputs))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(inputs))
	for i, input := range inputs {
		go func() {
			defer wg.Done()
			errors[i] = Access(ctx, input)
		}()
	}
	wg.Wait()

	for i, input := range inputs {
		ok := "\x1b[32mOK\x1b[m"
		if err := errors[i]; err != nil {
			ok = "\x1b[31mNG\x1b[m"
		}
		fmt.Println(ok, input.Type, input.Addr)
	}
}

type Input struct {
	Type string
	Addr string
}

func Access(ctx context.Context, input Input) error {
	switch input.Type {
	case "ping":
		pinger := probing.New(input.Addr)
		pinger.Count = 1
		pinger.Timeout = time.Second
		err := pinger.RunWithContext(ctx)
		return err
	case "tcp":
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", input.Addr)
		if err == nil {
			_ = conn.Close()
		}
		return err
	case "udp":
		msg := &dns.Msg{}
		msg.SetQuestion(dns.Fqdn("www.google.com"), dns.TypeA)
		_, _, err := (&dns.Client{}).ExchangeContext(ctx, msg, input.Addr)
		return err
	}
	panic("unexpected")
}
