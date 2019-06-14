package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func GetDataAppNginx(client *http.Client, data *Data) error {

	if _, ok := data.Instant["Processes.nginx.longname"]; !ok {
		return nil
	}

	namespace := "Applications.Nginx."

	config := ReadConfig("Nginx")
	location := config.GetOrDefault("location", "http://127.0.0.1/nginx_status")

	resp, err := client.Get(location)
	if err != nil {
		log.Printf("Cannot get http response data: %s", err)
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
		log.Printf("Cannot get http response data: %s", err)
		return err
	}

	respString := string(respBody)
	if strings.Index(respString, "server accepts handled requests") < 0 {
		fmt.Printf("Strange looking nginx response")
		return nil
	}

	exprActiveConnections := regexp.MustCompile(`Active connections: (\d+)`)
	exprTotalConnections := regexp.MustCompile(`(\d+) (\d+) (\d+)`)
	exprReadingWritingWaiting := regexp.MustCompile(`Reading: (\d+) Writing: (\d+) Waiting: (\d+)`)

	for _, l := range strings.Split(respString, "\n") {
		if m := exprActiveConnections.FindStringSubmatch(l); len(m) > 0 {
			// Active connections
			data.Longterm[namespace+"active"], _ = strconv.ParseUint(m[1], 10, 64)
		} else if m := exprTotalConnections.FindStringSubmatch(l); len(m) > 0 {
			// Total connections / requests
			data.Longterm[namespace+"accepted_cons"], _ = strconv.ParseUint(m[1], 10, 64)
			data.Longterm[namespace+"handled_cons"], _ = strconv.ParseUint(m[2], 10, 64)
			data.Longterm[namespace+"requests"], _ = strconv.ParseUint(m[3], 10, 64)
		} else if m := exprReadingWritingWaiting.FindStringSubmatch(l); len(m) > 0 {
			// Reading / writing / wqaiting
			data.Longterm[namespace+"reading"], _ = strconv.ParseUint(m[1], 10, 64)
			data.Longterm[namespace+"writing"], _ = strconv.ParseUint(m[2], 10, 64)
			data.Longterm[namespace+"waiting"], _ = strconv.ParseUint(m[3], 10, 64)
		}
	}

	// Server version
	version := resp.Header.Get("Server")
	if len(version) > 0 {
		data.Instant[namespace+"version"] = version
	}
	data.Instant[namespace+"status"] = 0
	data.Instant[namespace+"status_message"] = ""

	return nil
}
