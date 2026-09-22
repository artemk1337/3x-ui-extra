package controller

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestVKTurnControllerCallAndConfig(t *testing.T) {
	newHostTestDB(t)
	engine := gin.New()
	NewVKTurnController(engine.Group("/panel/api/vkturn"))

	inbound := model.Inbound{Tag: "wg-vk", Protocol: model.WireGuard, Enable: true, Port: 51820, Settings: `{"clients":[]}`}
	client := model.ClientRecord{Email: "vk@example.com", Enable: true, PublicKey: "public-key"}
	db := database.GetDB()
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientInbound{InboundId: inbound.Id, ClientId: client.Id}).Error; err != nil {
		t.Fatal(err)
	}

	created := doHostReq(t, engine, http.MethodPost, "/panel/api/vkturn/calls", map[string]any{
		"url": "https://vk.com/call/join/test", "enabled": true, "maxConfigs": 1,
	})
	if !created.Success {
		t.Fatalf("create call: %s", created.Msg)
	}
	optIn := doHostReq(t, engine, http.MethodPut, "/panel/api/vkturn/configs/1/vk@example.com", map[string]any{"enabled": true})
	if !optIn.Success {
		t.Fatalf("opt in: %s", optIn.Msg)
	}
	batch := doHostReq(t, engine, http.MethodPut, "/panel/api/vkturn/batch-configs/vk@example.com", map[string]any{
		"configs": []map[string]any{{"inboundId": inbound.Id, "enabled": true}},
	})
	if !batch.Success {
		t.Fatalf("batch opt in: %s", batch.Msg)
	}
	listed := doHostReq(t, engine, http.MethodGet, "/panel/api/vkturn/calls", nil)
	var calls []struct {
		Assigned int `json:"assigned"`
	}
	if err := json.Unmarshal(listed.Obj, &calls); err != nil || len(calls) != 1 || calls[0].Assigned != 1 {
		t.Fatalf("calls: %s, err=%v", listed.Obj, err)
	}
	proxy := doHostReq(t, engine, http.MethodPut, "/panel/api/vkturn/proxies/1", map[string]any{"listenPort": 56000, "enabled": false})
	if !proxy.Success {
		t.Fatalf("proxy: %s", proxy.Msg)
	}
}
