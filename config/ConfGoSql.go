package config

type ConfGo struct {
	GoSqlProfileEnabled bool
}

func (this *ConfGo) ApplyDefault(m map[string]string) {
	m["go.sql_profile_enabled"] = "true"
}
func (this *ConfGo) Apply(conf *Config) {
	this.GoSqlProfileEnabled = conf.Enabled && conf.GetBoolean("go.sql_profile_enabled", true)
}
