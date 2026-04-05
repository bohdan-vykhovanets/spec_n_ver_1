package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func (suite *KVTestSuite) TestDeleteNonExistentKeyPanics() {
	suite.Panics(func() {
		suite.store.Delete("missing-key")
	}, "Deleting a non-existent key should panic")
}

func (suite *KVTestSuite) TestGetItemParameterized() {
	_ = suite.store.Set("key1", "value1")
	_ = suite.store.Set("key2", "value2")

	cases := []struct {
		name           string
		key            string
		expectedStatus int
		expectedBody   string
	}{
		{"Valid Key 1", "key1", http.StatusOK, `{"value":"value1"}`},
		{"Valid Key 2", "key2", http.StatusOK, `{"value":"value2"}`},
		{"Missing Key", "key3", http.StatusNotFound, "item not found\n"},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			req, _ := http.NewRequest("GET", "/item/"+tc.key, nil)
			rr := httptest.NewRecorder()
			suite.router.ServeHTTP(rr, req)

			suite.Equal(tc.expectedStatus, rr.Code)

			if tc.expectedStatus == http.StatusOK {
				suite.JSONEq(tc.expectedBody, rr.Body.String())
			} else {
				suite.Equal(tc.expectedBody, rr.Body.String())
			}
		})
	}
}

func (suite *KVTestSuite) TestCreateAndGetAllItems() {
	item := Item{Key: "user1", Value: "Alice"}
	body, _ := json.Marshal(item)

	req, _ := http.NewRequest("POST", "/item", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	suite.router.ServeHTTP(rr, req)

	suite.Equal(http.StatusCreated, rr.Code)

	suite.NotNil(rr.Body)
	suite.True(rr.Code >= 200 && rr.Code < 300)

	_ = suite.store.Set("user2", "Bob")

	allData, err := suite.store.GetAll()
	suite.NoError(err, "Getting all items should not result in an error")

	expectedSubset := map[string]string{
		"user1": "Alice",
		"user2": "Bob",
	}

	suite.Subset(allData, expectedSubset, "Collection should contain the subset map")

	suite.Len(allData, 2)
}

func (suite *KVTestSuite) TestDeleteThroughHTTP() {
	_ = suite.store.Set("temp", "data")

	req, _ := http.NewRequest("DELETE", "/item/temp", nil)
	rr := httptest.NewRecorder()
	suite.router.ServeHTTP(rr, req)

	suite.Equal(http.StatusNoContent, rr.Code)

	reqPanic, _ := http.NewRequest("DELETE", "/item/missing", nil)
	rrPanic := httptest.NewRecorder()
	suite.router.ServeHTTP(rrPanic, reqPanic)

	suite.Equal(http.StatusInternalServerError, rrPanic.Code)
	suite.Contains(rrPanic.Body.String(), "key 'missing' does not exist")
}
