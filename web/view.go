package web

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
)

func View(tplPath string, data interface{}) (string, error) {
	tplContent, err := ioutil.ReadFile(tplPath)
	if err != nil {
		return "", errors.Wrap(err, "can not open template file")
	}

	funcMap := template.FuncMap{
		"starts_with": startsWith,
		"ends_with":   endsWith,
	}

	tpl, err := template.New("").Funcs(funcMap).Parse(string(tplContent))
	if err != nil {
		return "", errors.Wrap(err, "parse template failed")
	}

	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, data); err != nil {
		return "", errors.Wrap(err, "execute template failed")
	}

	return buffer.String(), nil
}

// startsWith 判断是字符串开始
func startsWith(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.HasPrefix(haystack, n) {
			return true
		}
	}

	return false
}

// endsWith 判断字符串结尾
func endsWith(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.HasSuffix(haystack, n) {
			return true
		}
	}

	return false
}
