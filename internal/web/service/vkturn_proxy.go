package service

import (
	"fmt"
	"net"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/vkturnproxy"
)

var vkProxyManager = vkturnproxy.NewManager()

type VKProxyView struct {
	model.VKProxy
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

func (VKTurnService) ProxyViews() ([]VKProxyView, error) {
	var rows []model.VKProxy
	if err := database.GetDB().Order("inbound_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	statusByInbound := make(map[int]vkturnproxy.Status)
	for _, status := range vkProxyManager.Statuses() {
		statusByInbound[status.Spec.InboundID] = status
	}
	out := make([]VKProxyView, 0, len(rows))
	for _, row := range rows {
		v := VKProxyView{VKProxy: row, State: "disabled"}
		if row.Enable {
			v.State = "stopped"
			var inbound model.Inbound
			if err := database.GetDB().First(&inbound, row.InboundId).Error; err != nil {
				v.Error = "WireGuard inbound was deleted"
			} else if !inbound.Enable {
				v.Error = "WireGuard inbound is disabled"
			} else if status, ok := statusByInbound[row.InboundId]; ok {
				if status.Running {
					v.State = "running"
				} else {
					v.Error = status.Error
				}
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func ReconcileVKProxyProcesses() error {
	var rows []model.VKProxy
	if err := database.GetDB().Where("enable = ?", true).Find(&rows).Error; err != nil {
		return err
	}
	specs := make([]vkturnproxy.Spec, 0, len(rows))
	for _, row := range rows {
		var inbound model.Inbound
		if err := database.GetDB().First(&inbound, row.InboundId).Error; err != nil {
			continue
		}
		if !inbound.Enable || inbound.Protocol != model.WireGuard || inbound.NodeID != nil || inbound.Port < 1 {
			continue
		}
		host := strings.TrimSpace(inbound.Listen)
		switch host {
		case "", "0.0.0.0":
			host = "127.0.0.1"
		case "::", "[::]":
			host = "::1"
		}
		specs = append(specs, vkturnproxy.Spec{
			InboundID: row.InboundId,
			Listen:    net.JoinHostPort("0.0.0.0", fmt.Sprint(row.ListenPort)),
			Connect:   net.JoinHostPort(host, fmt.Sprint(inbound.Port)),
		})
	}
	return vkProxyManager.Reconcile(specs)
}

func StopVKProxyProcesses() { vkProxyManager.StopAll() }
