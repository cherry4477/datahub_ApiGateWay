package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/asiainfoLDP/datahub_ApiGateWay/log"
	"github.com/asiainfoLDP/datahub_ApiGateWay/models"
	"github.com/asiainfoLDP/datahub_ApiGateWay/router"
	"github.com/asiainfoLDP/datahub_commons/httputil"
)

const SERVERPORT = 8092

var (
	logger = log.GetLogger()

	//init a router
	initRouter = router.InitRouter()
)

type Service struct {
	httpPort int
}

func newService(httpPort int) *Service {
	service := &Service{
		httpPort: httpPort,
	}

	return service
}

func main() {

	//new a router
	router.NewRouter(initRouter)

	//todo init db
	models.InitDB()
	models.InitRedis()

	service := newService(SERVERPORT)
	address := fmt.Sprintf(":%d", service.httpPort)
	logger.Debug("address: %v", address)

	logger.Info("Listening http at: %s", address)
	err := http.ListenAndServe(address, httputil.TimeoutHandler(initRouter, 35000*time.Millisecond, ""))
	if err != nil {
		logger.Error("http listen and server err: %v", err)
		return
	}

	return
}

func init() {
	//api.InitMQ()
}
