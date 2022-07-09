module github.com/Zamerykanizowana/replicated-file-system

go 1.18

require (
	github.com/google/uuid v1.1.2
	github.com/hanwen/go-fuse/v2 v2.1.0
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.27.0
	github.com/urfave/cli/v2 v2.8.1
	go.nanomsg.org/mangos/v3 v3.4.1
	go.uber.org/multierr v1.6.0
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
)

replace go.nanomsg.org/mangos/v3 => /home/mhawrus/studies/sem8/praca_inzynierska/mangos

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210927094055-39ccf1dd6fa6 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
)
