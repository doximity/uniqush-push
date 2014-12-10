package rest

type JsonError struct {
	Error   string `json:"error"`
	GoError string `json:"go_error"`
}
