package sgip

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	resp_code_ok             = 0
	resp_code_login_err      = 1
	resp_code_login_type_err = 4
	resp_code_para_err       = 5
)

type packeter interface {
	JudgeCommandHead(cmdType, cmdLen int) bool
	Decode(cmd []byte) bool
	Process(connStatus *int) respPacketer
	String() string
}

type respPacketer interface {
	Encode(buf []byte) int
	Decode(cmd []byte) bool
	String() string
}

type messageHead struct {
	length   int
	cmdType  int
	sequence [3]uint32
}

type messageReserved struct {
	reserve [8]byte
}

type bind struct {
	messageHead
	loginType     byte
	loginName     string
	loginPassword string
	messageReserved
}

type bindresp struct {
	messageHead
	result byte
	messageReserved
}

type unbind struct {
	messageHead
}

type unbindresp struct {
	messageHead
}

type deliver struct {
	messageHead
	userNumber string
	spNumber   string
	tppid      byte
	tpudhi     byte
	msgCoding  byte
	msgLength  int
	msgContent []byte
	messageReserved
}

type deliverresp struct {
	messageHead
	result byte
	messageReserved
}

type report struct {
	messageHead
	submitSeq  [12]byte
	reportType byte
	userNumber string
	state      byte
	errorCode  byte
	messageReserved
}

type reportresp struct {
	messageHead
	result byte
	messageReserved
}

type submit struct {
	messageHead
	submitInput
}

func (m *bind) JudgeCommandHead(cmdType, cmdLen int) bool {
	if cmdType != 1 || cmdLen != 0x3d {
		return false
	}
	return true
}

func newBind() bind {
	var b bind
	b.length = 20 + 1 + 16 + 16 + 8
	b.cmdType = 1
	seq := getNewSequence()
	copy(b.sequence[:], seq[:])

	b.loginType = 1
	b.loginName = sgipConfig.LoginUserName
	b.loginPassword = sgipConfig.LoginPassword

	for i := 0; i < len(b.reserve[:]); i++ {
		b.reserve[i] = 0
	}

	return b
}

func (m *bind) Decode(cmd []byte) bool {
	m.DecodeHead(cmd[0:20])
	m.loginType = cmd[20]
	m.loginName = decodeBytesString(cmd[21:37])
	m.loginPassword = decodeBytesString(cmd[37:53])
	copy(m.reserve[:], cmd[53:61])
	return true
}

func (m *bind) Encode(buf []byte) int {
	length := 20 + 1 + 16 + 16 + 8
	if len(buf) < length {
		return -1
	}

	m.messageHead.Encode(buf[0:20])
	buf[20] = m.loginType
	encodeStringBytes(m.loginName, buf[21:37])
	encodeStringBytes(m.loginPassword, buf[37:53])
	m.messageReserved.Encode(buf[53:])
	return length
}

func (m *bind) Process(connStatus *int) respPacketer {
	var resp bindresp
	*connStatus = CONN_STATUS_INIT
	if m.loginType != 2 { // SMG to SP
		resp.result = resp_code_login_type_err
	} else if m.loginName != sgipConfig.LoginUserName || m.loginPassword != sgipConfig.LoginPassword {
		resp.result = resp_code_login_err
	} else {
		resp.result = resp_code_ok
		*connStatus = CONN_STATUS_BIND
	}

	resp.SetHead(20+1+8, 0x80000001, m.sequence)

	return &resp
}

func (m *bind) String() string {
	buf := bytes.NewBufferString("Bind: ")
	buf.WriteString(m.messageHead.String())
	buf.WriteString(";")
	buf.WriteString(fmt.Sprintf("Login Type:%02X;", m.loginType))
	buf.WriteString(fmt.Sprintf("Login Name:%s;", m.loginName))
	buf.WriteString(fmt.Sprintf("Login Password:%s;", m.loginPassword))
	buf.WriteString(m.messageReserved.String())

	return buf.String()
}

func (br *bindresp) Encode(buf []byte) int {
	length := 20 + 1 + 8
	if len(buf) < length {
		return -1
	}

	br.messageHead.Encode(buf[0:20])
	buf[20] = br.result
	br.messageReserved.Encode(buf[21:])
	return length
}

func (br *bindresp) Decode(cmd []byte) bool {
	br.DecodeHead(cmd[0:20])
	br.result = cmd[20]
	copy(br.reserve[:], cmd[21:29])
	return true
}

