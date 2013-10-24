package sgip

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	CONN_STATUS_INIT  = 0
	CONN_STATUS_BIND  = 1
	CONN_STATUS_CLOSE = 2
)

type submitMessage struct {
	para         submitInput
	responseChan chan string
}

var submitChan chan submitMessage

func startTcpServer() {
	go tcpServerLoop()

	submitChan = make(chan submitMessage, sgipConfig.SubmitQueueDepth)

	for i := 0; i < sgipConfig.TcpClientCount; i++ {
		go tcpClientLoop()
	}
}

// tcp server goroutine
func tcpServerLoop() {
	port := fmt.Sprintf(":%d", sgipConfig.SpTcpListenPort)
	ln, err := net.Listen("tcp4", port)
	if err != nil {
		sgipConfig.Logger.Errorf("tcp listen error :%s", err.Error())
		panic(err.Error())
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			sgipConfig.Logger.Errorf("tcp accept error :%s", err.Error())
			continue
		}

		go handleTcpConnection(conn)
	}
}

// tcp client goroutine
func tcpClientLoop() {
	defer func() {
		if errRecover := recover(); errRecover != nil {
			sgipConfig.Logger.Errorf("recover in tcp client goroutine:%v", errRecover)
		}
	}()
	var buf [512]byte
	var conn net.Conn
	connActive := false
	for {
		submitMsg := <-submitChan
		sgipConfig.Logger.Debug("get a submit request in tcp client goroutine")

		// get the connection
		var err error
		conn, err = getConnection(conn, connActive)
		if err != nil {
			sgipConfig.Logger.Errorf("get connecttion error:%s", err.Error())
			submitMsg.responseChan <- ""
			connActive = false
			continue
		} else {
			connActive = true
		}

		// send submit
		s := newSubmit(&submitMsg.para)
		var submitLength int
		submitLength, err = s.Encode(buf[:])
		if err != nil {
			sgipConfig.Logger.Errorf("encode sumbit error:%s", err.Error())
			submitMsg.responseChan <- ""
			connActive = false
			continue
		}

		// because the SGP may close the tcp connection, so here may try 2 times.
		for i := 0; i < 2; i++ {
			err = sendBindSubmit(conn, buf[:submitLength], 3)
			if err != nil {
				conn.Close()
				sgipConfig.Logger.Debugf("send bind or submit error:%T %s", err, err.Error())

				connActive = false
				conn, err = getConnection(conn, connActive)
				if err != nil {
					break
				} else {
					connActive = true
				}
			} else {
				break
			}
		}

		if err != nil {
			sgipConfig.Logger.Errorf("send submit error:%s", err.Error())
			submitMsg.responseChan <- ""
			connActive = false
			continue
		}

		// submit successful, send back the sequence
		submitMsg.responseChan <- fmt.Sprintf("%08X%08X%08X", s.sequence[0], s.sequence[1], s.sequence[2])
	}
}

func getConnection(conn net.Conn, connActive bool) (net.Conn, error) {
	var err error
	if connActive == false {
		if conn, err = getNewConnection(); err != nil {
			return nil, err
		}

		// send bind
		var buf [128]byte
		bindMsg := newBind()
		bindLen := bindMsg.Encode(buf[:])
		if err = sendBindSubmit(conn, buf[:bindLen], 1); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

// send bind or submit, then wait the response
func sendBindSubmit(conn net.Conn, buf []byte, cmdType byte) error {
	conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(sgipConfig.WriteTimeoutSecond)))
	sgipConfig.Logger.Info("tcp client goroutine prepare to send: ", bytesToHexString(buf))
	_, err := conn.Write(buf)
	if err != nil {
		return err
	} else {
		conn.SetWriteDeadline(time.Time{})
	}

	sgipConfig.Logger.Debug("send over")

	// receive bind resp or submit resp
	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(sgipConfig.ReadTimeoutSecond)))
	var rcvLen int
	rcvLen, err = conn.Read(buf[:29])
	if err != nil {
		return err
	} else {
		sgipConfig.Logger.Info("tcp client rcv bytes: ", bytesToHexString(buf[:rcvLen]))
		conn.SetReadDeadline(time.Time{})
	}

	if rcvLen != 29 || buf[4] != 0x80 || buf[7] != cmdType || buf[20] != 0 {
		return fmt.Errorf("invalid response")
	}

	return nil
}

