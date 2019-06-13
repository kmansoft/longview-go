package main

import (
	_ "encoding/json"
)

func GetDataMemory(data *Data) error {
	psMemInfo, err := ReadProcFSFile("meminfo")
	if err != nil {
		return err
	}

	free, _ := psMemInfo.GetNumberValue("MemFree:")
	data.Longterm["Memory.real.free"] = free
	used, _ := psMemInfo.GetNumberValue("MemTotal:")
	used -= free
	data.Longterm["Memory.real.used"] = used
	data.Longterm["Memory.real.buffers"], _ = psMemInfo.GetNumberValue("Buffers:")
	data.Longterm["Memory.real.cache"], _ = psMemInfo.GetNumberValue("Cached:")

	free, _ = psMemInfo.GetNumberValue("SwapFree:")
	data.Longterm["Memory.swap.free"] = free
	used, _ = psMemInfo.GetNumberValue("SwapTotal:")
	used -= free
	data.Longterm["Memory.swap.used"] = used

	return nil
}
