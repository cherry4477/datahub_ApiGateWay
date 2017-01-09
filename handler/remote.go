package handler

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"bytes"
	"net/url"
)

var (
	API_SERVER         = Env("API_SERVER", false)
	API_PORT           = Env("API_PORT", false)
	USER_TP_ADMIN  int = 2
	USER_TP_UNKNOW     = -1
	AUTHORIZATION      = "Authorization"

	Username = Env("ADMIN_API_USERNAME", true)
	Password = Env("ADMIN_API_USER_PASSWORD", true)
)

type user struct {
	Loginname  string `json:"loginName,omitempty"`
	UserStatus int    `json:"userStatus,omitempty"`
}

type Plan struct {
	Units int `json:"units"`
	Used  int `json:"used"`
}

type Subscription struct {
	Subscription_id int64      `json:"subscriptionid,omitempty"`
	Pull_user_name  string     `json:"buyername,omitempty"`
	Repository_name string     `json:"repname,omitempty"`
	Dataitem_name   string     `json:"itemname,omitempty"`
	Supply_style    string     `json:"supply_style,omitempty"`
	Sign_time       *time.Time `json:"signtime,omitempty"`
	Expire_time     *time.Time `json:"expiretime,omitempty"`
	Plan            Plan       `json:"plan"`
}

type SubsResults struct {
	Total	int		`json:"total,omitempty"`
	Results	[]Subscription	`json:"results,omitempty"`
}

type SubsArray struct {
	Subs	[]Subscription	`json:"subs,omitempty"`
}

func getSubs(reponame, itemname, user string) ([]Subscription, error) {
	logger.Debug("getRealCreateUser BEGIN. login name %s.", user)
	user = url.QueryEscape(user)
	url := fmt.Sprintf("http://%s:%s/subscriptions/pull/%s/%s?username=%s", API_SERVER, API_PORT, reponame, itemname, user)
	token := getToken(Username, Password)
	b, err := httpGet(url, AUTHORIZATION, token)
	if err != nil {
		logger.Error("getSubs error:%v", err.Error())
		return nil, err
	}
	var result struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data SubsResults     `json:"data"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		logger.Error("unmarshal RealCreateUser err: %v", err.Error())
		return nil, err
	}

	return result.Data.Results, err
}

func updateSubs(sub Subscription, action string) (error) {
	logger.Debug("updateSubs BEGIN. subs id %s.", sub.Subscription_id)
	body := struct {
		Username	string	`json:"username"`
		Repname		string	`json:"repname"`
		Itemname	string	`json:"itemname"`
		Action		string	`json:"action"`
		Used		int	`json:"used"`
	}{Username : sub.Pull_user_name,Repname : sub.Repository_name,Itemname : sub.Dataitem_name}
	if action == "set_plan_used" {
		body.Action = action
		body.Used = sub.Plan.Used
	}
	if action == "set_retrieved" {
		body.Action = action
	}
	url := fmt.Sprintf("http://%s:%s/subscription/%s", API_SERVER, API_PORT, sub.Subscription_id)
	token := getToken(Username, Password)
	bodyjson , _ := json.Marshal(body)
	_ , err := HttpPutJson(url,bodyjson ,AUTHORIZATION, token)
	if err != nil {
		logger.Error("updateSub error:%v", err.Error())
		return err
	}
	return err
}

func httpGet(getUrl string, credential ...string) ([]byte, error) {
	var resp *http.Response
	var err error
	if len(credential) == 2 {
		req, err := http.NewRequest("GET", getUrl, nil)
		if err != nil {
			return nil, fmt.Errorf("[http] err %s, %s\n", getUrl, err)
		}
		req.Header.Set(credential[0], credential[1])
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			logger.Error("http get err:%s", err.Error())
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("[http get] status err %s, %d\n", getUrl, resp.StatusCode)
		}
	}else if len(credential) == 4 {
		req, err := http.NewRequest("GET", getUrl, nil)
		if err != nil {
			return nil, fmt.Errorf("[http] err %s, %s\n", getUrl, err)

		}
		req.Header.Set(credential[0], credential[1])
		req.Header.Set(credential[2], credential[3])
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			logger.Error("http get err:%s", err.Error())
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("[http get] status err %s, %d\n", getUrl, resp.StatusCode)
		}
	} else {
		resp, err = http.Get(getUrl)
		if err != nil {
			logger.Error("http get err:%s", err.Error())
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("[http get] status err %s, %d\n", getUrl, resp.StatusCode)
		}
	}
	return ioutil.ReadAll(resp.Body)
}

func HttpPutJson(putUrl string, body []byte, credential ...string) ([]byte, error) {
	var resp *http.Response
	var err error
	req, err := http.NewRequest("PUT", putUrl, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("[http] err %s, %s\n", putUrl, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(credential[0], credential[1])
	resp, err = http.DefaultClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("[http] err %s, %s\n", putUrl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[http] status err %s, %d\n", putUrl, resp.StatusCode)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[http] read err %s, %s\n", putUrl, err)
	}
	return b, nil
}

func Env(name string, required bool, showLog ...bool) string {
	s := os.Getenv(name)
	if required && s == "" {
		panic("env variable required, " + name)
	}
	if len(showLog) == 0 || showLog[0] {
		logger.Info("[env][%s] %s\n", name, s)
	}
	return s
}

func getToken(user, passwd string) string {
	passwdMd5 := getMd5(passwd)
	basic := fmt.Sprintf("Basic %s", string(base64Encode([]byte(fmt.Sprintf("%s:%s", user, passwdMd5)))))
	URL := fmt.Sprintf("http://%s:%s", API_SERVER, API_PORT)
	logger.Info("[Debug] http://%s basic %s", URL, basic)
	b, err := httpGet(URL, AUTHORIZATION, basic)
	logger.Info("[DEBUG] get token %s", string(b))
	if err != nil {
		logger.Error("get token err: %s", err.Error())
		return ""
	}
	var i interface{}
	if err := json.Unmarshal(b, &i); err != nil {
		logger.Error("unmarshal token err: %s", err.Error())
	}
	if m, ok := i.(map[string]interface{}); ok {
		if data, ok := m["data"].(map[string]interface{}); ok {
			if token, ok := data["token"].(string); ok {
				return "Token " + token
			}
		}
	}
	return ""
}

func valid(authuser, apiToken string) string {
	URL := fmt.Sprintf("http://%s:%s/valid", API_SERVER, API_PORT)
	logger.Info("[Debug] http://%s ", URL)
	b, err := httpGet(URL, AUTHORIZATION, "Token "+apiToken,"Authuser",authuser)
	logger.Info("[DEBUG] valid %s", string(b))
	if err != nil {
		logger.Error("valid err: %s", err.Error())
		return ""
	}
	var i interface{}
	if err := json.Unmarshal(b, &i); err != nil {
		logger.Error("unmarshal token err: %s", err.Error())
	}
	if m, ok := i.(map[string]interface{}); ok {
		if data, ok := m["data"].(map[string]interface{}); ok {
			if sregion, ok := data["sregion"].(string); ok {
				return sregion
			}
		}
	}
	return ""
}

func getMd5(content string) string {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(content))
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func base64Encode(src []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(src))
}
