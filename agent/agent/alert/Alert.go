package alert

import (
	"fmt"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

var LastHitMapVerEvent int64
var LastHitMapHorizEvent int64

func HitMapVertical(percent int32, level byte) *pack.EventPack {
	conf := config.GetConfig()
	if dateutil.SystemNow() < LastHitMapVerEvent+int64(conf.HitMapVerEventInterval) {
		return nil
	}
	ep := pack.NewEventPack()
	// fatal Alarm 발생
	ep.Level = level
	ep.Title = "HITMAP_VERTICAL"
	ep.Message = fmt.Sprintf("HitMap vertical %d ", percent)
	ep.Attr.Put("percent", fmt.Sprintf("%d", level))

	return ep
}

func HitMapHorizontal(hitmapTime int) *pack.EventPack {
	conf := config.GetConfig()
	if dateutil.SystemNow() < LastHitMapHorizEvent+int64(conf.HitMapHorizEventInterval) {
		return nil
	}
	ep := pack.NewEventPack()
	// fatal Alarm 발생
	ep.Level = pack.FATAL
	ep.Title = "HITMAP_HORIZONTAL"
	ep.Message = fmt.Sprintf("response time is %d ms ", conf.HitMapHorizEventDuration)
	ep.Attr.Put("hitmap_time", fmt.Sprintf("%d", hitmapTime))

	return ep
}
