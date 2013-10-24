package sgip

import (
	"github.com/cihub/seelog"
)

// Configure Parameter
type SgipConfig struct {
	// net parameter
	SgpIp              string
	SgpPort            int
	SpTcpListenPort    int // listen for SGP connection
	SpWebListenPort    int // listen for Submit request
	ReportCallbackUrl  string
	DeliverCallbackUrl string
	ReadTimeoutSecond  int
	WriteTimeoutSecond int
	SpAppIp            string // it defines which ip can request a submit

	// logger
	Logger seelog.LoggerInterface

	// goroutine parameter
	TcpClientCount   int // how many goroutines to send submit to SGP
	SubmitQueueDepth int

	// SGIP parameter
	AreaPhoneNo   uint32
	CorpId        uint32
	LoginUserName string
	LoginPassword string
}

var sgipConfig SgipConfig

// init the SGIP config para
func Init(config *SgipConfig) {
	sgipConfig = *config
}

// start sgip server.
// when this function return, the server is stop
func Start() {
	sgipConfig.Logger.Debug("sgip server start")
	startTcpServer()
	startWebServer()
	sgipConfig.Logger.Debug("sgip server stop")
}