func (br *bindresp) String() string {
	buf := bytes.NewBufferString("Bind_Resp: ")
	buf.WriteString(br.messageHead.String())
	buf.WriteString(";")
	buf.WriteString(fmt.Sprintf("result:%02X;", br.result))
	buf.WriteString(br.messageReserved.String())

	return buf.String()
}

func (m *unbind) JudgeCommandHead(cmdType, cmdLen int) bool {
	if cmdType != 2 || cmdLen != 20 {
		return false
	}
	return true
}

func (m *unbind) Decode(cmd []byte) bool {
	m.DecodeHead(cmd[0:20])
	return true
}

func (m *unbind) Process(connStatus *int) respPacketer {
	*connStatus = CONN_STATUS_CLOSE
	var resp unbindresp
	resp.SetHead(20, 0x80000002, m.sequence)
	return &resp
}

func (m *unbind) String() string {
	buf := bytes.NewBufferString("Unbind: ")
	buf.WriteString(m.messageHead.String())

	return buf.String()
}

func (r *unbindresp) Encode(buf []byte) int {
	length := 20
	if len(buf) < length {
		return -1
	}

	r.messageHead.Encode(buf[0:20])
	return length
}

func (r *unbindresp) Decode(cmd []byte) bool {
	r.DecodeHead(cmd[0:20])
	return true
}

func (r *unbindresp) String() string {
	buf := bytes.NewBufferString("Unbind_Resp: ")
	buf.WriteString(r.messageHead.String())

	return buf.String()
}

func (m *deliver) JudgeCommandHead(cmdType, cmdLen int) bool {
	if cmdType != 4 || cmdLen < 20+57 {
		return false
	}
	return true
}

func (m *deliver) Decode(cmd []byte) bool {
	m.DecodeHead(cmd[0:20])
	m.userNumber = decodeBytesString(cmd[20:41])
	m.spNumber = decodeBytesString(cmd[41:62])
	m.tppid = cmd[62]
	m.tpudhi = cmd[63]
	m.msgCoding = cmd[64]
	m.msgLength = bytesToIntBig(cmd[65:69])
	if m.msgLength+69+8 != len(cmd) {
		return false
	}
	m.msgContent = make([]byte, m.msgLength)
	copy(m.msgContent, cmd[69:69+m.msgLength])
	copy(m.reserve[:], cmd[69+m.msgLength:])
	return true
}

func (m *deliver) Process(connStatus *int) respPacketer {
	var resp deliverresp
	if *connStatus != CONN_STATUS_BIND {
		resp.result = resp_code_para_err
		resp.SetHead(20+1+8, 0x80000004, m.sequence)
		return &resp
	} else {
		resp.result = resp_code_ok
		resp.SetHead(20+1+8, 0x80000004, m.sequence)
	}

	// prepare to callback
	v := url.Values{}
	v.Set("userNumber", m.userNumber)
	v.Set("spNumber", m.spNumber)
	v.Set("tppid", fmt.Sprintf("%02X", m.tppid))
	v.Set("tpudhi", fmt.Sprintf("%02X", m.tpudhi))
	v.Set("msgCoding", fmt.Sprintf("%02X", m.msgCoding))
	v.Set("msgContent", bytesToHexString(m.msgContent))
	v.Set("reserve", bytesToHexString(m.reserve[:]))
	u, _ := url.Parse(sgipConfig.DeliverCallbackUrl)
	u.RawQuery = v.Encode()

	// callback
	go doCallback(u)

	return &resp
}

func (m *deliver) String() string {
	buf := bytes.NewBufferString("Deliver: ")
	buf.WriteString(m.messageHead.String())
	buf.WriteString(";")
	buf.WriteString(fmt.Sprintf("UserNumber:%s;", m.userNumber))
	buf.WriteString(fmt.Sprintf("spNumber:%s;", m.spNumber))
	buf.WriteString(fmt.Sprintf("tppid:%02X;", m.tppid))
	buf.WriteString(fmt.Sprintf("tpudhi:%02X;", m.tpudhi))
	buf.WriteString(fmt.Sprintf("Message Coding:%02X;", m.msgCoding))
	buf.WriteString(fmt.Sprintf("Message Length:%d;", m.msgLength))
	buf.WriteString(fmt.Sprintf("Message Content:%s;", bytesToHexString(m.msgContent)))
	buf.WriteString(m.messageReserved.String())

	return buf.String()
}

