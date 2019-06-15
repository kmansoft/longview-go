package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
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
			m := strings.Fields(line)
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

				if devName == "/dev/root" {
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

				prefix := fmt.Sprintf("Disk.%s.", devName)

				var statFs syscall.Statfs_t
				if syscall.Statfs(devPath, &statFs) == nil {

					data.Longterm[prefix+"fs.free"] = uint64(statFs.Bsize) * statFs.Bfree
					data.Longterm[prefix+"fs.total"] = uint64(statFs.Bsize) * statFs.Blocks
					data.Longterm[prefix+"fs.ifree"] = statFs.Ffree
					data.Longterm[prefix+"fs.itotal"] = statFs.Files
				}

				data.Instant[prefix+"fs.path"] = devPath
				data.Instant[prefix+"mounted"] = 1
			}
		}
	}

	psDiskStats, err := ReadProcFSFile("diskstats")
	if err != nil {
		return err
	}

	for _, l := range psDiskStats.GetAsLines() {
		s := strings.Fields(l)
		if len(s) >= 10 {

			/*
				TODO

					if (substr($device,0,8) eq '/dev/dm-') {
						# if the filesystem sees it under /dev
						if ( -b $device ) {
							unless (keys(%dev_mapper)) {
								%dev_mapper = map { substr(readlink($_),3) => substr($_,12); } (glob("/dev/mapper/*"));
							}
							if (exists($dev_mapper{substr($device,5)})) {
								$dataref->{INSTANT}->{"Disk.$e_device.label"} = $dev_mapper{substr($device,5)};
							}
						} else {
							unless (keys(%dev_mapper)) {
								%dev_mapper = map {
									my $rdev=(stat($_))[6];
									my $major_m = ($rdev & 03777400) >> 0000010;
									my $minor_m = ($rdev & 037774000377) >> 0000000;
									join('_', $major_m,$minor_m) => substr($_,12);
								} glob ("/dev/mapper/*");
							}
							if (exists($dev_mapper{$major."_".$minor})) {
								$dataref->{INSTANT}->{"Disk.$e_device.label"} = $dev_mapper{$major."_".$minor};
							}
						}
					} elsif ($device =~ m|(/dev/md\d+)(p\d+)?|) {

			*/

			devName := s[2]
			readCount, _ := strconv.ParseUint(s[3], 10, 64)
			readSectors, _ := strconv.ParseUint(s[5], 10, 64)
			writeCount, _ := strconv.ParseUint(s[7], 10, 64)
			writeSectors, _ := strconv.ParseUint(s[9], 10, 64)

			if readCount != 0 || writeCount != 0 {
				sectorSize := getHwSectorSize(devName)

				devName = "/dev/" + devName
				prefix := fmt.Sprintf("Disk.%s.", devName)

				data.Longterm[prefix+"reads"] = readCount
				data.Longterm[prefix+"writes"] = writeCount
				data.Longterm[prefix+"read_bytes"] = readSectors * sectorSize
				data.Longterm[prefix+"write_bytes"] = writeSectors * sectorSize

				data.Instant[prefix+"dm"] = 0
				data.Instant[prefix+"childof"] = 0
				data.Instant[prefix+"children"] = 0

				isSwap := 0
				if _, ok := swapDeviceMap[devName]; ok {
					isSwap = 1
				}

				data.Instant[prefix+"isswap"] = isSwap
				if _, ok := data.Instant[prefix+"mounted"]; !ok {
					data.Instant[prefix+"mounted"] = 0
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
			v := strings.Fields(l)
			if len(v) > 0 {
				devName := v[0]
				m[devName] = true
			}
		}
	}

	return m
}

func getHwSectorSize(devName string) uint64 {
	sectorSize := uint64(512)

	if size, err := getHwSectorSizeRaw(devName); err == nil {
		sectorSize = size
	} else {
		devParent := devName
		for {
			l := len(devParent)
			if l > 1 {
				if ch := devParent[l-1]; ch >= '0' && ch <= '9' {
					devParent = devParent[:l-1]
				} else {
					break
				}
			} else {
				break
			}
		}

		if devParent != devName {
			if size, err := getHwSectorSizeRaw(devName); err == nil {
				sectorSize = size
			}
		}
	}

	return sectorSize
}

func getHwSectorSizeRaw(devName string) (uint64, error) {
	data, err := ioutil.ReadFile("/sys/block/" + devName + "/queue/hw_sector_size")
	if err != nil {
		return 0, err
	}

	size, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return size, nil
}
