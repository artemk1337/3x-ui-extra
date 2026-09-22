package service

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VKTurnService struct{}

type VKCallInput struct {
	URL        string `json:"url"`
	Enabled    bool   `json:"enabled"`
	MaxConfigs int    `json:"maxConfigs"`
}

type VKCallView struct {
	model.VKCall
	Assigned  int `json:"assigned"`
	OverLimit int `json:"overLimit"`
}

type VKAssignmentView struct {
	InboundID int    `json:"inboundId"`
	Email     string `json:"email"`
	CallID    *int   `json:"callId"`
	Enabled   bool   `json:"enabled"`
}

type VKProxyInput struct {
	ListenPort int  `json:"listenPort"`
	Enabled    bool `json:"enabled"`
}

type VKConfigInput struct {
	InboundID int  `json:"inboundId"`
	Enabled   bool `json:"enabled"`
}

func normalizeVKCallURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "vk.com") || u.Port() != "" || u.User != nil || u.Fragment != "" {
		return "", errors.New("expected an https://vk.com/call/join/... URL")
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) != 3 || parts[0] != "call" || parts[1] != "join" || parts[2] == "" {
		return "", errors.New("expected an https://vk.com/call/join/... URL")
	}
	u.Host = "vk.com"
	return u.String(), nil
}

func (VKTurnService) ListCalls() ([]VKCallView, error) {
	var out []VKCallView
	err := runSerializedTx(func(tx *gorm.DB) error {
		if err := reconcileVKAssignmentsTx(tx); err != nil {
			return err
		}
		var calls []model.VKCall
		if err := tx.Order("id").Find(&calls).Error; err != nil {
			return err
		}
		var configs []model.VKConfig
		if err := tx.Where("enable = ? AND call_id IS NOT NULL", true).Find(&configs).Error; err != nil {
			return err
		}
		counts := make(map[int]int, len(calls))
		for _, c := range configs {
			counts[*c.CallId]++
		}
		out = make([]VKCallView, 0, len(calls))
		for _, c := range calls {
			v := VKCallView{VKCall: c, Assigned: counts[c.Id]}
			if c.MaxConfigs > 0 && v.Assigned > c.MaxConfigs {
				v.OverLimit = v.Assigned - c.MaxConfigs
			}
			out = append(out, v)
		}
		return nil
	})
	return out, err
}

func (VKTurnService) SaveCall(id int, input VKCallInput) (model.VKCall, error) {
	var out model.VKCall
	value, err := normalizeVKCallURL(input.URL)
	if err != nil {
		return out, err
	}
	if input.MaxConfigs < 0 {
		return out, errors.New("maxConfigs must be zero or positive")
	}
	err = runSerializedTx(func(tx *gorm.DB) error {
		if id == 0 {
			out = model.VKCall{URL: value, Enable: input.Enabled, MaxConfigs: input.MaxConfigs}
			if err := tx.Create(&out).Error; err != nil {
				return err
			}
		} else {
			if err := tx.First(&out, id).Error; err != nil {
				return err
			}
			out.URL, out.Enable, out.MaxConfigs = value, input.Enabled, input.MaxConfigs
			if err := tx.Model(&out).Updates(map[string]any{"url": value, "enable": input.Enabled, "max_configs": input.MaxConfigs}).Error; err != nil {
				return err
			}
		}
		return reconcileVKAssignmentsTx(tx)
	})
	return out, err
}

func (VKTurnService) DeleteCall(id int) error {
	return runSerializedTx(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.VKCall{}, id).Error; err != nil {
			return err
		}
		return reconcileVKAssignmentsTx(tx)
	})
}

func (VKTurnService) SetConfig(inboundID int, email string, enabled bool) error {
	return (VKTurnService{}).SetConfigs(email, []VKConfigInput{{InboundID: inboundID, Enabled: enabled}})
}

