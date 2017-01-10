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
	"encoding/json"
	"github.com/garyburd/redigo/redis"
)

const (
	Necessary   = 1
	NoNecessary = 0
	Https       = 1
	Http        = 0
	Verify      = 1
	NoVerify    = 0
	Delete      = 0
	Insert      = 1
	Update      = 2
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

	logger.Info("Begin do CreateRepo handler.")
	defer logger.Info("End do recharge handler.")
	user := r.Header.Get("User")
	if getUserStatus(user) != USER_TP_ADMIN {
		logger.Error("user auth fail")
		api.JsonResult(w, http.StatusBadRequest, api.GetError(api.ErrorCodeAuthFailed), nil)
		return
	}
	r.ParseForm()

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

	_, err = models.QueryApiInfo(db, repoName, itemName)
	if err != nil {
		if err == sql.ErrNoRows {
			//插入信息
			apiInfo, err = models.InsertApiInfo(db, apiInfo)
			if err != nil {
				logger.Error("insert apiinfo error:%v", err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
				return
			}
		} else {
			//其他错误
			logger.Error("query apiinfo error:%v", err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
			return
		}
	} else {
		//更新信息
		apiInfo, err = models.UpdateApiInfo(db, apiInfo)
		if err != nil {
			logger.Error("update apiinfo error:%v", err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
			return
		}
	}

	//获得apiinfoId
	ApiId := apiInfo.Id
	apiParams := apiItem.Params
	existParams , err := models.QueryParamList(db,ApiId)
	if err != nil {
		if err == sql.ErrNoRows {
			for _, param := range apiParams{
				param.ApiId = ApiId
				if err := models.InsertParam(db, param); err != nil {
					logger.Error("insert param error:%v", err.Error())
					api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiInfo, err.Error()), nil)
				}
			}
		} else {
			//其他错误
			logger.Error("query params error:%v", err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeQueryApiParamsInfo, err.Error()), nil)
			return
		}
	}

	existParams = inspectExist(apiParams,existParams)
	for _ ,existParam := range existParams{
		existParam.ApiId = ApiId
		if existParam.State == Insert {
			if err := models.InsertParam(db, existParam); err != nil {
				logger.Error("insert param error:%v", err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeInsertApiParamsInfo, err.Error()), nil)
			}
		}
		if existParam.State == Update {
			if err := models.UpdateParam(db, existParam); err != nil {
				logger.Error("update param error:%v", err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeUpdateApiParamsInfo, err.Error()), nil)
			}
		}
		if existParam.State == Delete {
			if err := models.DeleteParams(db, existParam); err != nil {
				logger.Error("delete param error:%v", err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeDeleteApiParamsInfo, err.Error()), nil)
			}
		}

	}

	api.JsonResult(w, http.StatusOK, nil, nil)
}

func QueryRepoListHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	logger.Info("Request url: GET %v.", r.URL)

	logger.Info("Begin get RepoList handler.")
	defer logger.Info("End get RepoList handler.")

	r.ParseForm()
	//authuser := r.Form.Get("authuser")
	//apiToken := r.Form.Get("apitoken")
	//sregion := valid(authuser,apiToken)
	//if sregion == "" {
	//	api.JsonResult(w, http.StatusBadRequest, api.GetError(api.ErrorCodeAuthFailed), nil)
	//	return
	//}
	//
	//user := sregion+"+"+authuser
	user := "datahub+datahub@asiainfo.com"
	db := models.GetDB()
	if db == nil {
		logger.Warn("Get db is nil.")
		api.JsonResult(w, http.StatusInternalServerError, api.GetError(api.ErrorCodeDbNotInitlized), nil)
		return
	}
	repoName := params.ByName("reponame")
	itemName := params.ByName("itemname")

	//查询api转发信息
	apiInfo, err := models.QueryApiInfo(db, repoName, itemName)
	if err != nil {
		logger.Error("RepoList query apiinfo error:%v", err.Error())
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
			logger.Error("RepoList query params error:%v", err.Error())
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
				if value != "" {
					query = query +"&"+ param.Name + "=" + value
				} else {
					//缺少必要参数
					logger.Error("RepoList invalidParameters error:%v", err.Error())
					api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeInvalidParameters, err.Error()), nil)
					return
				}
			}
			if param.Must == NoNecessary {
				//非必要参数
				value := r.Form.Get(param.Name)
				if value != "" {
					query = query +"&"+ param.Name + "=" + value
				}
			}
		}
	}

	redisDb := models.GPool.Get()
	defer redisDb.Close()
	apiSubs,err := redisDb.Do("GET",repoName+itemName+user)
	if err != nil {
		//报错查询redis中的信息失败
		logger.Error("RepoList redis get error:%v", err.Error())
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeRedisGetFail, err.Error()), nil)
		return
	}
	if apiSubs == nil {
		//查询订购服务中的订购信息
		subs, err := getSubs(repoName, itemName, user)
		if err != nil || len(subs)==0{
			//未订购报错退出
			logger.Error("RepoList get subscription error:%v", err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeNoOrder, err.Error()), nil)
			return
		}
		//提示subscription已经成功接收sub信息
		for i:=0 ; i<len(subs) ; i++  {
			subs[i].Pull_user_name = user
			subs[i].Repository_name = repoName
			subs[i].Dataitem_name = itemName
			err = updateSubs(subs[i],"set_retrieved")
			if err != nil {
				//json格式化出错
				logger.Error("RepoList update subscription retrieved error:%v", err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeUpdateSubs, err.Error()), nil)
				return
			}
		}
		subs = queryMinTime(subs)
		subsJson ,err := json.Marshal(subs)
		if err != nil {
			//json格式化出错
			logger.Error("RepoList subs json marshal error:%v", err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeJsonMarshalfail, err.Error()), nil)
			return
		}
		redisDb.Do("SET",repoName+itemName+user,subsJson)
		//插入查询的订购信息
	}
	//查询reids中的订购信息
	apiSubs,err = redisDb.Do("GET",repoName+itemName+user)
	if err != nil {
		//报错
		logger.Error("RepoList redis get subs error:%v", err.Error())
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeRedisGetFail, err.Error()), nil)
		return
	}

	subsArray := new(SubsArray)
	err = json.Unmarshal(apiSubs.([]byte),&subsArray.Subs)
	if err != nil {
		//报错
		logger.Error("RepoList subs json Unmarshal error:%v", err.Error())
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeJsonMarshalfail, err.Error()), nil)
		return
	}
	//请求地址
	reqUrl := apiInfo.UrlPath + query

	if apiInfo.IsHttps == Http {
		reqUrl = "http://"+apiInfo.UrlPath + query
	}else if apiInfo.IsHttps == Https {
		reqUrl = "https://"+apiInfo.UrlPath + query
	}


	i , err := httpGet(reqUrl)
	if err != nil {
		logger.Error("RepoList send url error:%v", err.Error())
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeUrlNotSupported, err.Error()), nil)
		return
	}
	var respBody interface{}
	if err := json.Unmarshal(i, &respBody); err != nil {
		logger.Error("unmarshal respbody error: %v", err.Error())
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeJsonMarshalfail, err.Error()), nil)
		return
	}

	//更新redis中的订购信息
	subsArray.Subs[0].Plan.Used += 1

	//检查是否完成订单
	if subsArray.Subs[0].Plan.Used == subsArray.Subs[0].Plan.Units {
		//更新订购服务中的订购信息
		err = updateSubs(subsArray.Subs[0],"set_plan_used")
		if err != nil {
			//更新sub失败
			logger.Error("RepoList update subscription set_plan_used error:%v", err.Error())
			api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeJsonMarshalfail, err.Error()), nil)
			return
		}
		if len(subsArray.Subs) == 1 {
			redisDb.Do("DEL",repoName+itemName+user)
		}else {
			subsArray.Subs = subsArray.Subs[1:]
			err = saveSlicesInredis(subsArray.Subs,redisDb,repoName,itemName,user)
			if err != nil {
				logger.Error("RepoList marshal subs err: %s", err.Error())
				api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeUpdateSubs, err.Error()), nil)
				return
			}
		}
	}
	err = saveSlicesInredis(subsArray.Subs,redisDb,repoName,itemName,user)
	if err != nil {
		logger.Error("RepoList marshal subs err: %s", err.Error())
		api.JsonResult(w, http.StatusBadRequest, api.GetError2(api.ErrorCodeUpdateSubs, err.Error()), nil)
		return
	}
	api.JsonResult(w, http.StatusOK, nil, respBody)

}

func saveSlicesInredis(subs []Subscription,redisDb redis.Conn,repoName,itemName,user string) error {
	subsJson ,err := json.Marshal(subs)
	if err != nil {
		return err
	}
	redisDb.Do("SET",repoName+itemName+user,subsJson)
	return nil
}

func queryMinTime(subsArray []Subscription)  []Subscription{
	if len(subsArray) <= 1 {
		return subsArray
	}
	mid, i := subsArray[0].Expire_time, 1
	head, tail := 0, len(subsArray)-1
	for head < tail {
		fmt.Println(subsArray)
		if subsArray[i].Expire_time.After(*mid) {
			subsArray[i], subsArray[tail] = subsArray[tail], subsArray[i]
			tail--
		} else {
			subsArray[i], subsArray[head] = subsArray[head], subsArray[i]
			head++
			i++
		}
	}
	subsArray[head].Expire_time = mid
	queryMinTime(subsArray[:head])
	queryMinTime(subsArray[head+1:])
	return subsArray
}

func inspectExist(apiParams , existParams []*models.Param) []*models.Param {
	for _ , existParam := range existParams {
		existParam.State = Delete
	}
	for _ , apiParam := range apiParams {
		apiParam.State = Insert
		if len(existParams) == 0 {
			existParams = append(existParams,apiParam)
		}else {
			lenth := len(existParams)
			for i := 0 ; i<lenth ; i++  {
				if apiParam.Name == existParams[i].Name {
					//fmt.Println("=====",k,"====",existParams[i].Name,"=====i:",i)
					//fmt.Println("=====",k,"====",apiParam.Name,"=====i:",i)
					existParams[i].State = Update
					existParams[i].Must = apiParam.Must
					existParams[i].Type = apiParam.Type
					break
				}else if i == len(existParams) - 1  {
					existParams = append(existParams,apiParam)
				}
			}
		}
	}
	return existParams
}