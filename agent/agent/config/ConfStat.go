package config

type ConfStat struct {
	StatEnabled    bool
	StatZipEnabled bool
	Stat1MEnabled  bool

	StatIpEnabled     bool
	StatIPURLEnabled  bool
	StatIPURLMaxCount int32

	StatDomainEnabled  bool
	StatDomainMaxCount int32

	StatMtraceEnabled  bool
	StatMtraceMaxCount int32

	StatLoginEnabled  bool
	StatLoginMaxCount int32

	StatRefererEnabled  bool
	StatRefererMaxCount int32
	// 0: full url, 1:domain(protocol+host), 2:uri, 3: domain + uri
	StatRefererFormat int32

	StatTxMaxCount        int32
	StatSqlMaxCount       int32
	StatHttpcMaxCount     int32
	StatErrorMaxCount     int32
	StatUseragentMaxCount int32
}

func (this *ConfStat) Apply(conf *Config) {

	this.StatEnabled = conf.Enabled && GetBoolean("stat_enabled", true)
	this.StatZipEnabled = this.StatEnabled && GetBoolean("stat_zip_enabled", true)
	this.Stat1MEnabled = this.StatEnabled && GetBoolean("stat_1m_enabled", false)

	this.StatIpEnabled = this.StatEnabled && GetBoolean("stat_ip_enabled", true)
	this.StatIPURLEnabled = GetBoolean("stat_ipurl_enabled", false)
	this.StatIPURLMaxCount = GetInt("stat_ipurl_max_count", 10000)

	this.StatDomainEnabled = this.StatEnabled && GetBoolean("stat_domain_enabled", true)
	this.StatDomainMaxCount = GetInt("stat_domain_max_count", 7000)

	this.StatMtraceEnabled = this.StatEnabled && GetBoolean("stat_mtrace_enabled", false)
	this.StatMtraceMaxCount = GetInt("stat_mtrace_max_count", 7000)

	this.StatLoginEnabled = this.StatEnabled && GetBoolean("stat_login_enabled", false)
	this.StatLoginMaxCount = GetInt("stat_login_max_count", 7000)

	this.StatRefererEnabled = this.StatEnabled && GetBoolean("stat_referer_enabled", false)
	this.StatRefererMaxCount = GetInt("stat_referer_max_count", 7000)
	this.StatRefererFormat = GetInt("stat_referer_format", 0)

	this.StatTxMaxCount = GetInt("stat_tx_max_count", 5000)
	this.StatSqlMaxCount = GetInt("stat_sql_max_count", 5000)
	this.StatHttpcMaxCount = GetInt("stat_httpc_max_count", 5000)
	this.StatErrorMaxCount = GetInt("stat_error_max_count", 1000)
	this.StatUseragentMaxCount = GetInt("stat_useragent_max_count", 500)

}
