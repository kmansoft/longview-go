package main

import (
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const (
	CLIENT_VERSION = "0.1.0"
)

func GetDataSysInfo(data *Data) error {
	err := getLinuxDistInfo(data)
	if err != nil {
		return err
	}

	return nil
}

func getLinuxDistInfo(data *Data) error {
	// Distro name and version
	bytes, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return err
	}

	for _, l := range strings.Split(string(bytes), "\n") {
		i := strings.Index(l, "=")
		if i > 0 {
			key := l[0:i]
			value := l[i+1:]

			j := len(value)
			if j >= 2 && value[0] == '"' && value[j-1] == '"' {
				value = value[1 : j-1]
			}

			if key == "NAME" {
				data.Instant["SysInfo.os.dist"] = value
			} else if key == "VERSION_ID" {
				data.Instant["SysInfo.os.distversion"] = value
			}
		}
	}

	// Kernel stuff (from uname)
	var u syscall.Utsname
	err = syscall.Uname(&u)
	if err == nil {
		data.Instant["SysInfo.kernel"] = charsToString(u.Sysname) + " " + charsToString(u.Release)
	}

	// OS and its architecture
	data.Instant["SysInfo.type"] = runtime.GOOS
	data.Instant["SysInfo.arch"] = runtime.GOARCH

	// Hostname
	host, err := os.Hostname()
	if err == nil {
		data.Instant["SysInfo.hostname"] = host
	}

	// Version of this client
	data.Instant["SysInfo.client"] = CLIENT_VERSION

	// Processor name
	psCpuInfo, err := ReadProcFSFile("cpuinfo")
	if err != nil {
		return err
	}
	modelName, _ := psCpuInfo.GetStringValue(`model name\s+:`)
	data.Instant["SysInfo.cpu.type"] = modelName

	// Processor count
	cpuCount := 0
	for _, l := range psCpuInfo.GetAsLines() {
		if strings.HasPrefix(l, "processor") {
			cpuCount += 1
		}
	}
	data.Instant["SysInfo.cpu.cores"] = cpuCount

	// Uptime
	psUptime, err := ReadProcFSFile("uptime")
	if err != nil {
		return err
	}
	line1 := strings.Split(psUptime.GetAsString(), " ")
	if len(line1) > 0 {
		uptime, _ := strconv.ParseFloat(line1[0], 64)
		data.Instant["Uptime"] = uptime
	}

	return nil
}

func charsToString(ca [65]int8) string {
	s := make([]byte, len(ca))
	var lens int
	for ; lens < len(ca); lens++ {
		if ca[lens] == 0 {
			break
		}
		s[lens] = uint8(ca[lens])
	}
	return string(s[0:lens])
}
