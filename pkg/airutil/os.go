package airutil

import "github.com/drone/envsubst"

func ExpandEnv(s string) string {
	val, _ := envsubst.EvalEnv(s)
	return val
}
