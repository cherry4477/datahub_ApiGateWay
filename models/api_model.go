package models

import (
	"database/sql"
	"fmt"
)

type ApiInfo struct {
	Id              int    `json:"id,omitempty"`
	Reqrepo         string `json:"reqrepo,omitempty"`
	Reqitem         string `json:"reqitem,omitempty"`
	CreateUser      string `json:"createuser,omitempty"`
	ReqResourcePage string `json:"reqresourcepage,omitempty"`
	UrlPath         string `json:"urlpath,omitempty"`
	ReqHostInfo     string `json:"reqhostinfo,omitempty"`
	ReqAppKey       string `json:"reqappkey,omitempty"`
	IsVerify        int    `json:"isverify,omitempty"`
	IsHttps         int    `json:"ishttps,omitempty"`
	QueryTimes      int64  `json:"querytimes,omitempty"`
	ReqType         int    `json:"reqtype,omitempty"`
	PostTemplate    string `json:"posttemplate,omitempty"`
}

type Param struct {
	Id    int    `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Must  int    `json:"must,omitempty"`
	Type  string `json:"type,omitempty"`
	ApiId int    `json:"apiId,omitempty"`
	State int    `json:"state,omitempty"`
}

type ApiItem struct {
	Url    string  `json:"url,omitempty"`
	Method int     `json:"method,omitempty"`
	Verify int     `json:"verify,omitempty"`
	Key    string  `json:"key,omitempty"`
	Params []*Param `json:"params,omitempty"`
}

//插入apiinfo
func InsertApiInfo(db *sql.DB, apiInfo *ApiInfo) (*ApiInfo, error) {
	logger.Info("Model begin insert ApiInfo")
	defer logger.Info("Model end insert ApiInfo")

	sqlstr := "insert into api_gateway_config (reqrepo,reqitem, reqresourcepage, urlpath, reqappkey," +
		"isverify, ishttps, querytimes,reqtype,createuser) " +
		"values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	_, err := db.Exec(sqlstr, apiInfo.Reqrepo, apiInfo.Reqitem, apiInfo.ReqResourcePage, apiInfo.UrlPath, apiInfo.ReqAppKey,
		apiInfo.IsVerify, apiInfo.IsHttps, apiInfo.QueryTimes, apiInfo.ReqType, apiInfo.CreateUser)

	if err != nil {
		return nil, err
	}
	apiInfo, err = QueryApiInfo(db, apiInfo.Reqrepo, apiInfo.Reqitem, apiInfo.CreateUser)

	return apiInfo, err

}

//更新apiinfo
func UpdateApiInfo(db *sql.DB, apiInfo *ApiInfo) (*ApiInfo, error) {
	logger.Info("Model begin update ApiInfo")
	defer logger.Info("Model end update ApiInfo")

	sqlstr := "update api_gateway_config set reqresourcepage = ? , urlpath = ?, reqappkey = ?," +
		"isverify = ?, ishttps = ?, querytimes = ?,reqtype = ? where reqrepo = ? and reqitem = ?"
	_, err := db.Exec(sqlstr, apiInfo.ReqResourcePage, apiInfo.UrlPath, apiInfo.ReqAppKey, apiInfo.IsVerify,
		apiInfo.IsHttps, apiInfo.QueryTimes, apiInfo.ReqType, apiInfo.Reqrepo, apiInfo.Reqitem)

	if err != nil {
		return nil, err
	}
	apiInfo, err = QueryApiInfo(db, apiInfo.Reqrepo, apiInfo.Reqitem, apiInfo.CreateUser)

	return apiInfo, err
}

//查询apiinfo
func QueryApiInfo(db *sql.DB, reqrepo, reqitem, user string) (*ApiInfo, error) {
	logger.Info("Model begin query ApiInfo")
	defer logger.Info("Model end query ApiInfo")

	apiInfo := new(ApiInfo)

	sqlstr := "select id,reqrepo,reqitem,reqresourcepage,urlpath,reqappkey," +
		" isverify,ishttps,querytimes,reqtype " +
		"from api_gateway_config where reqrepo = ? and reqitem = ? and createuser = ?"

	err := db.QueryRow(sqlstr, reqrepo, reqitem, user).Scan(&apiInfo.Id, &apiInfo.Reqrepo, &apiInfo.Reqitem,
		&apiInfo.ReqResourcePage, &apiInfo.UrlPath,
		&apiInfo.ReqAppKey, &apiInfo.IsVerify, &apiInfo.IsHttps, &apiInfo.QueryTimes, &apiInfo.ReqType)
	if err != nil {
		return nil, err
	}
	return apiInfo, nil
}

func InsertParam(db *sql.DB, param *Param) error {
	logger.Info("Model begin insert param")
	defer logger.Info("Model end insert param")

	sqlstr := "insert into api_param (name , must , type , apiid) value(?,?,?,?)"
	_, err := db.Exec(sqlstr, param.Name, param.Must, param.Type, param.ApiId)
	if err != nil {
		return err
	}
	return nil
}

func UpdateParam(db *sql.DB, param *Param) error {
	logger.Info("Model begin update param")
	defer logger.Info("Model end update param")

	sqlstr := "update api_param set name = ? , must = ? ,type = ? ,apiid = ? where id = ?"
	_, err := db.Exec(sqlstr, param.Name, param.Must, param.Type, param.ApiId,param.Id)
	if err != nil {
		return err
	}
	return nil
}

func QueryParamList(db *sql.DB, apiId int) ([]*Param, error) {
	logger.Debug("QueryParamList begin")

	sqlParams := make([]interface{}, 0, 2)
	sqlwhere := "apiId=?"

	sqlParams = append(sqlParams, apiId)

	params, err := queryParams(db, sqlwhere, sqlParams...)

	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return params, nil

}

func queryParams(db *sql.DB, sqlwhere string, sqlParams ...interface{}) ([]*Param, error) {

	logger.Info("Model begin QueryParamList")
	defer logger.Info("Model end QueryParamList")

	sqlwhereall := ""
	if sqlwhere != "" {
		sqlwhereall = fmt.Sprintf("where %s", sqlwhere)
	}
	sqlstr := fmt.Sprintf(`SELECT id ,name ,must , type
		FROM api_param
		%s`,
		sqlwhereall)

	logger.Info(">>> %v", sqlstr)
	rows, err := db.Query(sqlstr, sqlParams...)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	defer rows.Close()

	params := make([]*Param, 0, 32)
	for rows.Next() {
		param := &Param{}

		err := rows.Scan(&param.Id,&param.Name, &param.Must, &param.Type)

		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return params, nil
}

func DeleteParams(db *sql.DB, param *Param) error {
	logger.Info("Model begin delete param")
	defer logger.Info("Model end delete param")

	sqlstr := "delete from api_param where id = ?"
	_, err := db.Exec(sqlstr,param.Id)
	if err != nil {
		return err
	}
	return nil
}