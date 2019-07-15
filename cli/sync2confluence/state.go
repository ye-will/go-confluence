package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type tNode struct {
	ID       string `json:"id"`
	Hash     string `json:"hash"`
	ParentID string `json:"parentId"`
}

type tState struct {
	file string
	data map[string]*tNode
}

func newState(stateName string) tState {
	state := tState{stateName, make(map[string]*tNode)}
	data, err := ioutil.ReadFile(stateName)
	if err != nil {
		return state
	}
	json.Unmarshal(data, &state.data)
	return state
}

func (s *tState) getNode(nodeName string) *tNode {
	if nodeName == "" {
		return &tNode{}
	}
	_, exist := s.data[nodeName]
	if !exist {
		s.data[nodeName] = &tNode{}
	}
	return s.data[nodeName]
}

func (n *tNode) compageHash(data []byte) (string, bool) {
	h := sha1.New()
	h.Write(data)
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return hash, (n.Hash != hash)
}

func (s *tState) save() error {
	j, _ := json.MarshalIndent(s.data, "", "    ")
	return ioutil.WriteFile(s.file, j, 0755)
}
