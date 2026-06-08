package dto

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

type LoginForm struct {
	Phone    string `json:"phone" form:"phone" binding:"required"`
	Code     string `json:"code" form:"code"`
	Password string `json:"password" form:"password"`
}

type UserDTO struct {
	ID       uint64 `json:"id,string" redis:"id"`
	NickName string `json:"nickName" redis:"nickName"`
	Icon     string `json:"icon" redis:"icon"`
}

type ScrollResult struct {
	List    any   `json:"list"`
	MinTime int64 `json:"minTime"`
	Offset  int64 `json:"offset"`
}
