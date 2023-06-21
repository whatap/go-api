package watch

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	logsink_zip "github.com/whatap/go-api/agent/logsink/zip"
	"github.com/whatap/go-api/agent/util/logutil"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

type WatchLog struct {
	Id        string
	Activated bool

	file     *os.File
	FileName string
	FileInfo os.FileInfo
	FilePos  int64

	Words         []string
	CheckInterval int

	LastCheckTime int64

	encoding         string
	logsendThreshold int32

	trxLogFound bool
}

func NewWatchLog(id string) *WatchLog {
	p := new(WatchLog)
	p.Id = id
	p.Words = make([]string, 0)
	p.logsendThreshold = LogSendThreshold
	return p
}

func (wl *WatchLog) Config(id string, fileName string) {

	wl.FileName = fileName
	wl.file = nil
	if fi, err := os.Stat(fileName); err == nil {
		wl.FileInfo = fi
	} else {
		wl.FileInfo = nil
		return
	}
}

func (wl *WatchLog) Process() {
	if wl.trxLogFound {
		wl.processMultilineLogs()
	} else {
		wl.process()
	}

}

func (wl *WatchLog) processMultilineLogs() (err error) {

	fi, statErr := os.Stat(wl.FileName)
	if statErr != nil {
		err = statErr
		return
	}

	fileposthistime := fi.Size()

	if fileposthistime == wl.FilePos {
		return
	} else if fileposthistime < wl.FilePos {
		wl.FilePos = 0
	}

	f, fileErr := os.Open(wl.FileName)
	if fileErr != nil {
		err = fileErr
		return
	}

	defer f.Close()

	_, seekErr := f.Seek(wl.FilePos, io.SeekStart)
	if seekErr != nil {
		err = seekErr
		return
	}

	var scanner bufio.Scanner

	if wl.encoding == "euc-kr" {
		dec := transform.NewReader(f, korean.EUCKR.NewDecoder())
		scanner = *bufio.NewScanner(dec)
	} else {
		scanner = *bufio.NewScanner(f)
	}

	scanner.Split(bufio.ScanLines)

	multilinebuffer := bytes.Buffer{}
	var loopCondition = true
	lineCount := int32(0)
	for loopCondition {
		select {
		case <-time.After(time.Second * 1):
			loopCondition = false
		default:
			if !scanner.Scan() {
				loopCondition = false
				break
			}
			line := scanner.Text()
			lineCount += 1
			if validateTxHeader(line) || lineCount > LogSendThreshold {
				if multilinebuffer.Len() > SEND_THRESHOLD {
					wl.parseAndSend([]string{multilinebuffer.String()})
					multilinebuffer.Reset()
					lineCount = 0
				}
			}
			if multilinebuffer.Len() > 0 {
				multilinebuffer.WriteString(NEWLINE)
			}
			multilinebuffer.WriteString(line)
		}
	}

	// fmt.Println("<-----------------------loop-exit----------------------->")
	if multilinebuffer.Len() > 0 {
		wl.parseAndSend([]string{multilinebuffer.String()})

		multilinebuffer.Reset()
	}

	pos, seekErr := f.Seek(0, os.SEEK_CUR)
	err = seekErr
	if seekErr == nil {
		wl.FilePos = pos
	}

	return
}

