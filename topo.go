package main

import (
    "fmt"
    "os"
    "io/ioutil"
    "log"
    "gopkg.in/yaml.v2"
    "github.com/libgit2/git2go"
    "github.com/codeskyblue/go-sh"
)

type Provision struct {
    Action string
    State string
    Parameters map[string]string
}

type Config struct {
    TfRepo string `yaml:"tf_repo"`
    Provisions map[string]Provision
}

const TfDir = "./project"

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func cloneTfRepo(repoUrl string) {
    username := os.Getenv("TP_GIT_USER")
    password := os.Getenv("TP_GIT_PASSWORD")

    callbacks := git.RemoteCallbacks{
        CredentialsCallback: makeCredentialsCallback(username, password),
    }

    cloneOpts := &git.CloneOptions{
        FetchOptions: &git.FetchOptions{
            RemoteCallbacks: callbacks,
        },
    }

    if sh.Test("dir", TfDir) {
        sh.Command("rm", "-r", TfDir).Run()
    }

    fmt.Printf("git clone: %v\n", repoUrl)
    if _, err := git.Clone(repoUrl, TfDir, cloneOpts); err != nil {
        fmt.Println("clone error:", err)
    }
}

func makeCredentialsCallback(username, password string) git.CredentialsCallback {
    // If we're trying it means the credentials are invalid
    called := false
    return func(url string, username_from_url string, allowed_types git.CredType) (git.ErrorCode, *git.Cred) {
        if called {
            return git.ErrUser, nil
        }
        called = true
        errCode, cred := git.NewCredUserpassPlaintext(username, password)
        return git.ErrorCode(errCode), &cred
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

func tfApply() {
    sh.Command("terraform", "apply", sh.Dir(TfDir)).Run()
}

func tfDestroy() {
    sh.Command("terraform", "destroy", "-force", sh.Dir(TfDir)).Run()
}

func tfPlan() {
    sh.Command("terraform", "plan", sh.Dir(TfDir)).Run()
}

func main() {
    args := os.Args[1:]

    if len(args) < 1 {
        fmt.Println("usage: topo <config_file>")
        os.Exit(2)
    }

    configFile := args[0]
    fmt.Printf("topo configuration file: %v\n\n", configFile)

    config := getConfig(getConfigYaml(configFile))
    fmt.Printf("--- config:\n%v\n\n", config)

    cloneTfRepo(config.TfRepo)

    tfPlan()
}
