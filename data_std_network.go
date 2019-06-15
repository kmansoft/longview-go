package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

func GetDataNetwork(data *Data) error {

	psNetDev, err := ReadProcFSFile("net/dev")
	if err != nil {
		return err
	}

	eth := ""
	expr := regexp.MustCompile(`^\s+([a-zA-Z0-9]+):\s+(\d+)\s+(\d+)`)

	for _, l := range psNetDev.GetAsLines() {
		m := expr.FindStringSubmatch(l)
		if len(m) == 4 {
			rx, _ := strconv.ParseUint(m[2], 10, 64)
			tx, _ := strconv.ParseUint(m[3], 10, 64)

			if rx != 0 || tx != 0 {
				prefix := fmt.Sprintf("Network.Interface.%s.", m[1])

				data.Longterm[prefix+"rx_bytes"] = rx
				data.Longterm[prefix+"tx_bytes"] = tx
			}

			if eth == "" {
				eth = m[1]
			}
		}
	}

	// Linode uses eth0 but for testing we need to support other names (eno1 on Fedora)
	mac, err := ioutil.ReadFile("/sys/class/net/eth0/address")
	if err != nil && eth != "" {
		mac, err = ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/address", eth))
	}
	if err == nil && len(mac) > 0 {
		data.Instant["Network.mac_addr"] = strings.TrimSpace(string(mac))
	}

	return nil
}
