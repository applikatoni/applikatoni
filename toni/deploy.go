package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/flinc/applikatoni/deploy"
	"github.com/gorilla/websocket"
)

func streamDeploymentLog(c *Configuration, deploymentLogURL *url.URL) error {
	urlStr := deploymentLogURL.String()
	wsHeaders := http.Header{"Origin": {c.Host}, apiTokenHeaderName: {c.ApiToken}}

	wsConn, _, err := websocket.DefaultDialer.Dial(urlStr, wsHeaders)
	if err != nil {
		return err
	}

	logs := make(chan deploy.LogEntry)
	defer func() {
		close(logs)
	}()

	go deploy.ConsoleLogger(logs)

	for {
		entry := deploy.LogEntry{}
		err := wsConn.ReadJSON(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		logs <- entry
	}

	return nil
}

type UnexpectedResponse struct {
	ResponseCode int
	ResponseBody string
}

func (ur UnexpectedResponse) Error() string {
	if ur.ResponseCode == 302 {
		return "Not authorized"
	} else {
		return fmt.Sprintf("%d - %s", ur.ResponseCode, ur.ResponseBody)
	}
}

func createDeployment(c *Configuration, deploymentURL *url.URL, data url.Values) (string, error) {
	req, err := http.NewRequest("POST", deploymentURL.String(), bytes.NewBufferString(data.Encode()))
	req.Header.Set(apiTokenHeaderName, c.ApiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusSeeOther {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return "", UnexpectedResponse{resp.StatusCode, string(body)}
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("createDeployment: no Location header is response")
	}
	return location, nil
}

func buildDeploymentData(target, sha, branch, comment string, stages []string) url.Values {
	data := url.Values{}
	data.Set("target", target)
	data.Set("commitsha", commitSHA)
	data.Set("branch", branch)
	data.Set("comment", comment)
	for _, s := range stages {
		data.Add("stages[]", s)
	}
	return data
}

func buildDeploymentURL(hostURL, applicationName string) (*url.URL, error) {
	u, err := url.Parse(hostURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(applicationName, "deployments")
	return u, nil
}

func buildDeploymentLogURL(hostURL, deploymentPath string) (*url.URL, error) {
	u, err := url.Parse(hostURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(deploymentPath, "log")

	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}

	return u, nil
}

func buildDeploymentKillURL(hostURL, deploymentPath string) (*url.URL, error) {
	u, err := url.Parse(hostURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(deploymentPath, "kill")
	return u, nil
}

func killDeployment(c *Configuration, deploymentKillURL *url.URL) error {
	req, err := http.NewRequest("POST", deploymentKillURL.String(), &bytes.Buffer{})
	req.Header.Set(apiTokenHeaderName, c.ApiToken)

	_, err = http.DefaultTransport.RoundTrip(req)
	return err
}
