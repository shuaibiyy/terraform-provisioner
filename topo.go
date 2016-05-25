package main

import (
	"fmt"
	"github.com/codeskyblue/go-sh"
	"github.com/libgit2/git2go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

// Represents a Topo config provision.
type Provision struct {
	Action     string
	State      string
	Parameters map[string]string
}

// Represents a Topo config file.
type Config struct {
	TfRepo     string `yaml:"tf_repo"`
	S3Bucket   string `yaml:"s3_bucket"`
	Provisions map[string]Provision
}

const Projects = "./projects"

// Directory where the Terraform project is saved to.
const Original = Projects + "/original"

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Git clone the Terraform project.
func cloneTfProj(repoUrl string) {
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

	if sh.Test("dir", Original) {
		sh.Command("rm", "-r", Original).Run()
	}

	fmt.Printf("git clone: %v\n", repoUrl)
	if _, err := git.Clone(repoUrl, Original, cloneOpts); err != nil {
		fmt.Println("clone error:", err)
	}
}

// Gets called when authentication credentials are requested during git clone.
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

// Reads the yaml config passed to Topo and returns it as a string.
func getConfigYaml(configFile string) string {
	config, err := ioutil.ReadFile(configFile)
	check(err)
	return string(config)
}

// Takes the yaml config string and unmarshals it to a Config struct.
func getConfig(config string) Config {
	c := Config{}

	err := yaml.Unmarshal([]byte(config), &c)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return c
}

// Make copies of the Terraform project for each provision.
func mkProjCopies(c *Config) {
	fmt.Println()

	ch := make(chan string, len(c.Provisions))
	for k := range c.Provisions {
		s := k
		go func() {
			fmt.Printf("creating copy of project for: %v\n", s)
			sh.Command("cp", "-rf", Original, Projects+"/"+s).Run()
			ch <- s
		}()
	}
	for i := 0; i < len(c.Provisions); i++ {
		fmt.Printf("done copying: %v\n", <-ch)
	}
}

// Configure remote states for provisions.
func configureRemoteStates(c *Config) {
	ch := make(chan string, len(c.Provisions))
	for k := range c.Provisions {
		s := k
		go func() {
			fmt.Printf("initialize remote state file for: %v\n", s)
			sh.Command("terraform", "remote", "config", "-backend=s3",
				"-backend-config='bucket=" + c.S3Bucket + "'",
				"-backend-config='key=" + s + "/terraform.tfstate'",
				sh.Dir(Projects+"/"+s)).Run()
			ch <- s
		}()
	}
	for i := 0; i < len(c.Provisions); i++ {
		fmt.Printf("done configuring remote state: %v\n", <-ch)
	}
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

	cloneTfProj(config.TfRepo)
	mkProjCopies(&config)
	configureRemoteStates(&config)
}
