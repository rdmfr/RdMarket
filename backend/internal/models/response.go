package models

// ApiError represents structured API error response
type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ApiResponse represents standard envelope specified in section 13
type ApiResponse[T any] struct {
	Data  T                      `json:"data"`
	Meta  map[string]interface{} `json:"meta"`
	Error *ApiError              `json:"error"`
}

// NewSuccessResponse creates a successful envelope
func NewSuccessResponse[T any](data T, meta map[string]interface{}) ApiResponse[T] {
	if meta == nil {
		meta = make(map[string]interface{})
	}
	return ApiResponse[T]{
		Data:  data,
		Meta:  meta,
		Error: nil,
	}
}

// NewErrorResponse creates an error envelope
func NewErrorResponse(code, message string, meta map[string]interface{}) ApiResponse[interface{}] {
	if meta == nil {
		meta = make(map[string]interface{})
	}
	return ApiResponse[interface{}]{
		Data: nil,
		Meta: meta,
		Error: &ApiError{
			Code:    code,
			Message: message,
		},
	}
}
