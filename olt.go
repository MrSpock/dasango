package dasango

import (
	"fmt"
	"github.com/soniah/gosnmp"
	"net"
	//	"os"
	"strconv"
	"strings"
)

const (
	ONU_SERIAL   = "1.3.6.1.4.1.6296.101.23.3.1.1.4"
	ONU_RX_LEVEL = "1.3.6.1.4.1.6296.101.23.3.1.1.16"
)

type ONU struct {
	Id          int
	OltId       int
	Serial      string
	RxLevel     float32
	Description string
}

type OLT struct {
	Name        string
	IPAddress   *net.IPAddr
	Shelf_type  string
	ONUs        []ONU
	SNMPSession *gosnmp.GoSNMP
}

func MakeOLT(name string) *OLT {
	var olt = OLT{}
	olt.Name = name
	//	olt.ONUs = make([]ONU, 128)
	return &olt
}
func (o *OLT) ResolveIP() (err error) {
	o.IPAddress, err = net.ResolveIPAddr("ip", o.Name)
	return err
}
func (o *OLT) SetCommunity(community string) {
	o.SNMPSession.Community = community
}

func (o *OLT) Connect() (err error) {
	if o.IPAddress == nil {
		err = o.ResolveIP()
		if err != nil {
			return err
		}
	}
	o.SNMPSession.Target = o.IPAddress.IP.String()
	err = o.SNMPSession.Connect()
	//	defer o.SNMPSession.Conn.Close()
	return err
}
func (o *OLT) FindONU(olt_id int, onu_id int) (onu *ONU) {
	for _, v := range o.ONUs {
		if v.OltId == olt_id && onu.Id == onu_id {
			return onu
		}
	}
	return
}

func (o *OLT) AddONU(newonu ONU) []ONU {
	o.ONUs = append(o.ONUs, newonu)
	return o.ONUs
}
func (o *OLT) GetONUList() (err error) {
	var onus []ONU
	if o.SNMPSession.Conn == nil {
		fmt.Println("Establishing missing connection")
		err = o.Connect()
		if err != nil {
			return err
		}
	}
	err = o.SNMPSession.BulkWalk(ONU_SERIAL, func(pdu gosnmp.SnmpPDU) (err error) {
		olt_id, err := GetOnuOltId(pdu.Name)
		onu_id, err := GetOnuId(pdu.Name)
		if err != nil {
			fmt.Println(err)
			//fmt.Printf("%s = ", pdu.Name)
		} else {
			onu := o.FindONU(olt_id, onu_id)
			if onu == nil {
				//				fmt.Fprintf(os.Stdout, "New onu discovered - adding to ONU list (%d:%d %s)\n", olt_id, onu_id, string(pdu.Value.([]byte)))
				newonu := ONU{onu_id, olt_id, string(pdu.Value.([]byte)), -40, ""}
				onus = append(onus, newonu)
			} else {
				fmt.Println("ONU exists", onu)
			}
		}
		//		switch pdu.Type {
		//		case gosnmp.OctetString:
		//			b := pdu.Value.([]byte)
		//			fmt.Printf("%s\n", string(b))
		//		default:
		//			fmt.Printf("TYPE %d: %d\n", pdu.Type, gosnmp.ToBigInt(pdu.Value))
		//		}
		return err
	})
	o.ONUs = onus
	return err

}
func (olt *OLT) ReadONURxLevel(onu *ONU) (rxlevel float32, err error) {
	var onu_rx_oid []string
	oid := fmt.Sprintf("%s.%d.%d", ONU_RX_LEVEL, onu.OltId, onu.Id)
	onu_rx_oid = append(onu_rx_oid, oid)
	r, err := olt.SNMPSession.Get(onu_rx_oid)
	if err != nil {
		return
	}
	// ogarnac bledy po typie
	for _, variable := range r.Variables {
		//tmplevel := gosnmp.ToBigInt(variable.Value)
		rxlevel = float32((int(variable.Value.(int)))) / 10
		onu.RxLevel = rxlevel
		//rxlevel = float32(tmplevel)
	}
	return

}
func GetOnuOltId(oid string) (id int, err error) {
	elements := strings.Split(oid, ".")
	id, err = strconv.Atoi(elements[len(elements)-2])
	return
}
func GetOnuId(oid string) (id int, err error) {
	elements := strings.Split(oid, ".")
	id, err = strconv.Atoi(elements[len(elements)-1])
	return
}
