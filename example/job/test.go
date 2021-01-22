package job

import (
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/example/config"
)

func TestJob(conf *config.Config) {
	log.Info("Hello, test job!")
	log.Infof("mysql_conn: %s", conf.DB)
}

func TestTimeoutJob(conf *config.Config) {
	log.Info("Hello, test timeout job!")
	<-time.After(30 * time.Second)
	log.Infof("0000000000: %s", conf.DB)
}
