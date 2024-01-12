package config

type ConfGo struct {
	GoSqlProfileEnabled bool
	GoCounterEnabled    bool
	GoCounterInterval   int32
	GoCounterTimeout    int32

	GoRecoverEnabled bool

	GoUseGoroutineIDEnabled bool
}

func (this *ConfGo) ApplyDefault(m map[string]string) {
	m["go.sql_profile_enabled"] = "true"
	m["go.counter_enabled"] = "true"
	m["go.counter_interval"] = "5000"
	m["go.counter_timeout"] = "5000"
	m["go.use_goroutine_id_enabled"] = "false"
}
func (this *ConfGo) Apply(conf *Config) {
	this.GoSqlProfileEnabled = conf.Enabled && GetBoolean("go.sql_profile_enabled", true)
	this.GoCounterEnabled = conf.Enabled && GetBoolean("go.counter_enabled", true)
	this.GoCounterInterval = GetInt("go.counter_interval", 5000)
	this.GoCounterTimeout = GetInt("go.counter_interval", 5000)
	this.GoRecoverEnabled = GetBoolean("go.recover_enabled", false)

	this.GoUseGoroutineIDEnabled = conf.Enabled && GetBoolean("go.use_goroutine_id_enabled", false)
}
