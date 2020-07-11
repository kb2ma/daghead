package main

import (
	"bytes"
	"github.com/kb2ma/daghead/internal/log"
	"github.com/lunixbochs/struc"
	"github.com/mikepb/go-serial"
	"sync"
)

const HDLC_FLAG byte = 0x7E

const NOTIFICATION_ERROR int = 0

type IsSync struct {
	IsSync byte
}

// mote_id, component, error_code, arg1, arg2 = struct.unpack('>HBBhH'
type Notification struct {
	MoteId [2]byte
	Component byte
	Code byte
	Arg1 int16
	Arg2 uint16
}


// Status type 0 is_sync; example [83 56 66 0 0 61 189]
func readStatusFrame(statusType byte, data []byte) {
	if statusType == 0 {
		buf := bytes.NewBuffer(data)
		o := &IsSync{}
		err := struc.Unpack(buf, o)
		if err != nil {
			log.Printf(log.ERROR, "err? %d\n", err)
		} else {
			log.Printf(log.INFO, "is sync? %d\n", o.IsSync)
		}
	}
}

func readNotificationFrame(notificationLevel int, data []byte) {
	buf := bytes.NewBuffer(data)
	o := &Notification{}
	err := struc.Unpack(buf, o)
	if err != nil {
		log.Printf(log.ERROR, "err? %d\n", err)
	} else {
		log.Println(log.INFO, "got notification")
	}
}

func readDataFrame() {
	log.Println(log.INFO, "got data")
}

func readSerial(wg *sync.WaitGroup, port *serial.Port) {
	defer wg.Done()

	frameBuf := make([]byte, 0, 10)
	frameLen := 0
	isInFrame := false

	for true {
		buf := make([]byte, 1)
		_, err := port.Read(buf)
		if err != nil {
			log.Panic(err)
		}

		// At startup, position in stream from device is indeterminate. So must
		// synchronize on first sequence of 0x7E 0x7E flag bytes.
		if buf[0] == HDLC_FLAG {
			if isInFrame && (frameLen > 0) {
				log.Printf(log.DEBUG, "ending frame, len %d\n", frameLen)
				data := frameBuf[:frameLen]
				log.Println(log.DEBUG, data)
				switch data[0] {
				case 'S':
					readStatusFrame(data[3], data[4:4+frameLen-6])
				case 'E':
					readNotificationFrame(NOTIFICATION_ERROR, data[1:1+frameLen-3])
				case 'D':
					readDataFrame()
				}
				isInFrame = false
				frameLen = 0
			} else {
				log.Println(log.DEBUG, "starting frame")
				isInFrame = true
				frameBuf = make([]byte, 0, 10)
				frameLen = 0   // defensive; ensure zeroed
			}
		} else {
			frameBuf = append(frameBuf, buf[0])
			if isInFrame {
				frameLen++
			}
			//log.Println(buf)
		}
	}
}
