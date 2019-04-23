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

package promremote

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/m3db/m3/src/query/ts"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

const (
	// DefaultRemoteWrite is the default Prom remote write endpoint in m3coordinator.
	DefaultRemoteWrite = "http://localhost:7201/api/v1/prom/remote/write"

	defaulHTTPClientTimeout = time.Second * 30
)

// Tag are the metric tags.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Timeseries are made of tags and a datapoint.
type Timeseries struct {
	Tags      []Tag
	Datapoint ts.Datapoint
}

type client struct {
	writeURL   string
	httpClient *http.Client
}

type clientOptions struct {
	remoteWriteURL    string
	httpClientTimeout time.Duration
}

// M3Client is used to write timeseries data to m3coordinator.
type M3Client interface {
	// Write writes the Prom proto WriteRequest to the specified endpoint.
	Write(*prompb.WriteRequest) error
}

// ClientOptions defines available methods.
type ClientOptions interface {
	// SetWriteURL sets the URL which the client uses to write to m3coordinator.
	SetWriteURL(string) ClientOptions

	// WriteURL returns the URL which the client uses to write to m3coordinator.
	WriteURL() string

	// SetHTTPClientTimeout sets the timeout for the client.
	SetHTTPClientTimeout(time.Duration) ClientOptions

	//HTTPClientTimeout returns the timeout that is set for the client.
	HTTPClientTimeout() time.Duration
}

// NewClientOpts returns a default clientOptions struct.
func NewClientOpts() ClientOptions {
	return &clientOptions{
		remoteWriteURL:    DefaultRemoteWrite,
		httpClientTimeout: defaulHTTPClientTimeout,
	}
}

// NewClient creates a new remote write coordinator client.
func NewClient(opts ClientOptions) M3Client {
	return &client{
		writeURL: opts.WriteURL(),
		httpClient: &http.Client{
			Timeout: opts.HTTPClientTimeout(),
		},
	}
}

func (o *clientOptions) SetWriteURL(val string) ClientOptions {
	opts := *o
	opts.remoteWriteURL = val
	return &opts
}

func (o *clientOptions) WriteURL() string {
	return o.remoteWriteURL
}

func (o *clientOptions) SetHTTPClientTimeout(val time.Duration) ClientOptions {
	opts := *o
	opts.httpClientTimeout = val
	return &opts
}

func (o *clientOptions) HTTPClientTimeout() time.Duration {
	return o.httpClientTimeout
}

func (c *client) Write(promWR *prompb.WriteRequest) error {
	data, err := proto.Marshal(promWR)
	if err != nil {
		return fmt.Errorf("unable to marshal protobuf: %v", err)
	}

	encoded := snappy.Encode(nil, data)

	body := bytes.NewReader(encoded)
	req, err := http.NewRequest("POST", c.writeURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %v", err)
		}

		body := string(bodyBytes)
		return fmt.Errorf("expected 200 response code, instead got: %d, %s", resp.StatusCode, body)
	}

	return nil
}

// TSListToProtoWR converts a list of timeseries to a Prometheus proto write request.
func TSListToProtoWR(tsList []Timeseries) *prompb.WriteRequest {
	promTS := make([]*prompb.TimeSeries, len(tsList))

	for i, ts := range tsList {
		labels := make([]*prompb.Label, len(ts.Tags))
		for j, tag := range ts.Tags {
			labels[j] = &prompb.Label{Name: tag.Name, Value: tag.Value}
		}

		sample := []prompb.Sample{prompb.Sample{Value: ts.Datapoint.Value, Timestamp: ts.Datapoint.Timestamp.Unix()}}
		promTS[i] = &prompb.TimeSeries{Labels: labels, Samples: sample}
	}

	return &prompb.WriteRequest{
		Timeseries: promTS,
	}
}
