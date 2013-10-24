package sgip

import (
	"strconv"
	"sync"
	"time"
)

type msgSequence [3]uint32

var sequenceCounter uint32 = 0
var counterLock sync.Mutex

func getNewSequence() msgSequence {
	var seq msgSequence

	areaNo := sgipConfig.AreaPhoneNo
	if areaNo < 100 {
		areaNo *= 10
	}

	seq[0] = 3*1000000000 + areaNo*100000 + sgipConfig.CorpId
	t, _ := strconv.ParseUint(time.Now().Format("0102150405"), 10, 32)
	seq[1] = uint32(t)
	counterLock.Lock()
	seq[2] = sequenceCounter
	sequenceCounter++
	counterLock.Unlock()

	return seq
}

func (m msgSequence) fill(buf []byte) {
	if len(buf) != 12 {
		return
	}

	for i := 0; i < 3; i++ {
		intToBytesBig(int(m[i]), buf[i*4:i*4+4])
	}
}
