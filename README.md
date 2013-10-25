sgip
====

SGIP protocol v1.2 (China Unicom).  It implements a bridge from TCP to HTTP.

SGIP is used to send or receive SMS between SP and SGP. 
SGIP is based on TCP, but i like HTTP more than TCP. So this project implements a bridge from TCP to HTTP.
With this project, all the business logic code in SP would be based on HTTP.

Requirements
------------

* Go 1.1 or higher
* [seelog](https://github.com/cihub/seelog)

Build
-----

You need to create a golang project which has main package, then initialize the sgip package and start the sgip server.

For example:
```go
package main

import (
	"fmt"
	"github.com/cihub/seelog"
	"github.com/liuben/sgip"
	"os"
	"time"
)

func main() {
	// init seelog
	logger, err := seelog.LoggerFromConfigAsFile("/www/sgip/etc/seelog.xml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s -- seelog init error:%s\n", time.Now().String(), err.Error())
		panic("seelog init error")
	}
	seelog.ReplaceLogger(logger)
	defer seelog.Flush()

    // init sgip package
	var sgipConfig sgip.SgipConfig
	sgipConfig.SgpIp = "192.168.1.3"
	sgipConfig.SgpPort = 8881
	sgipConfig.SpTcpListenPort = 8801
	sgipConfig.SpWebListenPort = 8802
    sgipConfig.ReportCallbackUrl = "http://127.0.0.1/report"
    sgipConfig.DeliverCallbackUrl = "http://127.0.0.1/deliver"
	sgipConfig.ReadTimeoutSecond = 60
	sgipConfig.WriteTimeoutSecond = 10
    sgipConfig.SpAppIp = "127.0.0.1"
	sgipConfig.Logger = logger
	sgipConfig.TcpClientCount = 2
	sgipConfig.SubmitQueueDepth = 4
    sgipConfig.AreaPhoneNo = 10
    sgipConfig.CorpId = 12345
	sgipConfig.LoginUserName = "abcde"
	sgipConfig.LoginPassword = "abcde"
	sgip.Init(&sgipConfig)

    // start the sgip server
	sgip.Start()
}
```

Build & Run
```
go build -o sgip
./sgip or use supervisor
```

Usage
-----

When you start the sgip server, it will wait for the SGP connection and business logic request.

### Submit

Business logic can request the url as follow to submit a SMS:
```
http://127.0.0.1:8801/submit?spNumber=123456789&chargeNumber=&userNumber=861381123457&corpId=666666&serviceType=01020304&feeType=00&feeValue=0&givenValue=%00&agentFlag=00&mtFlag=02&priority=00&expireTime=%00&scheduleTime=%00&reportFlag=00&tppid=08&tpudhi=00&msgCoding=00&&msgContent=616263&reserve=0000000000000000
```

These fields use ascii, they could be shorter than SGIP protocol requirement:
* spNumber
* chargeNumber
* userNumber
* corpId
* serviceType
* feeValue
* givenValue
* expireTime
* scheduleTime

Thess fields use number in hex string format, 0F means 15, they should be as long as SGIP protocol requirement exactly:
* feeType
* agentFlag
* mtFlag
* priority
* reportFlag
* tppid
* tpudhi
* msgCoding
* msgContent
* reserve

The response of this HTTP request is in JSON format as
```
{"result":0,"sequence":"0102030405060708090A0B0C"}
```

Sequence is the submit's sequence number, which has 12 bytes. If failed, the result would be 1, and sequence would be empty string.

### Deliver

When sgip server receives a deliver, it will callback the business logic's web service.
```
http://127.0.0.1/deliver?msgCoding=00&msgContent=616263&spNumber=123456789&tppid=00&tpudhi=00&userNumber=8613811234567&reserve=0000000000000000
```

So business logic should implements the callback web service, and initilize it to DeliverCallbackUrl.

### Report

When sgip server receives a report, it will callback the business logic's web service.
```
http://127.0.0.1/report?errorCode=67&reportType=00&state=02&submitSeq=B44EC4FD3D1AEE6600000001&userNumber=8613811234567
```

So business logic should implements the callback web service, and initilize it to ReportCallbackUrl.


