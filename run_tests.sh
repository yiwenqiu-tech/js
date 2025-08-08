#!/bin/bash

# Set test environment variables
export HUNYUAN_TOKEN=test_token
export WX_APPID=test_appid
export WX_APP_SECRET=test_secret
export WX_TEMPLATE_ID=test_template_id
export MYSQL_DSN=test_mysql_dsn

# Run all tests
echo "Running all tests with test environment variables..."
go test ./... -v

echo "Tests completed!" 