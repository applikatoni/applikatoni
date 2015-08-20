package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
)

const (
	configurationFileName = ".toni.yml"
	apiTokenHeaderName    = "X-Api-Token"
	gitBranchCmd          = "git symbolic-ref --short HEAD"
	gitCommitCmd          = "git rev-parse HEAD"
)

var usage = `toni is the CLI for Applikatoni.

Usage:

	toni [-c=<commit SHA>] [-b=<commit branch>] -t <target> -m <comment>

Arguments:

	REQUIRED:

	-t		Target of the deployment

	-m		The deployment comment

	OPTIONAL:

	-c      The deployment commit SHA
			(if unspecified toni uses the current git HEAD)

	-b      The branch of the commit SHA
			(if unspecified toni uses the current git HEAD)

Configuration:

When starting up, toni tries to read the ".toni.yml" configuration file in the
current working directoy. The configuration MUST specify the HOST, APPLICATION,
API TOKEN and the STAGES of deployments.

An example ".toni.yml" looks like this:

	host: http://toni.shippingcompany.com
	application: shippingcompany-main-application
	api_token: 4fdd575f-FOOO-BAAR-af1e-ce3e9f75367d
	stages:
	  production:
		- CHECK_CONNECTION
		- PRE_DEPLOYMENT
		- CODE_DEPLOYMENT
		- POST_DEPLOYMENT
	  staging:
		- CHECK_CONNECTION
		- PRE_DEPLOYMENT
		- CODE_DEPLOYMENT
		- POST_DEPLOYMENT
`

var (
	target    string
	comment   string
	branch    string
	commitSHA string

	printHelp bool
)

var (
	currentDeploymentLocation string
	currentDeploymentMx       *sync.Mutex
)

func init() {
	flag.StringVar(&target, "t", "", "Target of the deployment [required]")
	flag.StringVar(&comment, "m", "", "Deployment comment [required]")
	flag.StringVar(&commitSHA, "c", "", "Deployment commit")
	flag.StringVar(&branch, "b", "", "Deployment branch")

	flag.BoolVar(&printHelp, "h", false, "Print the help and usage information")
}

func main() {
	flag.Parse()
	if printHelp {
		printUsage()
		os.Exit(0)
	}

	configurationFilePath, err := filepath.Abs(configurationFileName)
	if err != nil {
		configErrorExit("Error when trying to open configuration file (%s):\n",
			configurationFileName,
			err)
	}

	config, err := readConfiguration(configurationFilePath)
	if err != nil {
		configErrorExit("Error when trying to parse configuration file (%s):\n",
			configurationFileName,
			err)
	}

	err = config.Validate()
	if err != nil {
		configErrorExit("Configuration file is invalid (%s):\n",
			configurationFileName,
			err)
	}

	if target == "" {
		fmt.Fprintf(os.Stderr, "No target specified (use -t option to specify target)\n")
		os.Exit(1)
	}

	if _, ok := config.Stages[target]; !ok {
		fmt.Fprintf(os.Stderr, "Target %q not found in configuration file\n", target)
		os.Exit(1)
	}

	if comment == "" {
		fmt.Fprintf(os.Stderr, "No comment specified (use -m option to specify comment)\n")
		os.Exit(1)
	}

	if branch == "" {
		branch, err = runGitCmd(gitBranchCmd)
		if branch == "" || err != nil {
			fmt.Fprintf(os.Stderr, "Getting current branch with `%s` failed:\n", gitBranchCmd)
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	}

	if commitSHA == "" {
		commitSHA, err = runGitCmd(gitCommitCmd)
		if commitSHA == "" || err != nil {
			fmt.Fprintf(os.Stderr, "Getting current commit SHA with `%s` failed:\n", gitCommitCmd)
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	}

	currentDeploymentMx = &sync.Mutex{}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		killCurrentDeployment(config)
	}()

	fmt.Println("Creating deployment...\n")
	printDeploymentAttribute("host", config.Host)
	printDeploymentAttribute("application", config.Application)
	printDeploymentAttribute("target", target)
	printDeploymentAttribute("sha", commitSHA)
	printDeploymentAttribute("branch", branch)
	printDeploymentAttribute("comment", comment)
	printDeploymentAttribute("stages", strings.Join(config.Stages[target], ", "))

	deploymentURL, err := buildDeploymentURL(config.Host, config.Application)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating a deployment failed: %s\n", err)
		os.Exit(1)
	}

	data := buildDeploymentData(target, commitSHA, branch, comment, config.Stages[target])
	deploymentLocation, err := createDeployment(config, deploymentURL, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating a deployment failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("\nSuccessfully created. Streaming logs...\n")

	setCurrentDeploymentLocation(deploymentLocation)

	logURL, err := buildDeploymentLogURL(config.Host, deploymentLocation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "streaming deployment logs failed: %s\n", err)
		os.Exit(1)
	}

	err = streamDeploymentLog(config, logURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "streaming deployment logs failed: %s\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(usage)
}

func configErrorExit(message, configPath string, err error) {
	fmt.Fprintf(os.Stderr, message, configPath)
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func runGitCmd(gitCmd string) (string, error) {
	argv := strings.Split(gitCmd, " ")

	out, err := exec.Command(argv[0], argv[1:]...).Output()
	if err != nil {
		return "", err
	}

	return strings.Trim(string(out), "\n"), nil
}

func printDeploymentAttribute(name, val string) {
	fmt.Printf("\t%-15s = %s\n", name, val)
}

func setCurrentDeploymentLocation(location string) {
	currentDeploymentMx.Lock()
	currentDeploymentLocation = location
	currentDeploymentMx.Unlock()
}

func killCurrentDeployment(c *Configuration) {
	currentDeploymentMx.Lock()
	defer currentDeploymentMx.Unlock()

	if currentDeploymentLocation != "" {
		fmt.Printf("\nReceived INTERRUPT - ")
		fmt.Printf("Sending kill request to server (%s)...\n\n",
			currentDeploymentLocation)

		killURL, err := buildDeploymentKillURL(c.Host, currentDeploymentLocation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Killing deployment failed: %s\n", err)
			return
		}

		err = killDeployment(c, killURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Killing deployment failed: %s\n", err)
			return
		}
	}
}
