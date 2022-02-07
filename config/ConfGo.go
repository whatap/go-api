package config

type ConfGo struct {
	GoSqlProfileEnabled bool
	GoCounterEnabled    bool
	GoCounterInterval   int32
	GoCounterTimeout    int32
}

func (this *ConfGo) ApplyDefault(m map[string]string) {
	m["go.sql_profile_enabled"] = "true"
	m["go.counter_enabled"] = "true"
	m["go.counter_interval"] = "5000"
	m["go.counter_timeout"] = "5000"

}
func (this *ConfGo) Apply(conf *Config) {
	this.GoSqlProfileEnabled = conf.Enabled && conf.GetBoolean("go.sql_profile_enabled", true)
	this.GoCounterEnabled = conf.Enabled && conf.GetBoolean("go.counter_enabled", true)
	this.GoCounterInterval = conf.GetInt("go.counter_interval", 5000)
	this.GoCounterTimeout = conf.GetInt("go.counter_interval", 5000)
}
