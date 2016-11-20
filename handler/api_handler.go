package handler

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/asiainfoLDP/datahub_ApiGateWay/api"
	"github.com/asiainfoLDP/datahub_ApiGateWay/common"
	"github.com/asiainfoLDP/datahub_ApiGateWay/models"
	"github.com/julienschmidt/httprouter"
)

const (
	Necessary   = 1
	NoNecessary = 0
	Https       = 1
	Http        = 0
	Verify      = 1
	NoVerify    = 0
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func UpdateApiHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	logger.Info("Request url: PUT %v.", r.URL)

	var mode = "update"

	logger.Info("Begin do CreateRepo handler.")
	defer logger.Info("End do recharge handler.")

	//user := r.Header.Get("User")
	//
	//if getUserStatus(user) == USER_TP_UNKNOW {
	//	//用户不存在
	//	api.JsonResult(w, http.StatusBadRequest, api.GetError(api.ErrorCodeAuthFailed), nil)
	//	return
	//}
	user := "datahub+771435128@qq.com"
	repoName := params.ByName("reponame")
	itemName := params.ByName("itemname")

	db := models.GetDB()
	if db == nil {
		logger.Warn("Get db is nil.")
		api.JsonResult(w, http.StatusInternalServerError, api.GetError(api.ErrorCodeDbNotInitlized), nil)
		return
	}

	apiItem := &models.ApiItem{}
	err := common.ParseRequestJsonInto(r, apiItem)
	if err != nil {
		logger.Error("Parse body err: %v", err)
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeParseJsonFailed, err.Error()), nil)
		return
	}

	//设置apiinfo
	apiUrl, err := url.Parse(apiItem.Url)

	apiInfo := &models.ApiInfo{}
	apiInfo.Reqrepo = repoName
	apiInfo.Reqitem = itemName
	apiInfo.CreateUser = user
	apiInfo.ReqResourcePage = apiUrl.Host
	apiInfo.ReqHostInfo = apiUrl.Host
	apiInfo.UrlPath = apiUrl.Path
	apiInfo.ReqAppKey = apiItem.Key
	apiInfo.IsVerify = apiItem.Verify
	if apiUrl.Scheme == "https" {
		apiInfo.IsHttps = Https
	} else if apiUrl.Scheme == "http" {
		apiInfo.IsHttps = Http
	}
	apiInfo.ReqType = apiItem.Method
	apiInfo.QueryTimes = 100000

	_, err = models.QueryApiInfo(db, repoName, itemName, user)
	if err != nil {
		if err == sql.ErrNoRows {
			//这个item不存在 操作方式设置为插入
			mode = "insert"
			//插入信息
			apiInfo, err = models.InsertApiInfo(db, apiInfo)
			if err != nil {
				logger.Error(err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
				return
			}
		} else {
			//其他错误
			logger.Error(err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
			return
		}
	} else {
		//更新信息
		apiInfo, err = models.UpdateApiInfo(db, apiInfo)
		if err != nil {
			logger.Error(err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
			return
		}
	}

	//获得apiinfoId
	ApiId := apiInfo.Id
	apiParams := apiItem.Params
	fmt.Println("params : %v", apiParams)
	//设置参数
	for _, param := range apiParams {
		param.ApiId = ApiId
		if mode == "insert" {
			if err := models.InsertParam(db, &param); err != nil {
				logger.Error(err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
			}
		} else {
			if err := models.UpdateParam(db, &param); err != nil {
				logger.Error(err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
			}
		}
	}

	api.JsonResult(w, http.StatusOK, nil, nil)
}

func QueryRepoListHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	logger.Info("Request url: GET %v.", r.URL)

	logger.Info("Begin get RepoList handler.")
	defer logger.Info("End get RepoList handler.")

	user := r.Header.Get("User")

	if getUserStatus(user) == USER_TP_UNKNOW {
		//用户不存在
		api.JsonResult(w, http.StatusBadRequest, api.GetError(api.ErrorCodeAuthFailed), nil)
		return
	}

	db := models.GetDB()
	if db == nil {
		logger.Warn("Get db is nil.")
		api.JsonResult(w, http.StatusInternalServerError, api.GetError(api.ErrorCodeDbNotInitlized), nil)
		return
	}
	repoName := params.ByName("reponame")
	itemName := params.ByName("itemname")
	//查询api转发信息
	apiInfo, err := models.QueryApiInfo(db, repoName, itemName, user)
	if err != nil {
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
		return
	}

	//请求的问号参数
	var query string
	if apiInfo.IsVerify == Verify {
		//需要认证
		query = "?" + apiInfo.ReqAppKey
	} else if apiInfo.IsVerify == NoVerify {
		//不需要认证
		query = "?"
	}

	apiParams, err := models.QueryParamList(db, apiInfo.Id)
	if err != nil {
		if err != sql.ErrNoRows {
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiParamsInfo, err.Error()), nil)
			return
		}
	}
	if apiParams != nil {
		//参数不为空
		for _, param := range apiParams {
			if param.Must == Necessary {
				//必要参数
				value := r.Form.Get(param.Name)
				if value != nil {
					query = query + param.Name + "=" + value
				} else {
					//缺少必要参数
				}
			}
			if param.Must == NoNecessary {
				//非必要参数
				value := r.Form.Get(param.Name)
				query = query + param.Name + "=" + value
			}
		}
	}

	redisDb := models.GPool.Get()
	defer redisDb.Close()
	redisDb.Do("GET")
	//查询reids中的订购信息

	//查询订购服务中的订购信息
	err, subs := getSubs(repoName, itemName, user)
	if err != nil {
		//未订购
	}
	//更新redis中的订购信息

	//检查是否完成订单

	//更新订购服务中的订购信息

	api.JsonResult(w, http.StatusOK, nil, nil)

}
