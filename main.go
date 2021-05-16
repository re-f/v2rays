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

	tpl := template.New("outbound")
	tpl = template.Must(tpl.Parse(tplString))
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

var tplString = `{{range .}}{
    "sendThrough" : "0.0.0.0",
    "mux" : {
      "enabled" : false,
      "concurrency" : 8
    },
    "protocol" : "vmess",
    "settings" : {
      "vnext" : [
        {
          "address" : "{{.Add}}",
          "users" : [
            {
              "id" : "3c91d857-2d40-39b9-81c0-f6adde8037ff",
              "alterId" : 2,
              "security" : "auto",
              "level" : 0
            }
          ],
          "port" : {{.Port}}
        }
      ]
    },
    "tag" : "{{.Ps}}",
    "streamSettings" : {
      "wsSettings" : {
        "path" : "\{{.Path}}",
        "headers" : {
          "Host" : "{{.Host}}"
        }
      },
      "quicSettings" : {
        "key" : "",
        "header" : {
          "type" : "none"
        },
        "security" : "none"
      },
      "tlsSettings" : {
        "allowInsecure" : false,
        "alpn" : [
          "http\/1.1"
        ],
        "serverName" : "{{.Add}}",
        "allowInsecureCiphers" : false
      },
      "sockopt" : {

      },
      "httpSettings" : {
        "path" : "",
        "host" : [
          ""
        ]
      },
      "tcpSettings" : {
        "header" : {
          "type" : "none"
        }
      },
      "kcpSettings" : {
        "header" : {
          "type" : "none"
        },
        "mtu" : 1350,
        "congestion" : false,
        "tti" : 20,
        "uplinkCapacity" : 5,
        "writeBufferSize" : 1,
        "readBufferSize" : 1,
        "downlinkCapacity" : 20
      },
      "security" : "tls",
      "network" : "ws"
    }
  }
  {{end}}`
