package main

import (
	"io/ioutil"
	"kk/httpdigest"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const WOWZA_HOME = "/usr/local/WowzaStreamingEngine/"
const WOWZA_HOME_APPS = WOWZA_HOME + "applications/"
const WOWZA_HOME_CONF = WOWZA_HOME + "conf/"
const WOWZA_HOME_CONTENT = WOWZA_HOME + "content/"
const WOWZA_IP = "127.0.0.1"
const WOWZA_STREAM_API = "http://" + WOWZA_IP + ":8086" + "/streammanager/streamAction"
const WOWZA_ADMIN_USER = "rushmore"
const WOWZA_ADMIN_PASS = "rushmore"

func check(err error) {
	if err != nil {
		log.Fatal(err.Error())
		panic(err)
	}
}

func main() {
	streamId := "test346868"
	streamFile := streamId + ".stream"
	port := "10018"

	err := os.Mkdir(filepath.Join(WOWZA_HOME_APPS, streamId), 0777)
	check(err)
	err = os.Mkdir(filepath.Join(WOWZA_HOME_CONF, streamId), 0777)
	check(err)

	confDir := filepath.Join(WOWZA_HOME_CONF, streamId)
	dat, err := ioutil.ReadFile("Application.xml")
	check(err)
	application_config_path := filepath.Join(confDir, "Application.xml")
	write_err := ioutil.WriteFile(application_config_path, dat, 0644)

	streamLoc := "udp://" + WOWZA_IP + ":" + port + "\n"
	stream_path := filepath.Join(WOWZA_HOME_CONTENT, streamFile)
	write_err = ioutil.WriteFile(stream_path, []byte(streamLoc), 0644)
	check(write_err)
	startReq := url.Values{"action": {"startStream"},
		"appName": {streamId + "/_definst_"}, "streamName": {streamFile},
		"mediaCasterType": {"rtp"}, "vhostName": {"undefined"}}

	client := &http.Client{}

	req, _ := http.NewRequest("POST", WOWZA_STREAM_API, strings.NewReader(startReq.Encode()))
	username := "rushmore"
	password := "rushmore"
	resp, _ := client.Do(req)

	if resp.StatusCode == 401 {
		reqx, _ := http.NewRequest("POST", WOWZA_STREAM_API, strings.NewReader(startReq.Encode()))
		httpdigest.SetDigestAuth(reqx, username, password, resp, 1)
		_, erx := client.Do(reqx)
		check(erx)
	}
}
