package config

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

	LogSinkQueueSize      int32
	DebugLogSinkEnabled   bool
	LogSinkLineSize       int32
	DebugLogSinkLineLimit int32
	LogSinkZipEnabled     bool
	LogSinkFiles          []string
	LogSinkInterval       int32

	MaxBufferSize          int32
	MaxWaitTime            int32
	LogSinkZipMinSize      int32
	DebugLogSinkZipEnabled bool
	LogSinkZipLibpath      string

	TxIdTag          string
	AppLogCategory   string
	AppLogPattern    string
	LogSendThreshold int32

	LogSinkStopInterval int64
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

	//		logsink_stdout_enabled = conf.getBoolean("logsink_stdout_enabled", logsink_enabled);
	//		logsink_stderr_enabled = conf.getBoolean("logsink_stderr_enabled", logsink_enabled);
	//		logsink_logback_enabled = conf.getBoolean("logsink_logback_enabled", logsink_enabled);
	//		logsink_tomcat_enabled = conf.getBoolean("logsink_tomcat_enabled", logsink_enabled);
	//		logsink_custom_enabled = conf.getBoolean("logsink_custom_enabled", logsink_enabled);

	this.LogSinkQueueSize = GetInt("logsink_queue_size", 1000)
	this.LogSinkLineSize = GetInt("logsink_line_size", 512)
	this.DebugLogSinkEnabled = GetBoolean("debug_logsink_enabled", false)
	this.DebugLogSinkLineLimit = GetInt("debug_logsink_line_limit", 0)
	this.LogSinkZipEnabled = this.LogSinkEnabled && GetBoolean("logsink_zip_enabled", true)

	this.MaxBufferSize = GetInt("max_buffer_size", 1024*64)
	this.MaxWaitTime = GetInt("max_wait_time", 2000)
	this.LogSinkZipMinSize = GetInt("logsink_zip_min_size", 100)
	this.DebugLogSinkZipEnabled = GetBoolean("debug_logsink_zip_enabled", false)
	this.LogSinkZipLibpath = GetValue("logsink_zip_libpath")
	this.TxIdTag = GetValue("logsink_txidtag")
	this.AppLogCategory = GetValue("logsink_applogcategory")
	this.AppLogPattern = GetValue("logsink_applogpattern")
	this.LogSendThreshold = GetInt("logsink_sendthreshold", 500)

	//Interval until end time, default 30 minutes
	this.LogSinkStopInterval = GetLong("logsink_stop_interval", 60000*30)
}
