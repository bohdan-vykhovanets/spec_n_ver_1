package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type KVTestSuite struct {
	suite.Suite
	router *http.ServeMux
}

func (suite *KVTestSuite) SetupTest() {
	store = NewKVStore()
	suite.router = setupRouter()
}

func TestKVTestSuite(t *testing.T) {
	suite.Run(t, new(KVTestSuite))
}
