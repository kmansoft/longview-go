package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type DataProcess struct {
	PID uint64

	LongName, Name              string
	PPID, UID                   uint64
	User                        string
	RSS                         uint64
	CPU, Age                    uint64
	IOReadKBytes, IOWriteKBytes uint64
}

type DataProcessList struct {
	plist []*DataProcess

	uptimeJiffies, ticks float64
}

func GetDataProcessesPorts(data *Data) error {

	// Processes
	processList, err := newProcessList()
	if err != nil {
		return err
	}

	for _, proc := range processList.plist {
		if proc.PID == 2 || proc.PPID == 2 {
			continue
		}

		prefix := fmt.Sprintf("Processes.%s.", proc.Name)

		data.Instant[prefix+"longname"] = proc.LongName

		prefix = fmt.Sprintf("Processes.%s.%s.", proc.Name, proc.User)

		processList.addToCount(data.Longterm, prefix+"mem", proc.RSS)
		processList.addToCount(data.Longterm, prefix+"cpu", proc.CPU)
		processList.addToCount(data.Longterm, prefix+"count", 1)
		processList.addToCount(data.Longterm, prefix+"ioreadkbytes", proc.IOReadKBytes)
		processList.addToCount(data.Longterm, prefix+"iowritekbytes", proc.IOWriteKBytes)
	}

	/*

		TODO

			for my $key ( sort keys %{ $dataref->{LONGTERM} } ) {
				next unless $key =~ /^Processes/;
				if($key =~ /zz_age$/ ){
					delete $dataref->{LONGTERM}->{$key};
					next;
				}
				next if ($key =~ m/^Processes\.apt(?:-get|itude)/);
				my $age_key = $key;
				$age_key =~ s/[^\.]*$/zz_age/;
				delete $dataref->{LONGTERM}->{$key} if ($dataref->{LONGTERM}->{$age_key} < 60);
			}

	*/

	// Ports (network connections)
	networkList, err := newNetworkList()
	if err != nil {
		return err
	}

	active := make(map[string]*DataActive)
	listen := make(map[string]*DataListen)

	myPid := os.Getpid()

	for _, proc := range processList.plist {
		if proc.PID == 2 || proc.PPID == 2 {
			continue
		}

		if int(proc.PID) == myPid {
			continue
		}

		files, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", proc.PID))
		if err != nil {
			continue
		}

		for _, fd := range files {
			if rl, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", proc.PID, fd.Name())); err == nil {
				if len(rl) > 0 && strings.HasPrefix(rl, "socket:") {
					socket := rl[7:]
					sockl := len(socket)
					if sockl > 2 && socket[0] == '[' && socket[sockl-1] == ']' {
						socket = socket[1 : sockl-1]
						if inode, err := strconv.ParseUint(socket, 10, 64); err == nil {
							if network, ok := networkList.networkByINode[inode]; ok {

								if network.isListening {
									// Listening
									key := fmt.Sprintf("%s.%s.%s.%s.%d",
										proc.Name, proc.User,
										network.t, network.srcIP.String(), network.srcPort)
									if _, ok := listen[key]; !ok {
										listen[key] = &DataListen{
											Name: proc.Name, User: proc.User,
											T:     network.t,
											SrcIP: network.srcIP, SrcPort: network.srcPort}
									}
								} else {
									// Active
									key := fmt.Sprintf("%s.%s",
										proc.Name, proc.User)
									if activeItem, ok := active[key]; ok {
										activeItem.Count += 1
									} else {
										active[key] = &DataActive{
											Name: proc.Name, User: proc.User,
											Count: 1,
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	activeList := make([]*DataActive, 0)
	for _, activeItem := range active {
		activeList = append(activeList, activeItem)
	}

	listenList := make([]*DataListen, 0)
	for _, listenItem := range listen {
		listenList = append(listenList, listenItem)
	}

	data.Instant["Ports.active"] = activeList
	data.Instant["Ports.listening"] = listenList

	return nil
}

func newProcessList() (*DataProcessList, error) {
	list := &DataProcessList{
		plist: make([]*DataProcess, 0),
	}

	err := list.loadProcessList()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (list *DataProcessList) loadProcessList() error {
	expr := regexp.MustCompile(`\d+`)

	// Get uptime
	if uptime, err := ioutil.ReadFile("/proc/uptime"); err == nil {
		l := strings.Fields(string(uptime))
		if len(l) > 0 {
			if uptimeJiffies, err := strconv.ParseFloat(l[0], 64); err == nil {
				list.uptimeJiffies = uptimeJiffies
			}
		}
	}

	// Get ticks
	getconfCommand := exec.Command("getconf", "CLK_TCK")
	getconfOut := new(bytes.Buffer)
	getconfCommand.Stdout = getconfOut
	_ = getconfCommand.Run()
	if ticks, err := strconv.ParseFloat(strings.TrimSpace(getconfOut.String()), 64); err == nil {
		list.ticks = ticks
	}

	// Read /proc and save all processes
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return err
	}

	for _, f := range files {
		n := f.Name()
		if expr.MatchString(n) {
			pid, err := strconv.ParseUint(n, 10, 64)
			if err != nil {
				continue
			}

			item := list.getProcess(pid)
			if item != nil {
				list.plist = append(list.plist, item)
			}
		}
	}

	return nil
}

func (list *DataProcessList) getProcess(pid uint64) *DataProcess {
	prefix := fmt.Sprintf("/proc/%d/", pid)

	if status, err := ioutil.ReadFile(prefix + "status"); err == nil {
		if stat, err := ioutil.ReadFile(prefix + "stat"); err == nil {
			if cmdline, err := ioutil.ReadFile(prefix + "cmdline"); err == nil {
				io, err := ioutil.ReadFile(prefix + "io")
				if err != nil {
					io = nil
				}

				//if ( $line =~ m/^Name:\s+(.*)/ )         { $proc{name} = $1; next; }
				//if ( $line =~ m/^PPid:\s+(.*)/ )         { $proc{ppid} = $1; next; }
				//if ( $line =~ m/^Uid:\s+.*?\s+(.*?)\s/ ) { $proc{uid}  = $1; next; }
				//if ( $line =~ m/^VmRSS:\s+(.*)\s+kB/ )   { $proc{mem}  = $1; last; }

				item := &DataProcess{
					PID:      pid,
					LongName: list.getLongName(cmdline),
				}

				// Parse "status"
				for _, l := range strings.Split(string(status), "\n") {
					i := strings.IndexByte(l, ':')
					if i > 0 {
						key := l[:i]
						val := strings.TrimSpace(l[i+1:])

						switch key {
						case "Name":
							item.Name = val
						case "PPid":
							item.PPID, _ = strconv.ParseUint(val, 10, 64)
						case "Uid":
							item.UID, _ = strconv.ParseUint(list.getField(val, 1), 10, 64)
						case "VmRSS":
							item.RSS, _ = strconv.ParseUint(list.getField(val, 0), 10, 64)
						}
					}
				}

				if item.UID >= 0 {
					u, err := user.LookupId(strconv.FormatUint(item.UID, 10))
					if err == nil {
						item.User = u.Username
					}
				}

				// Parse "stat"
				f := strings.Fields(string(stat))
				if len(f) > 21 {
					userTime, _ := strconv.ParseUint(f[13], 10, 64)
					systemTime, _ := strconv.ParseUint(f[14], 10, 64)
					jiffies, _ := strconv.ParseUint(f[21], 10, 64)

					item.CPU = userTime + systemTime
					item.Age = list.procAge(jiffies)
				}

				// Parse "io"
				for _, l := range strings.Split(string(io), "\n") {
					i := strings.IndexByte(l, ':')
					if i > 0 {
						key := l[:i]
						val := strings.TrimSpace(l[i+1:])

						switch key {
						case "read_bytes":
							item.IOReadKBytes, _ = strconv.ParseUint(val, 10, 64)
						case "write_bytes":
							item.IOWriteKBytes, _ = strconv.ParseUint(val, 10, 64)
						}
					}
				}

				return item
			}
		}
	}

	return nil
}

func (list *DataProcessList) getLongName(l []byte) string {
	for i := 0; i < len(l); i += 1 {
		if l[i] == 0 {
			return string(l[0:i])
		}
	}
	return string(l)
}

func (list *DataProcessList) getField(s string, n int) string {
	l := strings.Fields(s)
	if len(l) > n {
		return l[n]
	}
	return ""
}

func (list *DataProcessList) procAge(startJiffies uint64) uint64 {
	if list.uptimeJiffies > 0 && list.ticks > 0 {
		currentJiffies := list.uptimeJiffies * list.ticks
		return uint64((currentJiffies - float64(startJiffies)) / list.ticks)
	}

	return 0
}

func (list *DataProcessList) addToCount(m map[string]interface{}, key string, value uint64) {
	if count, ok := m[key]; ok {
		m[key] = count.(uint64) + value
	} else {
		m[key] = value
	}
}

type DataNetworkList struct {
	needFlip       bool
	networkByINode map[uint64]DataNetwork
}

type DataNetwork struct {
	t                string
	srcIP, dstIP     net.IP
	srcPort, dstPort uint16
	isListening      bool
}

type DataListen struct {
	User    string `json:"user"`
	Name    string `json:"name"`
	T       string `json:"type"`
	SrcIP   net.IP `json:"ip"`
	SrcPort uint16 `json:"port"`
}

type DataActive struct {
	User  string `json:"user"`
	Name  string `json:"name"`
	Count uint64 `json:"count"`
}

func newNetworkList() (*DataNetworkList, error) {
	list := &DataNetworkList{}

	err := list.loadNetworkCache()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (list *DataNetworkList) loadNetworkCache() error {
	arch := runtime.GOARCH

	list.needFlip = arch == "amd64" || arch == "x86"
	list.networkByINode = make(map[uint64]DataNetwork)

	for _, t := range []string{"tcp", "tcp6", "udp", "udp6"} {
		data, err := ioutil.ReadFile(fmt.Sprintf("/proc/net/%s", t))
		if err != nil {
			return err
		}

		for i, l := range strings.Split(string(data), "\n") {
			if i == 0 {
				continue
			}

			f := strings.Fields(l)
			if len(f) >= 10 {
				inode, err := strconv.ParseUint(f[9], 10, 64)
				if err == nil && inode > 0 {
					srcIP, srcPort, err := list.parse(f[1])
					if err != nil {
						continue
					}

					dstIP, dstPort, err := list.parse(f[2])
					if err != nil {
						continue
					}

					network := DataNetwork{
						t:     t,
						srcIP: srcIP, srcPort: srcPort,
						dstIP: dstIP, dstPort: dstPort,
						isListening: dstIP.IsUnspecified() && dstPort == 0,
					}

					list.networkByINode[inode] = network
				}
			}
		}
	}

	return nil
}

var (
	errInvalidAddress = errors.New("Invalid addresss")
)

func (list *DataNetworkList) parse(s string) (net.IP, uint16, error) {
	i := strings.IndexByte(s, ':')
	if i <= 0 {
		return nil, 0, errInvalidAddress
	}

	addr := s[:i]
	port := s[i+1:]

	l := len(addr)
	if l != 8 && l != 32 {
		return nil, 0, errInvalidAddress
	}

	valueIP, err := hex.DecodeString(addr)
	if err != nil {
		return nil, 0, err
	}

	valuePort, err := strconv.ParseUint(port, 16, 16)
	if err != nil {
		return nil, 0, err
	}

	if list.needFlip {
		list.flip(valueIP)
	}

	return valueIP, uint16(valuePort), nil
}

func (list *DataNetworkList) flip(l []uint8) {
	i := 0
	j := len(l) - 1
	for i < j {
		l[i], l[j] = l[j], l[i]
		i += 1
		j -= 1
	}
}
