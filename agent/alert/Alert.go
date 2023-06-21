package alert

import (
	"github.com/whatap/golib/lang/pack"
)

func Alert(title, message string, level byte, attr map[string]string) *pack.EventPack {
	ep := pack.NewEventPack()
	// fatal Alarm 발생
	ep.Level = level
	ep.Title = title
	ep.Message = message

	for k, v := range attr {
		ep.Attr.Put(k, v)
	}

	return ep
}

//func HitMapVertical(percent int32, level byte) *pack.EventPack {
//	if dateutil.SystemNow() < LastHitMapVerEvent+int64(conf.HitMapVerEventInterval) {
//		return nil
//	}
//	ep := pack.NewEventPack()
//	// fatal Alarm 발생
//	ep.Level = level
//	ep.Title = "HITMAP_VERTICAL"
//	ep.Message = fmt.Sprintf("HitMap vertical %d %", percent)
//	ep.Attr.Put("percent", fmt.Sprintf("%s", level))
//
//	return ep
//}
//
//func HitMapHorizontal(hitmapTime int) *pack.EventPack {
//	if dateutil.SystemNow() < LastHitMapHorizEvent+int64(conf.HitMapHorizEventInterval) {
//		return nil
//	}
//	ep := pack.NewEventPack()
//	// fatal Alarm 발생
//	ep.Level = pack.FATAL
//	ep.Title = "HITMAP_HORIZONTAL"
//	ep.Message = fmt.Sprintf("response time is %d ms ", conf.HitMapHorizEventDuration)
//	ep.Attr.Put("hitmap_time", fmt.Sprintf("%d", hitmapTime))
//
//	return ep
//}
