package common

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"github.com/tealeg/xlsx"
)

const (
	StdLongMonth      = "January"
	StdMonth          = "Jan"
	StdNumMonth       = "1"
	StdZeroMonth      = "01"
	StdLongWeekDay    = "Monday"
	StdWeekDay        = "Mon"
	StdDay            = "2"
	StdUnderDay       = "_2"
	StdZeroDay        = "02"
	StdHour           = "15"
	StdHour12         = "3"
	StdZeroHour12     = "03"
	StdMinute         = "4"
	StdZeroMinute     = "04"
	StdSecond         = "5"
	StdZeroSecond     = "05"
	StdLongYear       = "2006"
	StdYear           = "06"
	StdPM             = "PM"
	Stdpm             = "pm"
	StdTZ             = "MST"
	StdISO8601TZ      = "Z0700"  // prints Z for UTC
	StdISO8601ColonTZ = "Z07:00" // prints Z for UTC
	StdNumTZ          = "-0700"  // always numeric
	StdNumShortTZ     = "-07"    // always numeric
	StdNumColonTZ     = "-07:00" // always numeric
)

type ProxyWriter struct {
	realWriter IWriter
}

func NewProxyWriter(writer IWriter) *ProxyWriter {
	return &ProxyWriter{realWriter: writer}
}
func(this *ProxyWriter) Write(p []byte) (n int, err error) {
	this.realWriter(string(p))
	return len(p), nil
}

type IWriter func(str string)

type CommandHandler func(w IWriter, param string, forceStop chan bool) error


const IsoDateFormat = StdLongYear + "-" + StdZeroMonth + "-" + StdZeroDay

func MapInterfaceInterfaceToMapStringString(input map[interface{}]interface{}) map[string]string {
	output := map[string]string{}
	for k, v := range input {
		output[fmt.Sprintf("%v",k)] = fmt.Sprintf("%v", v)
	}
	return output
}
func MapInterfaceStringToMapStringString(input map[interface{}]string) map[string]string {
	output := map[string]string{}
	for k, v := range input {
		output[fmt.Sprintf("%v",k)] = v
	}
	return output
}
func MapStringInterfaceToMapStringString(input map[string]interface{}) map[string]string {
	output := map[string]string{}
	for k, v := range input {
		output[k] = fmt.Sprintf("%v", v)
	}
	return output
}
func ArrayInterfaceToArrayString(input []interface{}) []string {
	var output []string
	for _, v := range input {
		output = append(output, fmt.Sprintf("%v", v))
	}
	return output
}
func MapStringStringToUrlValues(input map[string]string) url.Values {
	output := url.Values{}
	for k, v := range input {
		output.Set(k, v)
	}
	return output
}
func IsInteger(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}
func DiffJsonObjectString(expected string, actual string) (string, error) {
	if !IsJsonObjectString(expected) {
		return "", fmt.Errorf("DiffJsonObjectString only support json object string")
	}
	if !IsJsonObjectString(expected) {
		return "", fmt.Errorf("DiffJsonObjectString only support json object string")
	}
	differ := gojsondiff.New()
	diff, err := differ.Compare([]byte(actual), []byte(expected))
	if err != nil {
		return "", err
	}
	if !diff.Modified() {
		return "", nil
	}
	var aJson map[string]interface{}
	err = json.Unmarshal([]byte(expected), &aJson)
	if err != nil {
		return "", err
	}
	config := formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
		Coloring:       false,
	}
	formatter := formatter.NewAsciiFormatter(aJson, config)
	diffString, err := formatter.Format(diff)
	if err != nil {
		return "", err
	}
	return diffString, nil
}
//func ValidateJson(str []byte) bool {
//	var tmp []interface{}
//	if err := json.Unmarshal(str, &tmp); err == nil {
//		return true
//	}
//	tmp2 := map[string]interface{}{}
//	if err := json.Unmarshal(str, &tmp2); err == nil {
//		return true
//	}
//	tmp3 := interface{}{}
//	if err := json.Unmarshal(str, &tmp2); err == nil {
//		return true
//	}
//}
func IsJsonArrayString(str string) bool {
	var tmp []interface{}
	err := json.Unmarshal([]byte(str), &tmp)
	return err == nil
}
func IsJsonObjectString(str string) bool {
	var tmp map[string]interface{}
	err := json.Unmarshal([]byte(str), &tmp)
	return err == nil
}
func GetJsonValueFromKeyChain(body []byte, keyChain string) ([]byte, error) {
	pieces := strings.Split(keyChain, ".")
	var item interface{}
	for _, v := range pieces {
		if val, err := strconv.Atoi(v); err == nil {
			var tmp []interface{}
			err := json.Unmarshal(body, &tmp)
			if err != nil {
				return nil, err
			}
			item = tmp[val]
		} else {
			tmp := map[string]interface{}{}
			err := json.Unmarshal(body, &tmp)
			if err != nil {
				return nil, err
			}
			item = tmp[v]
		}
		b, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		body = b
	}
	return body, nil
}
func StringArrayMerge(m1, m2 map[string]string) map[string]string {
	res := map[string]string{}
	for k, v := range m1 {
		res[k] = v
	}
	for k, v := range m2 {
		res[k] = v
	}
	return res
}

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func FetchAll(db *sql.DB, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err //most likely a query error
	}
	defer rows.Close() // close when we leave the method..

	columnTypes, err := rows.ColumnTypes()
	columnNames, err := rows.Columns()

	var resRows []map[string]interface{}          //row container
	rawResult := make([][]byte, len(columnNames)) // Result is our slice string/int/whatever.
	dest := make([]interface{}, len(columnNames)) // An interface{} slice
	for i := range rawResult {
		dest[i] = &rawResult[i] // pointers to each string in the result interface slice
	}
	for rows.Next() {
		err = rows.Scan(dest...)
		if err != nil {
			fmt.Println("Failed to scan row", err)
			return nil, err
		}
		resRow := make(map[string]interface{})
		for i, raw := range rawResult {
			if raw == nil {
				resRow[columnNames[i]] = "" // usually NULL value, output "" (lazy)
			} else {
				switch columnTypes[i].DatabaseTypeName() { //handle database types. i guess there's a better way through columnTypes[i].scan.., but i need to get it working now
				case "NVARCHAR":
					resRow[columnNames[i]] = string(raw)
				case "INT":
					s := string(raw)
					ss, err := strconv.Atoi(s)
					if err != nil {
						fmt.Printf("cant convert to int")
					}
					resRow[columnNames[i]] = ss
				default:
					resRow[columnNames[i]] = string(raw) //tired to website
				}
			}
		}
		resRows = append(resRows, resRow)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return resRows, nil
}

