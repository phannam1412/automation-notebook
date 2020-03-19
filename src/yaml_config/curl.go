package yaml_config

import (
	"bytes"
	"common"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ICurl interface {
	RunForKey(string) error
	GetResponse() *http.Response
	GetResponseString() string
	GetItem(string) (*CurlItem, error)
	RunItem(item *CurlItem) error
	DisableVerbose()
	EnableVerbose()
	SetWriter(fn func(text string))
}

type ICurlWriter func(text string)

type CurlItem struct {
	AccessUrl                             string            `yaml:"access url"`
	UseBasicAuthentication                string                 `yaml:"use basic authentication"`
	UseBearerAuthorizationTokenFromConfig string                 `yaml:"use bearer authorization token from config"`
	SendRequestHeaders                    map[string]string      `yaml:"send request headers"`
	SendRawBody                           string                 `yaml:"send raw body"`
	SendFileFromPath               		  string                 `yaml:"send file from path"`
	SendEncodedRequestBodyInJavaStyle     map[string]interface{} `yaml:"send encoded request body in java style"`
	SendFormData                          map[string]interface{} `yaml:"send form data"`
	SendAdditionalPathParams              map[string]string      `yaml:"send additional params"`
	PatchBodyWithTheFollowingValues map[string]string `yaml:"patch body with the following values"`
	FinalRequestBody string
}

type CurlCollection struct {
	yml               map[string]CurlItem
	config            IConfig
	response          *http.Response
	additionalHeaders map[string]string
	verbose           bool
	writer            ICurlWriter
}

func NewCurlCollection(configService IConfig, data []byte) (*CurlCollection, error) {
	yml := map[string]CurlItem{}
	err := yaml.Unmarshal(data, &yml)
	if err != nil {
		return nil, err
	}
	return &CurlCollection{
		yml:               yml,
		config:            configService,
		response:          nil,
		additionalHeaders: nil,
		verbose:           true,
		writer: func(text string) {
			fmt.Printf(text)
		},
	}, nil
}

func (this *CurlCollection) SetWriter(fn func(text string)) {
	this.writer = fn
}

func (this *CurlCollection) GetKeys() []string {
	output := make([]string, 0, len(this.yml))
	for k := range this.yml {
		output = append(output, k)
	}
	return output
}

func (this *CurlCollection) DisableVerbose() {
	this.verbose = false
}

func (this *CurlCollection) EnableVerbose() {
	this.verbose = true
}

func (this *CurlItem) Run(config IConfig, verbose bool, writer ICurlWriter) (*http.Response, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	var req *http.Request
	var err error
	body := &bytes.Buffer{}
	method := "GET"

	accessUrl := this.AccessUrl
	if strings.Index(accessUrl, "GET ") == 0 ||
		strings.Index(accessUrl, "POST ") == 0 ||
		strings.Index(accessUrl, "PUT ") == 0 ||
		strings.Index(accessUrl, "PATCH ") == 0 ||
		strings.Index(accessUrl, "DELETE ") == 0 ||
		strings.Index(accessUrl, "OPTIONS ") == 0 {
		indexOfFirstWhitespace := strings.Index(accessUrl, " ")
		method = accessUrl[:indexOfFirstWhitespace]
		accessUrl = accessUrl[indexOfFirstWhitespace + 1:]
	}

	if len(this.SendFormData) > 0 {
		method = "POST"
		body = bytes.NewBufferString(common.MapStringStringToUrlValues(common.MapStringInterfaceToMapStringString(this.SendFormData)).Encode())
	}
	if len(this.SendEncodedRequestBodyInJavaStyle) > 0 {
		additional := ""
		for k, v := range this.SendEncodedRequestBodyInJavaStyle {
			if tmp, ok := v.([]interface{}); ok {
				tmp3 := common.ArrayInterfaceToArrayString(tmp)
				var tmp2 []string
				for _, v2 := range tmp3 {
					tmp2 = append(tmp2, k + "=" + v2)
				}
				additional = additional + "&" + strings.Join(tmp2, "&")
				delete(this.SendEncodedRequestBodyInJavaStyle, k)
			}
		}
		method = "POST"
		body = bytes.NewBufferString(common.MapStringStringToUrlValues(common.MapStringInterfaceToMapStringString(this.SendEncodedRequestBodyInJavaStyle)).Encode() + additional)
	}
	if this.SendRawBody != "" {
		body = bytes.NewBufferString(this.SendRawBody)
	}
	if len(this.SendAdditionalPathParams) > 0 {
		tmp, err := url.Parse(accessUrl)
		if err != nil {
			return nil, err
		}
		query := tmp.Query()
		for k, v := range this.SendAdditionalPathParams {
			query.Set(k, v)
		}
		tmp.RawQuery = query.Encode()
		accessUrl = tmp.String()
	}
	if this.PatchBodyWithTheFollowingValues != nil {
		for replaced, rule := range this.PatchBodyWithTheFollowingValues {
			rules := strings.Split(rule, ",")
			replacing := ""
			for _, v := range rules {
				if v == "[timestamp]" {
					replacing += fmt.Sprintf("%d", time.Now().Unix())
				} else {
					replacing += v
				}
			}
			body = bytes.NewBufferString(strings.Replace(body.String(), replaced, replacing, -1))
		}
	}
	req, err = http.NewRequest(method, accessUrl, body)
	if err != nil {
		return nil, err
	}
	if this.SendFileFromPath != "" {
		file, err := os.Open(this.SendFileFromPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		body = &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := common.CreateFormFileWithContentType("file", filepath.Base(file.Name()), writer, "text/csv")
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.Copy(part, file)
		common.PanicOnError(err)
		err = writer.Close()
		common.PanicOnError(err)
		// @todo should refactor to initialize at one place
		req, err = http.NewRequest(method, accessUrl, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
	}
	if len(this.SendFormData) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if len(this.SendEncodedRequestBodyInJavaStyle) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if this.UseBasicAuthentication != "" {
		req.Header.Set("Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte(this.UseBasicAuthentication)))
	}
	if this.UseBearerAuthorizationTokenFromConfig != "" {
		val, err := config.GetStringByKey(this.UseBearerAuthorizationTokenFromConfig)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", val)
	}
	for k, v := range this.SendRequestHeaders {
		req.Header.Set(k, v)
	}
	this.FinalRequestBody = body.String()
	return this.exec(verbose, writer, req)
}

func (this *CurlItem) exec(verbose bool, writer ICurlWriter, req *http.Request) (*http.Response, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if verbose {
		text, err := httputil.DumpRequest(req, true)
		if err != nil {
			return nil, err
		}
		writer(fmt.Sprintf("%s\n", text))
	}
	res, err := client.Do(req)
	if err != nil {
		return res, err
	}
	if verbose {
		text, err := httputil.DumpResponse(res, true)
		if err != nil {
			return nil, err
		}
		writer(fmt.Sprintf("%s\n", text))
	}
	return res, nil
}

func (this *CurlCollection) GetItem(formula string) (*CurlItem, error) {
	var info CurlItem
	var ok bool
	if info, ok = this.yml[formula]; !ok {
		return nil, fmt.Errorf("curl formula %s does not exist", formula)
	}
	return &info, nil
}

func (this *CurlCollection) RunItem(item *CurlItem) error {
	res, err := item.Run(this.config, this.verbose, this.writer)
	this.response = res
	return err
}

func (this *CurlCollection) RunForKey(formula string) error {
	item, err := this.GetItem(formula)
	if err != nil {
		return err
	}
	res, err := item.Run(this.config, this.verbose, this.writer)
	this.response = res
	return err
}

func (this *CurlCollection) GetResponse() *http.Response {
	return this.response
}

func (this *CurlCollection) GetResponseString() string {
	b, err := ioutil.ReadAll(this.response.Body)
	if err != nil {
		return ""
	}
	return string(b)
}
