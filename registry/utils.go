package registry

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func Debug(a ...interface{}) {
	fmt.Print("registry ")
	fmt.Println(a...)
}

func getProxy(proxy string) func(*http.Request) (*url.URL, error) {
	if len(proxy) > 0 {
		proxyURL, _ := url.Parse(proxy)
		return http.ProxyURL(proxyURL)
	}
	return http.ProxyFromEnvironment
}

func getAuthURL(s string) (string, error) {
	temp := strings.Split(s, ",")
	res := ""
	for i, t := range temp {
		t1 := strings.Split(t, "=")
		if len(t1) < 2 {
			return "", fmt.Errorf("invalid AuthURL")
		}
		if i == 0 {
			res += strings.Trim(t1[1], "\"") + "?"
		} else if i < len(temp)-1 {
			res += t1[0] + "=" + strings.Trim(t1[1], "\"") + "&"
		} else {
			res += t1[0] + "=" + strings.Trim(t1[1], "\"")
		}
	}
	return res, nil
}

func getTarFileSuffix(mediaType string) string {
	t := strings.Split(mediaType, ".")
	s := t[len(t)-1]

	switch s {
	case "gzip":
		return "tar.gz"
	case "zstd":
		return "tar.zst"
	default:
		return "tar"
	}
}
