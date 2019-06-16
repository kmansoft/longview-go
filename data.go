package main

type Data struct {
	Instant   map[string]interface{} `json:"INSTANT"`
	Longterm  map[string]interface{} `json:"LONGTERM"`
	Timestamp int64                  `json:"timestamp"`
}

type PostData struct {
	Version   string  `json:"version"`
	ApiKey    string  `json:"apikey"`
	Payload   []*Data `json:"payload"`
	Timestamp int64   `json:"timestamp"`
}

func (data *Data) HasProcess(pname string) bool {
	_, ok := data.Instant["Processes."+pname+".longname"]
	return ok
}
