package config

import (
	"strings"

	"github.com/whatap/go-api/agent/util/logutil"
)

type ConfLLM struct {
	LLMMode       bool
	LLMAccessKey  string
	LLMWhatapHost []string
}

func (this *ConfLLM) Load() {
	this.LLMMode = getBoolean("llm_enabled", false)

	this.LLMAccessKey = getValue("llm_accesskey")
	if strings.TrimSpace(this.LLMAccessKey) == "" {
		this.LLMAccessKey = getValue("llm_license")
	}
	this.LLMWhatapHost = getStringArray("llm.whatap.server.host", "/:,")
}

func (this *ConfLLM) ApplyChannelOverride(conf *Config) {
	if !this.LLMMode {
		return
	}
	accessKeyOverridden := false
	hostOverridden := false
	if strings.TrimSpace(this.LLMAccessKey) != "" {
		conf.AccessKey = this.LLMAccessKey
		conf.License = this.LLMAccessKey
		accessKeyOverridden = true
	}
	if this.LLMWhatapHost != nil && len(this.LLMWhatapHost) > 0 {
		conf.WhatapHost = this.LLMWhatapHost
		hostOverridden = true
	}
	switch {
	case accessKeyOverridden && hostOverridden:
		logutil.Println("WA216-LLM", "LLM Mode enabled: llm_accesskey + llm.whatap.server.host override applied")
	case accessKeyOverridden:
		logutil.Println("WA216-LLM", "LLM Mode enabled: llm_accesskey override applied (whatap.server.host inherited from main)")
	case hostOverridden:
		logutil.Println("WA216-LLM", "LLM Mode enabled: llm.whatap.server.host override applied (license inherited from main)")
	default:
		logutil.Println("WA216-LLM", "LLM Mode enabled: license + whatap.server.host inherited from main channel (no LLM-specific override)")
	}
}

func (this *ConfLLM) Apply(conf *Config) {
	this.Load()
	this.ApplyChannelOverride(conf)
}
