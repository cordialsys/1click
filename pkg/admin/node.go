package admin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cordialsys/panel/pkg/api"
	"github.com/cordialsys/panel/pkg/genesis"
)

func (node *Node) IsReady() bool {
	return node.Keys != nil && node.Keys.IsReady()
}

func NewInitFile(treasury *Treasury, node *Node) (genesis.TreasuryInitConfig, error) {
	if !node.IsReady() {
		return genesis.TreasuryInitConfig{}, fmt.Errorf("node is not initialized")
	}
	parts := strings.Split(string(node.Name), "/")
	last := parts[len(parts)-1]
	participant, err := strconv.ParseUint(last, 10, 64)
	if err != nil {
		return genesis.TreasuryInitConfig{}, fmt.Errorf("failed to parse participant: %v", err)

	}
	createTime, err := time.Parse(time.RFC3339, treasury.CreateTime)
	if err != nil {
		return genesis.TreasuryInitConfig{}, fmt.Errorf("failed to parse treasury create_time: %v", err)
	}

	return genesis.TreasuryInitConfig{
		Participant: genesis.Uint64(participant),
		CreateTime:  createTime,
		NodeId:      node.Keys.Node.Identity,
		Treasury: genesis.Treasury{
			Name:     treasury.Name,
			Software: api.DerefOrZero(treasury.InitialVersion),
			// This is a runtime field, do not need to set it.
			Network: "",
		},
		Validator: genesis.Validator{
			Name:      fmt.Sprintf("validators/%d", participant),
			PublicKey: node.Keys.Engine.Identity,
		},
		Signer: genesis.Signer{
			Name:         fmt.Sprintf("signers/%d", participant),
			Recipient:    node.Keys.Signer.Recipient,
			VerifyingKey: node.Keys.Signer.Identity,
			// not really used
			Socket: "",
			// runtime field, does not need to be set
			State: "",
			// These are hardcoded -- TODO perhaps this should be left blank and set by the blueprint instead??
			User: fmt.Sprintf("users/signer-%d", participant),
		},
	}, nil
}
