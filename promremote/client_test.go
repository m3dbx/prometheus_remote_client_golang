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
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	now = time.Now()
)

func TestPromRemoteClientWrite(t *testing.T) {
	overrideUserAgent := "overrideUserAgent"
	customHeaders := map[string]string{
		"M3-Metrics-Type": "unaggregated",
		"User-Agent":      overrideUserAgent,
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "application/x-protobuf", r.Header.Get("Content-Type"))
		assert.Equal(t, overrideUserAgent, r.Header.Get("User-Agent"))
		for k, v := range customHeaders {
			assert.Equal(t, v, r.Header.Get(k))
		}

		defer r.Body.Close()

		bodyBytes, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		decoded, err := snappy.Decode(nil, bodyBytes)
		require.NoError(t, err)

		newWR := &prompb.WriteRequest{}
		err = proto.Unmarshal(decoded, newWR)
		require.NoError(t, err)

		assert.Len(t, newWR.Timeseries, 1)
		assert.Len(t, newWR.Timeseries[0].Labels, 2)
		assert.Len(t, newWR.Timeseries[0].Samples, 1)
		assert.Equal(t, "__name__", newWR.Timeseries[0].Labels[0].Name)
		assert.Equal(t, "foo_bar", newWR.Timeseries[0].Labels[0].Value)
		assert.Equal(t, "biz", newWR.Timeseries[0].Labels[1].Name)
		assert.Equal(t, "baz", newWR.Timeseries[0].Labels[1].Value)
		assert.Equal(t, 1415.92, newWR.Timeseries[0].Samples[0].Value)
		assert.Equal(t, now.Unix(), newWR.Timeseries[0].Samples[0].Timestamp)
	}))

	defer testServer.Close()

	cfg := NewConfig(
		WriteURLOption(testServer.URL),
	)

	c, err := NewClient(cfg)
	require.NoError(t, err)

	tsList := TSList{
		{
			Labels: []Label{
				{
					Name:  "__name__",
					Value: "foo_bar",
				},
				{
					Name:  "biz",
					Value: "baz",
				},
			},
			Datapoint: Datapoint{
				Timestamp: now,
				Value:     1415.92,
			},
		},
	}

	r, err := c.WriteTimeSeries(context.Background(), tsList,
		WriteOptions{Headers: customHeaders})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestPromRemoteClientWriteNotHTTPOK(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	defer testServer.Close()

	cfg := NewConfig(
		WriteURLOption(testServer.URL),
	)

	c, err := NewClient(cfg)
	require.NoError(t, err)

	tsList := TSList{
		{
			Labels: []Label{
				{
					Name:  "__name__",
					Value: "foo_bar",
				},
				{
					Name:  "biz",
					Value: "baz",
				},
			},
			Datapoint: Datapoint{
				Timestamp: now,
				Value:     1415.92,
			},
		},
	}

	r, writeErr := c.WriteTimeSeries(context.Background(), tsList,
		WriteOptions{})
	require.Error(t, writeErr)
	require.Equal(t, http.StatusBadRequest, writeErr.StatusCode())
	require.Equal(t, http.StatusBadRequest, r.StatusCode)
}

func TestValidateConfig(t *testing.T) {
	cfg := NewConfig(
		HTTPClientTimeoutOption(-1 * time.Second),
	)

	_, err := NewClient(cfg)
	require.Error(t, err)

	cfg = NewConfig(
		WriteURLOption(""),
	)

	_, err = NewClient(cfg)
	require.Error(t, err)
}

func TestProvidedHTTPClient(t *testing.T) {
	cfg := NewConfig(
		HTTPClientOption(&http.Client{
			Timeout: 10 * time.Second,
		}),
	)

	_, err := NewClient(cfg)
	require.NoError(t, err)
}
