package envtb

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/leohubert/nginx-plex-auth-server/pkg/logtb"
)

func LoadEnvFile(filePaths ...string) {
	_ = godotenv.Load(filePaths...)
}

func GetString(key string, defaultValue string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultValue
}

func GetEnum(key string, allowedValues []string, defaultValue string) string {
	str := GetString(key, defaultValue)
	for _, v := range allowedValues {
		if str == v {
			return str
		}
	}
	panic(fmt.Errorf("invalid value %q for key %s valid values are %s", str, key, allowedValues))
}

func GetBool(key string, defaultValue bool) bool {
	defaultValueStr := "false"
	if defaultValue {
		defaultValueStr = "true"
	}

	str := GetEnum(key, []string{"true", "false"}, defaultValueStr)
	return str == "true"
}

func GetInt(key string, defaultValue int64) int64 {
	defaultValueStr := strconv.FormatInt(defaultValue, 10)
	str := GetString(key, defaultValueStr)
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

func GetDuration(key string, defaultDuration string) time.Duration {
	str := GetString(key, defaultDuration)
	res, err := time.ParseDuration(str)
	if err != nil {
		panic(fmt.Errorf("cannot parse duration %s: %w", str, err))
	}
	return res
}

func GetUrl(key string, defaultUrl string) *url.URL {
	str := GetString(key, defaultUrl)
	if str == "" {
		panic(fmt.Errorf("url %s cannot be empty", key))
	}

	res, err := url.Parse(str)
	if err != nil {
		panic(fmt.Errorf("cannot parse url %s: %w", str, err))
	}
	return res
}

func GetLogFormat(key string, defaultFormat logtb.Format) logtb.Format {
	str := GetEnum(key, []string{string(logtb.FormatPretty), string(logtb.FormatJSON)}, string(defaultFormat))
	return logtb.Format(str)
}
