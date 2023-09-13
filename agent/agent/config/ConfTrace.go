package config

type ConfTrace struct {
	TraceZipEnabled       bool
	TraceZipQueueSize     int
	TraceZipMaxWaitTime   int
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
}

func (this *ConfTrace) Apply(conf *Config) {

	this.TraceZipEnabled = GetBoolean("profile_zip_enabled", true)
	this.TraceZipQueueSize = int(GetInt("profile_zip_queue_size", 1000))
	this.TraceZipMaxWaitTime = int(GetInt("profile_zip_max_wait_time", 1000))
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
}
