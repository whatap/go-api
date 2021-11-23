package config

type ConfGrpc struct {
	GrpcProfileEnabled             bool
	GrpcProfileStreamClientEnabled bool
	GrpcProfileStreamServerEnabled bool
	GrpcProfileIgnoreMethod        []string
	GrpcProfileStreamMethod        []string
	GrpcProfileStreamIdentify      bool
	GrpcProfileStreamRate          int32
}

func (this *ConfGrpc) ApplyDefault(m map[string]string) {
	m["grpc_profile_enabled"] = "true"
	m["grpc_profile_stream_client_enabled"] = "true"
	m["grpc_profile_stream_server_enabled"] = "true"
	m["grpc_profile_ignore_method"] = ""
	m["grpc_profile_stream_method"] = ""
	m["grpc_profile_stream_identify"] = "false"
}
func (this *ConfGrpc) Apply(conf *Config) {
	this.GrpcProfileEnabled = conf.Enabled && conf.GetBoolean("grpc_profile_enabled", true)
	this.GrpcProfileStreamClientEnabled = conf.GrpcProfileEnabled && conf.GetBoolean("grpc_profile_stream_client_enabled", true)
	this.GrpcProfileStreamServerEnabled = conf.GrpcProfileEnabled && conf.GetBoolean("grpc_profile_stream_server_enabled", true)
	this.GrpcProfileIgnoreMethod = conf.GetStringArray("grpc_profile_ignore_method", ",")
	this.GrpcProfileStreamMethod = conf.GetStringArray("grpc_profile_stream_method", ",")
	this.GrpcProfileStreamIdentify = this.GrpcProfileEnabled && conf.GetBoolean("grpc_profile_stream_identify", false)
}
