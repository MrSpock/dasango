package main

import (
	"fmt"
	"github.com/mrspock/dasango"
	"github.com/soniah/gosnmp"
	//"log"
	"net"
	//"os"
	"time"
)

func main() {
	var err error
	//olt := dasango.MakeOLT("malkolt01")
	olt := dasango.OLT{Name: "malkolt01"}

	olt.IPAddress, err = net.ResolveIPAddr("ip", "10.9.0.9")
	if err != nil {
		fmt.Print("Unable to resolve IP")
	}
	olt.SNMPSession = &gosnmp.GoSNMP{}
	//olt.SNMPSession = gosnmp.Default
	//	olt.SNMPSession.Logger = log.New(os.Stderr, "", 0)
	olt.SNMPSession.Port = 161
	olt.SNMPSession.Version = gosnmp.Version2c
	olt.SNMPSession.Community = "public"
	olt.SNMPSession.Retries = 3
	olt.SNMPSession.Timeout = time.Duration(3) * time.Second
	err = olt.Connect()
	if err != nil {
		fmt.Println("Connetion problem")
	}

	err = olt.GetONUList()

	//	olt.AddONU(dasango.ONU{Id: 1, OltId: 2, Serial: "DSN0000", RxLevel: 0})
	if err != nil {
		fmt.Println(err)
	}
	//	fmt.Println("ONU list:")
	//	fmt.Println(olt.ONUs)
	for _, onu := range olt.ONUs {
		fmt.Printf("ONU %v RxLevel:", onu)

		_, err := olt.ReadONURxLevel(&onu)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(onu.RxLevel, "dB")

	}
}
