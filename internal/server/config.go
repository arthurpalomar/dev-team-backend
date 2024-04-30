package server

import (
	"encoding/json"
	"os"
)

type Config struct {
	SshHost             string `json:"sshHost"`
	SshPort             int    `json:"sshPort"`
	SshUser             string `json:"sshUser"`
	SshPass             string `json:"sshPass"`
	DbHost              string `json:"dbHost"`
	MysqlPort           string `json:"mysqlPort"`
	DbName              string `json:"dbName"`
	DbUser              string `json:"dbUser"`
	DbPass              string `json:"dbPass"`
	WorkerSpeed         int    `json:"workerSpeed"`
	WorkerQueue         int    `json:"workerQueue"`
	ActionLimitHour     uint   `json:"accountActionLimitHour"`
	TaskActionLimit     int64  `json:"taskActionLimit"`
	FileLog             string `json:"fileLog"`
	Port                string `json:"port"`
	Ssl                 bool   `json:"ssl"`
	SslCert             string `json:"sslCert"`
	SslKey              string `json:"sslKey"`
	UseTokens           bool   `json:"useTokens"`
	ApiTokens           string `json:"apiTokens"`
	CentrifugeHost      string `json:"CentrifugeHost"`
	CentrifugePort      string `json:"CentrifugePort"`
	CentrifugeKey       string `json:"CentrifugeKey"`
	CentrifugeSecret    string `json:"CentrifugeSecret"`
	VerifiedMinLenQueue int    `json:"verifiedMinLenQueue"`
	CommonMinLenQueue   int    `json:"commonMinLenQueue"`
	CookiePoolStatus    int    `json:"cookiePoolStatus"`
}

var GlobalConfig Config
var PathFile string

func ConfigLoad() {
	var err error

	if len(os.Args) > 1 {
		PathFile = os.Args[1]
	} else {
		PathFile = "./config.json"
	}

	configFile, err := os.Open(PathFile)
	defer configFile.Close()
	if err != nil {
		panic(err)
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&GlobalConfig)

	SetLogger(GlobalConfig.FileLog)
}