func (VKTurnService) SetConfigs(email string, configs []VKConfigInput) error {
	if len(configs) == 0 {
		return errors.New("configs must not be empty")
	}
	return runSerializedTx(func(tx *gorm.DB) error {
		var client model.ClientRecord
		if err := tx.Where("email = ?", email).First(&client).Error; err != nil {
			return err
		}
		seen := make(map[int]bool, len(configs))
		for _, config := range configs {
			if config.InboundID < 1 || seen[config.InboundID] {
				return errors.New("configs contain an invalid or duplicate inboundId")
			}
			seen[config.InboundID] = true
			var inbound model.Inbound
			if err := tx.First(&inbound, config.InboundID).Error; err != nil {
				return err
			}
			if inbound.Protocol != model.WireGuard || inbound.NodeID != nil {
				return errors.New("VK TURN requires a local WireGuard inbound")
			}
			var link model.ClientInbound
			if err := tx.Where("inbound_id = ? AND client_id = ?", config.InboundID, client.Id).First(&link).Error; err != nil {
				return err
			}
			row := model.VKConfig{InboundId: config.InboundID, ClientId: client.Id, Enable: config.Enabled}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "inbound_id"}, {Name: "client_id"}},
				DoUpdates: clause.Assignments(map[string]any{"enable": config.Enabled}),
			}).Create(&row).Error; err != nil {
				return err
			}
		}
		return reconcileVKAssignmentsTx(tx)
	})
}

func (VKTurnService) ListAssignments() ([]VKAssignmentView, error) {
	var out []VKAssignmentView
	err := runSerializedTx(func(tx *gorm.DB) error {
		if err := reconcileVKAssignmentsTx(tx); err != nil {
			return err
		}
		return tx.Table("vk_configs AS v").
			Select("v.inbound_id, clients.email, v.call_id, v.enable AS enabled").
			Joins("JOIN clients ON clients.id = v.client_id").
			Order("v.inbound_id, clients.email").Scan(&out).Error
	})
	return out, err
}

func (VKTurnService) ListProxies() ([]model.VKProxy, error) {
	var out []model.VKProxy
	err := database.GetDB().Order("inbound_id").Find(&out).Error
	return out, err
}

func (VKTurnService) SaveProxy(inboundID int, input VKProxyInput) (model.VKProxy, error) {
	var out model.VKProxy
	if input.ListenPort < 1 || input.ListenPort > 65535 {
		return out, errors.New("listenPort must be between 1 and 65535")
	}
	err := runSerializedTx(func(tx *gorm.DB) error {
		var inbound model.Inbound
		if err := tx.First(&inbound, inboundID).Error; err != nil {
			return err
		}
		if inbound.Protocol != model.WireGuard || inbound.NodeID != nil || inbound.Port < 1 {
			return errors.New("VK TURN requires a local WireGuard inbound with a UDP port")
		}
		if input.ListenPort == inbound.Port {
			return errors.New("VK TURN port must differ from the WireGuard port")
		}
		var count int64
		if err := tx.Model(&model.VKProxy{}).Where("listen_port = ? AND inbound_id <> ? AND enable = ?", input.ListenPort, inboundID, true).Count(&count).Error; err != nil {
			return err
		}
		if input.Enabled && count > 0 {
			return fmt.Errorf("UDP port %d is already used by another VK TURN proxy", input.ListenPort)
		}
		out = model.VKProxy{InboundId: inboundID, ListenPort: input.ListenPort, Enable: input.Enabled}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "inbound_id"}},
			DoUpdates: clause.Assignments(map[string]any{"listen_port": input.ListenPort, "enable": input.Enabled}),
		}).Create(&out).Error
	})
	return out, err
}

type vkConfigState struct {
	model.VKConfig
	ClientEnabled  bool  `gorm:"column:client_enabled"`
	InboundEnabled bool  `gorm:"column:inbound_enabled"`
	ExpiryTime     int64 `gorm:"column:expiry_time"`
}

