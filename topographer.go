package main

import (
    "fmt"
    "os"
    "io/ioutil"
    "log"
    "gopkg.in/yaml.v2"
    "github.com/libgit2/git2go"
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

func cloneGitRepo(repoUrl string) {
    username := os.Getenv("TP_GIT_USER")
    password := os.Getenv("TP_GIT_PASSWORD")
    folder := "./project"

    callbacks := git.RemoteCallbacks{
        CredentialsCallback: makeCredentialsCallback(username, password),
    }

    cloneOpts := &git.CloneOptions{
        FetchOptions: &git.FetchOptions{
            RemoteCallbacks: callbacks,
        },
    }

    fmt.Printf("git clone: %v\n", repoUrl)

    if _, err := git.Clone(repoUrl, folder, cloneOpts); err != nil {
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

func main() {
    args := os.Args[1:]

    if len(args) < 1 {
        fmt.Println("usage: topographer <config_file>")
        os.Exit(2)
    }

    configFile := args[0]
    fmt.Printf("topographer configuration file: %v\n\n", configFile)

    config := getConfig(getConfigYaml(configFile))
    fmt.Printf("--- config:\n%v\n\n", config)

    cloneGitRepo(config.GitRepo)
}

