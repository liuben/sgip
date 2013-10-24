package sgip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	SUBMIT_OK  = 0
	SUBMIT_ERR = 1
)

type submitResponse struct {
	Result   int    `json:"result"`
	Sequence string `json:"sequence"`
}

type submitInput struct {
	spNumber     string
	chargeNumber string
	userNumber   []string
	corpId       string
	serviceType  string
	feeType      byte
	feeValue     string
	givenValue   string
	agentFlag    byte
	mtFlag       byte
	priority     byte
	expireTime   string
	scheduleTime string
	reportFlag   byte
	tppid        byte
	tpudhi       byte
	msgCoding    byte
	msgContent   []byte
	reserve      []byte
}

func startWebServer() {
	http.HandleFunc("/submit", submitHandler)

	port := fmt.Sprintf(":%d", sgipConfig.SpWebListenPort)

	if err := http.ListenAndServe(port, nil); err != nil {
		sgipConfig.Logger.Errorf("web listen error :%s", err.Error())
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	sgipConfig.Logger.Infof("get submit request: %s", r.URL.String())

	var result submitResponse
	// check ip
	if checkClientIp(r.RemoteAddr, sgipConfig.SpAppIp) == false {
		result.Result = SUBMIT_ERR
		result.Sequence = ""
		res, _ := json.Marshal(result)
		fmt.Fprintf(w, string(res))
		return
	}

	// get input
	r.ParseForm()
	input := parseSubmit(&r.Form)
	if input == nil {
		sgipConfig.Logger.Warn("submit request is invalid")
		result.Result = SUBMIT_ERR
		result.Sequence = ""
		res, _ := json.Marshal(result)
		fmt.Fprintf(w, string(res))
		return
	}

	// send message to channel
	rc := make(chan string)
	msg := submitMessage{*input, rc}
	sgipConfig.Logger.Debug("web goroutine prepares to send msg to submitChan channel")
	submitChan <- msg
	sgipConfig.Logger.Debug("web goroutine prepares to wait the response from submitChan.Sequence channel")
	result.Sequence = <-rc
	sgipConfig.Logger.Debug("web goroutine gets the response from submitChan.Sequence channel")
	if len(result.Sequence) != 24 {
		result.Result = SUBMIT_ERR
	} else {
		result.Result = SUBMIT_OK
	}

	// return the response
	res, _ := json.Marshal(result)
	fmt.Fprintf(w, string(res))
}

func parseSubmit(form *url.Values) *submitInput {
	var s submitInput

	if str := form.Get("spNumber"); str != "" {
		s.spNumber = str
	} else {
		sgipConfig.Logger.Warn("no spNumber")
		return nil
	}

	s.chargeNumber = form.Get("chargeNumber")

	s.userNumber = make([]string, 0, 5)
	if uns, ok := (*form)["userNumber"]; ok == false {
		sgipConfig.Logger.Warn("no userNumber")
		return nil
	} else {
		for _, un := range uns {
			s.userNumber = append(s.userNumber, un)
		}
	}

	if str := form.Get("corpId"); str != "" {
		s.corpId = str
	} else {
		sgipConfig.Logger.Warn("no corpId")
		return nil
	}

	if str := form.Get("serviceType"); str != "" {
		s.serviceType = str
	} else {
		sgipConfig.Logger.Warn("no serviceType")
		return nil
	}

	if str := form.Get("feeType"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("feeType format is error")
			return nil
		}
		s.feeType = byte(v)
	} else {
		sgipConfig.Logger.Warn("no serviceType")
		return nil
	}

	if str := form.Get("feeValue"); str != "" {
		s.feeValue = str
	} else {
		sgipConfig.Logger.Warn("no feeValue")
		return nil
	}

	if str := form.Get("givenValue"); str != "" {
		s.givenValue = str
	} else {
		sgipConfig.Logger.Warn("no givenValue")
		return nil
	}

	if str := form.Get("agentFlag"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("agentFlag format is error")
			return nil
		}
		s.agentFlag = byte(v)
	} else {
		sgipConfig.Logger.Warn("no agentFlag")
		return nil
	}

	if str := form.Get("mtFlag"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("mtFlag format is error")
			return nil
		}
		s.mtFlag = byte(v)
	} else {
		sgipConfig.Logger.Warn("no mtFlag")
		return nil
	}

	if str := form.Get("priority"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("priority format is error")
			return nil
		}
		s.priority = byte(v)
	} else {
		sgipConfig.Logger.Warn("no priority")
		return nil
	}

	if str := form.Get("expireTime"); str != "" {
		s.expireTime = str
	} else {
		sgipConfig.Logger.Warn("no expireTime")
		return nil
	}

	if str := form.Get("scheduleTime"); str != "" {
		s.scheduleTime = str
	} else {
		sgipConfig.Logger.Warn("no scheduleTime")
		return nil
	}

	if str := form.Get("reportFlag"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("reportFlag format is error")
			return nil
		}
		s.reportFlag = byte(v)
	} else {
		sgipConfig.Logger.Warn("no reportFlag")
		return nil
	}

	if str := form.Get("tppid"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("tppid format is error")
			return nil
		}
		s.tppid = byte(v)
	} else {
		sgipConfig.Logger.Warn("no tppid")
		return nil
	}

	if str := form.Get("tpudhi"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("tpudhi format is error")
			return nil
		}
		s.tpudhi = byte(v)
	} else {
		sgipConfig.Logger.Warn("no tpudhi")
		return nil
	}

	if str := form.Get("msgCoding"); str != "" {
		v, err := strconv.ParseUint(str, 16, 8)
		if err != nil {
			sgipConfig.Logger.Warn("msgCoding format is error")
			return nil
		}
		s.msgCoding = byte(v)
	} else {
		sgipConfig.Logger.Warn("no msgCoding")
		return nil
	}

	if str := form.Get("msgContent"); str != "" {
		var err error
		s.msgContent, err = hexStringToBytes(str)
		if err != nil {
			sgipConfig.Logger.Warn("msgContent format is error")
			return nil
		}
	} else {
		sgipConfig.Logger.Warn("no msgContent")
		return nil
	}

	if str := form.Get("reserve"); str != "" {
		var err error
		s.reserve, err = hexStringToBytes(str)
		if err != nil || len(s.reserve) != 8 {
			sgipConfig.Logger.Warn("reserve format is error")
			return nil
		}
	} else {
		sgipConfig.Logger.Warn("no reserve")
		return nil
	}

	return &s
}
