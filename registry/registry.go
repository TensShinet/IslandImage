package registry

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
)

const (
	defaultRegistryHost = "index.docker.io"
	defaultImageTag     = "latest"
	defaultRepoName     = "library"
	defaultScheme       = "https"

	ManifestListAccept = "application/vnd.docker.distribution.manifest.list.v2+json"
	ManifestAccept     = "application/vnd.docker.distribution.manifest.v2+json"

	blobsFormat    = "%s://%s/v2/%s/%s/blobs/%s"
	ManifestFormat = "%s://%s/v2/%s/%s/manifests/%s"
)

var InvalidArchitecture = errors.New("invalid Architecture")

type Config struct {
	Username  string
	Password  string
	Proxy     string
	Insecure  bool
	ImageName string
	SaveDir   string
	UseHttp   bool
}

type registry struct {
	client *http.Client

	auth     authentication
	username string
	password string
	saveDir  string

	Scheme    string
	RepoName  string
	ImageName string
	ImageTag  string
	Host      string
}

type authentication struct {
	AccessToken  string    `json:"access_token,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	IssuedAt     time.Time `json:"issued_at,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	RefreshToken string    `json:"refresh_token, omitempty"`
}

// ImageName 有几种情况
// 1. index.docker.io/tensshinet/busybox
// 2. tensshinet/busybox
// 3. busybox
// 4. index.docker.io:5000/tensshinet/busybox
// 5. index.docker.io:5000/tensshinet/tensshinet/busybox
// 6. 非法
func New(conf Config) (*registry, error) {
	if conf.ImageName == "" {
		return nil, fmt.Errorf("invalid reference format")
	}
	if _, err := url.Parse("http://" + conf.ImageName); err != nil {
		return nil, fmt.Errorf("invalid reference format")
	}

	// 长度不可能为 0 最少为 1
	temp := strings.Split(conf.ImageName, "/")
	reg := &registry{
		saveDir:  conf.SaveDir,
		Host:     defaultRegistryHost,
		Scheme:   defaultScheme,
		RepoName: defaultRepoName,
		ImageTag: defaultImageTag,
	}
	if conf.UseHttp {
		reg.Scheme = "http"
	}
	reg.ImageName = temp[len(temp)-1]
	if strings.Contains(reg.ImageName, ":") {
		imageParts := strings.Split(reg.ImageName, ":")
		reg.ImageName = imageParts[0]
		reg.ImageTag = imageParts[1]
	}
	if len(temp) == 2 {
		reg.RepoName = temp[0]
	} else if len(temp) > 2 {
		reg.Host = temp[0]
		reg.RepoName = strings.Join(temp[1:len(temp)-1], "/")
	}

	reg.client = &http.Client{
		Transport: &http.Transport{
			Proxy:           getProxy(conf.Proxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: conf.Insecure},
		},
	}
	return reg, nil
}

func (reg *registry) tryGet(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", reg.auth.AccessToken))
	if headers != nil {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}
	res, err := reg.client.Do(req)
	if err != nil {
		return res, err
	}
	return res, err
}

// 先尝试拉取一下 如果是 401 那么就先认证
func (reg *registry) doGet(url string, headers map[string]string) (*http.Response, error) {
	res, err := reg.tryGet(url, headers)
	if err != nil {
		return res, err
	}
	if res.StatusCode != 200 && res.StatusCode != 401 {
		res.Body.Close()
		return nil, fmt.Errorf("HTTP Error: %s", res.Status)
	}
	if res.StatusCode == 401 {
		res.Body.Close()
		resHeaders := res.Header
		authURL, err := getAuthURL(resHeaders.Get("Www-Authenticate")[7:])
		if err != nil {
			return nil, err
		}
		if err := reg.GetToken(authURL); err != nil {
			return nil, err
		}
		res, err = reg.tryGet(url, headers)
		if err != nil || res.StatusCode != 200 {
			if res != nil {
				res.Body.Close()
				return nil, fmt.Errorf("HTTP status code %v error", res.StatusCode)
			} else {
				return nil, err
			}
		}
	}
	return res, nil
}

