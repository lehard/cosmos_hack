package httpapi

import "strconv"

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func sscanInt(s string, out *int64) (bool, error) {
	if s == "" {
		return false, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return false, err
	}
	*out = n
	return true, nil
}
