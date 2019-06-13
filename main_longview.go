package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

func main() {

	// Data item
	data := Data{
		Instant:   make(map[string]interface{}),
		Longterm:  make(map[string]interface{}),
		Timestamp: time.Now().Unix(),
	}

	// Fill it in
	GetDataMemory(&data)
	GetDataCPU(&data)
	GetDataSysInfo(&data)

	sendDataToServer(&data)
}

func sendDataToServer(data *Data) {

	// Add other smaller required fields
	post := PostData{
		Version:   "1.0",
		ApiKey:    "2C6F9D8D-068A-D23C-1B87639441717CB1",
		Payload:   make([]*Data, 0),
		Timestamp: time.Now().Unix(),
	}

	// Payload has one data item
	post.Payload = append(post.Payload, data)

	// Encode to JSON
	jsonData, _ := json.MarshalIndent(&post, "", "\t")
	fmt.Printf("Output:\n%s\n", jsonData)

	// Compress the JSON
	jsonCompressBuffer := bytes.Buffer{}

	jsonCompressWriter := gzip.NewWriter(&jsonCompressBuffer)
	_, _ = jsonCompressWriter.Write(jsonData)
	_ = jsonCompressWriter.Close()

	// Create a multipart form for upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			"data", "json.gz"))
	h.Set("Content-Type", "application/json")
	h.Set("Content-Encoding", "gzip")

	part, err := writer.CreatePart(h)
	_, _ = jsonCompressBuffer.WriteTo(part)
	_ = writer.Close()

	// Send it
	req, err := http.NewRequest("POST", "https://longview.linode.com/post", body)
	if err != nil {
		log.Printf("Cannot make new http request: %s", err)
		return
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Cannot get http response data: %s", err)
		return
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Parse response - which may tell us for how long to sleep or to terminate
	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)

	respBody, _ := ioutil.ReadAll(resp.Body)

	fmt.Println("response Body:", string(respBody))
}
