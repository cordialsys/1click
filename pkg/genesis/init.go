package genesis

import "time"

type TreasuryInitConfig struct {
	Participant Uint64    `json:"participant"`
	CreateTime  time.Time `json:"create_time"`
	NodeId      string    `json:"node_id"`
	Treasury    Treasury  `json:"treasury"`
	Validator   Validator `json:"validator"`
	Signer      Signer    `json:"signer"`
}
