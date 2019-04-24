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
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/m3db/m3/src/query/ts"
	"github.com/m3db/prometheus_remote_client_golang/promremote"
)

var (
	writeURL    *string
	tagsListVar tagList
	dpVar       dp
)

type tagList []promremote.Tag
type dp ts.Datapoint

func init() {
	writeURL = flag.String("u", promremote.DefaultRemoteWrite, "remote write endpoint")
	flag.Var(&tagsListVar, "t", "tag pair to include in metric. specify as key:value e.g. status_code:200")
	flag.Var(&dpVar, "d", "datapoint to add. specify as value(float),unixTimestamp(int) e.g. 14.23,1556026059. use `now` instead of timestamp for current time")

	flag.Parse()
}

func main() {
	tsList := []promremote.Timeseries{
		{
			Tags:      []promremote.Tag(tagsListVar),
			Datapoint: ts.Datapoint(dpVar),
		},
	}

	client := promremote.NewClient(promremote.NewClientOpts().SetWriteURL(*writeURL))

	promWR := promremote.TSListToProtoWR(tsList)
	if err := client.Write(promWR); err != nil {
		log.Fatal(err)
	}
}

func (t *tagList) String() string {
	return ""
}

func (t *tagList) Set(value string) error {
	tagPair := strings.Split(value, ":")
	if len(tagPair) != 2 {
		return fmt.Errorf("incorrect number of arguments to '-t': %d", len(tagPair))
	}

	tag := promremote.Tag{
		Name:  tagPair[0],
		Value: tagPair[1],
	}

	*t = append(*t, tag)

	return nil
}

func (d *dp) String() string {
	return ""
}

func (d *dp) Set(value string) error {
	dp := strings.Split(value, ",")
	if len(dp) != 2 {
		return fmt.Errorf("incorrect number of arguments to '-d': %d", len(dp))
	}

	val, err := strconv.ParseFloat(dp[0], 64)
	if err != nil {
		return fmt.Errorf("unable to parse value as float64: %s", dp[0])
	}

	var ts time.Time
	if strings.ToLower(dp[1]) == "now" {
		ts = time.Now()
	} else {
		i, err := strconv.Atoi(dp[1])
		if err != nil {
			return fmt.Errorf("unable to parse timestamp: %s", dp[1])
		}
		ts = time.Unix(int64(i), 0)
	}

	d.Timestamp = ts
	d.Value = val

	return nil
}
