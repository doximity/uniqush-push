package rest

import "fmt"

type JsonError struct {
	Error   string `json:"error"`
	GoError string `json:"go_error"`
}

func (json *JsonError) AddError(err string) {
	if json.Error != "" {
		json.Error = fmt.Sprintf("%v, ", json.Error)
	}
	json.Error = fmt.Sprintf("%v%v", json.Error, err)
}
