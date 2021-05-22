package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"
	"v2rayS/ticker"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	subscribeUrl string
	rootCmd      = &cobra.Command{
		Use:   "v2rayS",
		Short: "v2rayS",
		Long:  `v2rayS`,
		// SilenceUsage:  true,
		// SilenceErrors: true,
	}
	syncConfigCmd = &cobra.Command{
		Use:           "update",
		Short:         "update config from subscription",
		Long:          `update config from subscription`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateV2rayConifg()
		}}

	serverCmd = &cobra.Command{
		Use:           "server",
		Short:         "v2ray update config server",
		Long:          `v2ray update config server`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {

			ticker := ticker.NewTickerE(2*time.Second, 10*time.Second)

			go func() { //fake kill signal
				time.Sleep(30 * time.Second)
				ticker.Stop("received kill signal")
			}()

			<-ticker.Run(func(msg string) error {
				fmt.Println(msg)
				return updateV2rayConifg()
			})
			fmt.Println("Stopped")
		}}
)

func init() {
	serverCmd.PersistentFlags().StringVarP(&subscribeUrl, "subscribeUrl", "s", "", "subscrib url (required)")
	serverCmd.MarkPersistentFlagRequired("subscribeUrl")
	rootCmd.AddCommand(syncConfigCmd)
	rootCmd.AddCommand(serverCmd)

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

func killV2ray() error {
	oldV2rayPid := 0
	{ // find old v2ray pid
		pgrepPath, err := exec.LookPath("pgrep")
		if err != nil {
			return errors.Wrap(err, "can't find pgrep in $PATH, got error:")
		}

		v2rayPidRaw := ""
		if out, err := exec.Command(pgrepPath, "v2ray$").Output(); err != nil {
			return errors.Wrapf(err, "find old v2ray process got error, and output is %v", v2rayPidRaw)
		} else {
			v2rayPidRaw = strings.TrimSpace(string(out))
		}

		fmt.Printf("pgrep v2ray output: %v\n", v2rayPidRaw)

		if pLen := len(strings.Split(v2rayPidRaw, "\n")); pLen == 0 {
			fmt.Println("Warning: v2ray not runing before")
			return nil
		} else if pLen == 1 {
			pid, err := strconv.Atoi(v2rayPidRaw)
			if err != nil {
				return errors.Wrapf(err, "pgrep v2ray get non-numeric output: %v", v2rayPidRaw)
			}
			oldV2rayPid = pid
		} else {
			return errors.New(fmt.Sprintf("expect 1 v2ray process, but got %v", pLen))
		}
	}

	fmt.Printf("PID: %d, Name: v2ray will be killed.\n", oldV2rayPid)
	proc, err := os.FindProcess(oldV2rayPid)
	if err != nil {
		return errors.Wrap(err, "find v2ray(%v) got error: ")
	}
	// Kill the process
	return proc.Kill()
}

func updateV2rayConifg() error {
	if err := killV2ray(); err != nil {
		return errors.Wrap(err, "kill v2ray process got error: ")
	}
	vmessList, err := getVmssListUrlsFromUrl(subscribeUrl)
	if err != nil {
		return err
	}

	tplString, err := ioutil.ReadFile("config.json.tmpl")
	if err != nil {
		return errors.Wrap(err, "Read template file get error: ")
	}

	tpl := template.Must(template.New("outbound").Funcs(template.FuncMap{"separator": separator}).Parse(string(tplString)))
	confBytes := new(bytes.Buffer)
	if err := tpl.Execute(confBytes, vmessList); err != nil {
		return err
	}
	ioutil.WriteFile("config.json", confBytes.Bytes(), 0755)
	return nil
}
