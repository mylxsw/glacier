package job

import (
	"github.com/mylxsw/glacier/log"
	"time"

	"github.com/mylxsw/glacier/example/config"
)

func TestJob(conf *config.Config) {
	log.Debug("Hello, test job!")
	log.Debugf("mysql_conn: %s", conf.DB)
}

func TestTimeoutJob(conf *config.Config) {
	log.Debug("Hello, test timeout job!")
	<-time.After(10 * time.Second)
	log.Debugf("0000000000: %s", conf.DB)
}
