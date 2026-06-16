package middleware

import "strconv"

func parseUint(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}