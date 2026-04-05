package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Set(key, value string) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockStorage) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Delete(key string) {
	m.Called(key)
}

func (m *MockStorage) GetAll() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

type MockTestSuite struct {
	suite.Suite
	mockStore *MockStorage
	router    *http.ServeMux
}

func (suite *MockTestSuite) SetupTest() {
	suite.mockStore = new(MockStorage)
	suite.router = setupRouter(suite.mockStore)
}

func (suite *MockTestSuite) TestMockGeneratesException() {
	suite.mockStore.On("Delete", "crash-key").Panic("database connection lost")

	req, _ := http.NewRequest("DELETE", "/item/crash-key", nil)
	rr := httptest.NewRecorder()
	suite.router.ServeHTTP(rr, req)

	suite.Equal(http.StatusInternalServerError, rr.Code)
	suite.Contains(rr.Body.String(), "database connection lost")
	suite.mockStore.AssertExpectations(suite.T())
}

func (suite *MockTestSuite) TestOrderedCallsAndCount() {
	suite.mockStore.On("Get", "pop-key").Return("pop-value", nil).Once()
	suite.mockStore.On("Delete", "pop-key").Return().Once()

	req, _ := http.NewRequest("POST", "/item/pop/pop-key", nil)
	rr := httptest.NewRecorder()
	suite.router.ServeHTTP(rr, req)

	suite.Equal(http.StatusOK, rr.Code)

	suite.mockStore.AssertNumberOfCalls(suite.T(), "Get", 1)
	suite.mockStore.AssertNumberOfCalls(suite.T(), "Delete", 1)

	suite.Require().Len(suite.mockStore.Calls, 2)
	suite.Equal("Get", suite.mockStore.Calls[0].Method, "Get should be called first")
	suite.Equal("Delete", suite.mockStore.Calls[1].Method, "Delete should be called second")

	suite.mockStore.AssertExpectations(suite.T())
}

func (suite *MockTestSuite) TestParameterMatching() {
	validKeyMatcher := mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "user_")
	})

	suite.mockStore.On("Set", validKeyMatcher, mock.Anything).Return(nil)

	item := Item{Key: "user_123", Value: "John"}
	body, _ := json.Marshal(item)
	req, _ := http.NewRequest("POST", "/item", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	suite.router.ServeHTTP(rr, req)

	suite.Equal(http.StatusCreated, rr.Code)
	suite.mockStore.AssertCalled(suite.T(), "Set", "user_123", "John")
	suite.mockStore.AssertExpectations(suite.T())
}

func (suite *MockTestSuite) TestSubsequentCallsYieldDifferentAnswers() {
	suite.mockStore.On("Get", "status-key").Return("pending", nil).Once()
	suite.mockStore.On("Get", "status-key").Return("completed", nil).Once()
	suite.mockStore.On("Get", "status-key").Return("", errors.New("record expired")).Once()

	req1, _ := http.NewRequest("GET", "/item/status-key", nil)
	rr1 := httptest.NewRecorder()
	suite.router.ServeHTTP(rr1, req1)
	suite.JSONEq(`{"value":"pending"}`, rr1.Body.String())

	req2, _ := http.NewRequest("GET", "/item/status-key", nil)
	rr2 := httptest.NewRecorder()
	suite.router.ServeHTTP(rr2, req2)
	suite.JSONEq(`{"value":"completed"}`, rr2.Body.String())

	req3, _ := http.NewRequest("GET", "/item/status-key", nil)
	rr3 := httptest.NewRecorder()
	suite.router.ServeHTTP(rr3, req3)
	suite.Equal(http.StatusNotFound, rr3.Code)

	suite.mockStore.AssertExpectations(suite.T())
}

func TestMockTestSuite(t *testing.T) {
	suite.Run(t, new(MockTestSuite))
}
