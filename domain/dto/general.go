package dto

type Res struct {
	ResponseCode    string      `json:"response_code"`
	ResponseMessage string      `json:"response_message"`
	Meta            interface{} `json:"meta,omitempty"`
	Data            interface{} `json:"data,omitempty"`
}
