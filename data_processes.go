package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"os/user"
	"regexp"
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

func GetDataProcessList(data *Data) error {

	list, err := newProcessList()
	if err != nil {
		return err
	}

	for _, proc := range list.plist {
		if proc.PID == 2 || proc.PPID == 2 {
			continue
		}

		prefix := fmt.Sprintf("Processes.%s.", proc.Name)

		data.Instant[prefix+"longname"] = proc.LongName

		prefix = fmt.Sprintf("Processes.%s.%s.", proc.Name, proc.User)

		data.Longterm[prefix+"mem"] = proc.RSS
		data.Longterm[prefix+"cpu"] = proc.CPU

		list.addToCount(data.Longterm, prefix+"count", 1)
		list.addToCount(data.Longterm, prefix+"ioreadkbytes", proc.IOReadKBytes)
		list.addToCount(data.Longterm, prefix+"iowritekbytes", proc.IOWriteKBytes)
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

	return nil
}

func newProcessList() (*DataProcessList, error) {
	list := &DataProcessList{
		plist: make([]*DataProcess, 0),
	}

	err := list.fill()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (list *DataProcessList) fill() error {
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
	if ticks, err := strconv.ParseFloat(getconfOut.String(), 64); err == nil {
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

			item := list.getDataForProcess(pid)
			if item != nil {
				list.plist = append(list.plist, item)
			}
		}
	}

	return nil
}

func (list *DataProcessList) getDataForProcess(pid uint64) *DataProcess {
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
							item.PPID, _ = strconv.ParseUint(list.getField(val, 0), 10, 64)
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
