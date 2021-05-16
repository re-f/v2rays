package main

import (
	"fmt"
	"os"
	"text/template"
)

type Outbound struct {
	Address    string
	Port       int
	Tag        string
	Path       string
	Host       string
	ServerName string
}

func main() {
	out := Outbound{
		Address:    "v2ray.address",
		Port:       25001,
		Tag:        "node1",
		Path:       "v2ray",
		Host:       "v2ray.hos",
		ServerName: "v2ray.address",
	}
	outbounds := []Outbound{out}
	tpl := template.New("outbound")
	tpl = template.Must(tpl.Parse(tplString))

	if err := tpl.Execute(os.Stdout, outbounds); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
          "address" : "{{.Address}}",
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
    "tag" : "{{.Tag}}",
    "streamSettings" : {
      "wsSettings" : {
        "path" : "\/{{.Path}}",
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
        "serverName" : "{{.ServerName}}",
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
