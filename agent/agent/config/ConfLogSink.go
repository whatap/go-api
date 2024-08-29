package config

import (
	"encoding/json"
	"fmt"
)

type ConfLogSink struct {
	WatchLogEnabled       bool
	WatchLogCheckInterval int64

	WatchLogReadCount  int32
	WatchLogBufferSize int32
	WatchLogLineSize   int32
	WatchLogSendCount  int32

	LogSinkEnabled bool
	//	public static boolean logsink_stdout_enabled = logsink_enabled;
	//	public static boolean logsink_stderr_enabled = logsink_enabled;
	//	public static boolean logsink_logback_enabled = logsink_enabled;
	//	public static boolean logsink_tomcat_enabled = logsink_enabled;
	//	public static boolean logsink_custom_enabled= logsink_enabled;

	LogSinkQueueSize                   int32
	DebugLogSinkEnabled                bool
	LogSinkLineSize                    int32
	LogSinkLimitContentLogSilentTime   int32
	LogSinkLimitContentAlertSilentTime int32
	LogSinkLimitContentEnabled         bool
	DebugLogSinkLimitContentEnabled    bool
	LogsinkLimitContentAlertEnabled    bool
	LogSinkLimitContentLength          int32
	DebugLogSinkLineLimit              int32
	LogSinkZipEnabled                  bool
	LogSinkFiles                       []string
	LogSinkInterval                    int32

	LogSinkMaxBufferSize   int32
	LogSinkMaxWaitTime     int32
	LogSinkZipMinSize      int32
	DebugLogSinkZipEnabled bool
	LogSinkZipLibpath      string

	TxIdTag          string
	AppLogCategory   string
	AppLogPattern    string
	LogSendThreshold int32

	LogSinkStopInterval int64

	LogSinkStdOutEnabled bool
	LogSinkStdErrEnabled bool

	LogSinkCategoryStdErr string
	LogSinkCategoryStdOut string

	LogSinkTags     [][]string
	HasLogSinkTags_ bool
	LogSinkTagsStr  string

	// java
	// public static boolean _zip_real_enabled;
	// public static int logsink_zip_defer_time=30000;

	LogSinkRtEnabled           bool
	LogSinkRtDbcOkEnabled      bool
	LogSinkRtDbcErrorEnabled   bool
	LogSinkRtHttpcOkEnabled    bool
	LogSinkRtHttpcErrorEnabled bool
	LogSinkRtSocketEnabled     bool
	LogSinkRtErrorInterval     int64
	LogSinkRtOkInterval        int64

	LogSinkTraceEnabled         bool
	LogSinkTraceTxidEnabled     bool
	LogSinkTraceMtidEnabled     bool
	LogSinkTraceLoginEnabled    bool
	LogSinkTraceHttpHostEnabled bool

	LogSinkHighSecureEnabled bool
}

