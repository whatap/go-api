package net

import (
	"log"
	"time"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
)

func NetSyncTest(whatapip *string, license *string) {
	agentconfig := config.GetConfig()
	if len(*whatapip) > 0 {
		agentconfig.WhatapHost = stringutil.Tokenizer(*whatapip, "/")
	}
	if len(*license) > 0 {
		agentconfig.License = *license
	}

	session := new(TcpSession)
	defer func() {
		if x := recover(); x != nil {
			log.Fatal(x)
		} else {
			session.Close()
		}
	}()
	secure.GetSecurityMaster().WaitForInitFor(30)
	secure.GetSecuritySession()
	if secure.GetSecurityMaster().PCODE == 0 {
		log.Fatal("invalid license error", secure.GetSecurityMaster().Cypher)
	}
	if session.open() == false {
		log.Fatal("connection to ", *whatapip, " 6600 failed")
	}
	now := dateutil.Now()
	if session.Send(NET_TIME_SYNC, io.ToBytesLong(now), true) == false {
		log.Fatal("sending data to ", *whatapip, " 6600 failed")
	}
	isResponseReceived := false
	go func() {
		out := session.Read()
		if out != nil && out.Code == NET_TIME_SYNC {
			isResponseReceived = true
		}
	}()
	time.Sleep(3 * time.Second)

	if isResponseReceived == false {
		log.Fatal("Received no response")
	}
}
