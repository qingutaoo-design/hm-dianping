package middleware

import "strconv"

func fmtSscanf(value string, id *uint64) (int, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	*id = parsed
	return 1, nil
}
