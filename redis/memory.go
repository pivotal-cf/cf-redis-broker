package redis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var multipliers = map[string]int64{
	"":   1,
	"kb": 1024,
	"mb": 1024 * 1024,
	"gb": 1024 * 1024 * 1024,
}

func ParseMemoryStringToBytes(human string) (string, error) {
	human = strings.ToLower(human)
	human = strings.Replace(human, ",", "", -1)
	human = strings.Replace(human, " ", "", -1)

	numericPart := strings.TrimRight(human, "kmgb")
	unitPart := strings.TrimLeft(human, "0123456789")

	numeric, err := strconv.ParseInt(numericPart, 10, 64)

	if err != nil {
		return "", errors.New("cannot parse numeric part")
	}

	mult, found := multipliers[unitPart]

	if !found {
		return "", errors.New("cannot parse suffix")
	}

	return fmt.Sprintf("%d", numeric*mult), nil
}
