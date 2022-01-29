module ssh2lxd

go 1.16

replace google.golang.org/grpc/naming => google.golang.org/grpc v1.29.1

require (
	github.com/creack/pty v1.1.17
	github.com/gorilla/websocket v1.4.2
	github.com/lxc/lxd v0.0.0-20220126051716-203c3e15a0d5
	github.com/pkg/sftp v1.13.4
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
)
