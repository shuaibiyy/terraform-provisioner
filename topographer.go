package main

import (
    "fmt"
    "os"
    "io/ioutil"
    "log"
    "gopkg.in/yaml.v2"
)

type Provision struct {
    Action string
    State string
    Parameters map[string]string
}

type Config struct {
    GitRepo string `yaml:"git_repo"`
    Provisions map[string]Provision
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func getConfigYaml(configFile string) string {
    config, err := ioutil.ReadFile(configFile)
    check(err)
    return string(config)
}

func getConfig(config string) Config {
    c := Config{}

    err := yaml.Unmarshal([]byte(config), &c)
    if err != nil {
        log.Fatalf("error: %v", err)
    }
    return c
}

func main() {
    args := os.Args[1:]

    if len(args) < 1 {
        fmt.Println("usage: topographer <config_file>")
        os.Exit(2)
    }

    configFile := args[0]
    fmt.Printf("topographer configuration file: %v\n", configFile)

    config := getConfig(getConfigYaml(configFile))
    fmt.Printf("--- Config:\n%v\n\n", config)
}