// reconcileVKAssignmentsTx preserves valid assignments, moves only surplus
// when capacity becomes available elsewhere, then fills pending configs.
func reconcileVKAssignmentsTx(tx *gorm.DB) error {
	if err := tx.Exec(`DELETE FROM vk_configs WHERE NOT EXISTS (
		SELECT 1 FROM client_inbounds ci JOIN inbounds i ON i.id = ci.inbound_id
		WHERE ci.inbound_id = vk_configs.inbound_id AND ci.client_id = vk_configs.client_id
		AND i.protocol = 'wireguard' AND i.node_id IS NULL
	)`).Error; err != nil {
		return err
	}
	var calls []model.VKCall
	if err := tx.Where("enable = ?", true).Order("id").Find(&calls).Error; err != nil {
		return err
	}
	var rows []vkConfigState
	if err := tx.Table("vk_configs AS v").
		Select("v.*, c.enable AS client_enabled, c.expiry_time, i.enable AS inbound_enabled").
		Joins("JOIN clients c ON c.id = v.client_id").
		Joins("JOIN inbounds i ON i.id = v.inbound_id").
		Order("v.inbound_id, v.client_id").Scan(&rows).Error; err != nil {
		return err
	}
	originalIDs := make([]*int, len(rows))
	for i, row := range rows {
		if row.CallId != nil {
			originalIDs[i] = new(*row.CallId)
		}
	}
	validCalls := make(map[int]model.VKCall, len(calls))
	count := make(map[int]int, len(calls))
	for _, call := range calls {
		validCalls[call.Id] = call
	}
	now := time.Now().UnixMilli()
	for i := range rows {
		r := &rows[i]
		active := r.Enable && r.ClientEnabled && r.InboundEnabled && (r.ExpiryTime <= 0 || r.ExpiryTime > now)
		if !active || r.CallId == nil {
			if !active {
				r.CallId = nil
			}
			continue
		}
		if _, ok := validCalls[*r.CallId]; !ok {
			r.CallId = nil
			continue
		}
		count[*r.CallId]++
	}
	// Move surplus only when another call has room. Otherwise keep the stable
	// assignment and expose overLimit to the administrator.
	for _, call := range calls {
		if call.MaxConfigs == 0 {
			continue
		}
		for count[call.Id] > call.MaxConfigs {
			target := chooseVKCall(calls, count, call.Id, false)
			if target == 0 {
				break
			}
			for i := len(rows) - 1; i >= 0; i-- {
				if rows[i].CallId != nil && *rows[i].CallId == call.Id {
					rows[i].CallId = &target
					count[call.Id]--
					count[target]++
					break
				}
			}
		}
	}
	for i := range rows {
		r := &rows[i]
		if !r.Enable || !r.ClientEnabled || !r.InboundEnabled || (r.ExpiryTime > 0 && r.ExpiryTime <= now) || r.CallId != nil {
			continue
		}
		target := chooseVKCall(calls, count, 0, true)
		if target != 0 {
			r.CallId = &target
			count[target]++
		}
	}
	for i, r := range rows {
		if (r.CallId == nil && originalIDs[i] == nil) ||
			(r.CallId != nil && originalIDs[i] != nil && *r.CallId == *originalIDs[i]) {
			continue
		}
		if err := tx.Model(&model.VKConfig{}).
			Where("inbound_id = ? AND client_id = ?", r.InboundId, r.ClientId).
			Update("call_id", r.CallId).Error; err != nil {
			return err
		}
	}
	return nil
}

func chooseVKCall(calls []model.VKCall, count map[int]int, exclude int, allowOverflow bool) int {
	type candidate struct{ id, load, overflow int }
	choices := make([]candidate, 0, len(calls))
	for _, c := range calls {
		if c.Id == exclude {
			continue
		}
		if c.MaxConfigs > 0 && count[c.Id] >= c.MaxConfigs && !allowOverflow {
			continue
		}
		overflow := 0
		if c.MaxConfigs > 0 && count[c.Id] >= c.MaxConfigs {
			overflow = count[c.Id] - c.MaxConfigs + 1
		}
		choices = append(choices, candidate{c.Id, count[c.Id], overflow})
	}
	if len(choices) == 0 {
		return 0
	}
	sort.Slice(choices, func(i, j int) bool {
		a, b := choices[i], choices[j]
		if a.overflow != b.overflow {
			return a.overflow < b.overflow
		}
		if a.load != b.load {
			return a.load < b.load
		}
		return a.id < b.id
	})
	return choices[0].id
}

func ReconcileVKAssignments() error {
	return runSerializedTx(reconcileVKAssignmentsTx)
}
