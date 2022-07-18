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
}

func (this *ConfLogSink) Apply(conf *Config) {

	if this.LogSinkEnabled {
		this.WatchLogEnabled = true
	} else {
		this.WatchLogEnabled = conf.GetBoolean("watchlog_enabled", false)
	}

	this.LogSinkFiles = conf.GetStringArray("logsink.files", ",")
	this.LogSinkInterval = conf.GetInt("logsink_interval", 1000)

	this.WatchLogCheckInterval = conf.GetLong("watchlog_check_interval", int64(2000))
	this.WatchLogReadCount = conf.GetInt("watchlog_read_count", 4)
	if this.WatchLogReadCount < 1 {
		this.WatchLogReadCount = 1
	}
	this.WatchLogSendCount = conf.GetInt("watchlog_send_count", 0)
	this.WatchLogBufferSize = conf.GetInt("watchlog_buffer_size", 128*1024)
	this.WatchLogLineSize = conf.GetInt("watchlog_line_size", 512)

	this.LogSinkEnabled = conf.GetBoolean("logsink_enabled", false)
	//		logsink_stdout_enabled = conf.getBoolean("logsink_stdout_enabled", logsink_enabled);
	//		logsink_stderr_enabled = conf.getBoolean("logsink_stderr_enabled", logsink_enabled);
	//		logsink_logback_enabled = conf.getBoolean("logsink_logback_enabled", logsink_enabled);
	//		logsink_tomcat_enabled = conf.getBoolean("logsink_tomcat_enabled", logsink_enabled);
	//		logsink_custom_enabled = conf.getBoolean("logsink_custom_enabled", logsink_enabled);

	this.LogSinkQueueSize = conf.GetInt("logsink_queue_size", 1000)
	this.LogSinkLineSize = conf.GetInt("logsink_line_size", 512)
	this.DebugLogSinkEnabled = conf.GetBoolean("debug_logsink_enabled", false)
	this.DebugLogSinkLineLimit = conf.GetInt("debug_logsink_line_limit", 0)
	this.LogSinkZipEnabled = this.LogSinkEnabled && conf.GetBoolean("logsink_zip_enabled", false)

	this.MaxBufferSize = conf.GetInt("max_buffer_size", 1024*64)
	this.MaxWaitTime = conf.GetInt("max_wait_time", 2000)
	this.LogSinkZipMinSize = conf.GetInt("logsink_zip_min_size", 100)
	this.DebugLogSinkZipEnabled = conf.GetBoolean("debug_logsink_zip_enabled", false)
	this.LogSinkZipLibpath = conf.GetValue("logsink_zip_libpath")
}
