package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
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

type Authorization struct {
	Username, Password, Realm, Nonce, QOP, Opaque, Algorithm string
}

func GetAuthorization(username, password string, resp *http.Response) *Authorization {
	header := resp.Header.Get("www-authenticate")
	parts := strings.SplitN(header, " ", 2)
	parts = strings.Split(parts[1], ", ")
	opts := make(map[string]string)

	for _, part := range parts {
		fmt.Println("Part: ", part)
		vals := strings.SplitN(part, "=", 2)
		key := vals[0]
		val := strings.Trim(vals[1], "\",")
		opts[key] = val
	}

	auth := Authorization{
		username, password,
		opts["realm"], opts[" nonce"], opts["qop"], opts["opaque"], opts["algorithm"],
	}

	return &auth
}

func SetDigestAuth(r *http.Request, username, password string, resp *http.Response, nc int) {
	auth := GetAuthorization(username, password, resp)
	auth_str := GetAuthString(auth, r.URL, r.Method, nc)
	r.Header.Add("Authorization", auth_str)
	fmt.Println(r.Header)
}

func GetAuthString(auth *Authorization, url *url.URL, method string, nc int) string {
	a1 := auth.Username + ":" + auth.Realm + ":" + auth.Password
	h := md5.New()
	io.WriteString(h, a1)
	ha1 := hex.EncodeToString(h.Sum(nil))

	h = md5.New()
	a2 := method + ":" + url.Path
	io.WriteString(h, a2)
	ha2 := hex.EncodeToString(h.Sum(nil))

	nc_str := fmt.Sprintf("%08x", nc)
	hnc := RandomKey()

	respdig := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, auth.Nonce, nc_str, hnc, auth.QOP, ha2)
	h = md5.New()
	io.WriteString(h, respdig)
	respdig = hex.EncodeToString(h.Sum(nil))

	base := "username=\"%s\", realm=\"%s\", nonce=\"%s\", uri=\"%s\", response=\"%s\""
	base = fmt.Sprintf(base, auth.Username, auth.Realm, auth.Nonce, url.Path, respdig)
	if auth.Opaque != "" {
		base += fmt.Sprintf(", opaque=\"%s\"", auth.Opaque)
	}
	if auth.QOP != "" {
		base += fmt.Sprintf(", qop=%s, nc=%s, cnonce=\"%s\"", auth.QOP, nc_str, hnc)
	}
	if auth.Algorithm != "" {
		base += fmt.Sprintf(", algorithm=%s", auth.Algorithm)
	}

	return "Digest " + base
}

func RandomKey() string {
	k := make([]byte, 12)
	for bytes := 0; bytes < len(k); {
		n, err := rand.Read(k[bytes:])
		if err != nil {
			panic("rand.Read() failed")

		}
		bytes += n

	}
	return base64.StdEncoding.EncodeToString(k)

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
		SetDigestAuth(reqx, username, password, resp, 1)
		_, erx := client.Do(reqx)
		check(erx)
	}
}
