package pgconnections

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/runopsio/hoop/gateway/pgrest"
	"github.com/runopsio/hoop/gateway/storagev2/types"
)

type connections struct{}

func New() *connections { return &connections{} }
func (c *connections) FetchOneForExec(ctx pgrest.OrgContext, name string) (*types.Connection, error) {
	var conn pgrest.Connection
	err := pgrest.New("/connections?select=*,orgs(id,name)&org_id=eq.%v&name=eq.%v",
		ctx.GetOrgID(), name).
		FetchOne().
		DecodeInto(&conn)
	if err != nil {
		if err == pgrest.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if conn.LegacyAgentID != "" {
		conn.AgentID = conn.LegacyAgentID
	}
	return &types.Connection{
		Id:             conn.ID,
		OrgId:          conn.OrgID,
		Name:           conn.Name,
		Command:        conn.Command,
		Type:           conn.Type,
		SecretProvider: "database",
		SecretId:       "",
		CreatedById:    "",
		AgentId:        conn.AgentID,
	}, nil
}

func (c *connections) FetchByNames(ctx pgrest.OrgContext, connectionNames []string) (map[string]types.Connection, error) {
	var connList []pgrest.Connection
	err := pgrest.New("/connections?&org_id=eq.%v&name=in.(%s)",
		ctx.GetOrgID(), strings.Join(connectionNames, ",")).
		FetchAll().
		DecodeInto(&connList)
	if err != nil {
		if err == pgrest.ErrNotFound {
			return map[string]types.Connection{}, nil
		}
		return nil, err
	}
	var result = map[string]types.Connection{}
	for _, conn := range connList {
		if conn.LegacyAgentID != "" {
			conn.AgentID = conn.LegacyAgentID
		}
		result[conn.Name] = types.Connection{
			Id:             conn.ID,
			OrgId:          conn.OrgID,
			Name:           conn.Name,
			Command:        conn.Command,
			Type:           conn.Type,
			SecretProvider: "database",
			SecretId:       "",
			CreatedById:    "",
			AgentId:        conn.AgentID,
		}
	}
	return result, nil
}

func (c *connections) FetchByIDs(ctx pgrest.OrgContext, connectionIDs []string) (map[string]types.Connection, error) {
	var connList []pgrest.Connection
	itemMap := map[string]types.Connection{}
	err := pgrest.New("/connections?org_id=eq.%s", ctx.GetOrgID()).
		List().
		DecodeInto(&connList)
	if err != nil {
		if err == pgrest.ErrNotFound {
			return itemMap, nil
		}
		return nil, err
	}
	for _, conn := range connList {
		for _, connID := range connectionIDs {
			if conn.ID == connID {
				if conn.LegacyAgentID != "" {
					conn.AgentID = conn.LegacyAgentID
				}
				itemMap[connID] = types.Connection{
					Id:             conn.ID,
					OrgId:          conn.OrgID,
					Name:           conn.Name,
					Command:        conn.Command,
					Type:           conn.Type,
					SecretProvider: "database",
					SecretId:       "",
					CreatedById:    "",
					AgentId:        conn.AgentID,
				}
				break
			}
		}
	}
	return itemMap, nil
}

func (c *connections) FetchOne(ctx pgrest.OrgContext, name string) (*pgrest.Connection, error) {
	var conn pgrest.Connection
	err := pgrest.New("/connections?select=*,orgs(id,name)&org_id=eq.%v&name=eq.%v", ctx.GetOrgID(), name).
		FetchOne().
		DecodeInto(&conn)
	if err != nil {
		if err == pgrest.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if conn.LegacyAgentID != "" {
		conn.AgentID = conn.LegacyAgentID
	}
	return &conn, nil
}

func (a *connections) FetchOneByNameOrID(ctx pgrest.OrgContext, nameOrID string) (*pgrest.Connection, error) {
	client := pgrest.New("/connections?select=*,orgs(id,name)&org_id=eq.%v&name=eq.%v", ctx.GetOrgID(), nameOrID)
	if _, err := uuid.Parse(nameOrID); err == nil {
		client = pgrest.New("/connections?select=*,orgs(id,name)&org_id=eq.%v&id=eq.%v", ctx.GetOrgID(), nameOrID)
	}
	var conn pgrest.Connection
	if err := client.FetchOne().DecodeInto(&conn); err != nil {
		if err == pgrest.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if conn.LegacyAgentID != "" {
		conn.AgentID = conn.LegacyAgentID
	}
	return &conn, nil
}

func (c *connections) FetchAll(ctx pgrest.OrgContext) ([]pgrest.Connection, error) {
	var items []pgrest.Connection
	err := pgrest.New("/connections?select=*,orgs(id,name)&org_id=eq.%v", ctx.GetOrgID()).
		List().
		DecodeInto(&items)
	if err != nil && err != pgrest.ErrNotFound {
		return nil, err
	}
	return items, nil
}

func (c *connections) Delete(ctx pgrest.OrgContext, name string) error {
	return pgrest.New("/connections?org_id=eq.%v&name=eq.%v", ctx.GetOrgID(), name).
		Delete().
		Error()
}

func (c *connections) Create(ctx pgrest.OrgContext, conn pgrest.Connection) error {
	return pgrest.New("/rpc/update_connection").Create(map[string]any{
		"id":              conn.ID,
		"org_id":          ctx.GetOrgID(),
		"name":            conn.Name,
		"agent_id":        toAgentID(conn.AgentID),
		"legacy_agent_id": toLegacyAgentID(conn.LegacyAgentID),
		"type":            conn.Type,
		"command":         conn.Command,
		"envs":            conn.Envs,
	}).Error()
}

func (c *connections) Upsert(ctx pgrest.OrgContext, reqConn *types.Connection) error {
	return pgrest.New("/connections").Upsert(map[string]any{
		"id":              reqConn.Id,
		"org_id":          reqConn.OrgId,
		"name":            reqConn.Name,
		"command":         reqConn.Command,
		"type":            reqConn.Type,
		"agent_id":        toAgentID(reqConn.AgentId),
		"legacy_agent_id": toLegacyAgentID(reqConn.AgentId),
		"updated_at":      time.Now().UTC().Format(time.RFC3339),
	}).Error()
}

func toAgentID(agentID string) (v *string) {
	if _, err := uuid.Parse(agentID); err == nil {
		return &agentID
	}
	return
}

func toLegacyAgentID(agentID string) (v *string) {
	if _, err := uuid.Parse(agentID); err != nil {
		return &agentID
	}
	return
}