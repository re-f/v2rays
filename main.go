package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

func main() {
	subscribeUrl := ""
	vmessList, err := getVmssListUrlsFromUrl(subscribeUrl)
	ExitIfError(err)
	tplString, err := ioutil.ReadFile("config.json.tmpl")
	ExitIfError(errors.Wrap(err, "Read template file get error: "))
	tpl := template.Must(template.New("outbound").Funcs(template.FuncMap{"separator": separator}).Parse(string(tplString)))
	confBytes := new(bytes.Buffer)
	if err := tpl.Execute(confBytes, vmessList); err != nil {
		ExitIfError(err)
	}

	ioutil.WriteFile("config.json", confBytes.Bytes(), 0755)
}

func parseVmessUrl(vmessUrl string) (VmessInfo, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if len(vmessUrl) <= 0 {
		return VmessInfo{}, errors.New("Empty vmess URL string")
	}
	if !strings.HasPrefix(vmessUrl, "vmess://") {
		return VmessInfo{}, errors.New("Invalid vmess URL string")
	}
	vmDec, err := base64.StdEncoding.DecodeString(vmessUrl[8:])
	if err != nil && len(vmDec) > 0 {
		return VmessInfo{}, errors.Wrap(err, "Decode failed with vmess URL content")
	}
	var vmObj VmessInfo
	err = json.Unmarshal(vmDec, &vmObj)
	if err != nil {
		return VmessInfo{}, errors.Wrap(err, "Decode json failed")
	}
	return vmObj, nil
}

func getVmssListUrlsFromUrl(subUrl string) (vmssList []VmessInfo, err error) {
	resp, err := http.Get(subUrl)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get subscribe URL")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get content of URL")
	}

	decodeBytes, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil && decodeBytes == nil {
		return nil, errors.Wrap(err, "Decode failed with subscribe URL content")
	}
	vms := strings.Split(string(decodeBytes), "\n")
	vmssList = make([]VmessInfo, 0)
	for _, num := range vms {
		vmObj, err := parseVmessUrl(num)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", errors.Wrap(err, fmt.Sprint("Given vmess url: ", num)))
			continue
		}
		vmssList = append(vmssList, vmObj)
	}
	if len(vmssList) == 0 {
		return nil, errors.Errorf("Get empty vmess list form subscribe URL")
	}
	return vmssList, nil
}

/**
{{$s := separator ", "}}
{{range $key, $value := $}}
{{call $s}}key:{{$key}} value:{{$value}}

{{end}}
*/

func separator(s string) func() string {
	i := -1
	return func() string {
		i++
		if i == 0 {
			return ""
		}
		return s
	}
}
