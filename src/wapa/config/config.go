package config

/*
usage:

    configuration, _ := NewConfig("/tmp/.waparc")
    fmt.Printf("%+v", configuration)

content of `/tmp/.waparc`:

{
    "DBuser": "username",
    "DBpass": "password",
    "DBhost": "sql.yourhost.com",
    "DBport": "3306",
    "DBname": "dbname",
    "Encryption_key": "<some 32 bits encryption key>"
}

*/

import (
    "io/ioutil"
    "encoding/json"
)

type Config struct {
    DBuser string
    DBpass string
    DBhost string
    DBport string
    DBname string
    Encryption_key string
}

func NewConfig(configFile string) (*Config, error) {
    file, err := ioutil.ReadFile(configFile)
    if err != nil {
        return nil, err
    }
    var config Config
    json.Unmarshal(file, &config)
    return &config, nil
}
