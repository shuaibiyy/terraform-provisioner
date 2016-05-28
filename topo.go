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

// Allowed provision states.
type State string

const (
	Applied   State = "applied"
	Destroyed       = "destroyed"
	Changed         = "changed"
)

// Allowed provision actions.
type Action string

const (
	Apply   Action = "apply"
	Destroy        = "destroy"
)

// Represents a provision.
type Provision struct {
	Action     Action
	State      State
	Parameters map[string]string
}

// Represents a Topo config file.
type Config struct {
	TfRepo     string `yaml:"tf_repo"`
	S3Bucket   string `yaml:"s3_bucket"`
	Provisions map[string]Provision
}

// Information about a Terraform command.
type CmdInfo struct {
	name   string
	action Action
}

// Information about a copy command.
type CopyInfo struct {
	dest   string
	copied bool
}

// Topo configuration file.
var ConfigFile string

const Projects = "./projects"

// Directory where the Terraform project is saved to.
const Original = Projects + "/original"

// Git clone the Terraform project.
func cloneTfProj(repoUrl string, update bool) bool {
	username := os.Getenv("TP_GIT_USER")
	password := os.Getenv("TP_GIT_PASSWORD")
	callbacks := git.RemoteCallbacks{
		CredentialsCallback: getCredentialsCallback(username, password),
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
func getCredentialsCallback(username, password string) git.CredentialsCallback {
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
	func(e error) {
		if e != nil {
			panic(e)
		}
	}(err)
	backupConfig(configFile)
	return string(config)
}

func backupConfig(s string) {
	sh.Command("cp", s, s+".bak").Run()
}

// Takes the yaml config string and unmarshals it into a Config struct.
func getConfig(config string) Config {
	c := Config{}
	err := yaml.Unmarshal([]byte(config), &c)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return c
}

// Make copies of the Terraform project for each provision.
func mkProjCopies(c *Config, updated bool) {
	ch := make(chan CopyInfo, len(c.Provisions))
	for k := range c.Provisions {
		s := k
		go func() {
			// If there was no remote fetch, then there's no need to copy a project that already exists.
			if !updated {
				if sh.Test("dir", Projects+"/"+s) {
					ch <- CopyInfo{s, false}
					return
				}
			}
			log.Printf("creating copy of project for: %v\n", s)
			sh.Command("cp", "-rf", Original, Projects+"/"+s).Run()
			ch <- CopyInfo{s, true}
		}()
	}
	for i := 0; i < len(c.Provisions); i++ {
		msg := <-ch
		switch msg.copied {
		case true:
			log.Printf("done copying: %v\n", msg.dest)
		case false:
			log.Printf("copy already exists for: %v\n", msg.dest)
		}
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

// Returns a tuple of qualified and unqualified provisions.
func computeQualifiedProvisions(c *Config) (map[string]Provision, map[string]Provision) {
	unqualified := make(map[string]Provision)
	for k, v := range c.Provisions {
		if v.State == Changed && v.Action == Destroy {
			unqualified[k] = v
			delete(c.Provisions, k)
		}
		if v.State == Destroyed && v.Action == Destroy {
			unqualified[k] = v
			delete(c.Provisions, k)
		}
		if v.State == Applied && v.Action == Apply {
			unqualified[k] = v
			delete(c.Provisions, k)
		}
		if v.Action != Apply && v.Action != Destroy {
			unqualified[k] = v
			delete(c.Provisions, k)
		}
	}
	return c.Provisions, unqualified
}

// Run terraform commands for all provisions.
func provision(c *Config, uq map[string]Provision) bool {
	ch := make(chan CmdInfo, len(c.Provisions))
	for k, v := range c.Provisions {
		name := k
		switch v.Action {
		case Apply:
			args := prepareApplyArgs(&v)
			go runTfCmd(name, args, ch, v.Action)
		case Destroy:
			args := prepareDestroyArgs(&v)
			go runTfCmd(name, args, ch, v.Action)
		}
	}
	return updateProvisions(c, ch, uq)
}

func updateProvisions(c *Config, ch chan CmdInfo, uq map[string]Provision) bool {
	for i := 0; i < len(c.Provisions); i++ {
		msg := <-ch
		tmp := c.Provisions[msg.name]
		log.Printf("completed terraform %v on: %v\n", msg.action, msg.name)
		switch msg.action {
		case Apply:
			tmp.State = Applied
		case Destroy:
			tmp.State = Destroyed
		}
		c.Provisions[msg.name] = tmp
	}
	return saveConfig(c, uq)
}

func saveConfig(c *Config, uq map[string]Provision) bool {
	for k, v := range uq {
		c.Provisions[k] = v
	}
	d, err := yaml.Marshal(&c)
	if err != nil {
		log.Fatalf("error: %v", err)
		return false
	}
	err = ioutil.WriteFile(ConfigFile, d, 0644)
	if err != nil {
		log.Fatalf("error: %v", err)
		return false
	}
	log.Printf("--- config saved:\n%s\n---\n", string(d))
	return true
}

func runTfCmd(name string, cmdArgs []interface{}, c chan CmdInfo, action Action) {
	s := sh.NewSession()
	s.ShowCMD = true
	s.SetDir(Projects + "/" + name)
	s.Command("terraform", cmdArgs...).Run()
	c <- CmdInfo{name, action}
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
		mkProjCopies(c, cloneTfProj(c.TfRepo, val))
	}
	_, unqualified := computeQualifiedProvisions(c)
	configureRemoteStates(c)
	switch provision(c, unqualified) {
	case true:
		log.Println("topo succeeded!")
		os.Exit(0)
	case false:
		log.Println("topo failed :(")
		os.Exit(1)
	}
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

	ConfigFile = flag.Args()[0]
	log.Printf("topo configuration file: %v\n", ConfigFile)

	config := getConfig(getConfigYaml(ConfigFile))
	log.Printf("--- config read:\n%v\n---\n", config)

	topo(&config, flags)
}