func (wl *WatchLog) process() {
	defer func() {
		// 열린파일이 닫히는 것을 보장함 defer
		wl.file.Close()
		wl.file = nil
		if r := recover(); r != nil {
			logutil.Println("WA-LOGS-001", "Recover Process ", r, ",stack=", string(debug.Stack()))
		}
	}()

	//	if wl.FileInfo == nil {
	//		wl.FilePos = -1
	//		return
	//	}
	// 다시 파일 정보 확인
	if fi, err := os.Stat(wl.FileName); err == nil {
		wl.FileInfo = fi
	} else {
		wl.FileInfo = nil
		wl.FilePos = -1
		logutil.Println("WA-LOGS-002", "File not found ", wl.FileName, ", Error ", err)
	}

	if wl.FilePos < 0 {
		if wl.FileInfo != nil {
			wl.FilePos = wl.FileInfo.Size()
		}
		return
	}

	now := dateutil.SystemNow()
	if now < wl.LastCheckTime+int64(wl.CheckInterval) {
		return
	}
	wl.LastCheckTime = dateutil.SystemNow()

	if wl.FilePos > wl.FileInfo.Size() {
		wl.FilePos = wl.FileInfo.Size()
		return
	}

	if f, err := os.Open(wl.FileName); err == nil {
		wl.file = f
		// os.Stat 은 변화하는 파일 용량을 못 가져옴. Open 후 Stat 으로 가져와야 함
		if fi, err := f.Stat(); err == nil {
			wl.FileInfo = fi
		} else {
			wl.FileInfo = nil
			wl.FilePos = -1
			return
		}
	} else {
		logutil.Println("WA-LOGS-003", "Open Error ", ",err=", err)
	}

	fileLength := wl.FileInfo.Size()

	//conf := config.GetConfig()
	ConfLogSink := config.GetConfig().ConfLogSink

	for readCnt := 0; readCnt < int(ConfLogSink.WatchLogReadCount); readCnt++ {
		if wl.FilePos >= fileLength {
			return
		}

		//		if pos, err := f.Seek(wl.FilePos, os.SEEK_SET); err != nil {
		if _, err := wl.file.Seek(wl.FilePos, os.SEEK_SET); err != nil {
			logutil.Println("WA-LOGS-004", "Error SEEK_SET ", wl.FilePos, ",err=", err)
			return
		}
		// Read
		lines := wl.read(int(ConfLogSink.WatchLogLineSize))

		if cur, err := wl.file.Seek(0, os.SEEK_CUR); err != nil {
			logutil.Println("WA-LOGS-005", "Error SEEK_CUR ", err)
			return
		} else {
			wl.FilePos = cur
		}

		if len(lines) == 0 {
			return
		}

		wl.parseAndSend(lines)

		//		match := 0
		//		if wl.parseAndSend(lines) {
		//			match += 1
		//			if ConfLogSink.WatchLogSendCount > 0 && match >= int(ConfLogSink.WatchLogSendCount) {
		//				wl.FilePos = fileLength
		//				return
		//			}
		//		}
	}
}

func (wl *WatchLog) parseAndSend(buffer []string) bool {
	rt := false
	for _, line := range buffer {
		if len(wl.Words) > 0 {
			for _, word := range wl.Words {
				if strings.Index(line, word) > -1 {
					wl.send(word, wl, line)
					rt = true
				}
			}
		} else {
			wl.send("", wl, line)

		}
	}
	return rt
}

func (wl *WatchLog) read(lineLimit int) []string {
	//sT := dateutil.SystemNow()
	scanner := bufio.NewScanner(wl.file)
	scanner.Split(bufio.ScanLines)
	lineCount := 0
	result := make([]string, 0)

	for scanner.Scan() && lineCount < lineLimit {
		line := strings.TrimSpace(scanner.Text())
		result = append(result, line)
		lineCount++
	}
	//if wl.Debug {
	//logutil.Infoln("read elpased=", dateutil.SystemNow()-sT, ",len=", len(result))
	//}

	return result
}

func (wl *WatchLog) send(word string, wlog *WatchLog, line string) {
	p := pack.NewLogSinkPack()
	p.Time = dateutil.Now()
	p.Category = filepath.Base(wlog.Id)
	p.Tags.PutString("file", wlog.FileName)
	// Java ONAME, OKIND, ONODE
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()

	if secu.ONAME != "" {
		p.Tags.PutString("oname", secu.ONAME)
	}
	if conf.OKIND != 0 {
		p.Tags.PutString("okindName", conf.OKIND_NAME)
	}
	if conf.ONODE != 0 {
		p.Tags.PutString("onodeName", conf.ONODE_NAME)
	}

	wl.trxLogFound = ApplyAppLog(p, line) || wl.trxLogFound

	p.Content = line
	p.Line = wlog.FileInfo.Size()

	ConfLogSink := config.GetConfig().ConfLogSink
	if ConfLogSink.LogSinkZipEnabled {
		logsink_zip.GetInstance().Add(p)
	} else {
		data.SendFlush(p, false)
	}
}

func (wl *WatchLog) Activate() {
	if wl.Activated == false {
		if wl.FileInfo != nil {
			wl.FilePos = wl.FileInfo.Size()
		} else {
			wl.FilePos = -1
		}
	}
	wl.Activated = true
}

func (wl *WatchLog) Stop() {
	wl.Activated = false
}

func (wl *WatchLog) IsActive() bool {
	return wl.Activated
}

func (wl *WatchLog) String() string {
	return wl.ToString()
}
func (wl *WatchLog) ToString() string {

	return fmt.Sprintln("WatchLog [id=", wl.Id, ", activaqted=", wl.Activated, ", file=", wl.file,
		", file_pos=", wl.FilePos, ", words=", wl.Words, "]")
}

func (wl *WatchLog) Reset() {
	wl.FilePos = -1
}