func (r *deliverresp) Encode(buf []byte) int {
	length := 20 + 1 + 8
	if len(buf) < length {
		return -1
	}

	r.messageHead.Encode(buf[0:20])
	buf[20] = r.result
	r.messageReserved.Encode(buf[21:])
	return length
}

func (r *deliverresp) Decode(cmd []byte) bool {
	r.DecodeHead(cmd[0:20])
	r.result = cmd[20]
	copy(r.reserve[:], cmd[21:29])
	return true
}

func (r *deliverresp) String() string {
	buf := bytes.NewBufferString("Deliver_Resp: ")
	buf.WriteString(r.messageHead.String())
	buf.WriteString(";")
	buf.WriteString(fmt.Sprintf("result:%02X;", r.result))
	buf.WriteString(r.messageReserved.String())

	return buf.String()
}

func (m *report) JudgeCommandHead(cmdType, cmdLen int) bool {
	if cmdType != 5 || cmdLen != 20+44 {
		return false
	}
	return true
}

func (m *report) Decode(cmd []byte) bool {
	m.DecodeHead(cmd[0:20])
	copy(m.submitSeq[:], cmd[20:32])
	m.reportType = cmd[32]
	m.userNumber = decodeBytesString(cmd[33:54])
	m.state = cmd[54]
	m.errorCode = cmd[55]
	copy(m.reserve[:], cmd[56:])
	return true
}

func (m *report) Process(connStatus *int) respPacketer {
	var resp reportresp
	if *connStatus != CONN_STATUS_BIND {
		resp.result = resp_code_para_err
		resp.SetHead(20+1+8, 0x80000005, m.sequence)
		return &resp
	} else {
		resp.result = resp_code_ok
		resp.SetHead(20+1+8, 0x80000005, m.sequence)
	}

	// prepare to callback
	v := url.Values{}
	v.Set("submitSeq", bytesToHexString(m.submitSeq[:]))
	v.Set("reportType", fmt.Sprintf("%02X", m.reportType))
	v.Set("userNumber", m.userNumber)
	v.Set("state", fmt.Sprintf("%02X", m.state))
	v.Set("errorCode", fmt.Sprintf("%02X", m.errorCode))
	u, _ := url.Parse(sgipConfig.DeliverCallbackUrl)
	u.RawQuery = v.Encode()

	// callback
	go doCallback(u)

	return &resp
}

func (m *report) String() string {
	buf := bytes.NewBufferString("Report: ")
	buf.WriteString(m.messageHead.String())
	buf.WriteString(";")
	buf.WriteString(fmt.Sprintf("Submit Sequence:%s;", bytesToHexString(m.submitSeq[:])))
	buf.WriteString(fmt.Sprintf("Report Type:%02X;", m.reportType))
	buf.WriteString(fmt.Sprintf("User Number:%s;", m.userNumber))
	buf.WriteString(fmt.Sprintf("State:%02X;", m.state))
	buf.WriteString(fmt.Sprintf("Error Code:%02X;", m.errorCode))
	buf.WriteString(m.messageReserved.String())

	return buf.String()
}

func (r *reportresp) Encode(buf []byte) int {
	length := 20 + 1 + 8
	if len(buf) < length {
		return -1
	}

	r.messageHead.Encode(buf[0:20])
	buf[20] = r.result
	r.messageReserved.Encode(buf[21:])
	return length
}

func (r *reportresp) Decode(cmd []byte) bool {
	r.DecodeHead(cmd[0:20])
	r.result = cmd[20]
	copy(r.reserve[:], cmd[21:29])
	return true
}

func (r *reportresp) String() string {
	buf := bytes.NewBufferString("Report_Resp: ")
	buf.WriteString(r.messageHead.String())
	buf.WriteString(";")
	buf.WriteString(fmt.Sprintf("result:%02X;", r.result))
	buf.WriteString(r.messageReserved.String())

	return buf.String()
}

func newSubmit(input *submitInput) *submit {
	var s submit

	s.length = 20 + 21 + 21 + 1 + 21*len(input.userNumber) + 5 + 10 + 1 + 6 + 6 + 1 + 1 + 1 + 16 + 16 + 1 + 1 + 1 + 1 + 1 + 4 + len(input.msgContent) + 8
	s.cmdType = 3
	seq := getNewSequence()
	copy(s.sequence[:], seq[:])

	s.submitInput = *input

	return &s
}

