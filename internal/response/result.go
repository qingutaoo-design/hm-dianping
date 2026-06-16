package response

// Result 统一 API 响应格式
type Result struct {
	Success  bool   `json:"success"`
	ErrorMsg string `json:"errorMsg,omitempty"`
	Data     any    `json:"data,omitempty"`
	Total    *int64 `json:"total,omitempty"`
}

func OK(data ...any) Result {
	if len(data) == 0 {
		return Result{Success: true}
	}
	return Result{Success: true, Data: data[0]}
}

func OKList(data any, total int64) Result {
	return Result{Success: true, Data: data, Total: &total}
}

func Fail(message string) Result {
	return Result{Success: false, ErrorMsg: message}
}