package main

import (
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"os"
	"time"
)

const (
	protocolICMP = 1
)

func main() {
	host := "10.0.0.237"
	// Use icmp to get a *packetconn, note that we set `udp4` for network here
	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	// Generate an Echo message
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("Hello, are you there!"),
		},
	}
	wb, err := msg.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}
	// Send, note that here it must be a UDP address
	start := time.Now()
	if _, err := c.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP(host)}); err != nil {
		log.Fatal(err)
	}
	// Read the reply package
	reply := make([]byte, 1500)
	err = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		log.Fatal(err)
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		log.Fatal(err)
	}
	duration := time.Since(start)
	// The reply packet is an ICMP message, parsed first
	msg, err = icmp.ParseMessage(protocolICMP, reply[:n])
	if err != nil {
		log.Fatal(err)
	}
	// Print Results
	switch msg.Type {
	case ipv4.ICMPTypeEchoReply: // If it is an Echo Reply message
		echoReply, ok := msg.Body.(*icmp.Echo) // The message body is of type Echo
		if !ok {
			log.Fatal("invalid ICMP Echo Reply message")
			return
		}
		// Here you can judge by ID, Seq, remote address, the following one only uses two judgment conditions, it is risky
		// If another program also sends ICMP Echo with the same serial number, then it may be a reply packet from another program, but the chance of this is relatively small
		// If you add the ID judgment, it is accurate
		if peer.(*net.UDPAddr).IP.String() == host && echoReply.Seq == 1 {
			fmt.Printf("Reply from %s: seq=%d time=%v\n", host, msg.Body.(*icmp.Echo).Seq, duration)
			return
		}
	default:
		fmt.Printf("Unexpected ICMP message type: %v\n", msg.Type)
	}
}
