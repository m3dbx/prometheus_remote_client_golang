# promremote

`promremote` is a Prometheus remote write client written in Go.

## Installation

`go get -u github.com/m3db/prometheus_remote_client_golang`

## Use

`promremote` is used to send metrics to a Prometheus remote write endpoint such as that found in
[m3coordinator](http://m3db.github.io/m3/overview/components/#m3-coordinator).It can be pulled into
an existing codebase as a client library or used as a cli tool (`promremotecli`) for adhoc testing
purposes.

**WARNING:** You will need a program or app running that has a Prom remote write endpoint
registered.

### Client library

If you want to use `promremote` as a client library, you will need to construct the client yourself
and pass in the appropriate structs.

```golang
# create options and client
opts := promremote.NewClientOpts().SetWriteURL("http://localhost:7201/api/v1/prom/remote/write")
client := promremote.NewClient(opts)

tsList := []promremote.Timeseries{
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

promWR := promremote.TSListToProtoWR(tsList)
if err := client.Write(promWR); err != nil {
	log.Fatal(err)
}
```

### CLI

If you want to use `promremote` as a CLI, you can utilize the `promremotecli` tool located in
the `cmd/` directory. The tool takes in a series of tags and a datapoint then writes them to the
Prom remote write endpoint within `m3coordinator`. Below is an example showing a metric with two tags
(`__name__:foo_bar` and `biz:baz`) and a datapoint (val:`1415.92` timestamp:`now`).

**Note**: You can either specify a Unix timestamp (e.g. `1556026725`) or the keyword `now` as the
second parameter in the `-d` flag.

```bash
go run cmd/promremotecli.go -t=__name__:foo_bar -t=biz:baz -d=1415.92,now
```
