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
			if isInFrame && (len(frameBuf) > 0) {
				log.Printf(log.DEBUG, "ending frame, len %d\n", len(frameBuf))
				log.Println(log.DEBUG, frameBuf)
				switch frameBuf[0] {
				case 'S':
					readStatusFrame(frameBuf[3], frameBuf[4:4+len(frameBuf)-6])
				case 'E':
					readNotificationFrame(NOTIFICATION_ERROR, frameBuf[1:1+len(frameBuf)-3])
				case 'D':
					readDataFrame()
				}
				isInFrame = false
			} else {
				log.Println(log.DEBUG, "starting frame")
				isInFrame = true
				frameBuf = frameBuf[:0]
			}
		} else {
			if isInFrame {
				frameBuf = append(frameBuf, buf[0])
			}
		}
	}
}