func (s *submit) Encode(buf []byte) (int, error) {
	if len(buf) < s.length {
		return 0, fmt.Errorf("submit buffer is too small! need %d bytes", s.length)
	}

	s.messageHead.Encode(buf[0:20])
	encodeStringBytes(s.spNumber, buf[20:41])
	encodeStringBytes(s.chargeNumber, buf[41:62])
	buf[62] = byte(len(s.userNumber))
	index := 63
	for i := 0; i < len(s.userNumber); i++ {
		encodeStringBytes(s.userNumber[i], buf[index:index+21])
		index += 21
	}
	encodeStringBytes(s.corpId, buf[index:index+5])
	index += 5
	encodeStringBytes(s.serviceType, buf[index:index+10])
	index += 10
	buf[index] = s.feeType
	index++
	encodeStringBytes(s.feeValue, buf[index:index+6])
	index += 6
	encodeStringBytes(s.givenValue, buf[index:index+6])
	index += 6
	buf[index] = s.agentFlag
	index++
	buf[index] = s.mtFlag
	index++
	buf[index] = s.priority
	index++
	encodeStringBytes(s.expireTime, buf[index:index+16])
	index += 16
	encodeStringBytes(s.scheduleTime, buf[index:index+16])
	index += 16
	buf[index] = s.reportFlag
	index++
	buf[index] = s.tppid
	index++
	buf[index] = s.tpudhi
	index++
	buf[index] = s.msgCoding
	index++
	buf[index] = 0
	index++
	intToBytesBig(len(s.msgContent), buf[index:])
	index += 4
	copy(buf[index:index+len(s.msgContent)], s.msgContent)
	index += len(s.msgContent)
	copy(buf[index:index+8], s.reserve)
	index += 8
	return index, nil
}

func (h *messageHead) DecodeHead(cmd []byte) {
	h.length = bytesToIntBig(cmd[0:4])
	h.cmdType = bytesToIntBig(cmd[4:8])
	for i := 0; i < 3; i++ {
		h.sequence[i] = uint32(bytesToIntBig(cmd[8+i*4 : 8+(i+1)*4]))
	}
}

func (h *messageHead) SetHead(length int, cmdType int, seq [3]uint32) {
	h.length = length
	h.cmdType = cmdType
	copy(h.sequence[:], seq[:])
}

func (h *messageHead) String() string {
	return fmt.Sprintf("length:%08X, type:%08X, seq:%08X %08X %08X", h.length, h.cmdType, h.sequence[0], h.sequence[1], h.sequence[2])
}

func (h *messageHead) Encode(buf []byte) {
	intToBytesBig(h.length, buf[0:4])
	intToBytesBig(h.cmdType, buf[4:8])
	for i := 0; i < 3; i++ {
		intToBytesBig(int(h.sequence[i]), buf[8+i*4:])
	}
}

func (r *messageReserved) String() string {
	return fmt.Sprintf("reserve:%02X%02X%02X%02X%02X%02X%02X%02X", r.reserve[0], r.reserve[1], r.reserve[2], r.reserve[3], r.reserve[4], r.reserve[5], r.reserve[6], r.reserve[7])
}

func (r *messageReserved) Encode(buf []byte) {
	for i := 0; i < 8; i++ {
		buf[i] = 0
	}
}

func processRcvCommandHead(cmdType, cmdLen int) packeter {
	var pack packeter
	switch cmdType {
	case 1:
		pack = new(bind)
	case 2:
		pack = new(unbind)
	case 4:
		pack = new(deliver)
	case 5:
		pack = new(report)
	default:
		return nil
	}

	if pack.JudgeCommandHead(cmdType, cmdLen) == false {
		return nil
	}

	return pack
}

func doCallback(u *url.URL) {
	s := u.String()
	sgipConfig.Logger.Infof("callback:%s", s)
	resp, err := http.Get(s)
	if err != nil {
		sgipConfig.Logger.Errorf("callback error :%s", err.Error())
	} else {
		defer resp.Body.Close()
		var b []byte
		if b, err = ioutil.ReadAll(resp.Body); err != nil {
			sgipConfig.Logger.Errorf("callback response read error:%s", err.Error())
		} else {
			sgipConfig.Logger.Infof("callback response :%s", string(b))
		}
	}
}
