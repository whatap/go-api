package config

type ConfProfile struct {
	ProfileMissingTxidEnabled      bool
	ProfileMissingTxidMaxTime      int64
	ProfileMissingTxidInterval     int64
	ProfileMissingTxidLogEnabled   bool
	DebugProfileMissingTxidEnabled bool
}

func (this *ConfProfile) Apply(conf *Config) {

	this.ProfileMissingTxidEnabled = GetBoolean("profile_missing_txid_enabled", true)
	this.ProfileMissingTxidMaxTime = GetLong("profile_missing_txid_max_time", 30000)
	this.ProfileMissingTxidInterval = GetLong("profile_missing_txid_interval", 10000)
	this.ProfileMissingTxidLogEnabled = GetBoolean("profile_missing_txid_log_enabled", false)
	this.DebugProfileMissingTxidEnabled = GetBoolean("debug_profile_missing_txid_enabled", false)
}
