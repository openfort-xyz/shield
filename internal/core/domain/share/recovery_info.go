package share

type RecoveryInfo struct {
	Entropy    Entropy
	PasskeyID  *string
	PasskeyEnv *string
}
