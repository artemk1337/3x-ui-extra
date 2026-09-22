package service

import (
	"fmt"
	"sync"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func vkTestClient(t *testing.T, inboundID int, email string) {
	t.Helper()
	client := model.Client{Email: email, PublicKey: "test-public-key", Enable: true, SubID: email}
	if err := (&ClientService{}).ApplyInboundClientDelta(nil, inboundID, []model.Client{client}, nil); err != nil {
		t.Fatal(err)
	}
}

func vkTestCall(t *testing.T, suffix string, limit int) model.VKCall {
	t.Helper()
	row, err := (VKTurnService{}).SaveCall(0, VKCallInput{URL: "https://vk.com/call/join/" + suffix, Enabled: true, MaxConfigs: limit})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func vkAssignments(t *testing.T) map[string]*int {
	t.Helper()
	rows, err := (VKTurnService{}).ListAssignments()
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]*int, len(rows))
	for _, row := range rows {
		out[fmt.Sprintf("%d/%s", row.InboundID, row.Email)] = row.CallID
	}
	return out
}

func TestVKTurnCapacityReassignment(t *testing.T) {
	setupBulkDB(t)
	first := mkInbound(t, 30001, model.WireGuard, `{"clients":[]}`)
	second := mkInbound(t, 30002, model.WireGuard, `{"clients":[]}`)
	vkTestClient(t, first.Id, "alice@example.com")
	vkTestClient(t, second.Id, "alice@example.com")
	vkTestClient(t, first.Id, "bob@example.com")
	svc := VKTurnService{}
	callA := vkTestCall(t, "alpha", 2)
	for _, cfg := range []struct {
		inbound int
		email   string
	}{
		{first.Id, "alice@example.com"},
		{second.Id, "alice@example.com"},
		{first.Id, "bob@example.com"},
	} {
		if err := svc.SetConfig(cfg.inbound, cfg.email, true); err != nil {
			t.Fatal(err)
		}
	}
	calls, err := svc.ListCalls()
	if err != nil || len(calls) != 1 || calls[0].Assigned != 3 || calls[0].OverLimit != 1 {
		t.Fatalf("overflow calls = %+v, err = %v", calls, err)
	}
	callB := vkTestCall(t, "beta", 2)
	calls, err = svc.ListCalls()
	if err != nil || calls[0].Assigned != 2 || calls[0].OverLimit != 0 || calls[1].Assigned != 1 {
		t.Fatalf("rebalanced calls = %+v, err = %v", calls, err)
	}
	if _, err := svc.SaveCall(callA.Id, VKCallInput{URL: callA.URL, Enabled: true, MaxConfigs: 1}); err != nil {
		t.Fatal(err)
	}
	calls, err = svc.ListCalls()
	if err != nil || calls[0].Assigned != 1 || calls[1].Assigned != 2 {
		t.Fatalf("reduced capacity calls = %+v, err = %v", calls, err)
	}
	if _, err := svc.SaveCall(callA.Id, VKCallInput{URL: callA.URL, Enabled: false, MaxConfigs: 1}); err != nil {
		t.Fatal(err)
	}
	calls, err = svc.ListCalls()
	if err != nil || calls[0].Assigned != 0 || calls[1].Assigned != 3 || calls[1].OverLimit != 1 {
		t.Fatalf("disabled call = %+v, err = %v", calls, err)
	}
	for key, id := range vkAssignments(t) {
		if id == nil || *id != callB.Id {
			t.Errorf("%s assigned to %v, want call %d", key, id, callB.Id)
		}
	}
	if err := svc.DeleteCall(callB.Id); err != nil {
		t.Fatal(err)
	}
	for key, id := range vkAssignments(t) {
		if id != nil {
			t.Errorf("%s retained deleted call %d", key, *id)
		}
	}
}

func TestVKTurnDisabledConfigFreesPlace(t *testing.T) {
	setupBulkDB(t)
	inbound := mkInbound(t, 30003, model.WireGuard, `{"clients":[]}`)
	vkTestClient(t, inbound.Id, "alice@example.com")
	vkTestCall(t, "one", 1)
	svc := VKTurnService{}
	if err := svc.SetConfig(inbound.Id, "alice@example.com", true); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetConfig(inbound.Id, "alice@example.com", false); err != nil {
		t.Fatal(err)
	}
	calls, err := svc.ListCalls()
	if err != nil || calls[0].Assigned != 0 {
		t.Fatalf("disabled config still assigned: %+v, %v", calls, err)
	}
	if err := svc.SetConfig(inbound.Id, "alice@example.com", true); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.ClientRecord{}).Where("email = ?", "alice@example.com").Update("enable", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := ReconcileVKAssignments(); err != nil {
		t.Fatal(err)
	}
	calls, err = svc.ListCalls()
	if err != nil || calls[0].Assigned != 0 {
		t.Fatalf("disabled client still assigned: %+v, %v", calls, err)
	}
}