func (this *ConfLogSink) Apply(conf *Config) {

	this.LogSinkFiles = GetStringArray("logsink.files", ",")
	this.LogSinkInterval = GetInt("logsink_interval", 1000)

	this.WatchLogCheckInterval = GetLong("watchlog_check_interval", int64(2000))
	this.WatchLogReadCount = GetInt("watchlog_read_count", 4)
	if this.WatchLogReadCount < 1 {
		this.WatchLogReadCount = 1
	}
	this.WatchLogSendCount = GetInt("watchlog_send_count", 0)
	this.WatchLogBufferSize = GetInt("watchlog_buffer_size", 128*1024)
	this.WatchLogLineSize = GetInt("watchlog_line_size", 512)

	this.LogSinkEnabled = GetBoolean("logsink_enabled", false)

	if this.LogSinkEnabled {
		this.WatchLogEnabled = true
	} else {
		this.WatchLogEnabled = GetBoolean("watchlog_enabled", false)
	}

	this.LogSinkQueueSize = GetInt("logsink_queue_size", 1000)
	this.LogSinkLineSize = GetInt("logsink_line_size", 512)
	this.DebugLogSinkEnabled = GetBoolean("debug_logsink_enabled", false)
	this.DebugLogSinkLineLimit = GetInt("debug_logsink_line_limit", 0)
	this.LogSinkZipEnabled = this.LogSinkEnabled && GetBoolean("logsink_zip_enabled", true)

	this.LogSinkMaxBufferSize = GetInt("logsink_max_buffer_size", 1024*64)
	this.LogSinkMaxWaitTime = GetInt("logsink_max_wait_time", 2000)
	this.LogSinkZipMinSize = GetInt("logsink_zip_min_size", 100)
	this.DebugLogSinkZipEnabled = GetBoolean("debug_logsink_zip_enabled", false)
	this.LogSinkZipLibpath = GetValue("logsink_zip_libpath")
	this.TxIdTag = GetValue("logsink_txidtag")
	this.AppLogCategory = GetValue("logsink_applogcategory")
	this.AppLogPattern = GetValue("logsink_applogpattern")
	this.LogSendThreshold = GetInt("logsink_sendthreshold", 500)

	//Interval until end time, default 30 minutes
	this.LogSinkStopInterval = GetLong("logsink_stop_interval", 60000*30)

	this.LogSinkStdOutEnabled = this.LogSinkEnabled && GetBoolean("logsink_stdout_enabled", false)
	this.LogSinkStdErrEnabled = this.LogSinkEnabled && GetBoolean("logsink_stderr_enabled", false)

	this.LogSinkCategoryStdOut = GetValueDef("logsink_category_stdout", "AppStdOut")
	this.LogSinkCategoryStdErr = GetValueDef("logsink_category_stderr", "AppStdErr")

	this.LogSinkLimitContentEnabled = GetBoolean("logsink_limit_content_enabled", false)
	this.LogSinkLimitContentLogSilentTime = GetInt("logsink_limit_content_log_silent_time", 1000)
	this.LogSinkLimitContentAlertSilentTime = GetInt("logsink_limit_content_alert_silent_time", 10000)
	this.LogsinkLimitContentAlertEnabled = GetBoolean("logsink_limit_content_alert_enabled", true)
	this.LogSinkLimitContentLength = GetInt("logsink_limit_content_length", 1000000)
	this.DebugLogSinkLimitContentEnabled = GetBoolean("debug_logsink_limit_content_enabled", true)

	this.parseLogSinkTags(GetValueDef("logsink_tags", ""))

	this.LogSinkRtEnabled = this.LogSinkEnabled && GetBoolean("logsink_rt_enabled", false)
	this.LogSinkRtDbcOkEnabled = this.LogSinkEnabled && GetBoolean("logsink_rt_dbc_ok_enabled", this.LogSinkEnabled)
	this.LogSinkRtDbcErrorEnabled = this.LogSinkEnabled && GetBoolean("logsink_rt_dbc_error_enabled", this.LogSinkEnabled)
	this.LogSinkRtHttpcOkEnabled = this.LogSinkEnabled && GetBoolean("logsink_rt_httpc_ok_enabled", this.LogSinkEnabled)
	this.LogSinkRtHttpcErrorEnabled = this.LogSinkEnabled && GetBoolean("logsink_rt_httpc_error_enabled", this.LogSinkEnabled)
	this.LogSinkRtSocketEnabled = this.LogSinkEnabled && GetBoolean("logsink_rt_socket_enabled", this.LogSinkEnabled)
	this.LogSinkRtErrorInterval = GetLong("logsink_rt_error_interval", 5000)
	this.LogSinkRtOkInterval = GetLong("logsink_rt_ok_interval", 30000)

	this.LogSinkTraceTxidEnabled = GetBoolean("logsink_trace_txid_enabled", true)
	this.LogSinkTraceMtidEnabled = GetBoolean("logsink_trace_mtid_enabled", true)
	this.LogSinkTraceLoginEnabled = GetBoolean("logsink_trace_login_enabled", false)
	this.LogSinkTraceHttpHostEnabled = GetBoolean("logsink_trace_httphost_enabled", false)

	if GetBoolean("logtag_txid_enabled", false) {
		this.LogSinkTraceEnabled = GetBoolean("logsink_trace_enabled", true)
	} else {
		this.LogSinkTraceEnabled = GetBoolean("logsink_trace_enabled", false)
	}

	if GetBoolean("logtag_mtid_enabled", false) {
		this.LogSinkTraceEnabled = GetBoolean("logsink_trace_enabled", true)
	} else {
		this.LogSinkTraceEnabled = GetBoolean("logsink_trace_enabled", false)
	}

	this.LogSinkHighSecureEnabled = GetBoolean("logsink_high_secure_enabled", false)
}

func (this *ConfLogSink) parseLogSinkTags(tags string) {
	if tags == "" || len(tags) == 0 {
		this.HasLogSinkTags_ = false
		this.LogSinkTagsStr = ""
		if len(this.LogSinkTags) > 0 {
			this.LogSinkTags = make([][]string, 0)
		}
		return
	}
	if tags == this.LogSinkTagsStr {
		return
	}
	this.LogSinkTagsStr = tags

	var m map[string]interface{}
	out := make([][]string, 0)
	err := json.Unmarshal([]byte(this.LogSinkTagsStr), m)
	if err != nil {
		return
	}
	for k, v := range m {
		if v != nil {
			out = append(out, []string{k, fmt.Sprintf("%v", v)})
		}
	}
	this.LogSinkTags = out
	this.HasLogSinkTags_ = (len(out) > 0)
}
