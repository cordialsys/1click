package admin

func (k *Keys) IsReady() bool {
	if k == nil {
		return false
	}
	return k.Engine.Identity != "" && k.Node.Identity != "" && k.Signer.Identity != "" && k.Signer.Recipient != ""
}
