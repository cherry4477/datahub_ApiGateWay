package models

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/asiainfoLDP/datahub_ApiGateWay/log"
	"github.com/garyburd/redigo/redis"
	"github.com/miekg/dns"
)

const (
	Platform_Local      = "local"
	Platform_DaoCloud   = "daocloud"
	Platform_DaoCloudUT = "daocloud_ut"
	Platform_DataOS     = "dataos"
)

var (
	logger                       = log.GetLogger()
	Platform                     = Platform_Local
	masterName                   = "mymaster"
	sentinelTimeout              = time.Millisecond * 500
	GPool                        *redis.Pool
	redisConnStr                 string
	Service_Name_Redis           = Env("redis_service_name", false)
	DISCOVERY_CONSUL_SERVER_ADDR = Env("CONSUL_SERVER", false)
	DISCOVERY_CONSUL_SERVER_PORT = Env("CONSUL_DNS_PORT", false)
)

//================================================

type DbOrTx interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type dnsEntry struct {
	ip   string
	port string
}

//================================================

func Env(name string, required bool) string {
	s := os.Getenv(name)
	if required && s == "" {
		panic("env variable required, " + name)
	}
	logger.Info("[env]", name, s)
	return s
}

func InitDB() {

	for i := 0; i < 3; i++ {
		connectDB()

		if DB() == nil {
			select {
			case <-time.After(time.Second * 10):
				continue
			}
		} else {
			break
		}
	}

	if DB() == nil {
		logger.Error("dbInstance is nil.")
		return
	}

	upgradeDB()

	go updateDB()

	logger.Info("Init db succeed.")
	return
}

func InitRedis() {
	for i := 0; i < 3; i++ {
		initRedisPool()
		if GPool == nil {
			select {
			case <-time.After(time.Second * 10):
				logger.Info("sleep 10 secends")
				continue
			}
		} else {
			c := GPool.Get()

			if _, err := c.Do("PING"); err != nil {
				logger.Warn("err : ", err)
				GPool.Close()
				select {
				case <-time.After(time.Second * 10):
					continue
				}
			} else {
				c.Close()
				break
			}
		}
	}
	if GPool == nil {
		logger.Error("no redis connection")
		return
	}
}

func getRedisAddr() (string, string) {
	switch Platform {
	case Platform_DaoCloud:
		entryList := dnsExchange(Service_Name_Redis, DISCOVERY_CONSUL_SERVER_ADDR, DISCOVERY_CONSUL_SERVER_PORT)

		if len(entryList) > 0 {
			return entryList[0].ip, entryList[0].port
		}
	case Platform_DataOS:
		redisSentinelHost := os.Getenv(os.Getenv("ENV_NAME_REDIS_SENTINEL_HOST"))
		redisSentinelPort := os.Getenv(os.Getenv("ENV_NAME_REDIS_SENTINEL_PORT"))
		return getRedisMasterAddr(redisSentinelHost + ":" + redisSentinelPort)
	case Platform_Local:
		return os.Getenv("REDIS_PORT_6379_TCP_ADDR"), os.Getenv("REDIS_PORT_6379_TCP_PORT")
	}

	return "", ""
}

func dnsExchange(srvName, agentIp, agentPort string) []dnsEntry {
	Name := fmt.Sprintf("%s.service.consul", srvName)
	agentAddr := fmt.Sprintf("%s:%s", agentIp, agentPort)

	c := new(dns.Client)
	c.Net = "tcp"

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(Name), dns.TypeSRV)
	m.RecursionDesired = true

	result := []dnsEntry{}

	logger.Debug("Consul addr:", agentAddr)
	r, _, err := c.Exchange(m, agentAddr)
	if r == nil {
		logger.Error("dns query error: ", err.Error())
		return result
	}

	if r.Rcode != dns.RcodeSuccess {
		logger.Error("dns query error: ", r.Rcode)
		return result
	}

	for _, ex := range r.Extra {
		if tmp, ok := ex.(*dns.A); ok {
			result = append(result, dnsEntry{ip: tmp.A.String()})
		}
	}

	for i, an := range r.Answer {
		if tmp, ok := an.(*dns.SRV); ok {
			port := fmt.Sprintf("%d", tmp.Port)
			result[i].port = port
		}
	}

	return result
}

