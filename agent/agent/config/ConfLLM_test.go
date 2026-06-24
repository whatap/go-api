package config

import (
	"testing"
)

func TestConfLLM_DisabledLeavesChannelsUntouched(t *testing.T) {
	conf := &Config{
		AccessKey:  "main-key",
		License:    "main-license",
		WhatapHost: []string{"main-host"},
	}
	c := &ConfLLM{
		LLMMode:       false,
		LLMAccessKey:  "should-not-apply",
		LLMWhatapHost: []string{"should-not-apply"},
	}
	c.ApplyChannelOverride(conf)

	if conf.AccessKey != "main-key" {
		t.Fatalf("AccessKey should not change when LLMMode false, got %q", conf.AccessKey)
	}
	if conf.License != "main-license" {
		t.Fatalf("License should not change when LLMMode false, got %q", conf.License)
	}
	if len(conf.WhatapHost) != 1 || conf.WhatapHost[0] != "main-host" {
		t.Fatalf("WhatapHost should not change when LLMMode false, got %v", conf.WhatapHost)
	}
}

func TestConfLLM_EnabledOverridesChannels(t *testing.T) {
	conf := &Config{
		AccessKey:  "main-key",
		License:    "main-license",
		WhatapHost: []string{"main-host"},
	}
	c := &ConfLLM{
		LLMMode:       true,
		LLMAccessKey:  "llm-key",
		LLMWhatapHost: []string{"llm-host-1", "llm-host-2"},
	}
	c.ApplyChannelOverride(conf)

	if conf.AccessKey != "llm-key" {
		t.Fatalf("AccessKey: want llm-key, got %q", conf.AccessKey)
	}
	if conf.License != "llm-key" {
		t.Fatalf("License: want llm-key, got %q", conf.License)
	}
	if len(conf.WhatapHost) != 2 || conf.WhatapHost[0] != "llm-host-1" {
		t.Fatalf("WhatapHost: want [llm-host-1 llm-host-2], got %v", conf.WhatapHost)
	}
}

func TestConfLLM_EnabledWithoutKeysKeepsMain(t *testing.T) {
	// LLMMode=true 이지만 AccessKey/Host 가 비어 있으면 메인 값 유지 (apm-go-agent 동등)
	conf := &Config{
		AccessKey:  "main-key",
		License:    "main-license",
		WhatapHost: []string{"main-host"},
	}
	c := &ConfLLM{
		LLMMode:       true,
		LLMAccessKey:  "",
		LLMWhatapHost: nil,
	}
	c.ApplyChannelOverride(conf)

	if conf.AccessKey != "main-key" {
		t.Fatalf("AccessKey should stay main when llm_accesskey empty, got %q", conf.AccessKey)
	}
	if len(conf.WhatapHost) != 1 || conf.WhatapHost[0] != "main-host" {
		t.Fatalf("WhatapHost should stay main when llm.host empty, got %v", conf.WhatapHost)
	}
}

func TestConfLLM_EnabledWhitespaceAccessKeyIgnored(t *testing.T) {
	conf := &Config{AccessKey: "main-key"}
	c := &ConfLLM{
		LLMMode:      true,
		LLMAccessKey: "   ",
	}
	c.ApplyChannelOverride(conf)
	if conf.AccessKey != "main-key" {
		t.Fatalf("whitespace LLMAccessKey should not override, got %q", conf.AccessKey)
	}
}
