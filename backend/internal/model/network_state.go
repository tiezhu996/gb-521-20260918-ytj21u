package model

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

// NetworkState 是单行网络版本记录：任何通风网络内容变化都会提升 Revision，
// 方案批准时绑定当时的 Revision 与内容指纹，用于识别"批准后网络已变化"。
type NetworkState struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Revision  uint      `gorm:"not null;default:1" json:"revision"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (NetworkState) TableName() string { return "network_states" }

// ComputeNetworkFingerprint 对全部节点与巷道的求解相关内容生成确定性指纹。
// 不包含时间戳与乐观锁版本号，因此只有真实内容变化才会改变指纹；
// 输入顺序不影响结果，新增、停用与参数变化都会被捕获。
func ComputeNetworkFingerprint(nodes []VentilationNode, edges []AirwayEdge) string {
	parts := make([]string, 0, len(nodes)+len(edges))
	for _, node := range nodes {
		parts = append(parts, strings.Join([]string{
			"node",
			strconv.FormatUint(uint64(node.ID), 10),
			node.Code, node.NodeType,
			formatFingerprintFloat(node.ElevationM),
			formatFingerprintFloat(node.RequiredAirflowM3S),
			formatFingerprintFloat(node.PressurePa),
			node.Status,
		}, "|"))
	}
	for _, edge := range edges {
		parts = append(parts, strings.Join([]string{
			"edge",
			strconv.FormatUint(uint64(edge.ID), 10),
			edge.Code,
			strconv.FormatUint(uint64(edge.FromNodeID), 10),
			strconv.FormatUint(uint64(edge.ToNodeID), 10),
			formatFingerprintFloat(edge.ResistanceNS2M8),
			formatFingerprintFloat(edge.AreaM2),
			formatFingerprintFloat(edge.MaxVelocityMS),
			edge.DoorState,
			strconv.FormatBool(edge.Enabled),
			strconv.FormatBool(edge.CriticalPath),
		}, "|"))
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func formatFingerprintFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
