package yaml_config

import (
	"common"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type IAutomatedCheckWriter func(text string)

type AutomatedCheckCollection struct {
	yml                  map[string]AutomatedCheckItem
	config               IConfig
	curl                 ICurl
	res                  *http.Response
	body                 string
	writer               IAutomatedCheckWriter
}

type AutomatedCheckCurlItem struct {
	ConfigKey string `yaml:"config key"`
	AddPathParams map[string]interface{} `yaml:"add path params"`
}

type JsonItem struct {
	JsonKey string `yaml:"json key"`
	ExpectedJsonString string `yaml:"expected json string"`
}

type AutomatedCheckItemStep struct {
	DoHttpRequestFromCurlConfig *string           `yaml:"do http request from curl config"`
	DoHttpRequestFromCurl *AutomatedCheckCurlItem `yaml:"do http request from curl"`
	DoHttpRequest *CurlItem 					  `yaml:"do http request"`
	SeeHttpResponseCode *int                      `yaml:"see http response code"`
	SeedingDataWithImpex *string                  `yaml:"seeding data with impex"`
	RunAllImpexFromDirectory *string              `yaml:"run all impex from directory"`
	SeeJsonString *string                         `yaml:"see json string"`
	SeeSubstring *string                          `yaml:"see substring"`
	NotSeeSubstring *string                       `yaml:"not see substring"`
	QueryDatabase *string                          `yaml:"query database"`
	RunIntegrationTestFor *string                  `yaml:"run integration test for"`
	SeeJsonStringFor *JsonItem                     `yaml:"see json string for"`
}

type AutomatedCheckItem struct {
	StepsToVerify []AutomatedCheckItemStep `yaml:"steps to verify"`
	Group *string                          `yaml:"group"`
	config               IConfig
	curl                 ICurl
	res                  *http.Response
	body                 string
	writer               IAutomatedCheckWriter
	hybrisAdminCookie    string
	hybrisAdminCsrfToken string
}

func NewAutomatedCheckCollection(config IConfig, curl ICurl, data []byte) (*AutomatedCheckCollection, error) {
	yml := map[string]AutomatedCheckItem{}
	err := yaml.Unmarshal(data, &yml)
	if err != nil {
		return nil, err
	}
	curl.DisableVerbose()
	return &AutomatedCheckCollection{
		yml: yml,
		config: config,
		curl:   curl,
		res:    nil,
		body:   "",
		writer: func(text string) {
			fmt.Printf(text)
		},
	}, nil
}

func (this *AutomatedCheckCollection) GetItem(itemKey string) *AutomatedCheckItem {
	if v, ok := this.yml[itemKey]; ok {
		return &v
	}
	return nil
}

func (this *AutomatedCheckCollection) SetWriter(writer IAutomatedCheckWriter) {
	this.writer = writer
}

func (this *AutomatedCheckCollection) GetKeys() []string {
	output := make([]string, 0, len(this.yml))
	for k := range this.yml {
		output = append(output, k)
	}
	return output
}

func (this *AutomatedCheckCollection) RunGroup(group string) error {
	for k, v := range this.yml {
		if v.Group != nil && *v.Group == group {
			this.writer(fmt.Sprintf(">>> %s\n", k))
			err := v.Run(this.writer, this)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *AutomatedCheckItem) isLoggedIn() (bool, error) {
	hybrisAdminUrl, err := this.config.GetStringByKey("hybris admin url")
	if err != nil {
		return false, err
	}
	yml := fmt.Sprintf(`
access url: %s/
use method: get
send request headers:
  Cookie: %s
`, hybrisAdminUrl, this.hybrisAdminCookie)
	item := CurlItem{}
	err = yaml.Unmarshal([]byte(yml), &item)
	if err != nil {
		return false, err
	}
	err = this.curl.RunItem(&item)
	if err != nil {
		return false, err
	}
	return this.curl.GetResponse().StatusCode == 200, nil
}

func (this *AutomatedCheckItem) getCsrfTokenFromLoginPage() (string, error) {
	hybrisAdminUrl, err := this.config.GetStringByKey("hybris admin url")
	if err != nil {
		return "", err
	}
	yml := fmt.Sprintf(`
access url: %s/login.jsp
use method: get
`, hybrisAdminUrl)
	item := CurlItem{}
	err = yaml.Unmarshal([]byte(yml), &item)
	if err != nil {
		return "", err
	}
	err = this.curl.RunItem(&item)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(this.curl.GetResponse().Body)
	if err != nil {
		return "", err
	}
	value, exists := doc.Find("input[name='_csrf']").Attr("value")
	if !exists {
		return "", fmt.Errorf("csrf token does not exist")
	}
	return value, nil
}

func (this *AutomatedCheckItem) autoLoginToAdmin() error {
	res, err := this.isLoggedIn()
	if err != nil {
		return err
	}
	if res == true {
		return nil
	}
	hybrisAdminUrl, err := this.config.GetStringByKey("hybris admin url")
	if err != nil {
		return err
	}
	username, err := this.config.GetStringByKey("hybris admin username")
	if err != nil {
		return err
	}
	pass, err := this.config.GetStringByKey("hybris admin pass")
	if err != nil {
		return err
	}
	csrf, err := this.getCsrfTokenFromLoginPage()
	if err != nil {
		return err
	}
	yml := fmt.Sprintf(`
access url: %s/admin/j_spring_security_check
use method: post
send raw body: j_username=%s&j_password=%s&_spring_security_remember_me=on&_csrf=%s
`, hybrisAdminUrl, username, pass, csrf)
	item := CurlItem{}
	err = yaml.Unmarshal([]byte(yml), &item)
	if err != nil {
		return err
	}
	err = this.curl.RunItem(&item)
	if err != nil {
		return err
	}
	this.hybrisAdminCsrfToken = csrf
	location := this.curl.GetResponse().Header.Get("Location")
	if location != "/admin/" {
		return fmt.Errorf("login failed, got: %s", this.curl.GetResponseString())
	}
	cookie := this.curl.GetResponse().Header.Get("Set-Cookie")
	if !strings.Contains(cookie, "JSESSIONID") {
		return fmt.Errorf("missing Set-Cookie header from response")
	}
	pieces := strings.Split(cookie, ";")
	this.hybrisAdminCookie = pieces[0]
	return nil
}

func (this *AutomatedCheckItem) seedingDataWithImpex(impex string) error {
	hybrisAdminUrl, err := this.config.GetStringByKey("hybris admin url")
	if err != nil {
		return err
	}
	lines := strings.Split(impex, "\n")
	impex = ""
	for k, v := range lines {
		if k == 0 {
			lines[k] = strings.Trim(v, " ")
		} else {
			lines[k] = "    " + strings.Trim(v, " ")
		}
	}
	impex = strings.Join(lines, "\n")
	yml := fmt.Sprintf(`
access url: %s/console/impex/import
use method: post
send request headers:
  Cookie: %s
send encoded request body in java style:
  scriptContent: |
    %s
  validationEnum: IMPORT_STRICT
  maxThreads: 1
  encoding: UTF-8
  _legacyMode: on
  _enableCodeExecution: on
  _csrf:
    - %s
    - %s
  _distributedMode: on
  _sldEnabled: on
`, hybrisAdminUrl, this.hybrisAdminCookie, impex, this.hybrisAdminCsrfToken, this.hybrisAdminCsrfToken)
	item := CurlItem{}
	err = yaml.Unmarshal([]byte(yml), &item)
	if err != nil {
		return err
	}
	err = this.curl.RunItem(&item)
	if err != nil {
		return err
	}
	res := this.curl.GetResponse()
	if res.StatusCode != 200 {
		text, err := httputil.DumpResponse(res, true)
		if err != nil {
			return err
		}
		return fmt.Errorf("error when seeding data: %s, got: %s", impex, text)
	}
	text, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if strings.Contains(fmt.Sprintf("%s", text), "Import finished successfully") == false {
		return fmt.Errorf("expected: Import finished successfully, got: %s", text)
	}
	return nil
}

func (this *AutomatedCheckCollection) Run(formula string) error {
	item := this.GetItem(formula)
	if item == nil {
		return fmt.Errorf("automated check config key %s does not exist", formula)
	}
	return item.Run(this.writer, this)
}

func (this *AutomatedCheckItem) Run(writer IAutomatedCheckWriter, parent *AutomatedCheckCollection) error {
	//var this AutomatedCheckItem
	//var ok bool
	//if this, ok = this.yml[formula]; !ok {
	//	return fmt.Errorf("key does not exist")
	//}
	for _, step := range this.StepsToVerify {
		if step.RunIntegrationTestFor != nil {
			writer("- run integration test for: " + *step.RunIntegrationTestFor + "\n")
			item := parent.GetItem(*step.RunIntegrationTestFor)
			if item == nil {
				return fmt.Errorf("automated check config key %s does not exist", *step.RunIntegrationTestFor)
			}
			err := item.Run(writer, parent)
			if err != nil {
				return err
			}
		}
		if step.SeedingDataWithImpex != nil {
			writer("- seeding data with impex\n")
			err := this.seedingDataWithImpex(*step.SeedingDataWithImpex)
			if err != nil {
				return err
			}
		}
		if step.RunAllImpexFromDirectory != nil {
			writer("- run all impex from directory: " + *step.RunAllImpexFromDirectory + "\n")
			err := filepath.Walk(*step.RunAllImpexFromDirectory, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				b, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				writer("- executing impex: " + path + "\n")
				return this.seedingDataWithImpex(string(b))
			})
			if err != nil {
				return err
			}
		}
		// query database
		if step.QueryDatabase != nil {
			//err := this.autoLoginToAdmin()
			//if err != nil {
			//	return err
			//}
			writer("- query database\n")
			hybrisAdminUrl, err := parent.config.GetStringByKey("hybris admin url")
			if err != nil {
				return err
			}
			str := *step.QueryDatabase
			cmd := exec.Command("hsqldb-sqltool", "--autoCommit", "--inlineRc", "url=jdbc:hsqldb:hsql://127.0.0.1:9003/mydb,user=sa,password=", "--sql", str+";")
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("ERROR: %s, OUTPUT: %s\n", err.Error(), out)
			}
			err = parent.curl.RunItem(&CurlItem{
				AccessUrl:                             "POST " + hybrisAdminUrl + "/monitoring/cache/regionCache/clear",
				UseBasicAuthentication:                "",
				UseBearerAuthorizationTokenFromConfig: "",
				SendRawBody:                           "",
				SendEncodedRequestBodyInJavaStyle:     nil,
				SendFormData:                          nil,
				SendAdditionalPathParams:              nil,
			})
			if err != nil {
				return err
			}
			res := parent.curl.GetResponse()
			if res.StatusCode != 200 {
				dump, err := httputil.DumpResponse(res, true)
				if err != nil {
					return err
				}
				return fmt.Errorf("cannot clear hybris cache, dumping response: %s", string(dump))
			}
		}
		if step.DoHttpRequestFromCurlConfig != nil {
			parent.writer("- do http request from curl config\n")
			err := parent.curl.RunForKey(*step.DoHttpRequestFromCurlConfig)
			if err != nil {
				return err
			}
			this.res = this.curl.GetResponse()
			data, err := ioutil.ReadAll(this.res.Body)
			if err != nil {
				return err
			}
			this.body = fmt.Sprintf("%s", data)
			continue
		}
		if step.DoHttpRequestFromCurl != nil {
			this.writer("- do http request from curl\n")
			item, err := this.curl.GetItem(step.DoHttpRequestFromCurl.ConfigKey)
			if err != nil {
				return err
			}
			item.SendAdditionalPathParams = common.MapStringInterfaceToMapStringString(step.DoHttpRequestFromCurl.AddPathParams)
			this.curl.DisableVerbose()
			err = this.curl.RunItem(item)
			if err != nil {
				return err
			}
			this.res = this.curl.GetResponse()
			data, err := ioutil.ReadAll(this.res.Body)
			if err != nil {
				return err
			}
			this.body = fmt.Sprintf("%s", data)
			continue
		}
		if step.SeeHttpResponseCode != nil {
			this.writer("- see http response code\n")
			if this.res.StatusCode != *step.SeeHttpResponseCode {
				return fmt.Errorf("expected status code: %d, got: %d, response body: %s", *step.SeeHttpResponseCode, this.res.StatusCode, this.body)
			}
		}
		if step.SeeSubstring != nil {
			this.writer("- see substring\n")
			if strings.Contains(this.body, *step.SeeSubstring) == false {
				return fmt.Errorf("expected substring: %s, got response body: %s", *step.SeeSubstring, this.body)
			}
		}
		if step.SeeJsonString != nil {
			this.writer("- see json string\n")
			format, err := common.DiffJsonObjectString(*step.SeeJsonString, this.body)
			if err != nil {
				return err
			}
			if format != "" {
				return fmt.Errorf("expected json string: %s, got: %s", *step.SeeJsonString, format)
			}
		}
		if step.NotSeeSubstring != nil {
			this.writer("- not see substring\n")
			if strings.Contains(this.body, *step.NotSeeSubstring) != false {
				return fmt.Errorf("expected not containing substring: %s, got response body: %s", *step.NotSeeSubstring, this.body)
			}
		}
		if step.SeeJsonStringFor != nil {
			this.writer("- see json string for\n")
			body, err := common.GetJsonValueFromKeyChain([]byte(this.body), step.SeeJsonStringFor.JsonKey)
			if err != nil {
				return err
			}
			format, err := common.DiffJsonObjectString(step.SeeJsonStringFor.ExpectedJsonString, string(body))
			if err != nil {
				return err
			}
			if len(format) > 0 {
				return fmt.Errorf("expected json string: %s, got: %s", *step.SeeJsonString, format)
			}
		}
	}
	return nil
}

func VerifyHttpResponseCode(res *http.Response, expected int) {
	if res.StatusCode != expected {
		b, err := ioutil.ReadAll(res.Body)
		common.PanicOnError(err)
		panic(fmt.Errorf("expected response code: %d, got: %d, response body: %s\n", expected, res.StatusCode, string(b)))
	}
}
func VerifyHttpResponseBody(res *http.Response, expected string) {
	b, err := httputil.DumpResponse(res, true)
	common.PanicOnError(err)
	body := string(b)
	if body != expected {
		panic(fmt.Errorf("expected response body: %s, got: %s\n", expected, string(b)))
	}
}

func VerifyHttpResponseBodyContainSubstring(res *http.Response, expected string) {
	b, err := httputil.DumpResponse(res, true)
	common.PanicOnError(err)
	body := string(b)
	if !strings.Contains(body, expected) {
		panic(fmt.Errorf("expected response body containing substring: %s, got: %s\n", expected, string(b)))
	}
}

func VerifyArrayContainsStringKey(m map[string]interface{}, field string, expectedValue string) {
	actualValue := m[field].(string)
	if actualValue != expectedValue {
		panic(fmt.Errorf("expected %s = %s, got %s, data: %+v", field, expectedValue, actualValue, m))
	}
}
