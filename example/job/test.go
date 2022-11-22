package job

import (
	"time"

	"github.com/mylxsw/glacier/log"

	"github.com/mylxsw/glacier/example/config"
)

func TestJob(conf *config.Config) {
	log.Debug("[example] Hello, test job!")
	log.Debugf("[example] mysql_conn: %s", conf.DB)
}

func TestTimeoutJob(conf *config.Config) {
	log.Debug("[example] Hello, test timeout job!")
	<-time.After(3 * time.Second)
	log.Debugf("[example] 0000000000: %s", conf.DB)
}
