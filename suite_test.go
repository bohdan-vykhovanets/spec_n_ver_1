package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type KVTestSuite struct {
	suite.Suite
	store  *KVStore
	router *http.ServeMux
}

func (suite *KVTestSuite) SetupTest() {
	suite.store = NewKVStore()
	suite.router = setupRouter(suite.store)
}

func TestKVTestSuite(t *testing.T) {
	suite.Run(t, new(KVTestSuite))
}
