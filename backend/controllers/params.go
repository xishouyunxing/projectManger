package controllers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func parseUintParam(value string) (uint, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("invalid uint param")
	}
	return uint(parsed), nil
}

func parsePositiveIntQuery(value string, defaultValue, maxValue int, fieldName string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s参数格式错误", fieldName)
	}
	if maxValue > 0 && parsed > maxValue {
		parsed = maxValue
	}
	return parsed, nil
}

func parseRequiredString(value, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s参数不能为空", fieldName)
	}
	return trimmed, nil
}
