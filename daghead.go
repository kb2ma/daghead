package main

import (
    "bytes"
    "github.com/lunixbochs/struc"
    "github.com/mikepb/go-serial"
    "github.com/snksoft/crc"
    "log"
    "math/rand"
    "sync"
    "time"
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
            log.Printf("err? %d\n", err)
        } else {
            log.Printf("is sync? %d\n", o.IsSync)
        }
    }
}

func readNotificationFrame(notificationLevel int, data []byte) {
    buf := bytes.NewBuffer(data)
    o := &Notification{}
    err := struc.Unpack(buf, o)
    if err != nil {
        log.Printf("err? %d\n", err)
    } else {
        log.Println("got notification")
    }
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
                log.Printf("ending frame, len %d\n", frameLen)
                data := frameBuf[:frameLen]
                log.Println(data)
                if data[0] == 'S' {
                    readStatusFrame(data[3], data[4:4+frameLen-6])
                } else if data[0] == 'E' {
                    readNotificationFrame(NOTIFICATION_ERROR, data[1:1+frameLen-3])
                }
                isInFrame = false
                frameLen = 0
            } else {
                log.Println("starting frame")
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

func setDagRoot(wg *sync.WaitGroup, port *serial.Port) {
    defer wg.Done()
    data := [31]byte{ 0x7E, 'R', 'T', 0xBB, 0XBB, 0, 0, 0, 0, 0, 0, 0x1, 0x15, 0x38,
                     0xb6, 0x9a, 0x00, 0xbd, 0xa9, 0x17, 0x14, 0x50, 0x1c, 0xf6,
                     0x67, 0x76, 0x62, 0xc1, 0, 0, 0x7E }
    //data := [31]byte{ 0x7E, 'R', 'T', 0xBB, 0XBB, 0, 0, 0, 0, 0, 0, 0x1}
    //rand.Read(data[12:28])
    hash := crc.CalculateCRC(crc.X25, data[1:28])
    data[28] = byte(hash & 0xFF)
    data[29] = byte((hash & 0xFF00) >> 8)
    log.Printf("setDagRoot [% X]\n", data)

    _, err := port.Write(data[:])
    if err != nil {
        log.Panic(err)
    }
    //log.Printf("hash [%X] [%X]\n", hash & 0xFF, (hash & 0xFF00) >> 8)
}

func main() {
    options := serial.RawOptions
    options.BitRate = 19200
    options.FlowControl = serial.FLOWCONTROL_XONXOFF
    options.Mode = serial.MODE_READ_WRITE
    port, err := options.Open("/dev/ttyUSB0")
    if err != nil {
        log.Panic(err)
    }
    defer port.Close()
    rand.Seed(int64(time.Now().Nanosecond()))

    var wg sync.WaitGroup
    wg.Add(1)
    go readSerial(&wg, port)

    time.Sleep(5 * time.Second)
    wg.Add(1)
    go setDagRoot(&wg, port)
    wg.Wait()
}

