package httputil

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var ReLink = regexp.MustCompile(`<\s*a\s+href\s*=\s*"([^"]+)"`)

//根据给定的url返回url中的所有链接
//@param url string
//@return ([]string,error)
func GetLinks(url string) ([]string, error) {
	if len(url) == 0 {
		return nil, errors.New("The given url is empty string")
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//get links from body
	links := ReLink.FindAllStringSubmatch(string(body), -1)
	if len(links) == 0 {
		return nil, fmt.Errorf("get no link")
	}
	var retLinks []string
	//v是匹配的链接的字符串<h href="xxx" 以及链接xxx组成的slice
	for _, v := range links {
		if len(v) == 2 {
			fmt.Printf("get link: %s\n", v[1])
			retLinks = append(retLinks, v[1])
		}
	}
	return retLinks, nil
}

//根据http.Get的返回值(类型为*http.Response)来判断请求的URL是否是html页面
//@param res *http.Response
//@return (bool,error)
func IsHtml(res *http.Response) (bool, error) {
	if res == nil {
		return false, errors.New("res is nil!")
	}
	contentType, exist := res.Header["Content-Type"]
	if exist {
		for _, v := range contentType {
			if strings.Contains(v, "text/html") {
				return true, nil
			}
		}
		return false, nil
	} else {
		return false, errors.New("No Content-Type response from server!")
	}
}