func getRedisMasterAddr(sentinelAddr string) (string, string) {
	if len(sentinelAddr) == 0 {
		logger.Info("Redis sentinelAddr is nil.")
		return "", ""
	}
	conn, err := redis.DialTimeout("tcp", sentinelAddr, sentinelTimeout, sentinelTimeout, sentinelTimeout)
	if err != nil {
		logger.Error("redis.DialTimeout(\"tcp\", ", sentinelAddr, sentinelTimeout, err)
		return "", ""
	}
	defer conn.Close()

	masterName = os.Getenv(os.Getenv("ENV_NAME_REDIS_CLUSTER_NAME"))
	logger.Debug("ENV_NAME_REDIS_CLUSTER_NAME", masterName)
	redisMasterPair, err := redis.Strings(conn.Do("SENTINEL", "get-master-addr-by-name", masterName))
	if err != nil {
		logger.Error("conn.Do(\"SENTINEL\", \"get-master-addr-by-name\",", masterName, err)
		return "", ""
	}
	logger.Info("get new master redis addr: ", redisMasterPair)
	if len(redisMasterPair) != 2 {
		return "", ""
	}
	return redisMasterPair[0], redisMasterPair[1]
}

func initRedisPool() {
	//redisAddr := os.Getenv("REDIS_PORT_6379_TCP_ADDR")
	//redisPort := os.Getenv("REDIS_PORT_6379_TCP_PORT")
	logger.Debug("BEGIN")
	defer logger.Debug("END")
	redisAddr, redisPort := getRedisAddr()
	if len(redisAddr) == 0 {
		redisAddr = "127.0.0.1"
	}
	if len(redisPort) == 0 {
		redisPort = "6379"
	}

	redisConnStr = redisAddr + ":" + redisPort
	logger.Info("redis connecting tcp", redisConnStr)

	password := getRedisPassword()
	GPool = newPool(&redisConnStr, password)
	if GPool == nil {
		logger.Error("newPool error, gPool == nil")
	}
}

func newPool(server *string, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     100,
		MaxActive:   5000, // max number of connections
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {

			c, err := redis.Dial("tcp", *server)
			if err != nil {
				return nil, err
			}

			if len(password) != 0 {
				if _, err := c.Do("AUTH", password); err != nil {
					logger.Error("...")
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func getRedisPassword() string {
	switch Platform {
	case Platform_DataOS:
		return os.Getenv(os.Getenv("ENV_NAME_REDIS_PASSWORD"))
	}

	return ""
}

func updateDB() {
	var err error
	ticker := time.Tick(5 * time.Second)
	for range ticker {
		db := GetDB()
		if db == nil {
			connectDB()
		} else if err = db.Ping(); err != nil {
			db.Close()
			// setDB(nil) // draw snake feet
			connectDB()
		}
	}
}

func GetDB() *sql.DB {
	if IsServing() {
		dbMutex.Lock()
		defer dbMutex.Unlock()
		return dbInstance
	} else {
		return nil
	}
}

func setDB(db *sql.DB) {
	dbMutex.Lock()
	dbInstance = db
	dbMutex.Unlock()
}

var (
	dbInstance *sql.DB
	dbMutex    sync.Mutex
)

func DB() *sql.DB {
	return dbInstance
}

func connectDB() {
	DB_ADDR, DB_PORT := MysqlAddrPort()
	DB_DATABASE, DB_USER, DB_PASSWORD := MysqlDatabaseUsernamePassword()
	logger.Info("Mysql_addr: %s\n"+
		"Mysql_port: %s\n"+
		"Myql_database: %s\n"+
		"Mysql_user: %s\n"+
		"Mysql_password: %s", DB_ADDR, DB_PORT, DB_DATABASE, DB_USER, DB_PASSWORD)

	DB_URL := fmt.Sprintf(`%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true`, DB_USER, DB_PASSWORD, DB_ADDR, DB_PORT, DB_DATABASE)

	logger.Info("connect to %s.", DB_URL)
	db, err := sql.Open("mysql", DB_URL) // ! here, err is always nil, db is never nil.
	if err == nil {
		err = db.Ping()
	}

	if err != nil {
		logger.Error("connect db error: %s.", err)
		//logger.Alert("connect db error: %s.", err)
	} else {
		setDB(db)
	}
}

func upgradeDB() {
	err := TryToUpgradeDatabase(DB(), "datafoundry:data_integration", os.Getenv("MYSQL_CONFIG_DONT_UPGRADE_TABLES") != "yes") // don't change the name
	if err != nil {
		logger.Error("TryToUpgradeDatabase error: %v.", err)
	}
}

func MysqlAddrPort() (string, string) {
	return os.Getenv(os.Getenv("ENV_NAME_MYSQL_ADDR")),
		os.Getenv(os.Getenv("ENV_NAME_MYSQL_PORT"))
}

func MysqlDatabaseUsernamePassword() (string, string, string) {

	return os.Getenv(os.Getenv("ENV_NAME_MYSQL_DATABASE")),
		os.Getenv(os.Getenv("ENV_NAME_MYSQL_USER")),
		os.Getenv(os.Getenv("ENV_NAME_MYSQL_PASSWORD"))

}
