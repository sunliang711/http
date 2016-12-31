package httputil

import (
	"errors"
	"net/http"
	"strings"
)

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
