package job

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/example/config"
)

type TestJob struct{}

func NewTestJob() *TestJob {
	return &TestJob{}
}

func (TestJob) Handle() {
	log.Info("Hello, test job!")

	glacier.Container().MustResolve(func(conf *config.Config) {
		log.Infof("mysql_conn: %s", conf.MySQLURI)
	})
}
