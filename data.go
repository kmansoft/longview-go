package main

type Data struct {
	Longterm  map[string]interface{} `json:"LONGTERM"`
	Instant   map[string]interface{} `json:"INSTANT"`
	Timestamp int64                  `json:"timestamp"`
}

type PostData struct {
	Version   string  `json:"version"`
	ApiKey    string  `json:"apikey"`
	Payload   []*Data `json:"payload"`
	Timestamp int64   `json:"timestamp"`
}
