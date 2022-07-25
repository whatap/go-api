package config

type ConfGoGrpc struct {
	GoGrpcProfileEnabled             bool
	GoGrpcProfileStreamClientEnabled bool
	GoGrpcProfileStreamServerEnabled bool
	GoGrpcProfileIgnoreMethod        []string
	GoGrpcProfileStreamMethod        []string
	GoGrpcProfileStreamIdentify      bool
	GoGrpcProfileStreamRate          int32
}

func (this *ConfGoGrpc) ApplyDefault(m map[string]string) {
	m["go.grpc_profile_enabled"] = "true"
	m["go.grpc_profile_stream_client_enabled"] = "true"
	m["go.grpc_profile_stream_server_enabled"] = "true"
	m["go.grpc_profile_ignore_method"] = ""
	m["go.grpc_profile_stream_method"] = ""
	m["go.grpc_profile_stream_identify"] = "false"
}
func (this *ConfGoGrpc) Apply(conf *Config) {
	this.GoGrpcProfileEnabled = conf.Enabled && conf.GetBoolean("go.grpc_profile_enabled", true)
	this.GoGrpcProfileStreamClientEnabled = this.GoGrpcProfileEnabled && conf.GetBoolean("go.grpc_profile_stream_client_enabled", true)
	this.GoGrpcProfileStreamServerEnabled = this.GoGrpcProfileEnabled && conf.GetBoolean("go.grpc_profile_stream_server_enabled", true)
	this.GoGrpcProfileIgnoreMethod = conf.GetStringArray("go.grpc_profile_ignore_method", ",")
	this.GoGrpcProfileStreamMethod = conf.GetStringArray("go.grpc_profile_stream_method", ",")
	this.GoGrpcProfileStreamIdentify = this.GoGrpcProfileEnabled && conf.GetBoolean("go.grpc_profile_stream_identify", false)
}
