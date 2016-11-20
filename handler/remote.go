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

func getSubs(reponame, itemname, user string) ([]*Subscription, error) {
	logger.Debug("getRealCreateUser BEGIN. login name %s.", user)

	url := fmt.Sprintf("http://%s:%s/subscriptions/pull/%s/%s?username=s%", API_SERVER, API_PORT, reponame, itemname, user)
	token := getToken(Username, Password)
	b, err := httpGet(url, AUTHORIZATION, token)
	if err != nil {
		logger.Error("getSubs error:%v", err.Error())
		return nil, err
	}
	var result struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data []*Subscription `json:"data"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		logger.Error("unmarshal RealCreateUser err: %v", err.Error())
		return nil, err
	}

	return result.Data, err
}

func getUserStatus(createuser string) int {
	logger.Debug("getRealCreateUser BEGIN. login name %s.", createuser)

	url := fmt.Sprintf("http://%s:%s/users/%s", API_SERVER, API_PORT, createuser)
	b, err := httpGet(url)
	if err != nil {
		logger.Error("getFullCreateUser error:%v", err.Error())
		return USER_TP_UNKNOW
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data user   `json:"data"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		logger.Error("unmarshal RealCreateUser err: %v", err.Error())
		return USER_TP_UNKNOW
	}

	userStatus := result.Data.UserStatus
	return userStatus
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

func getMd5(content string) string {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(content))
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func base64Encode(src []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(src))
}
