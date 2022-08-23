package pkg

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

func length(s string) int {
	return len(s)
}

func listLength(s []string) int {
	return len(s)
}

func list(args ...interface{}) []interface{} {
	return args
}

func seq(start, end int) []int {
	var s []int
	for i := start; i <= end; i++ {
		s = append(s, i)
	}
	return s
}

func inttostring(s int) string {
	i := strconv.Itoa(s)
	return i
}

func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}
func nindent(spaces int, v string) string {
	return "\n" + indent(spaces, v)
}

func sub(max, min string) int {
	a, _ := strconv.Atoi(max)
	b, _ := strconv.Atoi(min)
	return a - b
}

func sum(num1, num2 string) int {
	a, _ := strconv.Atoi(num1)
	b, _ := strconv.Atoi(num2)
	return a + b
}

func readFile(file string) string {
	contents, _ := os.ReadFile(file)
	return strings.TrimSuffix(string(contents), "\n")
}

func unmarshallYamlFile(file string) map[interface{}]interface{} {
	yamlFile, _ := os.ReadFile(file)
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(yamlFile, &m)
	if err != nil {
		panic(err.Error())
	}
	return m

}

func keys(dicts ...interface{}) []string {
	var k []string
	for _, value := range dicts {
		for kk := range value.(map[interface{}]interface{}) {
			k = append(k, kk.(string))
		}
	}
	return k
}

func concat(v ...interface{}) string {
	v = removeNilElements(v)
	r := strings.TrimSpace(strings.Repeat("%v ", len(v)))
	return strings.Replace(fmt.Sprintf(r, v...), " ", "", -1)
}

func removeNilElements(v []interface{}) []interface{} {
	newSlice := make([]interface{}, 0, len(v))
	for _, i := range v {
		if i != nil {
			newSlice = append(newSlice, i)
		}
	}
	return newSlice
}

func base64decode(v string) string {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func dfault(d interface{}, given ...interface{}) interface{} {

	if empty(given) || empty(given[0]) {
		return d
	}
	return given[0]
}

// empty returns true if the given value has the zero value for its type.
func empty(given interface{}) bool {
	g := reflect.ValueOf(given)
	if !g.IsValid() {
		return true
	}

	// Basically adapted from text/template.isTrue
	switch g.Kind() {
	default:
		return g.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return g.Len() == 0
	case reflect.Bool:
		return !g.Bool()
	case reflect.Complex64, reflect.Complex128:
		return g.Complex() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return g.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return g.Float() == 0
	case reflect.Struct:
		return false
	}
}

func toYaml(v interface{}) string {
	o, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(o)
}

func uniqueSlice(s1, s2 []string) []string {
	var b []string
	var l []string

	var diff []string

	if len(s1) > len(s2) {
		b = s1
		l = s2
	}

	if len(s1) < len(s2) {
		b = s2
		l = s1
	}

	var found bool

	if len(s1) == len(s2) {
		for _, i := range s1 {
			for _, j := range s2 {
				if i == j {
					found = true
					break
				} else {
					found = false
				}
			}
			if !found {
				diff = append(diff, i)
			}
		}
		for _, i := range s2 {
			for _, j := range s1 {
				if i == j {
					found = true
					break
				} else {
					found = false
				}
			}
			if !found {
				diff = append(diff, i)
			}
		}
	}

	if len(s1) != len(s2) {
		for _, i := range b {
			for _, j := range l {
				if i == j {
					found = true
					break
				} else {
					found = false
				}
			}
			if !found {
				diff = append(diff, i)
			}
		}
	}
	return diff
}

func leftUniqueSlice(s1, s2 []string) []string {
	var b []string
	var l []string

	var diff []string

	if len(s1) > len(s2) {
		b = s1
		l = s2
	}

	if len(s1) < len(s2) {
		b = s2
		l = s1
	}

	var found bool

	if len(s1) == len(s2) {
		for _, i := range s2 {
			for _, j := range s1 {
				if i == j {
					found = true
					break
				} else {
					found = false
				}
			}
			if !found {
				diff = append(diff, i)
			}
		}
	}

	if len(s1) != len(s2) {
		for _, i := range b {
			for _, j := range l {
				if i == j {
					found = true
					break
				} else {
					found = false
				}
			}
			if !found {
				diff = append(diff, i)
			}
		}
	}
	return diff
}

func getFromURL(url, username, password string) (io.ReadCloser, error) {
	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if len(username) >= 1 && len(password) >= 0 {
		req.SetBasicAuth(username, password)
	}
	if err != nil {
		return nil, err
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func getYamlFromURL(url, username, password string) interface{} {
	if len(url) == 0 {
		var t interface{}
		return t
	}
	r, err := getFromURL(url, username, password)
	if err != nil {
		log.Fatalf(err.Error())
	}
	var t interface{}

	//todo IOUTIL is deprecated
	y, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf(err.Error())
	}

	err = yaml.Unmarshal(y, &t)
	if err != nil {
		log.Fatalf(err.Error())
	}

	return t
}

func replace(old, new, src string) string {
	return strings.Replace(src, old, new, -1)
}

func Template(templateFile, renderedFile string) {
	envMap := envToMap()
	funcMap := template.FuncMap{
		"upper":           strings.ToUpper,
		"lower":           strings.ToLower,
		"split":           strings.Split,
		"len":             length,
		"list":            list,
		"seq":             seq,
		"sub":             sub,
		"sum":             sum,
		"indent":          indent,
		"nindent":         nindent,
		"readFile":        readFile,
		"inttostring":     inttostring,
		"concat":          concat,
		"unmarshallYaml":  unmarshallYamlFile,
		"keys":            keys,
		"b64dec":          base64decode,
		"b64enc":          base64encode,
		"default":         dfault,
		"empty":           empty,
		"toyaml":          toYaml,
		"uniqueSlice":     uniqueSlice,
		"leftUniqueSlice": leftUniqueSlice,
		"listLen":         listLength,
		"getYamlFromURL":  getYamlFromURL,
		"replace":         replace,
	}

	fBase := path.Base(templateFile)
	t, err := template.New(fBase).Funcs(funcMap).ParseFiles(templateFile)
	if err != nil {
		log.Fatalf(err.Error())
	}
	err = t.Execute(os.Stdout, envMap)
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func envToMap() map[string]string {
	envMap := make(map[string]string)

	for _, v := range os.Environ() {
		envVar := strings.Split(v, "=")
		envMap[envVar[0]] = os.Getenv(envVar[0])
	}

	return envMap
}