func RunLinuxCommandByCsvWithDirectory(dir string, command string, writer IWriter, forceStop chan bool) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = dir
	proxyWriter := NewProxyWriter(writer)
	cmd.Stdout = proxyWriter
	cmd.Stderr = proxyWriter
	err := cmd.Start()
	if err != nil {
		return err
	}
	finishChan := make(chan error)
	go func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			finishChan <- err
			return
		}
		finishChan <- nil
	}(cmd)
	go func() {
		select {
		case <-forceStop:
			finishChan <- cmd.Process.Kill()
		}
	}()
	return <-finishChan
}

func RunLinuxCommandWithDirectory(dir string, command string, writer IWriter, forceStop chan bool) error {
	return RunLinuxCommandByCsvWithDirectory(dir, command, writer, forceStop)
}

func RunLinuxCommand(command string, writer IWriter, forceStop chan bool) error {
	return RunLinuxCommandWithDirectory("", command, writer, forceStop)
}

func RunBashScript(script string, workDir string, writer IWriter, forceStop chan bool) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(wd + "/test.sh", []byte(script), 0777)
	if err != nil {
		return err
	}
	if workDir != "" {
		return RunLinuxCommand("cd " + wd + " && ./test.sh", writer, forceStop)
	}
	return RunLinuxCommand("./test.sh", writer, forceStop)
}

func GenerateXLSXFromCSV(csvPath string, XLSXPath string, delimiter string) error {
	csvFile, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	if len(delimiter) > 0 {
		reader.Comma = rune(delimiter[0])
	} else {
		reader.Comma = rune(';')
	}
	xlsxFile := xlsx.NewFile()
	sheet, err := xlsxFile.AddSheet(csvPath)
	if err != nil {
		return err
	}
	fields, err := reader.Read()
	for err == nil {
		row := sheet.AddRow()
		for _, field := range fields {
			cell := row.AddCell()
			cell.Value = field
		}
		fields, err = reader.Read()
	}
	if err != nil {
		fmt.Printf(err.Error())
	}
	return xlsxFile.Save(XLSXPath)
}

// CreateFormFile is a convenience wrapper around CreatePart. It creates
// a new form-data header with the provided field name and file name.
func CreateFormFileWithContentType(fieldname, filename string, w *multipart.Writer, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			fieldname, filename))
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}
