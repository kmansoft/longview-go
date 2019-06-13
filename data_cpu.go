package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func GetDataCPU(data *Data) error {

	psStat, err := ReadProcFSFile("stat")
	if err != nil {
		return err
	}

	expr := regexp.MustCompile(`^cpu(\d+)`)

	for i, l := range psStat.GetAsLines() {
		if i > 0 {
			m := expr.FindStringSubmatch(l)
			if len(m) == 2 {
				if cpuN, err := strconv.ParseInt(m[1], 10, 32); err == nil && cpuN < 64 {
					r := strings.Split(l, " ")
					if len(r) >= 8 {
						user, _ := strconv.ParseUint(r[1], 10, 64)
						nice, _ := strconv.ParseUint(r[2], 10, 64)

						system, _ := strconv.ParseUint(r[3], 10, 64)
						wait, _ := strconv.ParseUint(r[5], 10, 64)

						prefix := fmt.Sprintf("CPU.cpu%d.", cpuN)
						data.Longterm[prefix+"user"] = user + nice
						data.Longterm[prefix+"system"] = system
						data.Longterm[prefix+"wait"] = wait
					}
				}
			} else {
				break
			}
		}
	}

	psLoadAvg, err := ReadProcFSFile("loadavg")
	line1 := strings.Split(psLoadAvg.GetAsString(), " ")
	if len(line1) > 1 {
		load, _ := strconv.ParseFloat(line1[0], 32)
		data.Longterm["Load"] = load
	}

	return nil
}
