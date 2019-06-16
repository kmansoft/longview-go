package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

const (
	WORKER_KEYS = "_SRWKDCLGI"
)

func GetDataAppApache(client *http.Client, data *Data) error {

	if !data.HasProcess("apache2") && !data.HasProcess("httpd") {
		return nil
	}

	namespace := "Applications.Apache."

	config := ReadConfig("Apache")
	location := config.GetOrDefault("location", "http://127.0.0.1/server-status?auto")

	resp, err := client.Get(location)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Parse response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return err
	}

	respString := string(respBody)
	if strings.Index(respString, "Scoreboard:") < 0 {
		fmt.Printf("Strange looking apache response")
		return nil
	}

	exprDataLine := regexp.MustCompile(`([^:]+):\s+(.+)`)

	scoreBoard := ""

	for _, l := range strings.Split(respString, "\n") {
		if m := exprDataLine.FindStringSubmatch(string(l)); len(m) == 3 {
			key := m[1]
			value := m[2]
			if key == "Scoreboard" {
				scoreBoard = value
			} else if key == "Total Accesses" || key == "Total kBytes" {
				data.Longterm[namespace+key] = value
			}
		}
	}

	// Workers
	fmt.Printf("Server scroreboard: %q\n", scoreBoard)
	for _, ch := range scoreBoard {
		skey := getWorkerKeyFromChar(ch)

		if skey != "" {
			skey = namespace + "Workers." + skey
			if sval, ok := data.Longterm[skey]; ok {
				data.Longterm[skey] = sval.(int) + 1
			} else {
				data.Longterm[skey] = 1
			}
		}
	}

	for _, ch := range WORKER_KEYS {
		skey := getWorkerKeyFromChar(ch)

		if skey != "" {
			skey = namespace + "Workers." + skey
			if _, ok := data.Longterm[skey]; !ok {
				data.Longterm[skey] = 0
			}
		}
	}

	// Server version
	version := resp.Header.Get("Server")
	if len(version) > 0 {
		data.Instant[namespace+"version"] = version
	}

	// Overall status
	data.Instant[namespace+"status"] = 0
	data.Instant[namespace+"status_message"] = ""

	return nil
}

func getWorkerKeyFromChar(ch int32) string {
	switch ch {
	case '_':
		return "Waiting for Connection"
	case 'S':
		return "Starting up"
	case 'R':
		return "Reading Request"
	case 'W':
		return "Sending Reply"
	case 'K':
		return "Keepalive"
	case 'D':
		return "DNS Lookup"
	case 'C':
		return "Closing connection"
	case 'L':
		return "Logging"
	case 'G':
		return "Gracefully finishing"
	case 'I':
		return "Idle cleanup of worker"
	}
	return ""
}
