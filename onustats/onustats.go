package main

import (
	"fmt"
	"github.com/mrspock/dasango"
	"github.com/mrspock/dasango/graph"
	//	"github.com/soniah/gosnmp"
	//"log"
	"net"
	"os"
)

func main() {
	var err error
	//olt := dasango.MakeOLT("malkolt01")
	olt := dasango.MakeOLT("malkolt01")

	olt.IPAddress, err = net.ResolveIPAddr("ip", "10.9.0.9")
	if err != nil {
		fmt.Print("Unable to resolve IP")
	}
	err = olt.Connect()
	if err != nil {
		fmt.Println("Connetion problem")
	}

	err = olt.GetONUList()

	//	olt.AddONU(dasango.ONU{Id: 1, OltId: 2, Serial: "DSN0000", RxLevel: 0})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("ONU list updated with", len(olt.ONUs), " new ONUs")
	onus, err := olt.GetONURxLevels()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetONURxLevels() error: %v", err)
	}
	for _, onu := range onus {
		filepath := fmt.Sprintf("%s-rxlevel.rrd", onu.Serial)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			err = graph.CreateRXLevelsRRD(filepath, 300)
			if err != nil {
				fmt.Println("Error creating graph:", err)
			}
		}
		//fmt.Printf("ONU %s RxLevel: %f\n", onu.Serial, onu.RxLevel)
	}
}
