package httputil

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	URL "net/url"
	"os"
	"path"
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
	isHtml, err := IsHtml(resp)
	if err != nil {
		return nil, err
	}
	if !isHtml {
		return nil, fmt.Errorf("The url: %s is not html page\n", url)
	}

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
			//过滤掉父目录
			if strings.Contains(v[1], "..") {
				continue
			}
			u, err := URL.Parse(url)
			if err != nil {
				return nil, fmt.Errorf("url %s is not valid\n", url)
			}
			u.Path = path.Join(u.Path, v[1])
			retLinks = append(retLinks, u.String())
		}
	}
	return retLinks, nil
}

//根据给定的url返回url中的所有链接,如果url不是text/html格式的，则把符合条件的url发送的channel中
//@param url string
//@param filterFunc func(url string) bool
//@param urlChan chan<- string
//@return ([]string,error)
func GetLinksAndFilter(url string, filterFunc func(url string) bool, urlChan chan<- string) ([]string, error) {
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
	isHtml, err := IsHtml(resp)
	if err != nil {
		return nil, err
	}
	if !isHtml {
		if filterFunc(url) {
			urlChan <- url
		}
		return nil, fmt.Errorf("The url: %s is not html page\n", url)
	}

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
			//过滤掉父目录
			if strings.Contains(v[1], "..") {
				continue
			}
			u, err := URL.Parse(url)
			if err != nil {
				return nil, fmt.Errorf("url %s is not valid\n", url)
			}
			u.Path = path.Join(u.Path, v[1])
			retLinks = append(retLinks, u.String())
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
			if strings.Contains(strings.ToLower(v), "text/html") {
				return true, nil
			}
		}
		return false, nil
	} else {
		return false, errors.New("No Content-Type response from server!")
	}
}

//下载指定的url
//@param url string
func Download(url string) error {
	if len(url) == 0 {
		return fmt.Errorf("url is empty\n")
	}

	//下载不能指定timeout，否则在timeout时间内没下完就会停止
	client := &http.Client{
	//Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	splits := strings.Split(url, "/")
	filename := splits[len(splits)-1]
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
