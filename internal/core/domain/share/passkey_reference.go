package share

import "regexp"

var PasskeyEnvPattern = regexp.MustCompile(`^name=([^;]+);os=([^;]+);osVersion=([^;]+);device=([^;]+)$`)

type PasskeyEnv struct {
	Name      *string
	OS        *string
	OSVersion *string
	Device    *string
}

type PasskeyReference struct {
	PasskeyID  string
	PasskeyEnv *PasskeyEnv
}