func (reg *registry) GetToken(u string) error {
	fmt.Println("GetToken ", u)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	if reg.username != "" && reg.password != "" {
		req.SetBasicAuth(reg.username, reg.password)
	}
	res, err := reg.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("HTTP Error: %s", res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var a = new(authentication)
	err = json.Unmarshal(body, &a)
	if err != nil {
		return err
	}
	reg.auth = *a
	return nil
}

func (reg *registry) GetManifests() (string, error) {
	headers := make(map[string]string)
	headers["Accept"] = ManifestListAccept
	manifestListURL := fmt.Sprintf(ManifestFormat, reg.Scheme, reg.Host, reg.RepoName, reg.ImageName, reg.ImageTag)
	res, err := reg.doGet(manifestListURL, headers)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	rawJSON, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	manifestList := ManifestList{}
	if err := json.Unmarshal(rawJSON, &manifestList); err != nil {
		return "", err
	}

	for _, m := range manifestList.Manifests {
		if m.Platform.Architecture == runtime.GOARCH && m.Platform.OS == "linux" {
			return m.Digest, nil
		}
	}

	return "", InvalidArchitecture
}

// 如果为空那么默认是 ImageTag
func (reg *registry) GetManifest(manifestDigest string) (*Manifest, error) {
	headers := make(map[string]string)
	if manifestDigest == "" {
		manifestDigest = reg.ImageTag
	}
	manifestURL := fmt.Sprintf(ManifestFormat, reg.Scheme, reg.Host, reg.RepoName, reg.ImageName, manifestDigest)
	fmt.Println("GetManifest ", manifestURL)
	headers["Accept"] = ManifestAccept
	res, err := reg.doGet(manifestURL, headers)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	rawJSON, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	m := new(Manifest)
	if err := json.Unmarshal(rawJSON, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func (reg *registry) GetConfig(manifest *Manifest) (string, error) {
	configFilePath := filepath.Join(reg.saveDir, fmt.Sprintf("%s.json", strings.TrimPrefix(manifest.Config.Digest, "sha256:")))
	out, err := os.Create(configFilePath)
	if err != nil {
		return "", fmt.Errorf("create config file failed")
	}
	defer out.Close()
	headers := make(map[string]string)
	configURL := fmt.Sprintf(blobsFormat, reg.Scheme, reg.Host, reg.RepoName, reg.ImageName, manifest.Config.Digest)
	headers["Accept"] = manifest.Config.MediaType

	res, err := reg.doGet(configURL, headers)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		return "", fmt.Errorf("writing config file failed")
	}

	return configFilePath, nil
}

func (reg *registry) GetLayers(layers []ManifestLayer) error {
	for _, layer := range layers {
		suffix := getTarFileSuffix(layer.MediaType)
		tarFilePath := filepath.Join(reg.saveDir, fmt.Sprintf("%s.%s", strings.TrimPrefix(layer.Digest, "sha256:"), suffix))
		out, err := os.Create(tarFilePath)
		if err != nil {
			return fmt.Errorf("create store file failed")
		}
		defer out.Close()

		headers := make(map[string]string)
		layerURL := fmt.Sprintf(blobsFormat, reg.Scheme, reg.Host, reg.RepoName, reg.ImageName, layer.Digest)
		headers["Accept"] = layer.MediaType
		res, err := reg.doGet(layerURL, headers)
		if err != nil {
			return err
		}
		bar := pb.New(layer.Size).Set(pb.Bytes, true)
		bar.SetWidth(50)
		bar.Start()
		reader := bar.NewProxyReader(res.Body)
		_, err = io.Copy(out, reader)
		bar.Finish()
		if err != nil {
			return fmt.Errorf("write tar file failed")
		}
		defer res.Body.Close()
	}
	return nil
}
