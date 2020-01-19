package httpUtil

import (
	"fmt"
	"io"
)

type state byte

const (
	stateStart state = iota
	stateSpacesAfterMethod
	statePath
	stateSpacesAfterPath
	stateHTTP
	stateMajor
	staeMinor
	stateCR
	stateLF
	stateEnd
)

const (
	// CR stands for \r
	CR = '\r'
	// LF stands for \n
	LF = '\n'
)

// RequestLine TODO
// 2019/10/04 11:41:55
type RequestLine struct {
	Method string
	Path   string
	Major  int
	Minor  int
}

// ParseRequestLine TODO
// 2019/10/03 12:56:50
func ParseRequestLine(r io.Reader) (requestLine *RequestLine, err error) {
	var (
		st    state = stateStart
		token []byte
		b     byte
		buf   = make([]byte, 1)
	)
	requestLine = &RequestLine{}

OUTTER_LOOP:
	for {
		_, err = io.ReadFull(r, buf)
		if err != nil {
			return
		}
		b = buf[0]

		switch st {
		case stateStart:
			if b == ' ' {
				st = stateSpacesAfterMethod
				requestLine.Method = string(token)
				token = []byte{}
				break
			}
			token = append(token, b)
		case stateSpacesAfterMethod:
			if b != ' ' {
				token = append(token, b)
				st = statePath
				break
			}

		case statePath:
			if b == ' ' {
				st = stateSpacesAfterPath
				requestLine.Path = string(token)
				token = []byte{}
				break
			}
			token = append(token, b)

		case stateSpacesAfterPath:
			if b != ' ' {
				st = stateHTTP
				token = append(token, b)
				break
			}

		case stateHTTP:
			if b == '/' {
				if string(token) != "HTTP" {
					err = fmt.Errorf("response status not http")
					return
				}
				token = []byte{}
				st = stateMajor
				break
			}
			token = append(token, b)
		case stateMajor:
			if b >= '0' && b <= '9' {
				requestLine.Major = requestLine.Major*10 + int(b-'0')
				break
			}
			if b == '.' {
				st = staeMinor
				break
			}
			err = fmt.Errorf("major not valid")
			return

		case staeMinor:
			if b >= '0' && b <= '9' {
				requestLine.Minor = requestLine.Minor*10 + int(b-'0')
				break
			}
			if b == CR {
				st = stateCR
				break
			}
			err = fmt.Errorf("minor not valid")
			return

		case stateCR:
			if b == LF {
				st = stateEnd
				break OUTTER_LOOP
			}
			err = fmt.Errorf("not lf after cr")
			return
		}

	}
	if st != stateEnd {
		err = fmt.Errorf("not end with stateEnd")
		return
	}
	return
}

const (
	stateKey state = iota
	stateSpacesAfterKey
	stateColon
	stateSpacesAfterColon
	stateValue
	stateCRLF
	stateCRLFCR
	//stateEnd
)

// ParseRequestHeaders TODO
// 2019/10/03 16:06:59
func ParseRequestHeaders(r io.Reader) (headers map[string]string, err error) {
	var (
		st    state = stateKey
		b     byte
		key   []byte
		value []byte
		buf   = make([]byte, 1)
	)
	headers = make(map[string]string)
OUTTER:
	for {
		_, err = io.ReadFull(r, buf)
		if err != nil {
			return
		}
		b = buf[0]
		switch st {
		case stateKey:
			if b == ' ' {
				st = stateSpacesAfterKey
				break
			}
			if b == ':' {
				st = stateColon
				break
			}
			key = append(key, b)

		case stateSpacesAfterKey:
			if b == ':' {
				st = stateColon
				break
			}
			if b != ' ' {
				err = fmt.Errorf("other char before colon")
				return
			}

		case stateColon:
			if b == ' ' {
				st = stateSpacesAfterColon
				break
			}
			st = stateValue

		case stateSpacesAfterColon:
			if b != ' ' {
				st = stateValue
				value = append(value, b)
			}

		case stateValue:
			if b == CR {
				headers[string(key)] = string(value)
				key = []byte{}
				value = []byte{}
				st = stateCR
				break
			}
			value = append(value, b)

		case stateCR:
			if b != LF {
				err = fmt.Errorf("not lf after cr")
				return
			}
			st = stateCRLF

		case stateCRLF:
			if b == CR {
				st = stateCRLFCR
				break
			}
			st = stateKey
			key = append(key, b)

		case stateCRLFCR:
			if b == LF {
				st = stateEnd
				break OUTTER
			}
			err = fmt.Errorf("not lf after crlfcr")
			return
		}
	}

	if st != stateEnd {
		err = fmt.Errorf("not end with stateEnd")
	}
	return
}
