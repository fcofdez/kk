package main

import (
	"encoding/json"
	"github.com/fcofdez/httpdigest"
	"github.com/go-martini/martini"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const WOWZA_HOME = "/usr/local/WowzaStreamingEngine/"
const WOWZA_HOME_APPS = WOWZA_HOME + "applications/"
const WOWZA_HOME_CONF = WOWZA_HOME + "conf/"
const WOWZA_HOME_CONTENT = WOWZA_HOME + "content/"
const WOWZA_IP = ""
const WOWZA_STREAM_API = "http://" + WOWZA_IP + ":8086" + "/streammanager/streamAction"
const WOWZA_ADMIN_USER = "rushmore"
const WOWZA_ADMIN_PASS = "rushmore"

var archives = make(map[string]int64)

func check(err error) {
	if err != nil {
		log.Fatal(err.Error())
		panic(err)
	}
}

func createDirs(streamId string) {
	err := os.Mkdir(filepath.Join(WOWZA_HOME_APPS, streamId), 0777)
	check(err)
	err = os.Mkdir(filepath.Join(WOWZA_HOME_CONF, streamId), 0777)
	check(err)
}

func removeDirs(streamId string) {
	err := os.RemoveAll(filepath.Join(WOWZA_HOME_APPS, streamId))
	check(err)
	err = os.RemoveAll(filepath.Join(WOWZA_HOME_CONF, streamId))
	check(err)
	streamFile := streamId + ".stream"
	stream_path := filepath.Join(WOWZA_HOME_CONTENT, streamFile)
	err = os.Remove(stream_path)
	check(err)
}

func createConfFiles(streamId, port string) string {
	streamFile := streamId + ".stream"
	confDir := filepath.Join(WOWZA_HOME_CONF, streamId)
	dat, err := ioutil.ReadFile("Application.xml")
	check(err)
	application_config_path := filepath.Join(confDir, "Application.xml")
	write_err := ioutil.WriteFile(application_config_path, dat, 0644)

	streamLoc := "udp://" + WOWZA_IP + ":" + port + "\n"
	stream_path := filepath.Join(WOWZA_HOME_CONTENT, streamFile)
	write_err = ioutil.WriteFile(stream_path, []byte(streamLoc), 0644)
	check(write_err)
	return streamFile
}

func createWowzaStream(streamId, streamFile string) {
	startVals := url.Values{"action": {"startStream"},
		"appName": {streamId + "/_definst_"}, "streamName": {streamFile},
		"mediaCasterType": {"rtp"}, "vhostName": {"undefined"}}
	authRequestWowza(startVals)
}

func removeWowzaStream(streamId, streamFile string) {
	deleteVals := url.Values{"action": {"deleteStream"},
		"appName": {streamId + "/_definst_"}, "streamName": {streamFile},
		"mediaCasterType": {"rtp"}, "vhostName": {"undefined"}}
	authRequestWowza(deleteVals)
}

func authRequestWowza(reqBody url.Values) {
	client := &http.Client{}

	req, _ := http.NewRequest("POST", WOWZA_STREAM_API, strings.NewReader(reqBody.Encode()))
	username := "rushmore"
	password := "rushmore"
	resp, _ := client.Do(req)

	if resp.StatusCode == 401 {
		reqx, _ := http.NewRequest("POST", WOWZA_STREAM_API, strings.NewReader(reqBody.Encode()))
		httpdigest.SetDigestAuth(reqx, username, password, resp, 1)
		_, erx := client.Do(reqx)
		check(erx)
	}
}

type Broadcast struct {
	Id string
}

func calculatePort(archiveId string) int64 {
	portId := strings.Split(archiveId, "-")[0]
	hexId, _ := strconv.ParseInt(portId, 16, 0)
	return 10000 + hexId%30011
}

func generateWowzaApp(streamId, port string) {
	createDirs(streamId)
	streamFile := createConfFiles(streamId, port)
	createWowzaStream(streamId, streamFile)
}

func deleteWowzaApp(streamId, port string) {
	removeWowzaStream(streamId, port)
	removeDirs(streamId)
}

func createLoggerFile() *log.Logger {
	file, err := os.OpenFile("/logs/wowzaapi.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file")
	}

	logger := log.New(file,
		"Log: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	return logger
}

func main() {
	logger := createLoggerFile()
	m := martini.Classic()
	m.Post("/streams/", func(c martini.Context, req *http.Request) (int, string) {
		decoder := json.NewDecoder(req.Body)
		var broadcast Broadcast
		decoder.Decode(&broadcast)
		if broadcast.Id == "" {
			logger.Println("Error: Not Archive id given.")
			return 400, "Not Archive id givenn."
		}
		port := calculatePort(broadcast.Id)
		streamId := broadcast.Id
		generateWowzaApp(streamId, strconv.FormatInt(port, 10))
		archives[broadcast.Id] = port
		logger.Println("Stream created:", broadcast.Id, port)
		return 200, ""
	})
	m.Delete("/streams/:archiveid", func(params martini.Params) (int, string) {
		port := calculatePort(params["archiveid"])
		streamId := params["archiveid"]
		if archives[streamId] == 0 {
			logger.Println("Error trying to delete stream with archive id: ", params["archiveid"])
			return 404, "Not found"
		}
		deleteWowzaApp(streamId, strconv.FormatInt(port, 10))
		logger.Println("Stream deleted:", params["archiveid"])
		delete(archives, params["archiveid"])
		return 200, ""
	})

	m.Run()
}
