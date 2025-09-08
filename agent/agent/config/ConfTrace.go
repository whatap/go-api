package config

import (
	// "strings"

	"github.com/whatap/golib/util/hmap"
)

type ConfTrace struct {
	TraceZipEnabled       bool
	TraceZipQueueSize     int
	TraceZipMaxWaitTime   int
	TraceZipMaxWait2Time  int
	TraceZipMaxBufferSize int
	TraceZipMinSize       int
	TraceTxSplitQueueSize int

	TraceStepNormalCount int
	TraceStepHeavyCount  int
	TraceStepMaxCount    int
	TraceStepHeavyTime   int

	CircularProfileEnabled      bool
	LargeProfileEnabled         bool
	SplitProfileEnabled         bool
	InternalTraceCollectingMode int

	DebugTraceZipEnabled  bool
	DebugTraceZipInterval int

	TxTextTxnameEnabled bool
	TxTextErrorEnabled  bool
	TxTextErrorLength   int

	TraceStatusErrorEnable bool
	TraceStatusErrorMode   int32
	StatusIgnore           *hmap.IntSet
	StatusIgnoreSet        *hmap.IntKeyLinkedMap
	StatusAlertIgnore      *hmap.IntSet
	StatusAlertIgnoreSet   *hmap.IntKeyLinkedMap

	HttpStatusErrorEnabled bool
	HttpStatusErrorMode    int32
	HttpcStatusIgnore      *hmap.IntSet
	HttpcStatusAlertIgnore *hmap.IntSet

	HttpcStatusURLIgnoreSet         *hmap.IntKeyLinkedMap
	HasHttpcStatusURLIgnoreSet      bool
	HttpcStatusURLAlertIgnoreSet    *hmap.IntKeyLinkedMap
	HasHttpcStatusURLAlertIgnoreSet bool
	HttpcStatusHostIgnoreSet        *hmap.IntKeyLinkedMap
	HasHttpcStatusHostIgnoreSet     bool
}

func (this *ConfTrace) Apply(conf *Config) {

	this.TraceZipEnabled = GetBoolean("profile_zip_enabled", true)
	this.TraceZipQueueSize = int(GetInt("profile_zip_queue_size", 1000))
	this.TraceZipMaxWaitTime = int(GetInt("profile_zip_max_wait_time", 1000))
	this.TraceZipMaxWait2Time = int(GetInt("profile_zip_max_wait2_time", 5000))
	this.TraceZipMaxBufferSize = int(GetInt("profile_zip_max_buffer_size", 1024*1024))
	this.TraceZipMinSize = int(GetInt("profile_zip_min_size", 100))
	this.TraceTxSplitQueueSize = int(GetInt("trace_txsplit_queue_size", 1000))

	this.TraceStepNormalCount = int(getInt("profile_step_normal_count", 800))
	this.TraceStepHeavyCount = int(getInt("profile_step_heavy_count", 1000))
	this.TraceStepMaxCount = int(getInt("profile_step_max_count", 1024))
	this.TraceStepHeavyTime = int(getInt("profile_step_heavy_time", 100))

	if this.TraceStepHeavyCount > this.TraceStepMaxCount {
		this.TraceStepHeavyCount = this.TraceStepMaxCount
	}
	if this.TraceStepNormalCount > this.TraceStepHeavyCount {
		this.TraceStepNormalCount = this.TraceStepHeavyCount
	}

	this.DebugTraceZipEnabled = GetBoolean("debug_profile_zip_enabled", false)
	this.DebugTraceZipInterval = int(GetInt("debug_profile_zip_interval", 5000))

	this.CircularProfileEnabled = getBoolean("circular_profile_enabled", false)
	this.LargeProfileEnabled = getBoolean("large_profile_enabled", false)
	this.SplitProfileEnabled = getBoolean("split_profile_enabled", false)

	if this.SplitProfileEnabled {
		this.InternalTraceCollectingMode = 4
	} else if this.LargeProfileEnabled == true {
		this.InternalTraceCollectingMode = 3 // huge profile
	} else if this.CircularProfileEnabled == true {
		this.InternalTraceCollectingMode = 2 // circle profile
	} else {
		this.InternalTraceCollectingMode = 1 // normal profile
	}

	this.TxTextTxnameEnabled = getBoolean("txtext_txname_enabled", true)
	this.TxTextErrorEnabled = getBoolean("txtext_error_enabled", true)
	this.TxTextErrorLength = int(getInt("txtext_error_length", 128))
	if this.TxTextErrorLength < 1 {
		this.TxTextErrorEnabled = false
	}

	this.TraceStatusErrorEnable = getBoolean("transaction_status_error_enable", false)
	// mode1 := getValue("transaction_status_mode", "")
	// if "info" == strings.ToLower(mode1) || "2" == strings.ToLower(mode1) {
	// 	this.transaction_status_error_mode = StatusErrorMode.TX_STATUS_INFO
	// } else {
	// 	this.transaction_status_error_mode = StatusErrorMode.TX_STATUS_NORMAL
	// }

	this.StatusIgnore = GetIntSet("status_ignore", "", ",")
	// this.StatusIgnoreSet = GetIntSet("status_ignore_set", "", ",") //load_status_ignore_set("status_ignore_set", this.status_ignore_set)
	// this.StatusAlertIgnore = GetIntSet("status_alert_ignore", "", ",")
	// this.StatusAlertIgnoreSet = GetIntSet("status_alert_ignore", "", ",") //load_status_ignore_set("status_alert_ignore_set", this.status_alert_ignore_set);

	// this.HttpStatusErrorEnabled = getBoolean("httpc_status_error_enable", true)
	// mode2: = getValue("httpc_status_mode", "");
	// if "info" == strings.ToLower(mode2)|| "1" == strings.ToLower(mode2)  {
	// 	this.httpc_status_error_mode = HttpCallStatusErrorMode.HTTPC_STATUS_INFO;
	// } else {
	// 	this.httpc_status_error_mode = HttpCallStatusErrorMode.HTTPC_STATUS_NORMAL;
	// }

	// this.HttpcStatusIgnore = getIntSet("httpc_status_ignore", "", ",")
	// this.HttpcStatusAlertIgnore = getIntSet("httpc_status_alert_ignore", "", ",")

	// this.HttpcStatusURLIgnoreSet = load_status_ignore_set("httpc_status_url_ignore_set", this.httpc_status_url_ignore_set)
	// this._has_httpc_status_url_ignore_set = this.httpc_status_url_ignore_set.size() > 0;

	// this.httpc_status_url_alert_ignore_set = load_status_ignore_set("httpc_status_url_alert_ignore_set", this.httpc_status_url_alert_ignore_set);
	// this._has_httpc_status_url_alert_ignore_set = this.httpc_status_url_alert_ignore_set.size() > 0;

	// this.httpc_status_host_ignore_set = load_status_ignore_set("httpc_status_host_ignore_set", this.httpc_status_host_ignore_set);
	// this._has_httpc_status_host_ignore_set = this.httpc_status_host_ignore_set.size() > 0;

}
