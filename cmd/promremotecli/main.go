// Copyright (c) 2019 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/m3db/prometheus_remote_client_golang/promremote"
)

type labelList []promremote.Label
type headerList []header
type dp promremote.Datapoint

type header struct {
	name  string
	value string
}

func main() {
	var (
		log                    = stdlog.New(os.Stderr, "promremotecli_log ", stdlog.LstdFlags)
		writeURLFlag           string
		labelsListFlag         labelList
		headerListFlag         headerList
		dpFlag                 dp
		insecureSkipVerifyFlag bool
	)

	flag.StringVar(&writeURLFlag, "u", promremote.DefaultRemoteWrite, "remote write endpoint")
	flag.Var(&labelsListFlag, "t", "label pair to include in metric. specify as key:value e.g. status_code:200")
	flag.Var(&headerListFlag, "h", "headers to set in the request, e.g. 'User-Agent: foo'")
	flag.Var(&dpFlag, "d", "datapoint to add. specify as unixTimestamp(int),value(float) e.g. 1556026059,14.23. use `now` instead of timestamp for current time")
	flag.BoolVar(&insecureSkipVerifyFlag, "i", promremote.DefaultInsecureSkipVerify, "skip verification of ssl certificates")

	flag.Parse()

	tsList := promremote.TSList{
		{
			Labels:    []promremote.Label(labelsListFlag),
			Datapoint: promremote.Datapoint(dpFlag),
		},
	}

	cfg := promremote.NewConfig(
		promremote.WriteURLOption(writeURLFlag),
		promremote.WriteInsecureSkipVerify(insecureSkipVerifyFlag),
	)

	client, err := promremote.NewClient(cfg)
	if err != nil {
		log.Fatal(fmt.Errorf("unable to construct client: %v", err))
	}

	var headers map[string]string
	log.Println("writing datapoint", dpFlag.String())
	log.Println("labelled", labelsListFlag.String())
	if len(headerListFlag) > 0 {
		log.Println("with headers", headerListFlag.String())
		headers = make(map[string]string, len(headerListFlag))
		for _, header := range headerListFlag {
			headers[header.name] = header.value
		}
	}
	log.Println("writing to", writeURLFlag)

	result, writeErr := client.WriteTimeSeries(context.Background(), tsList,
		promremote.WriteOptions{Headers: headers})
	if err := error(writeErr); err != nil {
		json.NewEncoder(os.Stdout).Encode(struct {
			Success    bool   `json:"success"`
			Error      string `json:"error"`
			StatusCode int    `json:"statusCode"`
		}{
			Success:    false,
			Error:      err.Error(),
			StatusCode: writeErr.StatusCode(),
		})
		os.Stdout.Sync()

		log.Fatal("write error", err)
	}

	json.NewEncoder(os.Stdout).Encode(struct {
		Success    bool `json:"success"`
		StatusCode int  `json:"statusCode"`
	}{
		Success:    true,
		StatusCode: result.StatusCode,
	})
	os.Stdout.Sync()

	log.Println("write success")
}

func (t *labelList) String() string {
	var labels [][]string
	for _, v := range []promremote.Label(*t) {
		labels = append(labels, []string{v.Name, v.Value})
	}
	return fmt.Sprintf("%v", labels)
}

func (t *labelList) Set(value string) error {
	labelPair := strings.Split(value, ":")
	if len(labelPair) != 2 {
		return fmt.Errorf("incorrect number of arguments to '-t': %d", len(labelPair))
	}

	label := promremote.Label{
		Name:  labelPair[0],
		Value: labelPair[1],
	}

	*t = append(*t, label)

	return nil
}

func (h *headerList) String() string {
	var headers [][]string
	for _, v := range []header(*h) {
		headers = append(headers, []string{v.name, v.value})
	}
	return fmt.Sprintf("%v", headers)
}

func (h *headerList) Set(value string) error {
	firstSplit := strings.Index(value, ":")
	if firstSplit == -1 {
		return fmt.Errorf("header missing separating colon: '%v'", value)
	}

	*h = append(*h, header{
		name:  strings.TrimSpace(value[:firstSplit]),
		value: strings.TrimSpace(value[firstSplit+1:]),
	})

	return nil
}

func (d *dp) String() string {
	return fmt.Sprintf("%v", []string{d.Timestamp.String(), fmt.Sprintf("%v", d.Value)})
}

func (d *dp) Set(value string) error {
	dp := strings.Split(value, ",")
	if len(dp) != 2 {
		return fmt.Errorf("incorrect number of arguments to '-d': %d", len(dp))
	}

	var ts time.Time
	if strings.ToLower(dp[0]) == "now" {
		ts = time.Now()
	} else {
		i, err := strconv.Atoi(dp[0])
		if err != nil {
			return fmt.Errorf("unable to parse timestamp: %s", dp[1])
		}
		ts = time.Unix(int64(i), 0)
	}

	val, err := strconv.ParseFloat(dp[1], 64)
	if err != nil {
		return fmt.Errorf("unable to parse value as float64: %s", dp[0])
	}

	d.Timestamp = ts
	d.Value = val

	return nil
}
