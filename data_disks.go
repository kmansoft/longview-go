package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"
)

func GetDataDisks(data *Data) error {

	// Get swap device names
	swapDeviceMap := getSwapDevicesAsMap()

	_ = swapDeviceMap

	// Parse mtab for mounted devices
	mtabData, err := ioutil.ReadFile("/etc/mtab")
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(mtabData), "\n") {
		if strings.HasPrefix(line, "/") {
			m := strings.Split(line, " ")
			if len(m) >= 2 {
				devName := m[0]
				devPath := m[1]

				/*
					TODO

							if ( $device =~ m|^/dev/mapper| ) {
							my $linkpath = readlink($device);
							if ($linkpath) {
								$device = abs_path("/dev/mapper/$linkpath");
							}
							else {
								my $rdev=(stat($device))[6];
								my $minor_m = ($rdev & 037774000377) >> 0000000;
								$device = "/dev/dm-$minor_m";
							}
						}
				*/

				if devName == "/dev/root" || devName == "/dev/sda2" {
					psCmdLine, err := ReadProcFSFile("cmdline")
					if err == nil {
						cmdLine := psCmdLine.GetAsString()
						rootBegin := strings.Index(cmdLine, "root=")
						if rootBegin >= 0 {
							rootEnd := strings.Index(cmdLine[rootBegin:], " ")
							if rootEnd < 0 {
								rootEnd = len(cmdLine) - rootBegin
							}
							rootEnd += rootBegin
							rootBegin += 5
							rootStr := cmdLine[rootBegin:rootEnd]
							devName = rootStr
						}
					}
				}

				if strings.HasPrefix(devName, "UUID=") {
					devLink, err := os.Readlink("/dev/disk/by-uuid/" + devName[5:])
					if err == nil {
						devName = path.Join("/dev/disk/by-uuid", devLink)
					}
				}

				var statFs syscall.Statfs_t
				if syscall.Statfs(devPath, &statFs) == nil {
					prefix := fmt.Sprintf("Disk.%s.", devName)

					data.Longterm[prefix+"fs.free"] = uint64(statFs.Bsize) * statFs.Bfree
					data.Longterm[prefix+"fs.total"] = uint64(statFs.Bsize) * statFs.Blocks
					data.Longterm[prefix+"fs.ifree"] = statFs.Ffree
					data.Longterm[prefix+"fs.itotal"] = statFs.Files

					data.Instant[prefix+"fs.path"] = devPath
					data.Instant[prefix+"fs.mounted"] = 1
				}
			}
		}
	}

	return nil
}

func getSwapDevicesAsMap() map[string]bool {
	m := make(map[string]bool)

	psSwaps, err := ReadProcFSFile("swaps")
	if err != nil {
		return m
	}

	for _, l := range psSwaps.GetAsLines() {
		if strings.HasPrefix(l, "/") {
			v := strings.Split(l, " ")
			if len(v) > 0 {
				devName := v[0]
				m[devName] = true
			}
		}
	}

	return m
}
