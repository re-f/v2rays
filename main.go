package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"
	"v2rayS/ticker"

	"github.com/facebookgo/pidfile"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
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

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			ticker := ticker.NewTickerE(2*time.Second, interval)
			go func() {
				sig := <-sigs
				ticker.Stop(fmt.Sprintf("received kill signal(%v)", sig))
			}()

			pidfile.Write()

			defer os.Remove(pidPath)

			<-ticker.Run(func(msg string) error {
				fmt.Println(msg)
				return updateV2rayConifg()
			})
			fmt.Println("Server stopped")
		}}
)
var (
	subscribeUrl string
	pidPath      string
	interval     time.Duration
	configPath   string
	tmplPath     string
)

func init() {

	ex, err := os.Executable()
	if err != nil {
		panic(fmt.Sprintf("find current execute file path got error: %v", err))
	}

	serverCmd.PersistentFlags().StringVarP(&subscribeUrl, "subscribeUrl", "s", "", "subscribe url (required)")

	initUpdateFlagFn := func(flagSet *pflag.FlagSet) {
		flagSet.DurationVarP(&interval, "interval", "i", 1*time.Hour, "update config interval")
		flagSet.StringVarP(&configPath, "config", "c", "/root/.config/v2ray/config.json", "target v2ray config.json path")
		flagSet.StringVarP(&tmplPath, "template", "t", filepath.Join(filepath.Dir(ex), "config.json.tmpl"), "config.json.tmpl file path")
	}
	initUpdateFlagFn(serverCmd.PersistentFlags())
	initUpdateFlagFn(syncConfigCmd.PersistentFlags())

	serverCmd.MarkPersistentFlagRequired("subscribeUrl")
	rootCmd.AddCommand(syncConfigCmd)
	rootCmd.AddCommand(serverCmd)
	pidDir, _ := os.Getwd()
	pidPath = filepath.Join(pidDir, "v2raS.pid")
	pidfile.SetPidfilePath(pidPath)
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
			// fmt.Fprintf(os.Stderr, "%v\n", errors.Wrap(err, fmt.Sprint("Given vmess url: ", num)))
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

func runV2ray() error {
	v2rayPath, err := exec.LookPath("v2ray")
	if err != nil {
		return errors.Wrap(err, "can't find v2ray in $PATH, you should install v2ray first")
	}
	cmd := exec.Command(v2rayPath, "-config", configPath)
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "run v2ray failed, got error :")
	}
	return nil
}

func killV2ray() error {
	oldV2rayPid := 0
	{ // find old v2ray pid
		pgrepPath, err := exec.LookPath("pgrep")
		if err != nil {
			return errors.Wrap(err, "can't find pgrep in $PATH, got error:")
		}

		out, err := exec.Command(pgrepPath, "v2ray$").Output()
		v2rayPidRaw := strings.TrimSpace(string(out))
		if err != nil {
			if strings.Contains(err.Error(), "exit status 1") { //
				// according to `man pgrep`, the better solution is to judge if exit code is equal to 1, but it is too rough to write in golang, so i use a bad IF condition
				fmt.Println("Warning: v2ray not runing, skip kill")
				return nil
			} else {
				return errors.Wrapf(err, "find old v2ray process got error, and output is %v", v2rayPidRaw)
			}
		}

		if pLen := len(strings.Split(v2rayPidRaw, "\n")); pLen == 1 {
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
	if err := proc.Kill(); err != nil {
		return errors.Wrap(err, "find v2ray(%v) got error: ")
	}
	pStat, err := proc.Wait()
	// Kill the process
	waitKillTimeOut := time.NewTimer(10 * time.Second)
	for {
		if pStat != nil {
			//map加锁、删除map中的key、解锁map
			break
		}
		select {
		case <-waitKillTimeOut.C:
			return errors.Wrap(err, "find v2ray(%v) got error: ")
		default:
			time.Sleep(10 * time.Nanosecond)
		}
	}
	return nil
}

func updateV2rayConifg() error {
	if err := killV2ray(); err != nil {
		return errors.Wrap(err, "kill v2ray process got error: ")
	}
	vmessList, err := getVmssListUrlsFromUrl(subscribeUrl)
	if err != nil {
		return err
	}

	if len(vmessList) == 0 {
		return errors.Errorf("get empty vmess url from subscribe url")
	}

	tplString, err := ioutil.ReadFile(tmplPath)
	if err != nil {
		return errors.Wrap(err, "Read template file get error: ")
	}

	tpl := template.Must(template.New("outbound").Funcs(template.FuncMap{"separator": separator}).Parse(string(tplString)))
	confBytes := new(bytes.Buffer)
	if err := tpl.Execute(confBytes, vmessList); err != nil {
		return err
	}
	ioutil.WriteFile(configPath, confBytes.Bytes(), 0755)
	fmt.Println("wrote down new ", configPath)

	return runV2ray()
}
