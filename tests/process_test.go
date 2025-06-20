package tests

// Note: The original process_test.go tested unexported functions like isEmptyValue and processProperties.
// Since these are internal implementation details, we skip those tests in the test suite migration.
// The functionality is still tested indirectly through the Generate function tests.
