# promremote

`promremote` is a Prometheus remote write client written in Go.

## Installation

`go get -u github.com/m3db/prometheus_remote_client_golang`

## Use

`promremote` is used to send metrics to a Prometheus remote write endpoint such as that found in
[m3coordinator](http://m3db.github.io/m3/overview/components/#m3-coordinator). It can be pulled into
an existing codebase as a client library or used as a cli tool (`cmd/main.go`) for ad hoc testing
purposes.

**IMPORTANT:** A running program or application that has a Prometheus remote write endpoint is required.

### Client library

To use `promremote` as a client library, the client must be constructed manually using structs.

```golang
// create config and client
cfg := promremote.NewConfig(
  promremote.WriteURLOption(writeURLFlag),
  promremote.HTTPClientTimeoutOption(60 * time.Second),
)

client, err := promremote.NewClient(cfg)
if err != nil {
  log.Fatal(fmt.Errorf("unable to construct client: %v", err))
}


tsList := promremote.TSList{
		{
			Tags: []Tag{
				{
					Name:  "__name__",
					Value: "foo_bar",
				},
				{
					Name:  "biz",
					Value: "baz",
				},
			},
			Datapoint: ts.Datapoint{
				Timestamp: time.Now(),
				Value:     1415.92,
			},
		},
	}

if err := client.WriteTimeSeries(tsList); err != nil {
	log.Fatal(err)
}
```

### CLI

If one wants to use `promremote` as a CLI, he or she can utilize the tool located in the `cmd/`
directory. The tool takes in a series of tags and a datapoint then writes them to the Prometheus
remote write endpoint within `m3coordinator`. Below is an example showing a metric with two tags
(`__name__:foo_bar` and `biz:baz`) and a datapoint (val:`1415.92` timestamp:`now`).

**Note**: One can either specify a Unix timestamp (e.g. `1556026725`) or the keyword `now` as the
second parameter in the `-d` flag.

```bash
go run cmd/main.go -t=__name__:foo_bar -t=biz:baz -d=1415.92,now
```
