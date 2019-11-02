package job

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/example/config"
)

func TestJob(conf *config.Config) {
	log.Info("Hello, test job!")
	log.Infof("mysql_conn: %s", conf.DB)
}