func TestVKTurnSetConfigsIsAtomic(t *testing.T) {
	setupBulkDB(t)
	first := mkInbound(t, 30005, model.WireGuard, `{"clients":[]}`)
	second := mkInbound(t, 30006, model.WireGuard, `{"clients":[]}`)
	vkTestClient(t, first.Id, "alice@example.com")
	vkTestClient(t, second.Id, "alice@example.com")
	call := vkTestCall(t, "batch", 2)
	svc := VKTurnService{}
	configs := []VKConfigInput{{InboundID: first.Id, Enabled: true}, {InboundID: -1, Enabled: true}}
	if err := svc.SetConfigs("alice@example.com", configs); err == nil {
		t.Fatal("invalid second configuration succeeded")
	}
	var count int64
	if err := database.GetDB().Model(&model.VKConfig{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("partial configuration saved: count=%d, err=%v", count, err)
	}
	configs[1].InboundID = second.Id
	if err := svc.SetConfigs("alice@example.com", configs); err != nil {
		t.Fatal(err)
	}
	for key, id := range vkAssignments(t) {
		if id == nil || *id != call.Id {
			t.Errorf("%s assigned to %v, want %d", key, id, call.Id)
		}
	}
}

func TestVKTurnClientEditorReconcilesEnableImmediately(t *testing.T) {
	setupBulkDB(t)
	client := model.Client{Email: "alice@example.com", ID: "11111111-1111-1111-1111-111111111111", PublicKey: "test-public-key", Enable: true, SubID: "alice@example.com"}
	inbound := mkInbound(t, 30007, model.WireGuard, clientsSettings(t, []model.Client{client}))
	if err := (&ClientService{}).SyncInbound(nil, inbound.Id, []model.Client{client}); err != nil {
		t.Fatal(err)
	}
	vkTestCall(t, "editor", 1)
	if err := (VKTurnService{}).SetConfig(inbound.Id, "alice@example.com", true); err != nil {
		t.Fatal(err)
	}
	svc := &ClientService{}
	rec, err := svc.GetRecordByEmail(nil, "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		updated := rec.ToClient()
		updated.Enable = enabled
		if _, err := svc.Update(&InboundService{}, rec.Id, *updated, 0); err != nil {
			t.Fatal(err)
		}
		var config model.VKConfig
		if err := database.GetDB().Where("inbound_id = ? AND client_id = ?", inbound.Id, rec.Id).First(&config).Error; err != nil {
			t.Fatal(err)
		}
		if (config.CallId != nil) != enabled {
			t.Fatalf("enabled=%t, call_id=%v", enabled, config.CallId)
		}
	}
}

func TestVKTurnCallURLValidation(t *testing.T) {
	for _, raw := range []string{"http://vk.com/call/join/x", "https://evil.com/call/join/x", "https://vk.com/call/join", "https://vk.com/call/join/x#fragment"} {
		if _, err := normalizeVKCallURL(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestVKTurnConcurrentConfigAssignment(t *testing.T) {
	setupBulkDB(t)
	inbound := mkInbound(t, 30004, model.WireGuard, `{"clients":[]}`)
	for i := range 10 {
		vkTestClient(t, inbound.Id, fmt.Sprintf("client%d@example.com", i))
	}
	vkTestCall(t, "concurrent-a", 5)
	vkTestCall(t, "concurrent-b", 5)
	var wg sync.WaitGroup
	errors := make(chan error, 10)
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- (VKTurnService{}).SetConfig(inbound.Id, fmt.Sprintf("client%d@example.com", i), true)
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	calls, err := (VKTurnService{}).ListCalls()
	if err != nil || len(calls) != 2 || calls[0].Assigned != 5 || calls[1].Assigned != 5 {
		t.Fatalf("concurrent allocation = %+v, err = %v", calls, err)
	}
}
