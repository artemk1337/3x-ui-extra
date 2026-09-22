package controller

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type VKTurnController struct {
	service service.VKTurnService
}

func NewVKTurnController(g *gin.RouterGroup) *VKTurnController {
	a := &VKTurnController{}
	g.GET("/calls", a.calls)
	g.POST("/calls", a.createCall)
	g.PUT("/calls/:id", a.updateCall)
	g.DELETE("/calls/:id", a.deleteCall)
	g.GET("/assignments", a.assignments)
	g.PUT("/batch-configs/:email", a.setConfigs)
	g.PUT("/configs/:inboundId/:email", a.setConfig)
	g.GET("/proxies", a.proxies)
	g.PUT("/proxies/:inboundId", a.setProxy)
	return a
}

func vkTurnID(c *gin.Context, key string) (int, error) {
	id, err := strconv.Atoi(c.Param(key))
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}

func (a *VKTurnController) calls(c *gin.Context) {
	rows, err := a.service.ListCalls()
	jsonObj(c, rows, err)
}

func (a *VKTurnController) saveCall(c *gin.Context, id int) {
	var input struct {
		URL        string `json:"url" form:"url"`
		Enabled    bool   `json:"enabled" form:"enabled"`
		MaxConfigs int    `json:"maxConfigs" form:"maxConfigs"`
	}
	if err := c.ShouldBind(&input); err != nil {
		jsonMsg(c, "Invalid call", err)
		return
	}
	row, err := a.service.SaveCall(id, service.VKCallInput{URL: input.URL, Enabled: input.Enabled, MaxConfigs: input.MaxConfigs})
	jsonObj(c, row, err)
}

func (a *VKTurnController) createCall(c *gin.Context) { a.saveCall(c, 0) }

func (a *VKTurnController) updateCall(c *gin.Context) {
	id, err := vkTurnID(c, "id")
	if err != nil {
		jsonMsg(c, "Invalid call", err)
		return
	}
	a.saveCall(c, id)
}

func (a *VKTurnController) deleteCall(c *gin.Context) {
	id, err := vkTurnID(c, "id")
	if err == nil {
		err = a.service.DeleteCall(id)
	}
	jsonMsg(c, "", err)
}

func (a *VKTurnController) assignments(c *gin.Context) {
	rows, err := a.service.ListAssignments()
	jsonObj(c, rows, err)
}

func (a *VKTurnController) setConfig(c *gin.Context) {
	id, err := vkTurnID(c, "inboundId")
	if err != nil {
		jsonMsg(c, "Invalid inbound", err)
		return
	}
	var input struct {
		Enabled bool `json:"enabled" form:"enabled"`
	}
	if err := c.ShouldBind(&input); err != nil {
		jsonMsg(c, "Invalid VK TURN configuration", err)
		return
	}
	jsonMsg(c, "", a.service.SetConfig(id, c.Param("email"), input.Enabled))
}

func (a *VKTurnController) setConfigs(c *gin.Context) {
	var input struct {
		Configs []service.VKConfigInput `json:"configs" binding:"required,min=1,dive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		jsonMsg(c, "Invalid VK TURN configurations", err)
		return
	}
	jsonMsg(c, "", a.service.SetConfigs(c.Param("email"), input.Configs))
}

func (a *VKTurnController) proxies(c *gin.Context) {
	rows, err := a.service.ProxyViews()
	jsonObj(c, rows, err)
}

func (a *VKTurnController) setProxy(c *gin.Context) {
	id, err := vkTurnID(c, "inboundId")
	if err != nil {
		jsonMsg(c, "Invalid inbound", err)
		return
	}
	var input struct {
		ListenPort int  `json:"listenPort" form:"listenPort"`
		Enabled    bool `json:"enabled" form:"enabled"`
	}
	if err := c.ShouldBind(&input); err != nil {
		jsonMsg(c, "Invalid VK TURN proxy", err)
		return
	}
	row, err := a.service.SaveProxy(id, service.VKProxyInput{ListenPort: input.ListenPort, Enabled: input.Enabled})
	if err == nil {
		service.ReconcileVKProxyProcesses()
	}
	jsonObj(c, row, err)
}
