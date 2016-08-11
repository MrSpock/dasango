package main

import (
	"flag"
	"fmt"
	"github.com/mrspock/dasango"
	"github.com/mrspock/dasango/graph"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native" // Native engine
	"log"
	//"log/syslog"
	"github.com/spf13/viper"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

//type CfgSQLConn struct {
//Hostname string
//Port     int
//Username string
//Password string
//Dbname   string
//}

func checkError(err error) {
	if err != nil {
		log.Println(err)
	}
}

func checkeSQLResult(rows []mysql.Row, res mysql.Result, err error) ([]mysql.Row,
	mysql.Result) {
	checkError(err)
	return rows, res
}

func main() {
	// cli options
	community := "public"
	rrdOutputDir := flag.String("rrdout", "/tmp/onu-rrds", "Output directory for auto created RRD files")
	oltList := flag.String("olt", "10.9.0.9:public", "Comma separated list of OLT's to query and community i.e: 10.0.0.1:public")
	timeout := flag.Int("timeout", 10, "Timeout for handling connections to OLT's")
	logfile := flag.String("log", "", "Optional log output (default stderr)")
	sqlenable := flag.Bool("sqlenable", false, "Enable OLD_ID ONU_ID updates for NCC database")
	flag.Parse()
	log.SetOutput(os.Stderr)
	viper.SetConfigName("onustats")       // name of config file (without extension)
	viper.AddConfigPath("/etc/onustats/") // path to look for the config file in
	viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
	viper.AddConfigPath(".")              // optionally look for config in the working directory
	viper.SetConfigType("json")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
		os.Exit(1)
	}

	if len(*logfile) > 0 {
		f, err := os.OpenFile(*logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
		defer f.Close()
		if err != nil {
			log.Println("Unable to open logfile:", err)
		} else {
			log.SetOutput(f)
		}
	}
	olts := strings.Split(*oltList, ",")
	finishCnt := len(olts)
	sqlchan := make(chan string, 10)
	if *sqlenable {
		log.Println("SQl updates enabled !")
		go processSQLCmd(sqlchan)
	}

	done := make(chan string, len(olts))
	for _, olt := range olts {
		oltData := strings.Split(olt, ":")
		if len(oltData) == 2 {
			community = oltData[1]
		}

		log.Println("Fetching data from", olt)
		go processOlt(oltData[0], community, *rrdOutputDir, olt, *sqlenable, sqlchan, done)
	}
	for {
		select {
		case msg := <-done:
			log.Println("Querying", msg, "finished")
			finishCnt--
			if finishCnt == 0 {
				log.Println("Done for all OLT's")
				os.Exit(0)
			}

		case <-time.After((time.Duration(*timeout) * time.Second)):
			log.Println("Timeout")
			os.Exit(1)
		}
	}

}

func processSQLCmd(data chan string) {
	var err error
	dbready := true
	db := mysql.New("tcp", "", fmt.Sprintf("%s:%f", viper.Get("sql.connection.hostname"), viper.Get("sql.connection.port")), viper.Get("sql.connection.username").(string), viper.Get("sql.connection.password").(string), viper.Get("sql.connection.dbname").(string))
	err = db.Connect()
	//defer db.Close()
	if err != nil {
		log.Fatalln("Database connection failed:", err)
		dbready = false
	}
	for {
		select {
		case msg := <-data:
			if dbready {
				fmt.Println("Executing", msg)
				checkeSQLResult(db.Query(msg))

			}
		}
	}
}

func processOlt(oltip string, snmpcommunity string, rrdout string, oltname string, updatedb bool, dbchan chan string, done chan string) {
	log.Println("Goroutine launched for OLT", oltip)
	var err error

	//olt := dasango.MakeOLT("malkolt01")
	olt := dasango.MakeOLT(oltname)
	olt.SNMPSession.Community = snmpcommunity

	olt.IPAddress, err = net.ResolveIPAddr("ip", oltip)
	if err != nil {
		log.Fatalln("Unable to resolve IP for", oltip)
	}
	err = olt.Connect()
	if err != nil {
		log.Fatalln("Connection problem with", oltip)
	}
	err = olt.GetONUList()

	//	olt.AddONU(dasango.ONU{Id: 1, OltId: 2, Serial: "DSN0000", RxLevel: 0})
	if err != nil {
		log.Println(err)
	}
	log.Println(len(olt.ONUs), "ONT's parsed at", oltip)
	onus, err := olt.GetONURxLevels()
	if err != nil {
		log.Println(fmt.Sprintf("GetONURxLevels(%s) error: %s", oltip, err))
	}
	for _, onu := range onus {
		filepath := path.Join(rrdout, fmt.Sprintf("%s-rxlevel.rrd", onu.Serial))
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			err = graph.CreateRXLevelsRRD(filepath, 60)
			if err != nil {
				log.Println("Error creating graph:", err)
			}
		}
		graph.UpdateRXLevelsRRD(filepath, int(onu.RxLevel*10))
		if updatedb {
			dbchan <- fmt.Sprintf("UPDATE %s set `%s` = '%d', onu_id = '%d', olt_ip = INET_ATON('%s') where serial ='%s'", viper.Get("sql.dbconfig.onutable"), viper.Get("sql.dbconfig.olt_id_col"), onu.OltId, onu.Id, oltip, onu.Serial)
		}
		//log.Println(onu.Serial, "->", onu.OltId, ":", onu.Id)
		//fmt.Printf("ONU %s RxLevel: %f\n", onu.Serial, onu.RxLevel)
	}
	done <- oltip
}
