package api

type ErrorResponse struct {
	Code         string `json:"errcode"`
	Message      string `json:"error"`
	InternalCode string `json:"mr_errcode"`
}

type UnknownContentTypeResponse struct {
	Value interface{}
}

func InternalServerError(message string) *ErrorResponse {
	return &ErrorResponse{"M_UNKNOWN", message, "M_UNKNOWN"}
}

func NotFoundError() *ErrorResponse {
	return &ErrorResponse{"M_NOT_FOUND", "Not found", "M_NOT_FOUND"}
}

func MethodNotAllowed() *ErrorResponse {
	return &ErrorResponse{"M_UNKNOWN", "Method Not Allowed", "M_METHOD_NOT_ALLOWED"}
}

func AuthFailed() *ErrorResponse {
	return &ErrorResponse{"M_UNKNOWN_TOKEN", "Authentication Failed", "M_UNKNOWN_TOKEN"}
}

func BadRequest(message string) *ErrorResponse {
	return &ErrorResponse{"M_UNKNOWN", message, "M_BAD_REQUEST"}
}
