package main

type VmessInfo struct {
	Ps   string `json:"ps"`
	Add  string `json:"add"`
	Port int    `json:"port"`
	ID   string `json:"id"`
	Aid  int    `json:"aid"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	TLS  string `json:"tls"`
	Path string `json:"path"`
}
