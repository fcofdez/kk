package digest

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