func getNewConnection() (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sgipConfig.SgpIp, sgipConfig.SgpPort))
	return conn, err
}

func handleTcpConnection(conn net.Conn) {
	defer conn.Close()

	// check remote ip
	if checkClientIp(conn.RemoteAddr().String(), sgipConfig.SgpIp) == false {
		return
	}

	// init status
	connStatus := CONN_STATUS_INIT

	// get buffer
	reader := bufio.NewReader(conn)
	var rcvbuf [512]byte
	var sndbuf [512]byte

	for {
		// read command length and command id
		conn.SetReadDeadline(time.Now().Add(time.Duration(sgipConfig.ReadTimeoutSecond) * time.Second))
		count, err := reader.Read(rcvbuf[0:8])
		logBytes(rcvbuf[0:count], "tcp server rcv :")
		if err != nil {
			sgipConfig.Logger.Warnf("tcp server receive error :%s", err.Error())
			return
		} else if count < 8 {
			sgipConfig.Logger.Warnf("tcp server receive not enough bytes")
			return
		}
		conn.SetReadDeadline(time.Time{})
		commandLength := bytesToIntBig(rcvbuf[:4])
		commandId := bytesToIntBig(rcvbuf[4:8])

		// judge commandId and commandLength
		rcvPacket := processRcvCommandHead(commandId, commandLength)
		if rcvPacket == nil {
			sgipConfig.Logger.Warn("can't parse receive bytes, so server would close connection")
			return
		}

		// receive the rest command bytes
		conn.SetReadDeadline(time.Now().Add(time.Duration(sgipConfig.ReadTimeoutSecond) * time.Second))
		count, err = reader.Read(rcvbuf[8:commandLength])
		logBytes(rcvbuf[8:8+count], "tcp server rcv :")
		if err != nil {
			sgipConfig.Logger.Warnf("tcp server receive error :%s", err.Error())
			return
		} else if count < commandLength-8 {
			sgipConfig.Logger.Warnf("tcp server receive not enough bytes")
			return
		}
		conn.SetReadDeadline(time.Time{})

		// process command
		if rcvPacket.Decode(rcvbuf[:commandLength]) == false {
			sgipConfig.Logger.Warnf("tcp server packet decode error")
		}
		sgipConfig.Logger.Debugf("tcp server rcv packet:%s", rcvPacket.String())
		resp := rcvPacket.Process(&connStatus)
		if resp == nil {
			sgipConfig.Logger.Warnf("tcp server packet process error")
			return
		}
		sgipConfig.Logger.Debugf("tcp server snd packet:%s", resp.String())

		// send response
		sndLen := resp.Encode(sndbuf[:])
		if sndLen < 0 {
			sgipConfig.Logger.Errorf("tcp server send buffer overflow")
			return
		}
		_, err = conn.Write(sndbuf[:sndLen])
		if err != nil {
			sgipConfig.Logger.Warnf("tcp server send error: %s", err.Error())
			return
		}
		logBytes(sndbuf[:sndLen], "tcp server snd : ")

		// close after send
		if connStatus == CONN_STATUS_CLOSE {
			sgipConfig.Logger.Info("close the connection")
			return
		}
	}
}

func checkClientIp(remoteAdd string, allowIp string) bool {
	sgipConfig.Logger.Infof("Connected by %s", remoteAdd)
	index := strings.Index(remoteAdd, ":")
	if index <= 0 {
		sgipConfig.Logger.Infof("Connected not allow by %s", remoteAdd)
		return false
	}
	remoteIp := remoteAdd[:index]
	if remoteIp != allowIp {
		sgipConfig.Logger.Infof("Connected not allow by %s", remoteAdd)
		return false
	}

	return true
}
