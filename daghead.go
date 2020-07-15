/*
daghead: RPL DODAG router app for an OpenWSN 6TiSCH network.

Reads incoming data from root mote, and performs routine management.

  * Prints error notifications from mote to daghead log.
  * If the root mote is not actually set as DODAG root, does so. Presently avoids
    use of Constrained Join Protocol for network motes by using a static network
    key hardcoded into mote firmware.

Since RPL operates in non-storing mode, reads ICMPv6 RPL messages to maintain a
routing table for the network motes.
 */
package main

import (
	"github.com/kb2ma/daghead/internal/log"
	"github.com/mikepb/go-serial"
	toml "github.com/pelletier/go-toml"
	"github.com/snksoft/crc"
	"sync"
	"time"
)

func setDagRoot(wg *sync.WaitGroup, port *serial.Port) {
	defer wg.Done()
	// Slice [12:28] (16 bytes) should be generated randomly; requires random seed also
	data := [31]byte{ 0x7E, 'R', 'T', 0xBB, 0XBB, 0, 0, 0, 0, 0, 0, 0x1, 0x15, 0x38,
					 0xb6, 0x9a, 0x00, 0xbd, 0xa9, 0x17, 0x14, 0x50, 0x1c, 0xf6,
					 0x67, 0x76, 0x62, 0xc1, 0, 0, 0x7E }
	hash := crc.CalculateCRC(crc.X25, data[1:28])
	data[28] = byte(hash & 0xFF)
	data[29] = byte((hash & 0xFF00) >> 8)
	log.Printf(log.INFO, "setDagRoot % X\n", data)

	_, err := port.Write(data[:])
	if err != nil {
		log.Panic(err)
	}
	//log.Printf("hash [%X] [%X]\n", hash & 0xFF, (hash & 0xFF00) >> 8)
}

func main() {
	// read config file for logging level
	config, err := toml.LoadFile("daghead.conf")
	if err != nil {
		log.Fatal(err)
	}
	levelStr := config.Get("log.level").(string)
	switch levelStr {
	case "ERROR":
		log.SetLevel(log.ERROR)
	case "WARN":
		log.SetLevel(log.WARN)
	case "DEBUG":
		log.SetLevel(log.DEBUG)
	default:
		log.SetLevel(log.INFO)
	}
	log.Println(log.INFO, "Starting daghead")

	// open serial port to root mote
	options := serial.RawOptions
	options.BitRate = 19200
	options.FlowControl = serial.FLOWCONTROL_XONXOFF
	options.Mode = serial.MODE_READ_WRITE
	port, err := options.Open("/dev/ttyUSB0")
	if err != nil {
		log.Panic(err)
	}
	defer port.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go readSerial(&wg, port)

	time.Sleep(5 * time.Second)
	wg.Add(1)
	go setDagRoot(&wg, port)
	wg.Wait()
}

