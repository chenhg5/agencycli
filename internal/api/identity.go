package api

import (
	"fmt"
	"strings"
)

// validateIdentity checks identity is "human" or an existing "project/agent".
func (s *Server) validateIdentity(identity, fieldName string) error {
	identity = strings.TrimSpace(identity)
	if identity == "human" {
		return nil
	}
	parts := strings.SplitN(identity, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid %s: expected human or project/agent", fieldName)
	}
	project, agentName := parts[0], parts[1]
	agents, err := s.st.ListAgents(project)
	if err != nil {
		return fmt.Errorf("list agents for %s: %w", project, err)
	}
	for _, a := range agents {
		if a.Name == agentName {
			return nil
		}
	}
	return fmt.Errorf("agent %q not found in project %q", agentName, project)
}

func (s *Server) agentExistsInProject(project, agentName string) bool {
	agents, err := s.st.ListAgents(project)
	if err != nil {
		return false
	}
	for _, a := range agents {
		if a.Name == agentName {
			return true
		}
	}
	return false
}
