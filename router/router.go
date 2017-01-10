package router

import (
	"net/http"
	"time"

	"github.com/asiainfoLDP/datahub_ApiGateWay/api"
	"github.com/asiainfoLDP/datahub_ApiGateWay/handler"
	"github.com/asiainfoLDP/datahub_ApiGateWay/log"
	"github.com/julienschmidt/httprouter"
)

const (
	Platform_Local  = "local"
	Platform_DataOS = "dataos"
)

var (
	Platform = Platform_DataOS
	logger   = log.GetLogger()
)

//==============================================================
//
//==============================================================

func handler_Index(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	api.JsonResult(w, http.StatusNotFound, api.GetError(api.ErrorCodeUrlNotSupported), nil)
}

func httpNotFound(w http.ResponseWriter, r *http.Request) {
	api.JsonResult(w, http.StatusNotFound, api.GetError(api.ErrorCodeUrlNotSupported), nil)
}

type HttpHandler struct {
	handler http.HandlerFunc
}

func (httpHandler *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if httpHandler.handler != nil {
		httpHandler.handler(w, r)
	}
}

//==============================================================
//
//==============================================================

func InitRouter() *httprouter.Router {
	router := httprouter.New()
	router.RedirectFixedPath = false
	router.RedirectTrailingSlash = false

	router.POST("/", handler_Index)
	router.DELETE("/", handler_Index)
	router.PUT("/", handler_Index)
	router.GET("/", handler_Index)

	router.NotFound = &HttpHandler{httpNotFound}
	router.MethodNotAllowed = &HttpHandler{httpNotFound}

	return router
}

func NewRouter(router *httprouter.Router) {
	logger.Info("new router.")
	router.PUT("/apiinfo/:reponame/:itemname", api.TimeoutHandle(35000*time.Millisecond, handler.UpdateApiHandler))
	router.GET("/apiinfo/:reponame/:itemname", api.TimeoutHandle(35000*time.Millisecond, handler.QueryRepoListHandler))
}
