package main

import (
	"flag"
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
func cloneTfProj(repoUrl string, update bool) bool {
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

	// If repo exists and update flag is set to true, remove project directory and clone.
	if sh.Test("dir", Original) {
		if update {
			sh.Command("rm", "-r", Projects).Run()
		} else {
			return false
		}
	}

	log.Printf("git clone: %v\n", repoUrl)
	if _, err := git.Clone(repoUrl, Original, cloneOpts); err != nil {
		log.Println("clone error: ", err)
	}
	return true
}

// Gets called when authentication credentials are requested during git clone.
func makeCredentialsCallback(username, password string) git.CredentialsCallback {
	// If we're trying, it means the credentials are invalid.
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
	ch := make(chan string, len(c.Provisions))
	for k := range c.Provisions {
		s := k
		go func() {
			log.Printf("creating copy of project for: %v\n", s)
			sh.Command("cp", "-rf", Original, Projects+"/"+s).Run()
			ch <- s
		}()
	}
	for i := 0; i < len(c.Provisions); i++ {
		log.Printf("done copying: %v\n", <-ch)
	}
}

// Configure remote states for provisions.
func configureRemoteStates(c *Config) {
	ch := make(chan string, len(c.Provisions))
	for k := range c.Provisions {
		name := k
		go func() {
			log.Printf("initialize remote state file for: %v\n", name)
			sh.Command("terraform", "remote", "config", "-backend=s3",
				"-backend-config", "bucket="+c.S3Bucket,
				"-backend-config", "key="+name+"/terraform.tfstate",
				sh.Dir(Projects+"/"+name)).Run()
			ch <- name
		}()
	}
	for i := 0; i < len(c.Provisions); i++ {
		log.Printf("done configuring remote state: %v\n", <-ch)
	}
}

func computeQualifiedConfig(c *Config) *Config {
	for k, v := range c.Provisions {
		if v.State == "changed" && v.Action == "destroy" {
			delete(c.Provisions, k)
		}
		if v.State == "destroyed" && v.Action == "destroy" {
			delete(c.Provisions, k)
		}
		if v.State == "applied" && v.Action == "apply" {
			delete(c.Provisions, k)
		}
	}
	return c
}

// Run terraform commands for all provisions.
func provision(c *Config) {
	ch := make(chan string, len(c.Provisions))
	for k, v := range c.Provisions {
		name := k
		switch v.Action {
		case "apply":
			args := prepareApplyArgs(&v)
			go runTfCmd(name, args, ch)
		case "destroy":
			args := prepareDestroyArgs(&v)
			go runTfCmd(name, args, ch)
		default:
			log.Printf("unknown action '%v' for provision '%v'. skipping...\n", v.Action, name)
		}
	}
	log.Printf("completed terraform command for: %v\n", <-ch)
}

func runTfCmd(name string, cmdArgs []interface{}, c chan string) {
	s := sh.NewSession()
	s.ShowCMD = true
	s.SetDir(Projects + "/" + name)
	s.Command("terraform", cmdArgs...).Run()
	c <- name
}

func prepareDestroyArgs(p *Provision) []interface{} {
	return prepareArgs([]string{"destroy", "-force"}, p)
}

func prepareApplyArgs(p *Provision) []interface{} {
	return prepareArgs([]string{"apply"}, p)
}

func prepareArgs(action []string, p *Provision) []interface{} {
	args := []string{}
	args = append(args, action...)
	for k1, v1 := range p.Parameters {
		args = append(args, "-var", k1+"="+v1)
	}
	cmdArgs := make([]interface{}, len(args))
	for i, v := range args {
		cmdArgs[i] = v
	}
	return cmdArgs
}

func topo(c *Config, flags map[string]interface{}) {
	if val, ok := flags["update"].(bool); ok {
		if cloneTfProj(c.TfRepo, val) {
			mkProjCopies(c)
		}
	}
	computeQualifiedConfig(c)
	configureRemoteStates(c)
	provision(c)
}

func main() {
	flags := make(map[string]interface{})
	update := flag.Bool("update", false, "guarantees that the terraform project will be fetched from remote")

	flag.Parse()
	flags["update"] = *update

	if len(flag.Args()) < 1 {
		log.Println("usage: topo [flags...] <config_file>")
		os.Exit(2)
	}

	configFile := flag.Args()[0]
	log.Printf("topo configuration file: %v\n", configFile)

	config := getConfig(getConfigYaml(configFile))
	log.Printf("--- config:\n%v\n", config)

	topo(&config, flags)
}
